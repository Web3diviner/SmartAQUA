"""Deterministic water-quality assessment.

13_CODING_AND_ENGINEERING_STANDARDS.md: deterministic logic must be testable
without AI. Nothing in this module calls a model.

The central rule: a measurement that was not taken produces a finding with
status `unknown`. It never produces `ok`. Silence is not evidence of safety
(04_AQUADOC_RAG_LLM.md section 9).
"""

from __future__ import annotations

from dataclasses import dataclass

from app.schemas.chat import RuleFinding
from app.schemas.common import MEASUREMENT_LABELS, MeasurementKey
from app.schemas.farm_context import WaterQuality

RULES_VERSION = "water_quality/v1"

STATUS_OK = "ok"
STATUS_WATCH = "watch"
STATUS_CONCERN = "concern"
STATUS_UNKNOWN = "unknown"


@dataclass(frozen=True)
class Threshold:
    """Comfort and tolerance bands for one measurement.

    `comfort` is the range where no action is indicated. Outside `tolerance` is
    a concern. Between the two is worth watching.

    These are general warm-water freshwater bands (the current deployment is
    African catfish, Clarias gariepinus). Species-specific tables belong here
    when Stage 8 adds more sensors and more species.
    """

    key: MeasurementKey
    comfort: tuple[float, float]
    tolerance: tuple[float, float]
    unit: str
    note_low: str
    note_high: str


THRESHOLDS: tuple[Threshold, ...] = (
    Threshold(
        key=MeasurementKey.TEMPERATURE_C,
        comfort=(26.0, 30.0),
        tolerance=(22.0, 32.0),
        unit="degC",
        note_low="Below the comfort band; metabolism and feed intake fall as temperature drops.",
        note_high="Above the comfort band; oxygen solubility falls as temperature rises.",
    ),
    Threshold(
        key=MeasurementKey.PH,
        comfort=(6.5, 8.5),
        tolerance=(6.0, 9.0),
        unit="pH",
        note_low="Acidic relative to the comfort band.",
        note_high="Alkaline relative to the comfort band; raises un-ionised ammonia toxicity.",
    ),
    Threshold(
        key=MeasurementKey.DISSOLVED_OXYGEN_MG_L,
        comfort=(5.0, 12.0),
        tolerance=(3.0, 15.0),
        unit="mg/L",
        note_low="Below the comfort band; low dissolved oxygen suppresses feeding first.",
        note_high="Unusually high; check for supersaturation and sensor calibration.",
    ),
    Threshold(
        key=MeasurementKey.TURBIDITY_NTU,
        comfort=(0.0, 30.0),
        tolerance=(0.0, 80.0),
        unit="NTU",
        note_low="Below the comfort band.",
        note_high="High turbidity; check suspended solids, algal load, and feed waste.",
    ),
    Threshold(
        key=MeasurementKey.AMMONIA_MG_L,
        comfort=(0.0, 0.5),
        tolerance=(0.0, 1.0),
        unit="mg/L",
        note_low="Below the comfort band.",
        note_high="Elevated total ammonia (TAN); toxicity rises sharply with pH and temperature.",
    ),
    Threshold(
        key=MeasurementKey.NITRITE_MG_L,
        comfort=(0.0, 0.2),
        tolerance=(0.0, 0.5),
        unit="mg/L",
        note_low="Below the comfort band.",
        note_high="Elevated nitrite (NO2); causes brown blood disease by oxidizing hemoglobin.",
    ),
    Threshold(
        key=MeasurementKey.NITRATE_MG_L,
        comfort=(0.0, 50.0),
        tolerance=(0.0, 100.0),
        unit="mg/L",
        note_low="Below the comfort band.",
        note_high="Elevated nitrate (NO3); indicates biofilter buildup or overdue water exchange.",
    ),
    Threshold(
        key=MeasurementKey.UN_IONIZED_AMMONIA_MG_L,
        comfort=(0.0, 0.02),
        tolerance=(0.0, 0.05),
        unit="mg/L",
        note_low="Below the comfort band.",
        note_high="Lethal un-ionized ammonia (NH3) toxicity; causes gill hyperplasia and acute mortality.",
    ),
    Threshold(
        key=MeasurementKey.ORP_MV,
        comfort=(250.0, 400.0),
        tolerance=(200.0, 450.0),
        unit="mV",
        note_low="Low ORP indicates high organic waste, low dissolved oxygen, or failing biofilter.",
        note_high="Very high ORP; verify ozone generator dosing or sensor calibration.",
    ),
    Threshold(
        key=MeasurementKey.SALINITY_PPT,
        comfort=(0.5, 3.0),
        tolerance=(0.0, 5.0),
        unit="ppt",
        note_low="Near-zero salinity.",
        note_high="Elevated salinity for freshwater stock; confirm treatment bath concentration.",
    ),
    Threshold(
        key=MeasurementKey.TDS_PPM,
        comfort=(100.0, 1000.0),
        tolerance=(50.0, 2000.0),
        unit="ppm",
        note_low="Low mineral content.",
        note_high="High total dissolved solids; check mineral accumulation or salinity.",
    ),
    Threshold(
        key=MeasurementKey.WATER_LEVEL_CM,
        comfort=(60.0, 150.0),
        tolerance=(40.0, 180.0),
        unit="cm",
        note_low="Water level below safe threshold; risk of overcrowding and oxygen depletion.",
        note_high="Water level near overflow line; risk of fish escape or flood.",
    ),
    Threshold(
        key=MeasurementKey.ALKALINITY_MG_L,
        comfort=(80.0, 200.0),
        tolerance=(50.0, 300.0),
        unit="mg/L",
        note_low="Low alkalinity reduces pH buffering capacity; high risk of fatal nighttime pH crash.",
        note_high="High alkalinity.",
    ),
    Threshold(
        key=MeasurementKey.HARDNESS_MG_L,
        comfort=(50.0, 150.0),
        tolerance=(30.0, 250.0),
        unit="mg/L",
        note_low="Low water hardness (soft water); can impair egg development and bone mineralization.",
        note_high="Very hard water.",
    ),
)

