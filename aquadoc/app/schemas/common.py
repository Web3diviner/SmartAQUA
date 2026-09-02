"""Enumerations and value objects shared across the AquaDoc API."""

from __future__ import annotations

from enum import StrEnum


class Intent(StrEnum):
    """What kind of question the farmer asked.

    Drives metadata filtering and prompt selection (04_AQUADOC_RAG_LLM.md sections 7-8).
    """

    GENERAL_AQUACULTURE = "general_aquaculture"
    FARM_ASSESSMENT = "farm_assessment"
    FEEDING = "feeding"
    WATER_QUALITY = "water_quality"
    DISEASE = "disease"
    UNKNOWN = "unknown"


class RiskLevel(StrEnum):
    """How urgent the situation appears.

    Deliberately separate from RecommendationTier: risk describes the situation,
    tier describes what approval an action needs.
    """

    INFORMATIONAL = "informational"
    WATCH = "watch"
    ELEVATED = "elevated"
    HIGH = "high"


class RecommendationTier(StrEnum):
    """Approval requirement for a recommended action.

    14_AQUADOC_SAFETY_AND_GOVERNANCE.md section 5.
    """

    TIER_0_INFORMATIONAL = "tier_0_informational"
    TIER_1_ADVISORY = "tier_1_advisory"
    TIER_2_LOW_RISK_OPERATIONAL = "tier_2_low_risk_operational"
    TIER_3_HIGH_RISK = "tier_3_high_risk"


class ConfidenceBand(StrEnum):
    """Farmer-facing confidence label.

    15_AQUADOC_FRONTEND.md section 13: farmers see a band, developers may also
    see the numeric value. Do not imply false scientific precision.
    """

    LOW = "low"
    MODERATE = "moderate"
    HIGH = "high"


class EvidenceLevel(StrEnum):
    """04_AQUADOC_RAG_LLM.md section 5."""

    A_OFFICIAL_GUIDELINE = "A"
    B_PEER_REVIEWED = "B"
    C_TEXTBOOK = "C"
    D_VERIFIED_EXPERT_CASE = "D"
    E_USER_REPORT = "E"


class ReviewStatus(StrEnum):
    """Knowledge governance states (14_AQUADOC_SAFETY_AND_GOVERNANCE.md section 9).

    Only APPROVED documents may be retrieved in production.
    """

    PENDING = "pending"
    APPROVED = "approved"
    DEPRECATED = "deprecated"
    REJECTED = "rejected"


class MeasurementKey(StrEnum):
    """Water-quality measurements AquaDoc knows how to reason about.

    Used as the vocabulary for `missing_data`, so the frontend can label
    unavailable measurements consistently.
    """

    TEMPERATURE_C = "temperature_c"
    PH = "ph"
    DISSOLVED_OXYGEN_MG_L = "dissolved_oxygen_mg_l"
    TURBIDITY_NTU = "turbidity_ntu"
    AMMONIA_MG_L = "ammonia_mg_l"
    NITRITE_MG_L = "nitrite_mg_l"
    NITRATE_MG_L = "nitrate_mg_l"
    UN_IONIZED_AMMONIA_MG_L = "un_ionized_ammonia_mg_l"
    ORP_MV = "orp_mv"
    SALINITY_PPT = "salinity_ppt"
    TDS_PPM = "tds_ppm"
    WATER_LEVEL_CM = "water_level_cm"
    ALKALINITY_MG_L = "alkalinity_mg_l"
    HARDNESS_MG_L = "hardness_mg_l"


#: Human-readable labels for missing-data display.
MEASUREMENT_LABELS: dict[str, str] = {
    MeasurementKey.TEMPERATURE_C: "Temperature",
    MeasurementKey.PH: "pH",
    MeasurementKey.DISSOLVED_OXYGEN_MG_L: "Dissolved Oxygen",
    MeasurementKey.TURBIDITY_NTU: "Turbidity",
    MeasurementKey.AMMONIA_MG_L: "Total Ammonia (TAN)",
    MeasurementKey.NITRITE_MG_L: "Nitrite (NO2)",
    MeasurementKey.NITRATE_MG_L: "Nitrate (NO3)",
    MeasurementKey.UN_IONIZED_AMMONIA_MG_L: "Un-ionized Ammonia (NH3)",
    MeasurementKey.ORP_MV: "Redox / ORP",
    MeasurementKey.SALINITY_PPT: "Salinity",
    MeasurementKey.TDS_PPM: "Total Dissolved Solids (TDS)",
    MeasurementKey.WATER_LEVEL_CM: "Water Level",
    MeasurementKey.ALKALINITY_MG_L: "Alkalinity (CaCO3)",
    MeasurementKey.HARDNESS_MG_L: "Hardness (CaCO3)",
}


def confidence_band(score: float) -> ConfidenceBand:
    """Map a numeric confidence to a farmer-facing band.

    15_AQUADOC_FRONTEND.md section 13 initial mapping:
        0.00-0.49 low, 0.50-0.74 moderate, 0.75-1.00 high.
    Tune with evaluation data before changing these boundaries.
    """
    if score < 0.50:
        return ConfidenceBand.LOW
    if score < 0.75:
        return ConfidenceBand.MODERATE
    return ConfidenceBand.HIGH
