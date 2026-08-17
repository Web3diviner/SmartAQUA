"""Citation construction.

15_AQUADOC_FRONTEND.md section 12: every grounded answer displays supporting
sources.

Citations are built from what retrieval actually returned — never from anything
the model wrote. The model can reference a source by ID, but it cannot invent a
source, a page number, or an evidence level, because those fields are only ever
copied from the database row.
"""

from __future__ import annotations

from app.rag.reranking import Candidate
from app.schemas.chat import SourceReference

_EXCERPT_CHARS = 320

#: Stable, short IDs the model uses to reference sources in its answer.
#: Positional rather than UUID-based so they stay readable in a prompt.
CITATION_PREFIX = "S"


def citation_id(index: int) -> str:
    return f"{CITATION_PREFIX}{index + 1}"


def build_source_references(
    candidates: list[Candidate],
    *,
    include_full_text: bool = False,
) -> list[SourceReference]:
    """Convert selected candidates into citations.

    Args:
        candidates: the selected passages, in presentation order.
        include_full_text: developer mode only — attaches the whole chunk so the
            Retrieval Inspector can show exactly what the model was given.
    """
    references: list[SourceReference] = []
    for index, candidate in enumerate(candidates):
        references.append(
            SourceReference(
                chunk_id=citation_id(index),
                document_id=candidate.document_id,
                title=candidate.title,
                source=candidate.source,
                author=candidate.author,
                year=candidate.year,
                page=candidate.page_number,
                section=candidate.section,
                evidence_level=candidate.evidence_level,
                excerpt=_excerpt(candidate.content),
                score=_clamp(candidate.similarity),
                chunk_text=candidate.content if include_full_text else None,
            )
        )
    return references


def _excerpt(content: str) -> str:
    """A readable snippet, cut on a word boundary."""
    collapsed = " ".join(content.split())
    if len(collapsed) <= _EXCERPT_CHARS:
        return collapsed
    cut = collapsed[:_EXCERPT_CHARS]
    boundary = cut.rfind(" ")
    return (cut[:boundary] if boundary > 0 else cut).rstrip() + "..."


def _clamp(value: float) -> float:
    return round(min(1.0, max(0.0, value)), 4)


def filter_reported_sources(
    references: list[SourceReference],
    reported_ids: list[str],
) -> list[str]:
    """Keep only source IDs that actually exist.

    The model may cite an ID that was never supplied. Silently dropping unknown
    IDs is correct here — a dangling citation in the UI is worse than no
    citation, and the answer text itself is unaffected.
    """
    known = {reference.chunk_id for reference in references}
    return [source_id for source_id in reported_ids if source_id in known]
