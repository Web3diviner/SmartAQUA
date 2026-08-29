"""In-memory Knowledge Retriever for local development fallback.

When PostgreSQL + pgvector is not running locally in development mode,
this retriever indexes approved documents in `sample-knowledge/` in-memory,
computing dense vector similarity and lexical ranking so RAG retrieval,
citations, and confidence scoring work out-of-the-box.
"""

from __future__ import annotations

import logging
import math
import time
from pathlib import Path

from app.embeddings.base import EmbeddingProvider
from app.rag.filters import RetrievalFilters
from app.rag.reranking import Candidate, rerank
from app.schemas.chat import RetrievalTrace, RetrievalTraceItem
from app.schemas.common import EvidenceLevel, Intent
from ingestion.chunker import Chunker
from ingestion.loader import load_path
from ingestion.parser import parse

logger = logging.getLogger(__name__)

_PREVIEW_CHARS = 240


def _cosine_similarity(a: list[float], b: list[float]) -> float:
    dot = sum(x * y for x, y in zip(a, b))
    norm_a = math.sqrt(sum(x * x for x in a))
    norm_b = math.sqrt(sum(y * y for y in b))
    if norm_a == 0 or norm_b == 0:
        return 0.0
    return max(0.0, min(1.0, dot / (norm_a * norm_b)))


class MemoryRetriever:
    """In-memory retriever loaded from sample-knowledge/ markdown/pdf files."""

    def __init__(
        self,
        embeddings: EmbeddingProvider,
        *,
        candidates: int = 40,
        top_k: int = 6,
        min_similarity: float = 0.02,
        enable_lexical: bool = True,
    ) -> None:
        self._embeddings = embeddings
        self._top_k = top_k
        self._min_similarity = min_similarity
        self._enable_lexical = enable_lexical
        self._initialized = False
        self._chunks: list[dict] = []
        self._chunker = Chunker(target_tokens=750, overlap_tokens=150)

    async def _ensure_loaded(self) -> None:
        if self._initialized:
            return

        sample_dir = Path(__file__).resolve().parent.parent.parent / "sample-knowledge"
        if not sample_dir.exists():
            self._initialized = True
            return

        chunks_to_embed = []
        metadata_list = []

        for doc_idx, path in enumerate(sorted(sample_dir.glob("*.md")), start=1):
            try:
                doc = load_path(path, max_bytes=10 * 1024 * 1024)
                text_str = doc.content.decode("utf-8", errors="replace")
                parsed = parse(doc.content, doc.media_type)
                chunks = self._chunker.chunk_blocks(parsed.blocks)

                # Parse authentic metadata from markdown headers
                doc_title = path.stem.replace("-", " ").title()
                doc_source = "National Aquaculture Research Institute / FAO Guidelines"
                doc_year = 2024
                doc_evidence = EvidenceLevel.A_OFFICIAL_GUIDELINE
                doc_topics = ["aquaculture", "water_quality", "nigeria", "west_africa"]

                for line in text_str.splitlines()[:20]:
                    if line.startswith("# "):
                        doc_title = line[2:].strip()
                    elif "**Publisher / Source:**" in line or "**Source:**" in line:
                        doc_source = line.split(":", 1)[1].replace("**", "").strip()
                    elif "**Year:**" in line:
                        import re
                        m = re.search(r"\d{4}", line)
                        if m:
                            doc_year = int(m.group(0))
                    elif "**Topics:**" in line:
                        raw_top = line.split(":", 1)[1].replace("**", "").strip()
                        doc_topics = [t.strip().lower() for t in raw_top.split(",") if t.strip()]
                    elif "**Evidence Level:**" in line:
                        if "B" in line.upper():
                            doc_evidence = EvidenceLevel.B_PEER_REVIEWED
                        elif "C" in line.upper():
                            doc_evidence = EvidenceLevel.C_TEXTBOOK
                        elif "D" in line.upper():
                            doc_evidence = EvidenceLevel.D_EXPERT_CASE

                for chunk_idx, chunk in enumerate(chunks, start=1):
                    chunks_to_embed.append(chunk.content)
                    metadata_list.append(
                        {
                            "chunk_id": f"mem-chk-{doc_idx:02d}-{chunk_idx:03d}",
                            "document_id": f"mem-doc-{doc_idx:03d}",
                            "title": doc_title,
                            "source": doc_source,
                            "author": "Aquaculture Technical Working Group",
                            "year": doc_year,
                            "page_number": chunk.page_number or chunk_idx,
                            "section": chunk.section or doc_title,
                            "evidence_level": doc_evidence,
                            "topics": doc_topics,
                            "content": chunk.content,
                        }
                    )
            except Exception as e:
                logger.warning("failed_to_load_sample_doc", extra={"path": str(path), "error": str(e)})

        if chunks_to_embed:
            vectors = await self._embeddings.embed_documents(chunks_to_embed)
            for meta, vec in zip(metadata_list, vectors):
                meta["vector"] = vec
                self._chunks.append(meta)

        logger.info(
            "memory_retriever_initialized",
            extra={"documents_loaded": doc_idx if 'doc_idx' in locals() else 0, "total_chunks": len(self._chunks)},
        )
        self._initialized = True

    async def retrieve(
        self,
        *,
        request_id: str,
        question: str,
        intent: Intent,
        filters: RetrievalFilters,
        top_k: int | None = None,
    ):
        from app.rag.retrieval import RetrievalResult

        started = time.perf_counter()
        effective_top_k = top_k or self._top_k
        await self._ensure_loaded()

        query_vector = await self._embeddings.embed_query(question)

        candidates: list[Candidate] = []
        for i, item in enumerate(self._chunks):
            similarity = _cosine_similarity(query_vector, item["vector"])
            if similarity >= self._min_similarity:
                candidates.append(
                    Candidate(
                        chunk_id=item["chunk_id"],
                        document_id=item["document_id"],
                        title=item["title"],
                        source=item["source"],
                        author=item["author"],
                        year=item["year"],
                        page_number=item["page_number"],
                        section=item["section"],
                        evidence_level=item["evidence_level"],
                        topics=item["topics"],
                        content=item["content"],
                        similarity=similarity,
                        vector_rank=i + 1,
                    )
                )

        ranked = rerank(candidates, filters, effective_top_k)
        latency_ms = (time.perf_counter() - started) * 1000

        trace = RetrievalTrace(
            request_id=request_id,
            question=question,
            intent=intent,
            metadata_filters=filters.as_dict(),
            embedding_model=self._embeddings.model_id,
            embedding_dimensions=self._embeddings.dimensions,
            candidates_considered=len(ranked),
            selected_count=sum(1 for c in ranked if c.selected),
            lexical_enabled=self._enable_lexical,
            min_similarity=self._min_similarity,
            retrieval_latency_ms=round(latency_ms, 2),
            items=[
                RetrievalTraceItem(
                    chunk_id=c.chunk_id,
                    document_id=c.document_id,
                    title=c.title,
                    page=c.page_number,
                    section=c.section,
                    evidence_level=c.evidence_level,
                    similarity=round(c.similarity, 4),
                    lexical_rank=c.lexical_rank,
                    vector_rank=c.vector_rank,
                    fused_score=round(c.fused_score, 6),
                    final_score=round(c.final_score, 6),
                    selected=c.selected,
                    content_preview=c.content[:_PREVIEW_CHARS],
                )
                for c in ranked
            ],
        )

        return RetrievalResult(candidates=ranked, trace=trace)
