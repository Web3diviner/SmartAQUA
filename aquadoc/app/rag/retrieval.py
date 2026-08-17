"""Semantic + lexical retrieval over approved knowledge chunks.

Pipeline (04_AQUADOC_RAG_LLM.md section 7):

    question -> intent -> metadata filter -> query embedding
             -> vector similarity search
             -> lexical search (optional, fused)
             -> reranking + source-quality weighting
             -> top context chunks

Every SQL statement re-asserts `review_status = 'approved'`. That is intentional
duplication: the filter layer already excludes non-approved documents, but a
retrieval path that could ever return deprecated or rejected knowledge is a
governance failure (14_AQUADOC_SAFETY_AND_GOVERNANCE.md section 9), so the
guarantee is enforced where the rows are actually read.
"""

from __future__ import annotations

import logging
import time
from dataclasses import dataclass

from sqlalchemy import text
from sqlalchemy.exc import SQLAlchemyError
from sqlalchemy.ext.asyncio import AsyncSession

from app.embeddings.base import EmbeddingProvider
from app.errors import RetrievalError
from app.rag.filters import RetrievalFilters
from app.rag.reranking import Candidate, rerank
from app.schemas.common import EvidenceLevel, Intent
from app.schemas.chat import RetrievalTrace, RetrievalTraceItem

logger = logging.getLogger(__name__)

_PREVIEW_CHARS = 240

_SELECT_COLUMNS = """
    c.id::text            AS chunk_id,
    c.document_id::text   AS document_id,
    c.content             AS content,
    c.page_number         AS page_number,
    c.section             AS section,
    d.title               AS title,
    d.source              AS source,
    d.author              AS author,
    d.year                AS year,
    d.evidence_level::text AS evidence_level,
    d.topic               AS topics
"""


@dataclass(frozen=True)
class RetrievalResult:
    """Selected passages plus the full trace of what was considered."""

    candidates: list[Candidate]
    trace: RetrievalTrace

    @property
    def selected(self) -> list[Candidate]:
        return [candidate for candidate in self.candidates if candidate.selected]


