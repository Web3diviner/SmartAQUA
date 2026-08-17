"""Developer API — backs the temporary React client and nothing else.

15_AQUADOC_FRONTEND.md section 7 defines this surface, and section 19 is
explicit that it is temporary: the Flutter app will consume `/internal/v1`
through the Go backend instead. Keeping developer tooling on its own prefix
means removing it later is a route-file deletion, not a refactor.

Every route requires the development token, which the settings validator
prevents from existing in production. Retrieval traces, prompt internals, and
raw chunk text are developer tools; a farmer-facing caller never receives them.
"""

from __future__ import annotations

import logging
from datetime import UTC, datetime
from typing import Annotated
from uuid import UUID

from fastapi import APIRouter, Depends, File, Form, Query, UploadFile, status
from sqlalchemy import func, select

from app.api.deps import (
    DatabaseDep,
    DevCallerDep,
    OrchestratorDep,
    SettingsDep,
    get_ingestion_service,
)
from app.db import Database
from app.errors import DocumentNotFoundError
from app.models import KnowledgeChunk, KnowledgeDocument
from app.prompts import PROMPT_VERSIONS
from app.rules import RULES_VERSION
from app.rules.water_quality import THRESHOLDS
from app.schemas.chat import ChatRequest, ChatResponse, KnowledgeSearchRequest, RetrievalTrace
from app.schemas.common import EvidenceLevel, ReviewStatus
from app.schemas.knowledge import (
    DocumentListResponse,
    DocumentMetadata,
    DocumentSummary,
    IngestResult,
    ReviewDecisionRequest,
)
from app.schemas.system import ConfigResponse
from ingestion.loader import load_bytes
from ingestion.service import IngestionService

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/dev/v1", tags=["developer"])

IngestionDep = Annotated[IngestionService, Depends(get_ingestion_service)]


# -- chat ---------------------------------------------------------------------


@router.post(
    "/chat",
    response_model=ChatResponse,
    summary="Chat with developer diagnostics attached",
)
async def dev_chat(
    payload: ChatRequest,
    caller: DevCallerDep,
    orchestrator: OrchestratorDep,
) -> ChatResponse:
    """Same pipeline as `/internal/v1/aquadoc/chat`, with internals attached.

    Deliberately the identical orchestrator call rather than a parallel debug
    path — diagnostics from a different code path would be worse than none.
    `retrieval_trace` and `confidence_breakdown` are populated here and omitted
    for service callers.
    """
    outcome = await orchestrator.chat(payload, include_debug=True)
    return outcome.response


@router.get(
    "/debug/retrieval/{request_id}",
    response_model=RetrievalTrace,
    summary="Replay a stored retrieval trace",
)
async def get_retrieval_trace(
    request_id: str,
    caller: DevCallerDep,
    orchestrator: OrchestratorDep,
) -> RetrievalTrace:
    """Back the Retrieval Inspector (15_AQUADOC_FRONTEND.md section 4)."""
    return await orchestrator.get_retrieval_trace(request_id)


@router.post(
    "/knowledge/search",
    summary="Retrieval without generation",
)
async def dev_knowledge_search(
    payload: KnowledgeSearchRequest,
    caller: DevCallerDep,
    orchestrator: OrchestratorDep,
) -> dict:
    """Evaluate retrieval on its own, with the full trace attached."""
    references, trace = await orchestrator.search_knowledge(
        query=payload.query,
        top_k=payload.top_k,
        filters=payload.filters,
        include_full_text=True,
    )
    return {
        "request_id": trace.request_id,
        "query": payload.query,
        "results": [reference.model_dump(mode="json") for reference in references],
        "trace": trace.model_dump(mode="json"),
    }


# -- knowledge base -----------------------------------------------------------


