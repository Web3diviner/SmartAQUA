"""LLM provider abstraction.

04_AQUADOC_RAG_LLM.md section 10: the application depends on `LLMProvider`,
never on a vendor SDK. 13_CODING_AND_ENGINEERING_STANDARDS.md: AI calls must be
isolated behind interfaces, and all external calls require explicit timeouts.
"""

from __future__ import annotations

from abc import ABC, abstractmethod
from dataclasses import dataclass, field
from typing import Any, Literal

Role = Literal["user", "assistant"]


@dataclass(frozen=True)
class LLMMessage:
    role: Role
    content: str


@dataclass(frozen=True)
class LLMRequest:
    """A provider-agnostic generation request."""

    system: str
    messages: list[LLMMessage]
    #: JSON Schema the response must satisfy. Providers that support native
    #: structured output enforce it; others must still return matching JSON.
    json_schema: dict[str, Any] | None = None
    max_tokens: int = 8000
    effort: str = "high"
    timeout_seconds: float = 120.0


@dataclass(frozen=True)
class LLMUsage:
    input_tokens: int | None = None
    output_tokens: int | None = None


@dataclass(frozen=True)
class LLMResponse:
    """A provider-agnostic generation result."""

    text: str
    #: Parsed JSON when the request carried a schema.
    parsed: dict[str, Any] | None = None
    model: str = ""
    provider: str = ""
    stop_reason: str | None = None
    usage: LLMUsage = field(default_factory=LLMUsage)
    latency_ms: float = 0.0


class LLMProvider(ABC):
    """Interface every language-model backend implements."""

    #: Stable identifier recorded in provenance.
    name: str = "abstract"

    @property
    @abstractmethod
    def model_id(self) -> str:
        """Model identifier recorded in provenance."""
        raise NotImplementedError

    @abstractmethod
    async def generate(self, request: LLMRequest) -> LLMResponse:
        """Generate a response.

        Raises:
            LLMProviderError: transport, auth, or upstream failure.
            LLMRefusalError: the provider's safety layer declined the request.
        """
        raise NotImplementedError

    async def aclose(self) -> None:
        """Release provider resources. Override when holding a client."""
        return None
