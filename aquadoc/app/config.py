"""Application configuration.

All configuration is read from the environment (13_CODING_AND_ENGINEERING_STANDARDS.md:
"configuration through environment/config files", "no secrets in code").
"""

from __future__ import annotations

from functools import lru_cache
from typing import Literal

from pydantic import Field, field_validator, model_validator
from pydantic_settings import BaseSettings, SettingsConfigDict

AppEnv = Literal["development", "staging", "production"]
LLMProviderName = Literal["claude", "groq", "echo"]
EmbeddingProviderName = Literal["hashing", "voyage"]
EffortLevel = Literal["low", "medium", "high", "xhigh", "max"]


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        extra="ignore",
        case_sensitive=False,
    )

    # -- Application ---------------------------------------------------------
    service_name: str = "aquadoc"
    service_version: str = "0.1.0"
    app_env: AppEnv = "development"
    log_level: str = "INFO"
    host: str = "0.0.0.0"  # noqa: S104 - bound inside a container/private network
    port: int = 8001
    cors_allow_origins: str = "http://localhost:5173"

    # -- Database ------------------------------------------------------------
    database_url: str = "postgresql+asyncpg://aquadoc:aquadoc@localhost:5432/aquadoc"
    database_pool_size: int = 5
    database_max_overflow: int = 5

    # -- Authentication ------------------------------------------------------
    aquadoc_internal_service_secret: str = ""
    aquadoc_dev_token: str = ""

    # -- LLM provider --------------------------------------------------------
    llm_provider: LLMProviderName = "groq"
    llm_model: str = "openai/gpt-oss-120b"
    llm_max_tokens: int = 2000
    llm_effort: EffortLevel = "high"
    llm_timeout_seconds: float = 120.0
    llm_enable_refusal_fallback: bool = True
    anthropic_api_key: str = ""
    groq_api_key: str = ""
    groq_base_url: str = "https://api.groq.com/openai/v1"

    # -- Embedding provider --------------------------------------------------
    embedding_provider: EmbeddingProviderName = "hashing"
    embedding_dimensions: int = 1024
    embedding_timeout_seconds: float = 60.0
    voyage_model: str = "voyage-3"
    voyage_api_key: str = ""

    # -- Retrieval -----------------------------------------------------------
    retrieval_candidates: int = Field(default=40, ge=1, le=200)
    retrieval_top_k: int = Field(default=6, ge=1, le=50)
    retrieval_min_similarity: float = Field(default=0.15, ge=0.0, le=1.0)
    retrieval_enable_lexical: bool = True

    # -- Ingestion -----------------------------------------------------------
    chunk_target_tokens: int = Field(default=750, ge=100, le=4000)
    chunk_overlap_tokens: int = Field(default=150, ge=0, le=1000)
    max_upload_bytes: int = Field(default=25 * 1024 * 1024, ge=1024)

    @property
    def is_production(self) -> bool:
        return self.app_env == "production"

    @property
    def dev_routes_enabled(self) -> bool:
        """Dev routes exist only in development, and only with a token configured.

        15_AQUADOC_FRONTEND.md section 9: the development token must be
        "impossible to use in production".
        """
        return self.app_env == "development" and bool(self.aquadoc_dev_token)

    #: Alias used at router-mount time. Same rule, read at a different layer.
    @property
    def enable_dev_api(self) -> bool:
        return self.dev_routes_enabled

    @property
    def cors_origins(self) -> list[str]:
        return [origin.strip() for origin in self.cors_allow_origins.split(",") if origin.strip()]

    @field_validator("port", mode="before")
    @classmethod
    def _parse_port(cls, value: object) -> int:
        if value is None or str(value).strip() == "":
            return 8001
        try:
            return int(value)
        except (ValueError, TypeError):
            return 8001

    @field_validator("database_url", mode="before")
    @classmethod
    def _sanitize_database_url(cls, value: str | None) -> str:
        if not value or not str(value).strip():
            return "postgresql+asyncpg://aquadoc:aquadoc@localhost:5432/aquadoc"
        url = str(value).strip().strip('"\'')
        # Automatically fix standard postgresql:// or postgres:// to asyncpg
        if url.startswith("postgres://"):
            url = "postgresql+asyncpg://" + url[len("postgres://"):]
        elif url.startswith("postgresql://") and not url.startswith("postgresql+asyncpg://"):
            url = "postgresql+asyncpg://" + url[len("postgresql://"):]
        
        # Remove empty port like @hostname:/database -> @hostname/database (in host section only)
        if "://" in url:
            scheme, rest = url.split("://", 1)
            import re
            rest = re.sub(r':(?=/|$|\?)', '', rest)
            url = f"{scheme}://{rest}"
        return url

    @field_validator("chunk_overlap_tokens")
    @classmethod
    def _overlap_below_target(cls, value: int, info) -> int:  # type: ignore[no-untyped-def]
        target = info.data.get("chunk_target_tokens", 750)
        if value >= target:
            raise ValueError("chunk_overlap_tokens must be smaller than chunk_target_tokens")
        return value

    @model_validator(mode="after")
    def _validate_production_gates(self) -> Settings:
        """Fail fast rather than start a production service in an unsafe shape.

        12_PRODUCTION_READINESS_CHECKLIST.md / 07_SECURITY_ARCHITECTURE.md section 12:
        no release with default credentials or dev auth reachable in production.
        """
        if not self.is_production:
            return self

        problems: list[str] = []
        if self.aquadoc_dev_token:
            problems.append("AQUADOC_DEV_TOKEN must be empty when APP_ENV=production")
        if not self.aquadoc_internal_service_secret:
            problems.append("AQUADOC_INTERNAL_SERVICE_SECRET is required in production")
        if len(self.aquadoc_internal_service_secret) < 32:
            problems.append("AQUADOC_INTERNAL_SERVICE_SECRET must be at least 32 characters")
        if self.llm_provider == "echo":
            problems.append("LLM_PROVIDER=echo is a development stub and cannot run in production")
        if self.embedding_provider == "hashing":
            problems.append(
                "EMBEDDING_PROVIDER=hashing is a development stub and cannot run in production"
            )
        if self.llm_provider == "claude" and not self.anthropic_api_key:
            problems.append("ANTHROPIC_API_KEY is required when LLM_PROVIDER=claude")
        if self.llm_provider == "groq" and not self.groq_api_key:
            problems.append("GROQ_API_KEY is required when LLM_PROVIDER=groq")
        if self.embedding_provider == "voyage" and not self.voyage_api_key:
            problems.append("VOYAGE_API_KEY is required when EMBEDDING_PROVIDER=voyage")

        if problems:
            raise ValueError("Unsafe production configuration: " + "; ".join(problems))
        return self


@lru_cache(maxsize=1)
def get_settings() -> Settings:
    return Settings()