@router.post(
    "/knowledge/documents",
    response_model=IngestResult,
    status_code=status.HTTP_201_CREATED,
    summary="Upload and ingest a knowledge document",
)
async def upload_document(
    caller: DevCallerDep,
    database: DatabaseDep,
    settings: SettingsDep,
    ingestion: IngestionDep,
    file: Annotated[UploadFile, File(description="PDF, TXT, or Markdown.")],
    title: Annotated[str, Form()],
    source: Annotated[str, Form()],
    document_type: Annotated[str, Form()],
    evidence_level: Annotated[EvidenceLevel, Form()],
    author: Annotated[str | None, Form()] = None,
    year: Annotated[int | None, Form()] = None,
    owner: Annotated[str | None, Form()] = None,
    species: Annotated[str, Form()] = "",
    life_stage: Annotated[str, Form()] = "",
    topic: Annotated[str, Form()] = "",
    disease: Annotated[str, Form()] = "",
    region: Annotated[str, Form()] = "",
    replace_existing: Annotated[bool, Form()] = False,
) -> IngestResult:
    """Ingest a document into `pending` review status.

    Ingesting is not approving: the document is parsed, chunked, embedded and
    stored, but stays out of retrieval until a human approves it
    (14_AQUADOC_SAFETY_AND_GOVERNANCE.md section 9).

    List-valued metadata arrives comma-separated because multipart form data
    has no native list type.
    """
    content = await file.read()
    document = load_bytes(
        content,
        filename=file.filename or "upload",
        max_bytes=settings.max_upload_bytes,
    )

    metadata = DocumentMetadata(
        title=title,
        source=source,
        author=author,
        year=year,
        document_type=document_type,
        species=_split_csv(species),
        life_stage=_split_csv(life_stage),
        topic=_split_csv(topic),
        disease=_split_csv(disease),
        region=_split_csv(region),
        evidence_level=evidence_level,
        owner=owner or caller.subject,
    )

    async with database.session() as session:
        result = await ingestion.ingest(
            session,
            document=document,
            metadata=metadata,
            replace_existing=replace_existing,
        )

    logger.info(
        "document_uploaded",
        extra={"document_id": result.document_id, "actor": caller.subject},
    )
    return result


@router.get(
    "/knowledge/documents",
    response_model=DocumentListResponse,
    summary="List knowledge documents",
)
async def list_documents(
    caller: DevCallerDep,
    database: DatabaseDep,
    review_status: Annotated[ReviewStatus | None, Query()] = None,
    species: Annotated[str | None, Query()] = None,
    topic: Annotated[str | None, Query()] = None,
    limit: Annotated[int, Query(ge=1, le=200)] = 50,
    offset: Annotated[int, Query(ge=0)] = 0,
) -> DocumentListResponse:
    """Back the Knowledge Base screen (15_AQUADOC_FRONTEND.md section 4)."""
    async with database.session() as session:
        query = select(KnowledgeDocument)
        count_query = select(func.count()).select_from(KnowledgeDocument)

        for condition in _list_conditions(review_status, species, topic):
            query = query.where(condition)
            count_query = count_query.where(condition)

        total = (await session.execute(count_query)).scalar_one()
        rows = (
            (
                await session.execute(
                    query.order_by(KnowledgeDocument.created_at.desc())
                    .limit(limit)
                    .offset(offset)
                )
            )
            .scalars()
            .all()
        )
        chunk_counts = await _chunk_counts(session, [row.id for row in rows])

    return DocumentListResponse(
        total=total,
        limit=limit,
        offset=offset,
        documents=[_to_summary(row, chunk_counts.get(row.id, 0)) for row in rows],
    )


@router.get(
    "/knowledge/documents/{document_id}",
    response_model=DocumentSummary,
    summary="Inspect one knowledge document",
)
async def get_document(
    document_id: UUID,
    caller: DevCallerDep,
    database: DatabaseDep,
) -> DocumentSummary:
    async with database.session() as session:
        row = await _load_document(session, document_id)
        counts = await _chunk_counts(session, [row.id])
        return _to_summary(row, counts.get(row.id, 0))


@router.post(
    "/knowledge/documents/{document_id}/approve",
    response_model=DocumentSummary,
    summary="Approve a document for production retrieval",
)
async def approve_document(
    document_id: UUID,
    caller: DevCallerDep,
    database: DatabaseDep,
    decision: ReviewDecisionRequest | None = None,
) -> DocumentSummary:
    """Move a document to `approved`, making it retrievable.

    This is the gate that lets content reach farmers, so the reviewer and
    timestamp are recorded — approval must be attributable
    (14_AQUADOC_SAFETY_AND_GOVERNANCE.md section 9).
    """
    return await _set_review_status(
        database, document_id, ReviewStatus.APPROVED, caller.subject, decision
    )


@router.post(
    "/knowledge/documents/{document_id}/reject",
    response_model=DocumentSummary,
    summary="Reject a document",
)
async def reject_document(
    document_id: UUID,
    caller: DevCallerDep,
    database: DatabaseDep,
    decision: ReviewDecisionRequest | None = None,
) -> DocumentSummary:
    return await _set_review_status(
        database, document_id, ReviewStatus.REJECTED, caller.subject, decision
    )


@router.post(
    "/knowledge/documents/{document_id}/deprecate",
    response_model=DocumentSummary,
    summary="Deprecate a previously approved document",
)
async def deprecate_document(
    document_id: UUID,
    caller: DevCallerDep,
    database: DatabaseDep,
    decision: ReviewDecisionRequest | None = None,
) -> DocumentSummary:
    """Withdraw a document from retrieval without deleting it.

    Deprecated documents stop being retrieved but stay stored, so any past
    answer that cited them remains auditable.
    """
    return await _set_review_status(
        database, document_id, ReviewStatus.DEPRECATED, caller.subject, decision
    )


