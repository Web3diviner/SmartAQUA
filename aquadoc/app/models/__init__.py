"""SQLAlchemy ORM models.

Mirrors migrations/0001_initial.sql. Schema changes require a migration file
plus a rollback (13_CODING_AND_ENGINEERING_STANDARDS.md, "Migrations").
"""

from __future__ import annotations

import uuid
from datetime import datetime
from typing import Any

from pgvector.sqlalchemy import Vector
from sqlalchemy import (
    ARRAY,
    Enum as SAEnum,
    ForeignKey,
    Integer,
    String,
    Text,
    UniqueConstraint,
    func,
)
from sqlalchemy.dialects.postgresql import JSONB, UUID as PGUUID
from sqlalchemy.orm import Mapped, mapped_column, relationship

from app.db import Base
from app.schemas.common import EvidenceLevel, ReviewStatus

#: Must match Settings.embedding_dimensions and the vector(N) in the migration.
EMBEDDING_DIMENSIONS = 1024


def _uuid_pk() -> Mapped[uuid.UUID]:
    return mapped_column(PGUUID(as_uuid=True), primary_key=True, default=uuid.uuid4)


class KnowledgeDocument(Base):
    __tablename__ = "knowledge_documents"
    __table_args__ = (UniqueConstraint("checksum", "version", name="knowledge_documents_checksum_version_key"),)

    id: Mapped[uuid.UUID] = _uuid_pk()
    title: Mapped[str] = mapped_column(Text, nullable=False)
    source: Mapped[str] = mapped_column(Text, nullable=False)
    author: Mapped[str | None] = mapped_column(Text)
    year: Mapped[int | None] = mapped_column(Integer)
    document_type: Mapped[str] = mapped_column(Text, nullable=False)
    species: Mapped[list[str]] = mapped_column(ARRAY(Text), nullable=False, default=list)
    life_stage: Mapped[list[str]] = mapped_column(ARRAY(Text), nullable=False, default=list)
    topic: Mapped[list[str]] = mapped_column(ARRAY(Text), nullable=False, default=list)
    disease: Mapped[list[str]] = mapped_column(ARRAY(Text), nullable=False, default=list)
    region: Mapped[list[str]] = mapped_column(ARRAY(Text), nullable=False, default=list)
    evidence_level: Mapped[EvidenceLevel] = mapped_column(
        SAEnum(EvidenceLevel, name="knowledge_evidence_level", values_callable=lambda e: [m.value for m in e]),
        nullable=False,
    )
    review_status: Mapped[ReviewStatus] = mapped_column(
        SAEnum(ReviewStatus, name="knowledge_review_status", values_callable=lambda e: [m.value for m in e]),
        nullable=False,
        default=ReviewStatus.PENDING,
    )
    owner: Mapped[str | None] = mapped_column(Text)
    file_url: Mapped[str | None] = mapped_column(Text)
    checksum: Mapped[str] = mapped_column(Text, nullable=False)
    version: Mapped[int] = mapped_column(Integer, nullable=False, default=1)
    chunk_count: Mapped[int] = mapped_column(Integer, nullable=False, default=0)
    ingest_status: Mapped[str] = mapped_column(Text, nullable=False, default="pending")
    ingest_error: Mapped[str | None] = mapped_column(Text)
    ingested_at: Mapped[datetime | None] = mapped_column()
    reviewed_at: Mapped[datetime | None] = mapped_column()
    reviewed_by: Mapped[str | None] = mapped_column(Text)
    review_note: Mapped[str | None] = mapped_column(Text)
    created_at: Mapped[datetime] = mapped_column(server_default=func.now(), nullable=False)
    updated_at: Mapped[datetime] = mapped_column(
        server_default=func.now(), onupdate=func.now(), nullable=False
    )

    chunks: Mapped[list[KnowledgeChunk]] = relationship(
        back_populates="document", cascade="all, delete-orphan", lazy="selectin"
    )


