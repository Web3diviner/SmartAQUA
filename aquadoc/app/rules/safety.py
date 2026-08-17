"""Safety guardrails applied to model output.

This module is the enforcement point for the non-negotiable rules in
00_README.md and 14_AQUADOC_SAFETY_AND_GOVERNANCE.md:

  - AI output must never overwrite deterministic safety rules.
  - AquaDoc produces recommendations; the platform produces commands.
  - Health output is decision support, never guaranteed diagnosis.
  - High-risk or uncertain cases support expert escalation.

Everything here runs *after* the model responds and can only make the response
more conservative — raise a tier, raise risk, force escalation, strip an action.
It can never relax a constraint. That direction is the whole point: a model that
returns confident, permissive output cannot talk its way past these checks.
"""

from __future__ import annotations

import re
from dataclasses import dataclass

from app.rules.water_quality import STATUS_CONCERN, STATUS_WATCH
from app.schemas.chat import RecommendedAction, RuleFinding
from app.schemas.common import RecommendationTier, RiskLevel

RULES_VERSION = "safety/v1"

_RISK_ORDER: dict[RiskLevel, int] = {
    RiskLevel.INFORMATIONAL: 0,
    RiskLevel.WATCH: 1,
    RiskLevel.ELEVATED: 2,
    RiskLevel.HIGH: 3,
}

_TIER_ORDER: dict[RecommendationTier, int] = {
    RecommendationTier.TIER_0_INFORMATIONAL: 0,
    RecommendationTier.TIER_1_ADVISORY: 1,
    RecommendationTier.TIER_2_LOW_RISK_OPERATIONAL: 2,
    RecommendationTier.TIER_3_HIGH_RISK: 3,
}

#: Phrasing that would be read as a device instruction rather than a proposal.
#: Any action matching these is escalated to Tier 3 so a human decides.
_ACTUATION_PATTERNS = (
    re.compile(r"\b(?:i (?:will|'ll)|i am going to|now) (?:start|stop|run|trigger)\b", re.I),
    re.compile(r"\b(?:starting|stopping|triggering|dispensing|activating) the (?:feeder|motor|auger|pump)\b", re.I),
    re.compile(r"\bsend(?:ing)? (?:a )?command\b", re.I),
)

#: Claims of certainty a decision-support system must not make.
#: 14_AQUADOC_SAFETY_AND_GOVERNANCE.md section 2.
_OVERCLAIM_PATTERNS: tuple[tuple[re.Pattern[str], str], ...] = (
    (
        re.compile(r"\b(?:definitely|certainly|undoubtedly) (?:have|has|is|are)\b", re.I),
        "The answer asserted certainty that the available evidence does not support.",
    ),
    (
        re.compile(r"\bconfirmed diagnosis\b", re.I),
        "The answer referred to a confirmed diagnosis without laboratory or expert confirmation.",
    ),
    (
        re.compile(r"\b(?:lab|laboratory)[- ]confirmed\b", re.I),
        "The answer implied laboratory confirmation that AquaDoc cannot provide.",
    ),
    (
        re.compile(r"\bthis is (?:100%|guaranteed)\b", re.I),
        "The answer claimed a guarantee.",
    ),
)

#: Actions that change how much or whether fish are fed carry operational risk.
_OPERATIONAL_KEYWORDS = re.compile(
    r"\b(?:reduce|increase|adjust|change|skip|delay|suspend|stop|halt|withhold)\b.*"
    r"\b(?:feed|feeding|ration|schedule)\b",
    re.I,
)

#: Actions with a materially harder-to-reverse outcome.
_HIGH_RISK_KEYWORDS = re.compile(
    r"\b(?:treat|treatment|medicate|antibiotic|chemical|salt bath|formalin|harvest|cull|"
    r"drain|restock|suspend feeding for)\b",
    re.I,
)

#: Mortality in 24h that forces escalation regardless of anything else.
MORTALITY_ESCALATION_THRESHOLD = 10

#: Confidence below which a case with any health signal must be escalated.
LOW_CONFIDENCE_ESCALATION_THRESHOLD = 0.5


@dataclass
class SafetyOutcome:
    """The post-guardrail response shape."""

    risk_level: RiskLevel
    actions: list[RecommendedAction]
    expert_escalation: bool
    escalation_reasons: list[str]
    warnings: list[str]


