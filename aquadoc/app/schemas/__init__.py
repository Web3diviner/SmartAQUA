"""Pydantic request/response schemas.

13_CODING_AND_ENGINEERING_STANDARDS.md: use Pydantic schemas for requests,
responses, recommendation structures, and provider payloads; avoid passing
unstructured dictionaries across the codebase.
"""

from app.schemas.chat import (
    ChatRequest,
    ChatResponse,
    ErrorBody,
    ErrorResponse,
    KnowledgeSearchRequest,
    KnowledgeSearchResponse,
    PossibleCause,
    Provenance,
    RecommendedAction,
    RetrievalTrace,
    RetrievalTraceItem,
    RuleFinding,
    SourceReference,
)
from app.schemas.common import (
    ConfidenceBand,
    EvidenceLevel,
    Intent,
    MeasurementKey,
    RecommendationTier,
    ReviewStatus,
    RiskLevel,
    confidence_band,
)
from app.schemas.farm_context import (
    FarmContext,
    FeedingContext,
    HealthContext,
    WaterQuality,
)
from app.schemas.knowledge import (
    DocumentListResponse,
    DocumentMetadata,
    DocumentSummary,
    IngestResult,
    ReviewDecisionRequest,
)

__all__ = [
    "ChatRequest",
    "ChatResponse",
    "ConfidenceBand",
    "DocumentListResponse",
    "DocumentMetadata",
    "DocumentSummary",
    "ErrorBody",
    "ErrorResponse",
    "EvidenceLevel",
    "FarmContext",
    "FeedingContext",
    "HealthContext",
    "IngestResult",
    "Intent",
    "KnowledgeSearchRequest",
    "KnowledgeSearchResponse",
    "MeasurementKey",
    "PossibleCause",
    "Provenance",
    "RecommendationTier",
    "RecommendedAction",
    "RetrievalTrace",
    "RetrievalTraceItem",
    "ReviewDecisionRequest",
    "ReviewStatus",
    "RiskLevel",
    "RuleFinding",
    "SourceReference",
    "WaterQuality",
    "confidence_band",
]
