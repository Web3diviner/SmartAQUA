"""Deterministic feeding calculations, including the Q10 temperature response.

01_SYSTEM_ARCHITECTURE.md places Q10 in the Decision Brain — deterministic
rules, not the language model. AquaDoc computes it so it can *explain* feeding
behaviour; the Go backend remains the authority for actual ration decisions and
the ESP32 keeps its own safety interlocks.

No function here emits a device command.
"""

from __future__ import annotations

from dataclasses import dataclass

from app.schemas.chat import RuleFinding
from app.schemas.farm_context import FarmContext

RULES_VERSION = "feeding/v1"

#: Q10 coefficient: the factor by which metabolic rate changes per 10 degC.
#: 2.0 is the conventional starting value for warm-water finfish. Calibrate
#: against real feeding-response data before treating it as precise.
Q10_COEFFICIENT = 2.0

#: Temperature the ration baseline is defined at.
REFERENCE_TEMPERATURE_C = 28.0

#: Feeding rate as a percentage of biomass per day, by average weight class.
#: Smaller fish eat proportionally more.
FEED_RATE_BY_WEIGHT: tuple[tuple[float, float], ...] = (
    (5.0, 0.10),
    (20.0, 0.07),
    (50.0, 0.05),
    (150.0, 0.035),
    (300.0, 0.025),
    (float("inf"), 0.018),
)

#: Below this, appetite is suppressed enough that normal rations risk waste.
APPETITE_SUPPRESSION_BELOW_C = 22.0
#: Above this, oxygen demand rises enough to make heavy feeding risky.
HEAT_STRESS_ABOVE_C = 32.0


@dataclass(frozen=True)
class FeedingAssessment:
    """Computed feeding picture. `None` fields mean "not computable"."""

    biomass_kg: float | None
    baseline_rate_fraction: float | None
    baseline_ration_g: float | None
    q10_factor: float | None
    temperature_adjusted_ration_g: float | None
    observed_ration_g: float | None
    ration_ratio: float | None
    findings: list[RuleFinding]


def q10_factor(temperature_c: float, *, reference_c: float = REFERENCE_TEMPERATURE_C) -> float:
    """Metabolic scaling factor relative to the reference temperature.

    factor = Q10 ** ((T - T_ref) / 10)
    """
    return float(Q10_COEFFICIENT ** ((temperature_c - reference_c) / 10.0))


def feed_rate_fraction(average_weight_g: float) -> float:
    """Daily feed as a fraction of biomass, from the weight class table."""
    for upper_bound, rate in FEED_RATE_BY_WEIGHT:
        if average_weight_g <= upper_bound:
            return rate
    return FEED_RATE_BY_WEIGHT[-1][1]