THRESHOLDS_BY_KEY: dict[MeasurementKey, Threshold] = {t.key: t for t in THRESHOLDS}


def evaluate_water_quality(water: WaterQuality) -> list[RuleFinding]:
    """Evaluate every known measurement. One finding per measurement.

    Measurements that were not taken yield `unknown`, so the caller can tell the
    farmer explicitly what could not be assessed rather than omitting it.
    """
    findings: list[RuleFinding] = []

    for threshold in THRESHOLDS:
        label = MEASUREMENT_LABELS[threshold.key]
        value = getattr(water, threshold.key.value, None)

        if value is None:
            findings.append(
                RuleFinding(
                    rule_id=f"water_quality.{threshold.key.value}",
                    rule_version=RULES_VERSION,
                    status=STATUS_UNKNOWN,
                    summary=f"{label} is not currently measured and cannot be evaluated.",
                    measurement=threshold.key.value,
                    observed=None,
                    expected_range=threshold.comfort,
                )
            )
            continue

        comfort_low, comfort_high = threshold.comfort
        tolerance_low, tolerance_high = threshold.tolerance

        if comfort_low <= value <= comfort_high:
            status, note = STATUS_OK, "Within the comfort band."
        elif value < tolerance_low or value > tolerance_high:
            status = STATUS_CONCERN
            note = threshold.note_low if value < tolerance_low else threshold.note_high
        else:
            status = STATUS_WATCH
            note = threshold.note_low if value < comfort_low else threshold.note_high

        findings.append(
            RuleFinding(
                rule_id=f"water_quality.{threshold.key.value}",
                rule_version=RULES_VERSION,
                status=status,
                summary=(
                    f"{label} is {value:g} {threshold.unit} "
                    f"(comfort {comfort_low:g}-{comfort_high:g}). {note}"
                ),
                measurement=threshold.key.value,
                observed=float(value),
                expected_range=threshold.comfort,
            )
        )

    return findings


def worst_status(findings: list[RuleFinding]) -> str:
    """The most severe status present, ignoring `unknown`.

    `unknown` is not ranked as severity: it means "not assessed", which the
    caller surfaces separately via missing-data reporting.
    """
    order = {STATUS_OK: 0, STATUS_WATCH: 1, STATUS_CONCERN: 2}
    severities = [order[f.status] for f in findings if f.status in order]
    if not severities:
        return STATUS_UNKNOWN
    return {0: STATUS_OK, 1: STATUS_WATCH, 2: STATUS_CONCERN}[max(severities)]