class Retriever:
    """Reads approved knowledge chunks. Owns no business logic beyond ranking."""

    def __init__(
        self,
        session: AsyncSession,
        embeddings: EmbeddingProvider,
        *,
        candidates: int = 40,
        top_k: int = 6,
        min_similarity: float = 0.15,
        enable_lexical: bool = True,
    ) -> None:
        self._session = session
        self._embeddings = embeddings
        self._candidate_limit = candidates
        self._top_k = top_k
        self._min_similarity = min_similarity
        self._enable_lexical = enable_lexical

    async def retrieve(
        self,
        *,
        request_id: str,
        question: str,
        intent: Intent,
        filters: RetrievalFilters,
        top_k: int | None = None,
    ) -> RetrievalResult:
        started = time.perf_counter()
        effective_top_k = top_k or self._top_k

        query_vector = await self._embeddings.embed_query(question)

        try:
            vector_hits = await self._vector_search(query_vector, filters)
            lexical_hits = (
                await self._lexical_search(question, filters) if self._enable_lexical else {}
            )
        except SQLAlchemyError as exc:
            logger.exception("retrieval_query_failed", extra={"intent": intent.value})
            raise RetrievalError("Knowledge retrieval query failed.") from exc

        merged = self._merge(vector_hits, lexical_hits)
        ranked = rerank(merged, filters, effective_top_k)
        latency_ms = (time.perf_counter() - started) * 1000

        trace = RetrievalTrace(
            request_id=request_id,
            question=question,
            intent=intent,
            metadata_filters=filters.as_dict(),
            embedding_model=self._embeddings.model_id,
            embedding_dimensions=self._embeddings.dimensions,
            candidates_considered=len(ranked),
            selected_count=sum(1 for candidate in ranked if candidate.selected),
            lexical_enabled=self._enable_lexical,
            min_similarity=self._min_similarity,
            retrieval_latency_ms=round(latency_ms, 2),
            items=[
                RetrievalTraceItem(
                    chunk_id=candidate.chunk_id,
                    document_id=candidate.document_id,
                    title=candidate.title,
                    page=candidate.page_number,
                    section=candidate.section,
                    evidence_level=candidate.evidence_level,
                    similarity=round(candidate.similarity, 4),
                    lexical_rank=candidate.lexical_rank,
                    vector_rank=candidate.vector_rank,
                    fused_score=round(candidate.fused_score, 6),
                    final_score=round(candidate.final_score, 6),
                    selected=candidate.selected,
                    content_preview=candidate.content[:_PREVIEW_CHARS],
                )
                for candidate in ranked
            ],
        )
        return RetrievalResult(candidates=ranked, trace=trace)

    # -- queries -------------------------------------------------------------

    async def _vector_search(
        self, query_vector: list[float], filters: RetrievalFilters
    ) -> list[Candidate]:
        where_sql, params = self._filter_sql(filters)
        params["query_vector"] = self._vector_literal(query_vector)
        params["limit"] = self._candidate_limit
        params["min_similarity"] = self._min_similarity

        # `<=>` is pgvector cosine distance; similarity = 1 - distance.
        sql = text(
            f"""
            SELECT {_SELECT_COLUMNS},
                   1 - (c.embedding <=> CAST(:query_vector AS vector)) AS similarity
            FROM knowledge_chunks c
            JOIN knowledge_documents d ON d.id = c.document_id
            WHERE d.review_status = 'approved'
              AND c.embedding IS NOT NULL
              {where_sql}
              AND 1 - (c.embedding <=> CAST(:query_vector AS vector)) >= :min_similarity
            ORDER BY c.embedding <=> CAST(:query_vector AS vector)
            LIMIT :limit
            """
        )
        rows = (await self._session.execute(sql, params)).mappings().all()
        return [
            self._to_candidate(row, similarity=float(row["similarity"]), vector_rank=index + 1)
            for index, row in enumerate(rows)
        ]

    async def _lexical_search(self, question: str, filters: RetrievalFilters) -> dict[str, int]:
        """Return chunk_id -> lexical rank for keyword matches.

        Catches exact terms — species names, compound names, unit strings — that
        a dense embedding can smear away.
        """
        where_sql, params = self._filter_sql(filters)
        params["question"] = question
        params["limit"] = self._candidate_limit

        sql = text(
            f"""
            SELECT c.id::text AS chunk_id
            FROM knowledge_chunks c
            JOIN knowledge_documents d ON d.id = c.document_id
            WHERE d.review_status = 'approved'
              AND c.content_tsv @@ websearch_to_tsquery('english', :question)
              {where_sql}
            ORDER BY ts_rank(c.content_tsv, websearch_to_tsquery('english', :question)) DESC
            LIMIT :limit
            """
        )
        rows = (await self._session.execute(sql, params)).mappings().all()
        return {row["chunk_id"]: index + 1 for index, row in enumerate(rows)}

    # -- helpers -------------------------------------------------------------

    @staticmethod
    def _filter_sql(filters: RetrievalFilters) -> tuple[str, dict[str, object]]:
        """Build the array-overlap predicates.

        Keys come from an allowlist in `filters.build_filters`, and values are
        always bound parameters — never string-interpolated.
        """
        clauses: list[str] = []
        params: dict[str, object] = {}
        for field_name, values in filters.array_filters.items():
            param = f"filter_{field_name}"
            clauses.append(f"AND d.{field_name} && CAST(:{param} AS text[])")
            params[param] = values
        return "\n              ".join(clauses), params

    @staticmethod
    def _vector_literal(vector: list[float]) -> str:
        """pgvector's text input format."""
        return "[" + ",".join(f"{value:.8f}" for value in vector) + "]"

    @staticmethod
    def _to_candidate(row: dict, *, similarity: float, vector_rank: int | None) -> Candidate:
        return Candidate(
            chunk_id=row["chunk_id"],
            document_id=row["document_id"],
            title=row["title"],
            source=row["source"],
            author=row["author"],
            year=row["year"],
            page_number=row["page_number"],
            section=row["section"],
            evidence_level=EvidenceLevel(row["evidence_level"]),
            topics=list(row["topics"] or []),
            content=row["content"],
            similarity=similarity,
            vector_rank=vector_rank,
        )

    @staticmethod
    def _merge(vector_hits: list[Candidate], lexical_ranks: dict[str, int]) -> list[Candidate]:
        """Attach lexical ranks to the vector candidate set.

        Lexical-only hits are not promoted into the candidate pool: without an
        embedding they cannot be scored for semantic relevance, and a pure
        keyword match is a weak grounding signal on its own.
        """
        for candidate in vector_hits:
            candidate.lexical_rank = lexical_ranks.get(candidate.chunk_id)
        return vector_hits