# -- configuration ------------------------------------------------------------


@router.get(
    "/config",
    response_model=ConfigResponse,
    summary="Inspect the active retrieval, model, and rule configuration",
)
async def get_config(
    caller: DevCallerDep,
    settings: SettingsDep,
    request_orchestrator: OrchestratorDep,
) -> ConfigResponse:
    return ConfigResponse(
        environment=settings.app_env,
        llm_provider=settings.llm_provider,
        llm_model=settings.llm_model,
        llm_effort=settings.llm_effort,
        embedding_provider=settings.embedding_provider,
        embedding_model=request_orchestrator.embedding_model_id,
        embedding_dimensions=settings.embedding_dimensions,
        retrieval_candidates=settings.retrieval_candidates,
        retrieval_top_k=settings.retrieval_top_k,
        retrieval_min_similarity=settings.retrieval_min_similarity,
        retrieval_lexical_enabled=settings.retrieval_enable_lexical,
        chunk_target_tokens=settings.chunk_target_tokens,
        chunk_overlap_tokens=settings.chunk_overlap_tokens,
        rules_version=RULES_VERSION,
        prompt_versions={intent.value: version for intent, version in PROMPT_VERSIONS.items()},
        water_quality_parameters=[threshold.key.value for threshold in THRESHOLDS],
    )


# -- internals ----------------------------------------------------------------


def _split_csv(raw: str) -> list[str]:
    return [item.strip() for item in raw.split(",") if item.strip()]


def _list_conditions(
    review_status: ReviewStatus | None,
    species: str | None,
    topic: str | None,
) -> list:
    conditions = []
    if review_status is not None:
        conditions.append(KnowledgeDocument.review_status == review_status)
    if species:
        conditions.append(KnowledgeDocument.species.any(species))
    if topic:
        conditions.append(KnowledgeDocument.topic.any(topic))
    return conditions


async def _chunk_counts(session, document_ids: list[UUID]) -> dict[UUID, int]:
    """Count chunks that actually carry an embedding.

    The denormalised `chunk_count` records what was chunked; this records what
    is retrievable. They differ when embedding partially failed.
    """
    if not document_ids:
        return {}
    rows = (
        await session.execute(
            select(KnowledgeChunk.document_id, func.count())
            .where(
                KnowledgeChunk.document_id.in_(document_ids),
                KnowledgeChunk.embedding.is_not(None),
            )
            .group_by(KnowledgeChunk.document_id)
        )
    ).all()
    return {document_id: count for document_id, count in rows}


async def _load_document(session, document_id: UUID) -> KnowledgeDocument:
    row = (
        await session.execute(
            select(KnowledgeDocument).where(KnowledgeDocument.id == document_id)
        )
    ).scalar_one_or_none()
    if row is None:
        raise DocumentNotFoundError(f"No knowledge document with id {document_id}.")
    return row


async def _set_review_status(
    database: Database,
    document_id: UUID,
    new_status: ReviewStatus,
    actor: str,
    decision: ReviewDecisionRequest | None,
) -> DocumentSummary:
    async with database.session() as session:
        row = await _load_document(session, document_id)
        previous = row.review_status

        row.review_status = new_status
        row.reviewed_by = actor
        row.reviewed_at = datetime.now(UTC)
        row.review_note = decision.note if decision else None

        counts = await _chunk_counts(session, [row.id])

        # Review-status changes are audited: this is the decision that controls
        # what knowledge can reach a farmer.
        logger.info(
            "knowledge_review_status_changed",
            extra={
                "document_id": str(row.id),
                "from_status": previous.value,
                "to_status": new_status.value,
                "actor": actor,
            },
        )
        return _to_summary(row, counts.get(row.id, 0))


def _to_summary(row: KnowledgeDocument, embedded_chunks: int) -> DocumentSummary:
    return DocumentSummary(
        id=str(row.id),
        title=row.title,
        source=row.source,
        author=row.author,
        year=row.year,
        document_type=row.document_type,
        species=row.species or [],
        life_stage=row.life_stage or [],
        topic=row.topic or [],
        disease=row.disease or [],
        region=row.region or [],
        evidence_level=row.evidence_level,
        review_status=row.review_status,
        owner=row.owner,
        checksum=row.checksum,
        version=row.version,
        chunk_count=embedded_chunks,
        ingest_status=row.ingest_status,
        ingest_error=row.ingest_error,
        ingested_at=row.ingested_at,
        reviewed_at=row.reviewed_at,
        reviewed_by=row.reviewed_by,
        review_note=row.review_note,
        created_at=row.created_at,
        updated_at=row.updated_at,
    )
