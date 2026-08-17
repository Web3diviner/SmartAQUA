"""Chat request/response contract.

This is the contract that must stay stable so the temporary React frontend can
later be replaced by the Flutter integration without rebuilding RAG or the LLM
layer (15_AQUADOC_FRONTEND.md section 1 and section 19).
"""

from __future__ import annotations

from datetime import UTC, datetime
from typing import Any

from pydantic import BaseModel, ConfigDict, Field

from app.schemas.common import (
    ConfidenceBand,
    EvidenceLevel,
    Intent,
    RecommendationTier,
    RiskLevel,
)
from app.schemas.farm_context import FarmContext


class ChatRequest(BaseModel):
    """05_API_AND_SERVICE_CONTRACTS.md, "Chat Request"."""

    model_config = ConfigDict(extra="forbid")

    request_id: str | None = Field(
        default=None,
        max_length=128,
        description="Caller-supplied correlation ID. Generated if absent.",
    )
    user_id: str = Field(min_length=1, max_length=128)
    conversation_id: str | None = None
    question: str = Field(min_length=1, max_length=4000)
    farm_context: FarmContext | None = None
    # Free-form metadata filters, e.g. {"species": ["Clarias gariepinus"]}.
    filters: dict[str, list[str]] = Field(default_factory=dict)


class SourceReference(BaseModel):
    """A retrieved passage, presented as a citation.

    15_AQUADOC_FRONTEND.md section 12: every grounded answer displays supporting
    sources. `page` is nullable because not every document type is paginated.
    """

    model_config = ConfigDict(extra="forbid")

    chunk_id: str
    document_id: str
    title: str
    source: str
    author: str | None = None
    year: int | None = None
    page: int | None = None
    section: str | None = None
    evidence_level: EvidenceLevel
    excerpt: str
    score: float = Field(ge=0.0, le=1.0)
    # Only populated in developer mode — the full retrieved chunk.
    chunk_text: str | None = None


class PossibleCause(BaseModel):
    model_config = ConfigDict(extra="forbid")

    name: str
    confidence: float = Field(ge=0.0, le=1.0)
    explanation: str | None = None
    supporting_source_ids: list[str] = Field(default_factory=list)


class RecommendedAction(BaseModel):
    """A proposal, never a device command.

    Non-negotiable design rule (00_README.md): AquaDoc produces recommendations;
    the platform produces commands.
    """

    model_config = ConfigDict(extra="forbid")

    action: str
    tier: RecommendationTier
    reason: str
    requires_approval: bool
    urgency: RiskLevel = RiskLevel.INFORMATIONAL


class RuleFinding(BaseModel):
    """Output of a deterministic rule, evaluated without the LLM.

    Deterministic logic must be testable without AI
    (13_CODING_AND_ENGINEERING_STANDARDS.md).
    """

    model_config = ConfigDict(extra="forbid")

    rule_id: str
    rule_version: str
    status: str = Field(description="ok | watch | concern | unknown")
    summary: str
    measurement: str | None = None
    observed: float | None = None
    expected_range: tuple[float, float] | None = None


class RetrievalTraceItem(BaseModel):
    """One retrieval candidate, with the scores that ranked it."""

    model_config = ConfigDict(extra="forbid")

    chunk_id: str
    document_id: str
    title: str
    page: int | None = None
    section: str | None = None
    evidence_level: EvidenceLevel
    similarity: float
    lexical_rank: int | None = None
    vector_rank: int | None = None
    fused_score: float
    final_score: float
    selected: bool
    content_preview: str


class RetrievalTrace(BaseModel):
    """Everything the Retrieval Inspector needs (15_AQUADOC_FRONTEND.md section 4)."""

    model_config = ConfigDict(extra="forbid")

    request_id: str
    question: str
    intent: Intent
    metadata_filters: dict[str, list[str]] = Field(default_factory=dict)
    embedding_model: str
    embedding_dimensions: int
    candidates_considered: int
    selected_count: int
    lexical_enabled: bool
    min_similarity: float
    items: list[RetrievalTraceItem] = Field(default_factory=list)
    retrieval_latency_ms: float = 0.0


class Provenance(BaseModel):
    """Recorded for every production response.

    14_AQUADOC_SAFETY_AND_GOVERNANCE.md section 7 requires input context,
    retrieved sources, rule versions, model versions, prompt version, timestamp
    and confidence to be traceable.
    """

    model_config = ConfigDict(extra="forbid")

    prompt_version: str
    llm_model: str
    llm_provider: str
    embedding_model: str
    embedding_provider: str
    rules_version: str
    retrieval_source_ids: list[str] = Field(default_factory=list)
    farm_context_supplied: bool = False
    farm_context_completeness: float = 0.0
    generated_at: datetime = Field(default_factory=lambda: datetime.now(UTC))
    llm_latency_ms: float = 0.0
    total_latency_ms: float = 0.0
    input_tokens: int | None = None
    output_tokens: int | None = None


class ChatResponse(BaseModel):
    """04_AQUADOC_RAG_LLM.md section 12, "Structured Responses".

    Free-form text alone is not enough — every field below exists so the client
    can render uncertainty, missing data, and provenance rather than presenting
    a bare paragraph as fact.
    """

    model_config = ConfigDict(extra="forbid")

    request_id: str
    conversation_id: str
    answer: str
    intent: Intent
    risk_level: RiskLevel
    confidence: float = Field(ge=0.0, le=1.0)
    confidence_band: ConfidenceBand
    possible_causes: list[PossibleCause] = Field(default_factory=list)
    recommended_actions: list[RecommendedAction] = Field(default_factory=list)
    #: Measurement keys AquaDoc could not evaluate. Rendered as "Not available",
    #: never as 0 (15_AQUADOC_FRONTEND.md section 14).
    missing_data: list[str] = Field(default_factory=list)
    missing_data_labels: list[str] = Field(default_factory=list)
    expert_escalation: bool = False
    escalation_reasons: list[str] = Field(default_factory=list)
    sources: list[SourceReference] = Field(default_factory=list)
    rule_findings: list[RuleFinding] = Field(default_factory=list)
    warnings: list[str] = Field(default_factory=list)
    provenance: Provenance
    #: Populated only for developer callers.
    retrieval_trace: RetrievalTrace | None = None
    #: Per-component confidence inputs. Developer mode only — farmers see the
    #: band, not the arithmetic (15_AQUADOC_FRONTEND.md section 13).
    confidence_breakdown: dict[str, float | None] | None = None


class KnowledgeSearchRequest(BaseModel):
    """POST /internal/v1/knowledge/search"""

    model_config = ConfigDict(extra="forbid")

    query: str = Field(min_length=1, max_length=2000)
    top_k: int = Field(default=6, ge=1, le=50)
    filters: dict[str, list[str]] = Field(default_factory=dict)


class KnowledgeSearchResponse(BaseModel):
    model_config = ConfigDict(extra="forbid")

    request_id: str
    query: str
    results: list[SourceReference] = Field(default_factory=list)
    embedding_model: str


class ErrorBody(BaseModel):
    model_config = ConfigDict(extra="forbid")

    code: str
    message: str
    request_id: str
    details: dict[str, Any] = Field(default_factory=dict)


class ErrorResponse(BaseModel):
    """05_API_AND_SERVICE_CONTRACTS.md, "Error Format"."""

    model_config = ConfigDict(extra="forbid")

    error: ErrorBody
