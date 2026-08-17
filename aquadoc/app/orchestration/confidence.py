"""Confidence scoring.

14_AQUADOC_SAFETY_AND_GOVERNANCE.md section 3: confidence must not be arbitrary
LLM self-confidence alone. It combines:

  - retrieval relevance
  - evidence quality
  - completeness of farm data
  - rule agreement
  - model confidence

The model's own number is one weighted input among five, capped so that a
confident-sounding answer over thin retrieval and missing measurements cannot
report high confidence.
"""

from __future__ import annotations

from dataclasses import asdict, dataclass

from app.rag.reranking import Candidate
from app.rules.water_quality import STATUS_CONCERN, STATUS_OK, STATUS_UNKNOWN, STATUS_WATCH
from app.schemas.chat import RuleFinding
from app.schemas.common import EvidenceLevel, Intent

#: Weights sum to 1.0. Retrieval carries the most weight because an answer with
#: no supporting passages is ungrounded regardless of how sure the model sounds.
WEIGHT_RETRIEVAL = 0.30
WEIGHT_EVIDENCE = 0.20
WEIGHT_DATA_COMPLETENESS = 0.20
WEIGHT_RULE_AGREEMENT = 0.15
WEIGHT_MODEL = 0.15

#: Evidence-level quality scores (04_AQUADOC_RAG_LLM.md section 5).
_EVIDENCE_QUALITY: dict[EvidenceLevel, float] = {
    EvidenceLevel.A_OFFICIAL_GUIDELINE: 1.00,
    EvidenceLevel.B_PEER_REVIEWED: 0.90,
    EvidenceLevel.C_TEXTBOOK: 0.80,
    EvidenceLevel.D_VERIFIED_EXPERT_CASE: 0.65,
    EvidenceLevel.E_USER_REPORT: 0.45,
}

#: Ceiling when nothing was retrieved. An ungrounded answer can never be
#: presented as "high" confidence.
NO_SOURCE_CEILING = 0.35

#: Ceiling for a farm-specific assessment made on very incomplete data.
THIN_CONTEXT_CEILING = 0.60
THIN_CONTEXT_THRESHOLD = 0.4

#: Intents that require farm data to answer well.
_CONTEXT_DEPENDENT = frozenset(
    {Intent.FARM_ASSESSMENT, Intent.WATER_QUALITY, Intent.FEEDING, Intent.DISEASE}
)


@dataclass(frozen=True)
class ConfidenceBreakdown:
    """Per-component scores, exposed in developer mode for tuning."""

    retrieval_relevance: float
    evidence_quality: float
    data_completeness: float
    rule_agreement: float
    model_confidence: float
    weighted_score: float
    applied_ceiling: float | None
    final_score: float

    def as_dict(self) -> dict[str, float | None]:
        return asdict(self)


def score_confidence(
    *,
    intent: Intent,
    candidates: list[Candidate],
    findings: list[RuleFinding],
    context_completeness: float,
    model_confidence: float,
    has_farm_context: bool,
) -> ConfidenceBreakdown:
    """Compute the final confidence score and its components."""
    retrieval = _retrieval_relevance(candidates)
    evidence = _evidence_quality(candidates)
    completeness = context_completeness if has_farm_context else _neutral_completeness(intent)
    agreement = _rule_agreement(findings)
    model = _clamp(model_confidence)

    weighted = (
        WEIGHT_RETRIEVAL * retrieval
        + WEIGHT_EVIDENCE * evidence
        + WEIGHT_DATA_COMPLETENESS * completeness
        + WEIGHT_RULE_AGREEMENT * agreement
        + WEIGHT_MODEL * model
    )

    ceiling = _ceiling(intent, candidates, completeness, has_farm_context)
    final = min(weighted, ceiling) if ceiling is not None else weighted

    return ConfidenceBreakdown(
        retrieval_relevance=round(retrieval, 4),
        evidence_quality=round(evidence, 4),
        data_completeness=round(completeness, 4),
        rule_agreement=round(agreement, 4),
        model_confidence=round(model, 4),
        weighted_score=round(weighted, 4),
        applied_ceiling=ceiling,
        final_score=round(_clamp(final), 4),
    )


def _retrieval_relevance(candidates: list[Candidate]) -> float:
    """Similarity of the selected passages, weighted toward the best matches.

    Uses the top three: a strong lead passage matters more than a long tail of
    marginal ones.
    """
    selected = [candidate for candidate in candidates if candidate.selected]
    if not selected:
        return 0.0

    top = sorted((candidate.similarity for candidate in selected), reverse=True)[:3]
    weights = [0.5, 0.3, 0.2][: len(top)]
    total_weight = sum(weights)
    return _clamp(sum(s * w for s, w in zip(top, weights, strict=True)) / total_weight)


def _evidence_quality(candidates: list[Candidate]) -> float:
    """Mean evidence quality of the selected sources."""
    selected = [candidate for candidate in candidates if candidate.selected]
    if not selected:
        return 0.0
    scores = [_EVIDENCE_QUALITY.get(candidate.evidence_level, 0.4) for candidate in selected]
    return _clamp(sum(scores) / len(scores))


def _rule_agreement(findings: list[RuleFinding]) -> float:
    """How settled the deterministic picture is.

    A clean `ok` picture is high agreement; a mix of concerns and unknowns is
    low. `unknown` counts against agreement because it means the rules could not
    assess that dimension at all.
    """
    if not findings:
        return 0.5  # neutral — no deterministic signal either way

    weights = {STATUS_OK: 1.0, STATUS_WATCH: 0.6, STATUS_CONCERN: 0.35, STATUS_UNKNOWN: 0.3}
    scores = [weights.get(finding.status, 0.5) for finding in findings]
    return _clamp(sum(scores) / len(scores))


def _neutral_completeness(intent: Intent) -> float:
    """Completeness score when no farm context was supplied.

    Educational questions need no farm data, so completeness is not a penalty.
    A farm-specific question asked without farm data is a different matter.
    """
    return 0.3 if intent in _CONTEXT_DEPENDENT else 1.0


def _ceiling(
    intent: Intent,
    candidates: list[Candidate],
    completeness: float,
    has_farm_context: bool,
) -> float | None:
    if not any(candidate.selected for candidate in candidates):
        return NO_SOURCE_CEILING
    if (
        intent in _CONTEXT_DEPENDENT
        and has_farm_context
        and completeness < THIN_CONTEXT_THRESHOLD
    ):
        return THIN_CONTEXT_CEILING
    return None


def _clamp(value: float) -> float:
    return min(1.0, max(0.0, float(value)))
