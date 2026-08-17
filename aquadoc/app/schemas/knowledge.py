"""Knowledge-document administration contract.

Backs the Knowledge Base screen (15_AQUADOC_FRONTEND.md section 4) and the
governance requirements in 14_AQUADOC_SAFETY_AND_GOVERNANCE.md section 9.
"""

from __future__ import annotations

from datetime import datetime

from pydantic import BaseModel, ConfigDict, Field

from app.schemas.common import EvidenceLevel, ReviewStatus


class DocumentMetadata(BaseModel):
    """Metadata required for every ingested document (04_AQUADOC_RAG_LLM.md section 4)."""

    model_config = ConfigDict(extra="forbid")

    title: str = Field(min_length=1, max_length=500)
    source: str = Field(min_length=1, max_length=500)
    author: str | None = Field(default=None, max_length=300)
    year: int | None = Field(default=None, ge=1800, le=2200)
    document_type: str = Field(
        min_length=1,
        max_length=100,
        description="guideline | research_paper | manual | expert_case | user_report | other",
    )
    species: list[str] = Field(default_factory=list)
    life_stage: list[str] = Field(default_factory=list)
    topic: list[str] = Field(default_factory=list)
    disease: list[str] = Field(default_factory=list)
    region: list[str] = Field(default_factory=list)
    evidence_level: EvidenceLevel
    owner: str | None = Field(default=None, max_length=200)


class DocumentSummary(BaseModel):
    model_config = ConfigDict(extra="forbid")

    id: str
    title: str
    source: str
    author: str | None = None
    year: int | None = None
    document_type: str
    species: list[str] = Field(default_factory=list)
    life_stage: list[str] = Field(default_factory=list)
    topic: list[str] = Field(default_factory=list)
    disease: list[str] = Field(default_factory=list)
    region: list[str] = Field(default_factory=list)
    evidence_level: EvidenceLevel
    review_status: ReviewStatus
    owner: str | None = None
    checksum: str
    version: int
    chunk_count: int
    ingest_status: str
    ingest_error: str | None = None
    ingested_at: datetime | None = None
    reviewed_at: datetime | None = None
    reviewed_by: str | None = None
    review_note: str | None = None
    created_at: datetime
    updated_at: datetime


class DocumentListResponse(BaseModel):
    model_config = ConfigDict(extra="forbid")

    documents: list[DocumentSummary] = Field(default_factory=list)
    total: int
    limit: int = 50
    offset: int = 0


class ReviewDecisionRequest(BaseModel):
    model_config = ConfigDict(extra="forbid")

    note: str | None = Field(default=None, max_length=2000)


class IngestResult(BaseModel):
    model_config = ConfigDict(extra="forbid")

    document_id: str
    title: str
    checksum: str
    chunk_count: int
    embedded_chunks: int
    review_status: ReviewStatus
    warnings: list[str] = Field(default_factory=list)
