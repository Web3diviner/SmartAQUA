"""The ingestion pipeline.

04_AQUADOC_RAG_LLM.md section 4:

    approved document -> checksum -> parse -> clean -> preserve page/section
                      -> chunk -> metadata tagging -> embedding -> pgvector
                      -> review status

Two governance properties are enforced here rather than left to the caller:

1. Ingested documents land in `pending`, never `approved`. Approval is a
   separate, audited human decision (14_AQUADOC_SAFETY_AND_GOVERNANCE.md
   section 9). Ingesting a document must not make it retrievable.

2. The embedding model is recorded on every chunk. Vectors from different
   models are not comparable, so a model change requires a re-ingest and the
   stored model ID is how that is detected.
"""

from __future__ import annotations

import logging
from dataclasses import dataclass
from datetime import UTC, datetime

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from app.embeddings.base import EmbeddingProvider
from app.errors import UploadRejectedError
from app.models import KnowledgeChunk, KnowledgeDocument
from app.schemas.common import ReviewStatus
from app.schemas.knowledge import DocumentMetadata, IngestResult
from ingestion.chunker import Chunk, Chunker
from ingestion.cleaner import clean_text, find_repeated_lines, strip_repeated_lines
from ingestion.loader import LoadedDocument
from ingestion.parser import ParsedBlock, parse

logger = logging.getLogger(__name__)

#: Embedding calls are batched to bound memory and provider request size.
_EMBED_BATCH = 32


@dataclass(frozen=True)
class IngestionConfig:
    chunk_target_tokens: int = 750
    chunk_overlap_tokens: int = 150


class IngestionService:
    """Turns an uploaded document into embedded, retrievable chunks."""

    def __init__(
        self,
        *,
        embeddings: EmbeddingProvider,
        config: IngestionConfig | None = None,
    ) -> None:
        self._embeddings = embeddings
        self._config = config or IngestionConfig()
        self._chunker = Chunker(
            target_tokens=self._config.chunk_target_tokens,
            overlap_tokens=self._config.chunk_overlap_tokens,
        )

    async def ingest(
        self,
        session: AsyncSession,
        *,
        document: LoadedDocument,
        metadata: DocumentMetadata,
        replace_existing: bool = False,
    ) -> IngestResult:
        warnings: list[str] = []

        existing = await self._find_by_checksum(session, document.checksum)
        if existing is not None and not replace_existing:
            raise UploadRejectedError(
                f"This document has already been ingested (id={existing.id}, "
                f"status={existing.review_status.value}). Pass replace_existing "
                f"to re-ingest it as a new version."
            )

        version = (existing.version + 1) if existing is not None else 1
        if existing is not None:
            warnings.append(
                f"Re-ingesting as version {version}; the previous version remains "
                f"stored for audit."
            )

        # -- parse and clean --------------------------------------------------
        parsed = parse(document.content, document.media_type)
        if parsed.is_empty:
            raise UploadRejectedError("The document contained no extractable text.")

        cleaned_blocks = self._clean_blocks(parsed.blocks)

        # -- chunk -------------------------------------------------------------
        chunks = self._chunker.chunk_blocks(cleaned_blocks)
        if not chunks:
            raise UploadRejectedError("The document produced no usable chunks after cleaning.")

        # -- persist the document record --------------------------------------
        record = KnowledgeDocument(
            title=metadata.title,
            source=metadata.source,
            author=metadata.author,
            year=metadata.year,
            document_type=metadata.document_type,
            species=metadata.species,
            life_stage=metadata.life_stage,
            topic=metadata.topic,
            disease=metadata.disease,
            region=metadata.region,
            evidence_level=metadata.evidence_level,
            # Always pending. Approval is a separate human decision.
            review_status=ReviewStatus.PENDING,
            owner=metadata.owner,
            file_url=document.storage_name,
            checksum=document.checksum,
            version=version,
            chunk_count=len(chunks),
            ingest_status="in_progress",
        )
        session.add(record)
        await session.flush()

        # -- embed and store ---------------------------------------------------
        embedded = await self._embed_and_store(session, record, chunks, metadata)

        record.ingest_status = "complete"
        record.ingested_at = datetime.now(UTC)
        record.chunk_count = len(chunks)

        if embedded != len(chunks):
            warnings.append(
                f"{len(chunks) - embedded} chunk(s) could not be embedded and will "
                f"not be retrievable."
            )

        logger.info(
            "document_ingested",
            extra={
                "document_id": str(record.id),
                "chunks": len(chunks),
                "embedded": embedded,
                "version": version,
            },
        )

        return IngestResult(
            document_id=str(record.id),
            title=record.title,
            checksum=record.checksum,
            chunk_count=len(chunks),
            embedded_chunks=embedded,
            review_status=record.review_status,
            warnings=warnings,
        )

    # -- internals -----------------------------------------------------------

    @staticmethod
    def _clean_blocks(blocks: list[ParsedBlock]) -> list[ParsedBlock]:
        """Clean each block, removing running headers detected across pages."""
        repeated = find_repeated_lines(block.text for block in blocks)

        cleaned: list[ParsedBlock] = []
        for block in blocks:
            text = strip_repeated_lines(clean_text(block.text), repeated)
            if text.strip():
                cleaned.append(
                    ParsedBlock(text=text, page_number=block.page_number, section=block.section)
                )
        return cleaned

    async def _embed_and_store(
        self,
        session: AsyncSession,
        record: KnowledgeDocument,
        chunks: list[Chunk],
        metadata: DocumentMetadata,
    ) -> int:
        embedded_count = 0

        for start in range(0, len(chunks), _EMBED_BATCH):
            batch = chunks[start : start + _EMBED_BATCH]
            texts = [self._embedding_text(chunk, metadata) for chunk in batch]
            vectors = await self._embeddings.embed_documents(texts)

            for chunk, vector in zip(batch, vectors, strict=True):
                session.add(
                    KnowledgeChunk(
                        document_id=record.id,
                        chunk_index=chunk.index,
                        content=chunk.content,
                        token_estimate=chunk.token_estimate,
                        page_number=chunk.page_number,
                        section=chunk.section,
                        metadata_json={
                            "species": metadata.species,
                            "topic": metadata.topic,
                            "disease": metadata.disease,
                            "life_stage": metadata.life_stage,
                            "evidence_level": metadata.evidence_level.value,
                        },
                        embedding=vector,
                        embedding_model=self._embeddings.model_id,
                    )
                )
                embedded_count += 1

        return embedded_count

    @staticmethod
    def _embedding_text(chunk: Chunk, metadata: DocumentMetadata) -> str:
        """Text actually sent to the embedding model.

        Prefixing the title and section gives an otherwise context-free chunk
        enough anchoring to match a topical query — a passage about "reduce the
        ration by 20%" is much more findable when it carries "Feeding
        Management" alongside it.
        """
        parts = [metadata.title]
        if chunk.section:
            parts.append(chunk.section)
        parts.append(chunk.content)
        return "\n\n".join(parts)

    @staticmethod
    async def _find_by_checksum(
        session: AsyncSession, checksum: str
    ) -> KnowledgeDocument | None:
        result = await session.execute(
            select(KnowledgeDocument)
            .where(KnowledgeDocument.checksum == checksum)
            .order_by(KnowledgeDocument.version.desc())
            .limit(1)
        )
        return result.scalar_one_or_none()
