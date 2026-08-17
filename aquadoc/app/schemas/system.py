"""System-level schemas: health, readiness, service metadata."""

from __future__ import annotations

from pydantic import BaseModel, ConfigDict, Field


class HealthResponse(BaseModel):
    """Liveness. Reports identity only — never configuration."""

    model_config = ConfigDict(extra="forbid")

    status: str
    service: str
    version: str
    environment: str


class ComponentHealth(BaseModel):
    model_config = ConfigDict(extra="forbid")

    name: str
    status: str = Field(description="ok | error")
    detail: str | None = None
    latency_ms: float | None = None


class ReadinessResponse(HealthResponse):
    model_config = ConfigDict(extra="forbid")

    ready: bool
    components: list[ComponentHealth] = Field(default_factory=list)


class ConfigResponse(BaseModel):
    """The knobs that change answers, exposed to the developer console.

    04_AQUADOC_RAG_LLM.md section 6 says to tune chunking and retrieval against
    evaluation rather than fixing them permanently — which requires the values
    actually in effect to be visible. Contains no secrets: provider names and
    model IDs only, never keys or connection strings.
    """

    model_config = ConfigDict(extra="forbid")

    environment: str
    llm_provider: str
    llm_model: str
    llm_effort: str
    embedding_provider: str
    embedding_model: str
    embedding_dimensions: int
    retrieval_candidates: int
    retrieval_top_k: int
    retrieval_min_similarity: float
    retrieval_lexical_enabled: bool
    chunk_target_tokens: int
    chunk_overlap_tokens: int
    rules_version: str
    prompt_versions: dict[str, str] = Field(default_factory=dict)
    water_quality_parameters: list[str] = Field(default_factory=list)