class KnowledgeChunk(Base):
    __tablename__ = "knowledge_chunks"
    __table_args__ = (
        UniqueConstraint("document_id", "chunk_index", name="knowledge_chunks_document_index_key"),
    )

    id: Mapped[uuid.UUID] = _uuid_pk()
    document_id: Mapped[uuid.UUID] = mapped_column(
        PGUUID(as_uuid=True), ForeignKey("knowledge_documents.id", ondelete="CASCADE"), nullable=False
    )
    chunk_index: Mapped[int] = mapped_column(Integer, nullable=False)
    content: Mapped[str] = mapped_column(Text, nullable=False)
    token_estimate: Mapped[int] = mapped_column(Integer, nullable=False, default=0)
    page_number: Mapped[int | None] = mapped_column(Integer)
    section: Mapped[str | None] = mapped_column(Text)
    metadata_json: Mapped[dict[str, Any]] = mapped_column(JSONB, nullable=False, default=dict)
    embedding: Mapped[list[float] | None] = mapped_column(Vector(EMBEDDING_DIMENSIONS))
    embedding_model: Mapped[str | None] = mapped_column(Text)
    created_at: Mapped[datetime] = mapped_column(server_default=func.now(), nullable=False)

    document: Mapped[KnowledgeDocument] = relationship(back_populates="chunks")


class AquaDocConversation(Base):
    __tablename__ = "aquadoc_conversations"

    id: Mapped[uuid.UUID] = _uuid_pk()
    user_id: Mapped[str] = mapped_column(Text, nullable=False)
    farm_id: Mapped[str | None] = mapped_column(Text)
    pond_id: Mapped[str | None] = mapped_column(Text)
    production_cycle_id: Mapped[str | None] = mapped_column(Text)
    title: Mapped[str | None] = mapped_column(Text)
    created_at: Mapped[datetime] = mapped_column(server_default=func.now(), nullable=False)
    updated_at: Mapped[datetime] = mapped_column(
        server_default=func.now(), onupdate=func.now(), nullable=False
    )

    messages: Mapped[list[AquaDocMessage]] = relationship(
        back_populates="conversation", cascade="all, delete-orphan", lazy="selectin"
    )


class AquaDocMessage(Base):
    __tablename__ = "aquadoc_messages"

    id: Mapped[uuid.UUID] = _uuid_pk()
    conversation_id: Mapped[uuid.UUID] = mapped_column(
        PGUUID(as_uuid=True), ForeignKey("aquadoc_conversations.id", ondelete="CASCADE"), nullable=False
    )
    role: Mapped[str] = mapped_column(
        SAEnum("user", "assistant", "system", name="aquadoc_message_role"), nullable=False
    )
    content: Mapped[str] = mapped_column(Text, nullable=False)
    structured_payload_json: Mapped[dict[str, Any] | None] = mapped_column(JSONB)
    request_id: Mapped[str | None] = mapped_column(Text)
    created_at: Mapped[datetime] = mapped_column(server_default=func.now(), nullable=False)

    conversation: Mapped[AquaDocConversation] = relationship(back_populates="messages")


class AquaDocRetrievalTrace(Base):
    """Persisted so GET /dev/v1/debug/retrieval/{request_id} can replay it."""

    __tablename__ = "aquadoc_retrieval_traces"

    request_id: Mapped[str] = mapped_column(String(128), primary_key=True)
    conversation_id: Mapped[uuid.UUID | None] = mapped_column(
        PGUUID(as_uuid=True), ForeignKey("aquadoc_conversations.id", ondelete="SET NULL")
    )
    question: Mapped[str] = mapped_column(Text, nullable=False)
    intent: Mapped[str] = mapped_column(Text, nullable=False)
    trace_json: Mapped[dict[str, Any]] = mapped_column(JSONB, nullable=False)
    created_at: Mapped[datetime] = mapped_column(server_default=func.now(), nullable=False)


class AquaDocAuditLog(Base):
    __tablename__ = "aquadoc_audit_logs"

    id: Mapped[uuid.UUID] = _uuid_pk()
    actor_type: Mapped[str] = mapped_column(Text, nullable=False)
    actor_id: Mapped[str | None] = mapped_column(Text)
    action: Mapped[str] = mapped_column(Text, nullable=False)
    resource_type: Mapped[str] = mapped_column(Text, nullable=False)
    resource_id: Mapped[str | None] = mapped_column(Text)
    before_json: Mapped[dict[str, Any] | None] = mapped_column(JSONB)
    after_json: Mapped[dict[str, Any] | None] = mapped_column(JSONB)
    request_id: Mapped[str | None] = mapped_column(Text)
    user_agent: Mapped[str | None] = mapped_column(Text)
    created_at: Mapped[datetime] = mapped_column(server_default=func.now(), nullable=False)
