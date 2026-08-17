"""Safety guardrails.

These encode the non-negotiable design rules from 00_README.md and
14_AQUADOC_SAFETY_AND_GOVERNANCE.md. The property under test throughout is
one-directional: the guardrail layer can make a response more conservative,
never less. A model that returns confident, permissive output must not be able
to talk its way past these checks.
"""

from __future__ import annotations

from app.rules.safety import (
    MORTALITY_ESCALATION_THRESHOLD,
    classify_action_tier,
    enforce,
    requires_approval,
)
from app.rules.water_quality import STATUS_CONCERN, STATUS_OK, STATUS_WATCH
from app.schemas.chat import RecommendedAction, RuleFinding
from app.schemas.common import RecommendationTier, RiskLevel


def _finding(status: str, rule_id: str = "water_quality.ph") -> RuleFinding:
    return RuleFinding(
        rule_id=rule_id, rule_version="test/v1", status=status, summary="test finding"
    )


def _action(
    text: str,
    tier: RecommendationTier = RecommendationTier.TIER_0_INFORMATIONAL,
) -> RecommendedAction:
    return RecommendedAction(
        action=text, tier=tier, reason="because", requires_approval=False
    )


def _enforce(**overrides):
    defaults = dict(
        answer="A neutral answer.",
        model_risk_level=RiskLevel.INFORMATIONAL,
        model_actions=[],
        model_escalation=False,
        model_escalation_reasons=[],
        rule_findings=[],
        confidence=0.8,
        mortality_24h=None,
        has_health_signal=False,
    )
    return enforce(**{**defaults, **overrides})


class TestActionTiering:
    def test_treatment_advice_is_high_risk(self) -> None:
        tier = classify_action_tier(
            "Apply a salt bath treatment to the affected fish",
            RecommendationTier.TIER_0_INFORMATIONAL,
        )
        assert tier is RecommendationTier.TIER_3_HIGH_RISK

    def test_ration_change_is_operational(self) -> None:
        tier = classify_action_tier(
            "Reduce the daily ration by 20%", RecommendationTier.TIER_0_INFORMATIONAL
        )
        assert tier is RecommendationTier.TIER_2_LOW_RISK_OPERATIONAL

    def test_observation_stays_advisory(self) -> None:
        tier = classify_action_tier(
            "Inspect the pond and check sensor calibration",
            RecommendationTier.TIER_0_INFORMATIONAL,
        )
        assert tier is RecommendationTier.TIER_1_ADVISORY

    def test_tier_is_never_lowered_below_what_the_model_claimed(self) -> None:
        """A model may over-classify risk. That judgement is respected."""
        tier = classify_action_tier(
            "Observe the fish at feeding time", RecommendationTier.TIER_3_HIGH_RISK
        )
        assert tier is RecommendationTier.TIER_3_HIGH_RISK

    def test_command_like_phrasing_escalates_to_high_risk(self) -> None:
        """AquaDoc proposes; the platform commands.

        Anything phrased as actuation is forced to a tier requiring an explicit
        human decision.
        """
        for text in (
            "I will start the feeder now",
            "Triggering the auger to dispense feed",
            "Send a command to the device",
        ):
            assert (
                classify_action_tier(text, RecommendationTier.TIER_0_INFORMATIONAL)
                is RecommendationTier.TIER_3_HIGH_RISK
            ), text

    def test_approval_required_only_for_physical_action(self) -> None:
        assert not requires_approval(RecommendationTier.TIER_0_INFORMATIONAL)
        assert not requires_approval(RecommendationTier.TIER_1_ADVISORY)
        assert requires_approval(RecommendationTier.TIER_2_LOW_RISK_OPERATIONAL)
        assert requires_approval(RecommendationTier.TIER_3_HIGH_RISK)


class TestDeterministicRulesOverrideTheModel:
    def test_rule_concern_raises_risk_the_model_understated(self) -> None:
        """"AI output must never overwrite deterministic safety rules." """
        outcome = _enforce(
            model_risk_level=RiskLevel.INFORMATIONAL,
            rule_findings=[_finding(STATUS_CONCERN)],
        )
        assert outcome.risk_level is RiskLevel.ELEVATED

    def test_rule_watch_raises_risk_to_watch(self) -> None:
        outcome = _enforce(rule_findings=[_finding(STATUS_WATCH)])
        assert outcome.risk_level is RiskLevel.WATCH

    def test_model_risk_is_kept_when_higher_than_the_rules(self) -> None:
        outcome = _enforce(
            model_risk_level=RiskLevel.HIGH,
            rule_findings=[_finding(STATUS_OK)],
        )
        assert outcome.risk_level is RiskLevel.HIGH


class TestOverclaimDetection:
    def test_certainty_claim_is_flagged_and_escalated(self) -> None:
        outcome = _enforce(answer="Your fish definitely have columnaris.")

        assert outcome.warnings
        assert outcome.expert_escalation
        assert outcome.risk_level is not RiskLevel.INFORMATIONAL

    def test_confirmed_diagnosis_language_is_flagged(self) -> None:
        outcome = _enforce(answer="This is a confirmed diagnosis of columnaris.")
        assert outcome.warnings

    def test_lab_confirmation_claim_is_flagged(self) -> None:
        outcome = _enforce(answer="These are lab-confirmed results.")
        assert outcome.warnings

    def test_hedged_language_passes_clean(self) -> None:
        """The wording 14_AQUADOC_SAFETY_AND_GOVERNANCE.md section 2 prefers."""
        outcome = _enforce(
            answer="The signs are consistent with columnaris, but other causes remain possible."
        )
        assert not outcome.warnings
        assert not outcome.expert_escalation


class TestEscalation:
    def test_high_mortality_forces_escalation(self) -> None:
        outcome = _enforce(
            mortality_24h=MORTALITY_ESCALATION_THRESHOLD,
            has_health_signal=True,
        )
        assert outcome.expert_escalation
        assert outcome.risk_level is RiskLevel.HIGH

    def test_low_confidence_on_a_health_case_escalates(self) -> None:
        outcome = _enforce(confidence=0.3, has_health_signal=True)
        assert outcome.expert_escalation

    def test_low_confidence_without_health_signal_does_not_escalate(self) -> None:
        """An uncertain educational answer is not a case for a veterinarian."""
        outcome = _enforce(confidence=0.3, has_health_signal=False)
        assert not outcome.expert_escalation

    def test_high_risk_action_forces_escalation(self) -> None:
        outcome = _enforce(model_actions=[_action("Begin antibiotic treatment")])

        assert outcome.expert_escalation
        assert outcome.actions[0].tier is RecommendationTier.TIER_3_HIGH_RISK
        assert outcome.actions[0].requires_approval

    def test_escalation_reasons_are_deduplicated(self) -> None:
        outcome = _enforce(
            mortality_24h=50,
            has_health_signal=True,
            model_escalation_reasons=["The overall risk level is high."],
        )
        assert len(outcome.escalation_reasons) == len(set(outcome.escalation_reasons))


class TestAnswerTextIsNeverRewritten:
    def test_overclaiming_answer_is_flagged_not_edited(self) -> None:
        """Editing the text would break the provenance record.

        The stored answer must be what the model actually said, so an overclaim
        is surfaced as a warning rather than silently rewritten.
        """
        answer = "Your fish definitely have columnaris."
        outcome = _enforce(answer=answer)

        assert outcome.warnings
        # `enforce` returns no modified answer; the caller keeps the original.
        assert not hasattr(outcome, "answer")
