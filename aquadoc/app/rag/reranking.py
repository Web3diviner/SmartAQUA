"""Reranking and source-quality weighting.

04_AQUADOC_RAG_LLM.md section 7: vector search -> optional lexical/hybrid ->
reranking -> source-quality weighting -> top context chunks.

Deliberately deterministic arithmetic rather than a neural cross-encoder: it is
testable without AI (13_CODING_AND_ENGINEERING_STANDARDS.md), it is cheap, and
its behaviour is explainable in the Retrieval Inspector. A learned reranker can
replace `rerank` later without changing callers.
"""

from __future__ import annotations

from dataclasses import dataclass, replace

from app.rag.filters import RetrievalFilters
from app.schemas.common import EvidenceLevel

#: Ranked-list fusion constant. 60 is the value from the original RRF paper and
#: damps the influence of any single ranking's top position.
RRF_K = 60.0

#: Source-quality weights (04_AQUADOC_RAG_LLM.md section 5).
EVIDENCE_WEIGHTS: dict[EvidenceLevel, float] = {
    EvidenceLevel.A_OFFICIAL_GUIDELINE: 1.00,
    EvidenceLevel.B_PEER_REVIEWED: 0.95,
    EvidenceLevel.C_TEXTBOOK: 0.90,
    EvidenceLevel.D_VERIFIED_EXPERT_CASE: 0.80,
    EvidenceLevel.E_USER_REPORT: 0.65,
}

#: For high-risk questions the spread widens, so a user report cannot outrank a
#: guideline on a marginally better similarity score.
HIGH_RISK_EVIDENCE_WEIGHTS: dict[EvidenceLevel, float] = {
    EvidenceLevel.A_OFFICIAL_GUIDELINE: 1.00,
    EvidenceLevel.B_PEER_REVIEWED: 0.92,
    EvidenceLevel.C_TEXTBOOK: 0.82,
    EvidenceLevel.D_VERIFIED_EXPERT_CASE: 0.65,
    EvidenceLevel.E_USER_REPORT: 0.35,
}

_TOPIC_HINT_BOOST = 0.06
_MAX_CHUNKS_PER_DOCUMENT = 3


@dataclass
class Candidate:
    """A retrieval candidate as it moves through scoring."""

    chunk_id: str
    document_id: str
    title: str
    source: str
    author: str | None
    year: int | None
    page_number: int | None
    section: str | None
    evidence_level: EvidenceLevel
    topics: list[str]
    content: str
    similarity: float
    vector_rank: int | None = None
    lexical_rank: int | None = None
    fused_score: float = 0.0
    final_score: float = 0.0
    selected: bool = False


def reciprocal_rank_fusion(candidates: list[Candidate]) -> list[Candidate]:
    """Fuse the vector and lexical rankings into one score.

    RRF combines rankings without needing the two score scales to be
    comparable — cosine similarity and `ts_rank` are not.
    """
    fused: list[Candidate] = []
    for candidate in candidates:
        score = 0.0
        if candidate.vector_rank is not None:
            score += 1.0 / (RRF_K + candidate.vector_rank)
        if candidate.lexical_rank is not None:
            score += 1.0 / (RRF_K + candidate.lexical_rank)
        fused.append(replace(candidate, fused_score=score))
    return fused


def apply_source_quality(candidates: list[Candidate], filters: RetrievalFilters) -> list[Candidate]:
    """Weight fused scores by evidence level and topic alignment."""
    weights = HIGH_RISK_EVIDENCE_WEIGHTS if filters.prefer_high_evidence else EVIDENCE_WEIGHTS
    hints = {hint.lower() for hint in filters.topic_hints}

    weighted: list[Candidate] = []
    for candidate in candidates:
        weight = weights.get(candidate.evidence_level, 0.5)
        score = candidate.fused_score * weight

        if hints and any(topic.lower() in hints for topic in candidate.topics):
            score *= 1.0 + _TOPIC_HINT_BOOST

        weighted.append(replace(candidate, final_score=score))
    return weighted


def enforce_document_diversity(
    candidates: list[Candidate],
    top_k: int,
    max_per_document: int = _MAX_CHUNKS_PER_DOCUMENT,
) -> list[Candidate]:
    """Select the top K, capping how much any single document can contribute.

    Without this a single long document crowds out every other source, which
    makes citations look broad while the grounding is actually narrow.
    """
    selected: list[Candidate] = []
    per_document: dict[str, int] = {}

    for candidate in candidates:
        if len(selected) >= top_k:
            break
        used = per_document.get(candidate.document_id, 0)
        if used >= max_per_document:
            continue
        per_document[candidate.document_id] = used + 1
        selected.append(candidate)

    # Backfill only if the diversity cap left the result set short.
    if len(selected) < top_k:
        chosen = {candidate.chunk_id for candidate in selected}
        for candidate in candidates:
            if len(selected) >= top_k:
                break
            if candidate.chunk_id not in chosen:
                selected.append(candidate)

    return selected


def rerank(
    candidates: list[Candidate],
    filters: RetrievalFilters,
    top_k: int,
) -> list[Candidate]:
    """Full reranking pass. Returns every candidate, `selected` flagged.

    All candidates are returned (not just the winners) so the Retrieval
    Inspector can show what was considered and rejected.
    """
    fused = reciprocal_rank_fusion(candidates)
    weighted = apply_source_quality(fused, filters)
    weighted.sort(key=lambda candidate: (-candidate.final_score, -candidate.similarity))

    winners = {candidate.chunk_id for candidate in enforce_document_diversity(weighted, top_k)}
    return [replace(candidate, selected=candidate.chunk_id in winners) for candidate in weighted]