def assess_feeding(context: FarmContext) -> FeedingAssessment:
    """Compute the feeding picture from whatever context is available.

    Every intermediate value is independently optional. A missing average weight
    means no baseline ration — not a guessed one.
    """
    findings: list[RuleFinding] = []
    biomass_kg = context.derived_biomass_kg()
    temperature_c = context.water.temperature_c
    average_weight_g = context.average_weight_g
    observed_ration_g = context.feeding.daily_ration_g

    baseline_rate: float | None = None
    baseline_ration_g: float | None = None
    if average_weight_g is not None:
        baseline_rate = feed_rate_fraction(average_weight_g)
        if biomass_kg is not None:
            baseline_ration_g = round(biomass_kg * 1000.0 * baseline_rate, 1)

    factor: float | None = None
    adjusted_ration_g: float | None = None
    if temperature_c is None:
        findings.append(
            RuleFinding(
                rule_id="feeding.q10",
                rule_version=RULES_VERSION,
                status="unknown",
                summary=(
                    "Water temperature is not available, so the Q10 metabolic adjustment "
                    "cannot be calculated."
                ),
                measurement="temperature_c",
            )
        )
    else:
        factor = round(q10_factor(temperature_c), 4)
        if baseline_ration_g is not None:
            adjusted_ration_g = round(baseline_ration_g * factor, 1)
        findings.append(
            RuleFinding(
                rule_id="feeding.q10",
                rule_version=RULES_VERSION,
                status="ok",
                summary=(
                    f"At {temperature_c:g} degC the Q10 factor is {factor:g} relative to the "
                    f"{REFERENCE_TEMPERATURE_C:g} degC reference "
                    f"(Q10={Q10_COEFFICIENT:g}), so expected intake scales by that factor."
                ),
                measurement="temperature_c",
                observed=float(temperature_c),
            )
        )
        findings.extend(_temperature_appetite_findings(temperature_c))

    ration_ratio: float | None = None
    if observed_ration_g is not None and adjusted_ration_g:
        ration_ratio = round(observed_ration_g / adjusted_ration_g, 3)
        findings.append(_ration_comparison_finding(observed_ration_g, adjusted_ration_g, ration_ratio))
    elif observed_ration_g is None:
        findings.append(
            RuleFinding(
                rule_id="feeding.ration_comparison",
                rule_version=RULES_VERSION,
                status="unknown",
                summary="No daily ration was supplied, so actual feeding cannot be compared.",
            )
        )
    elif baseline_ration_g is None:
        findings.append(
            RuleFinding(
                rule_id="feeding.ration_comparison",
                rule_version=RULES_VERSION,
                status="unknown",
                summary=(
                    "Population and average weight are needed to compute an expected ration; "
                    "the reported ration cannot be compared."
                ),
            )
        )

    if context.feeding.feed_acceptance:
        findings.append(_acceptance_finding(context.feeding.feed_acceptance))

    return FeedingAssessment(
        biomass_kg=biomass_kg,
        baseline_rate_fraction=baseline_rate,
        baseline_ration_g=baseline_ration_g,
        q10_factor=factor,
        temperature_adjusted_ration_g=adjusted_ration_g,
        observed_ration_g=observed_ration_g,
        ration_ratio=ration_ratio,
        findings=findings,
    )


def _temperature_appetite_findings(temperature_c: float) -> list[RuleFinding]:
    if temperature_c < APPETITE_SUPPRESSION_BELOW_C:
        return [
            RuleFinding(
                rule_id="feeding.appetite_temperature",
                rule_version=RULES_VERSION,
                status="concern",
                summary=(
                    f"At {temperature_c:g} degC appetite is suppressed. Reduced intake is an "
                    f"expected thermal response rather than a sign of disease on its own."
                ),
                measurement="temperature_c",
                observed=float(temperature_c),
                expected_range=(APPETITE_SUPPRESSION_BELOW_C, HEAT_STRESS_ABOVE_C),
            )
        ]
    if temperature_c > HEAT_STRESS_ABOVE_C:
        return [
            RuleFinding(
                rule_id="feeding.appetite_temperature",
                rule_version=RULES_VERSION,
                status="concern",
                summary=(
                    f"At {temperature_c:g} degC thermal stress is likely and oxygen demand is "
                    f"elevated. Heavy feeding compounds the oxygen load."
                ),
                measurement="temperature_c",
                observed=float(temperature_c),
                expected_range=(APPETITE_SUPPRESSION_BELOW_C, HEAT_STRESS_ABOVE_C),
            )
        ]
    return []


def _ration_comparison_finding(observed_g: float, expected_g: float, ratio: float) -> RuleFinding:
    if ratio < 0.6:
        status = "watch"
        note = "Substantially below the temperature-adjusted expectation."
    elif ratio > 1.5:
        status = "concern"
        note = "Substantially above the temperature-adjusted expectation; risk of feed waste."
    else:
        status = "ok"
        note = "Broadly consistent with the temperature-adjusted expectation."

    return RuleFinding(
        rule_id="feeding.ration_comparison",
        rule_version=RULES_VERSION,
        status=status,
        summary=(
            f"Reported ration {observed_g:g} g/day against a temperature-adjusted "
            f"expectation of {expected_g:g} g/day (ratio {ratio:g}). {note}"
        ),
        observed=float(observed_g),
    )


def _acceptance_finding(acceptance: str) -> RuleFinding:
    normalised = acceptance.strip().lower()
    status = "concern" if normalised in {"refused", "none", "poor"} else (
        "watch" if normalised in {"reduced", "low", "slow"} else "ok"
    )
    return RuleFinding(
        rule_id="feeding.acceptance",
        rule_version=RULES_VERSION,
        status=status,
        summary=f"Farmer-reported feed acceptance: {acceptance}.",
    )