def classify_action_tier(action_text: str, proposed: RecommendationTier) -> RecommendationTier:
    """Assign an approval tier, never below what the text warrants.

    The model proposes a tier; this raises it when the wording implies more
    risk than claimed. It never lowers one.
    """
    derived = RecommendationTier.TIER_1_ADVISORY

    if _HIGH_RISK_KEYWORDS.search(action_text) or any(
        pattern.search(action_text) for pattern in _ACTUATION_PATTERNS
    ):
        derived = RecommendationTier.TIER_3_HIGH_RISK
    elif _OPERATIONAL_KEYWORDS.search(action_text):
        derived = RecommendationTier.TIER_2_LOW_RISK_OPERATIONAL

    return max(proposed, derived, key=lambda tier: _TIER_ORDER[tier])


def requires_approval(tier: RecommendationTier) -> bool:
    """Tiers 0 and 1 propose no physical action, so they need no approval."""
    return _TIER_ORDER[tier] >= _TIER_ORDER[RecommendationTier.TIER_2_LOW_RISK_OPERATIONAL]


def enforce(
    *,
    answer: str,
    model_risk_level: RiskLevel,
    model_actions: list[RecommendedAction],
    model_escalation: bool,
    model_escalation_reasons: list[str],
    rule_findings: list[RuleFinding],
    confidence: float,
    mortality_24h: int | None,
    has_health_signal: bool,
) -> SafetyOutcome:
    """Apply every guardrail. Output is never less conservative than input."""
    warnings: list[str] = []
    escalation_reasons = list(model_escalation_reasons)

    # 1. Deterministic rules set a floor on risk that the model cannot lower.
    #    "AI output must never overwrite deterministic safety rules."
    risk_level = max(
        model_risk_level,
        _rule_risk_floor(rule_findings),
        key=lambda level: _RISK_ORDER[level],
    )

    # 2. Re-tier every action from its own text.
    actions: list[RecommendedAction] = []
    for action in model_actions:
        tier = classify_action_tier(f"{action.action} {action.reason}", action.tier)
        if tier != action.tier:
            warnings.append(
                f"Recommended action was reclassified to {tier.value} based on its content."
            )
        actions.append(
            RecommendedAction(
                action=action.action,
                tier=tier,
                reason=action.reason,
                requires_approval=requires_approval(tier),
                urgency=action.urgency,
            )
        )

    escalate = model_escalation

    # 3. Overclaiming is a warning, not a rewrite. Editing the answer text would
    #    make the stored response differ from what the model actually said,
    #    which breaks the provenance record. An answer that claims more
    #    certainty than the evidence supports is exactly the case a human
    #    should review, so it also escalates.
    for pattern, message in _OVERCLAIM_PATTERNS:
        if pattern.search(answer):
            warnings.append(message)
            escalation_reasons.append("Answer language exceeded the supportable level of certainty.")
            escalate = True
            risk_level = max(risk_level, RiskLevel.ELEVATED, key=lambda level: _RISK_ORDER[level])

    # 4. Escalation triggers (14_AQUADOC_SAFETY_AND_GOVERNANCE.md section 4).
    if mortality_24h is not None and mortality_24h >= MORTALITY_ESCALATION_THRESHOLD:
        escalate = True
        escalation_reasons.append(
            f"Reported mortality in the last 24 hours ({mortality_24h}) is at or above the "
            f"escalation threshold ({MORTALITY_ESCALATION_THRESHOLD})."
        )
        risk_level = max(risk_level, RiskLevel.HIGH, key=lambda level: _RISK_ORDER[level])

    if has_health_signal and confidence < LOW_CONFIDENCE_ESCALATION_THRESHOLD:
        escalate = True
        escalation_reasons.append(
            "Confidence is low for a case involving health signs; expert review is advised."
        )

    if any(action.tier is RecommendationTier.TIER_3_HIGH_RISK for action in actions):
        escalate = True
        escalation_reasons.append(
            "A high-risk action was proposed; it requires an explicit human decision."
        )

    if _RISK_ORDER[risk_level] >= _RISK_ORDER[RiskLevel.HIGH]:
        escalate = True
        escalation_reasons.append("The overall risk level is high.")

    return SafetyOutcome(
        risk_level=risk_level,
        actions=actions,
        expert_escalation=escalate,
        escalation_reasons=_dedupe(escalation_reasons),
        warnings=_dedupe(warnings),
    )


def _rule_risk_floor(findings: list[RuleFinding]) -> RiskLevel:
    """The minimum risk level the deterministic findings justify."""
    if any(finding.status == STATUS_CONCERN for finding in findings):
        return RiskLevel.ELEVATED
    if any(finding.status == STATUS_WATCH for finding in findings):
        return RiskLevel.WATCH
    return RiskLevel.INFORMATIONAL


def _dedupe(values: list[str]) -> list[str]:
    """Order-preserving deduplication."""
    seen: set[str] = set()
    result: list[str] = []
    for value in values:
        if value not in seen:
            seen.add(value)
            result.append(value)
    return result
