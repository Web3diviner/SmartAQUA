"""Metadata filters applied before vector search.

04_AQUADOC_RAG_LLM.md section 7: intent classification -> metadata filter ->
query embedding -> search.

The review-status filter is the load-bearing one. 14_AQUADOC_SAFETY_AND_GOVERNANCE.md
section 9: deprecated and rejected documents must not be retrieved in
production. That is enforced here and re-asserted in the SQL, so a caller
cannot widen it by passing a crafted filter.
"""

from __future__ import annotations

from dataclasses import dataclass, field

from app.schemas.common import Intent, ReviewStatus

#: Array columns on knowledge_documents that callers may filter on.
FILTERABLE_ARRAY_FIELDS = frozenset({"species", "life_stage", "topic", "disease", "region"})

#: Topics worth boosting per intent. Soft signals for reranking — never a hard
#: filter, because a well-matched passage tagged differently is still useful.
INTENT_TOPIC_HINTS: dict[Intent, tuple[str, ...]] = {
    Intent.FEEDING: ("feeding", "nutrition", "feed", "ration", "fcr"),
    Intent.WATER_QUALITY: ("water_quality", "oxygen", "ph", "ammonia", "temperature"),
    Intent.DISEASE: ("disease", "health", "pathology", "parasites", "treatment"),
    Intent.FARM_ASSESSMENT: ("management", "husbandry", "production", "water_quality", "feeding"),
    Intent.GENERAL_AQUACULTURE: (),
    Intent.UNKNOWN: (),
}

#: Questions where a weak source is worse than a cautious answer. For these,
#: the retriever prefers higher evidence levels (04_AQUADOC_RAG_LLM.md section 5).
HIGH_RISK_INTENTS = frozenset({Intent.DISEASE})


@dataclass(frozen=True)
class RetrievalFilters:
    """Resolved filter set for one retrieval call."""

    #: Always exactly {APPROVED} in production. Not caller-controllable.
    review_statuses: frozenset[ReviewStatus] = frozenset({ReviewStatus.APPROVED})
    array_filters: dict[str, list[str]] = field(default_factory=dict)
    topic_hints: tuple[str, ...] = ()
    prefer_high_evidence: bool = False

    def as_dict(self) -> dict[str, list[str]]:
        """Filter view for the retrieval trace / debug UI."""
        payload: dict[str, list[str]] = {
            "review_status": sorted(status.value for status in self.review_statuses)
        }
        payload.update({key: list(values) for key, values in self.array_filters.items()})
        if self.topic_hints:
            payload["topic_hints"] = list(self.topic_hints)
        return payload


def build_filters(
    intent: Intent,
    requested: dict[str, list[str]] | None = None,
    *,
    species: str | None = None,
) -> RetrievalFilters:
    """Resolve the filter set for a question.

    Caller-supplied filters are allowlisted against `FILTERABLE_ARRAY_FIELDS`;
    anything else is dropped rather than passed through to SQL.
    """
    array_filters: dict[str, list[str]] = {}

    for key, values in (requested or {}).items():
        normalised_key = key.strip().lower()
        if normalised_key not in FILTERABLE_ARRAY_FIELDS:
            continue
        cleaned = [value.strip() for value in values if isinstance(value, str) and value.strip()]
        if cleaned:
            array_filters[normalised_key] = cleaned

    # A pond's species narrows retrieval usefully, but only when the caller has
    # not already filtered on species themselves.
    if species and "species" not in array_filters:
        array_filters["species"] = [species]

    return RetrievalFilters(
        review_statuses=frozenset({ReviewStatus.APPROVED}),
        array_filters=array_filters,
        topic_hints=INTENT_TOPIC_HINTS.get(intent, ()),
        prefer_high_evidence=intent in HIGH_RISK_INTENTS,
    )
