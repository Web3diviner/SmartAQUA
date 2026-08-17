"""Farm context contract.

The Go backend computes and supplies this (05_API_AND_SERVICE_CONTRACTS.md,
"Context Contract"). AquaDoc consumes a computed pond state, not raw database
rows (01_SYSTEM_ARCHITECTURE.md section 8).

The single most important rule in this module: a measurement that is `None` is
UNKNOWN. It must never be coerced to 0, defaulted to a "typical" value, or
treated as normal (04_AQUADOC_RAG_LLM.md section 9).
"""

from __future__ import annotations

from datetime import datetime

from pydantic import BaseModel, ConfigDict, Field

from app.schemas.common import MEASUREMENT_LABELS, MeasurementKey


class WaterQuality(BaseModel):
    """Water measurements. `None` means "not measured", never "fine"."""

    model_config = ConfigDict(extra="forbid")

    temperature_c: float | None = Field(default=None, ge=-5, le=60)
    ph: float | None = Field(default=None, ge=0, le=14)
    dissolved_oxygen_mg_l: float | None = Field(default=None, ge=0, le=30)
    turbidity_ntu: float | None = Field(default=None, ge=0, le=5000)
    ammonia_mg_l: float | None = Field(default=None, ge=0, le=100)
    nitrite_mg_l: float | None = Field(default=None, ge=0, le=100)
    measured_at: datetime | None = None

    def available(self) -> dict[str, float]:
        """Measurements that were actually taken."""
        return {
            key: value
            for key in MeasurementKey
            if isinstance(value := getattr(self, key, None), (int, float))
        }

    def missing(self) -> list[str]:
        """Measurement keys with no value, in a stable declaration order."""
        return [key.value for key in MeasurementKey if getattr(self, key, None) is None]

    def missing_labels(self) -> list[str]:
        return [MEASUREMENT_LABELS[key] for key in self.missing()]


class FeedingContext(BaseModel):
    model_config = ConfigDict(extra="forbid")

    daily_ration_g: float | None = Field(default=None, ge=0)
    last_feeding_at: datetime | None = None
    last_feeding_g: float | None = Field(default=None, ge=0)
    feeds_per_day: int | None = Field(default=None, ge=0, le=24)
    feed_acceptance: str | None = Field(
        default=None,
        description="Farmer-observed acceptance, e.g. 'normal', 'reduced', 'refused'.",
    )


class HealthContext(BaseModel):
    model_config = ConfigDict(extra="forbid")

    mortality_24h: int | None = Field(default=None, ge=0)
    mortality_7d: int | None = Field(default=None, ge=0)
    active_disease_case: bool | None = None
    reported_symptoms: list[str] = Field(default_factory=list)


class FarmContext(BaseModel):
    """Computed pond state supplied by the Go backend, or simulated in dev."""

    model_config = ConfigDict(extra="forbid")

    farm_id: str | None = None
    pond_id: str | None = None
    production_cycle_id: str | None = None
    farm_name: str | None = None
    pond_name: str | None = None
    species: str | None = None
    life_stage: str | None = None
    population: int | None = Field(default=None, ge=0)
    average_weight_g: float | None = Field(default=None, ge=0)
    biomass_kg: float | None = Field(default=None, ge=0)
    pond_volume_liters: float | None = Field(default=None, ge=0)
    water: WaterQuality = Field(default_factory=WaterQuality)
    feeding: FeedingContext = Field(default_factory=FeedingContext)
    health: HealthContext = Field(default_factory=HealthContext)

    def derived_biomass_kg(self) -> float | None:
        """Biomass from population x average weight, when not supplied directly.

        Returns None rather than guessing when either input is unknown.
        """
        if self.biomass_kg is not None:
            return self.biomass_kg
        if self.population is None or self.average_weight_g is None:
            return None
        return round(self.population * self.average_weight_g / 1000.0, 3)

    def completeness(self) -> float:
        """Fraction of the context fields that matter for assessment and are known.

        Feeds the confidence calculation — an assessment made on thin data should
        not report high confidence (14_AQUADOC_SAFETY_AND_GOVERNANCE.md section 3).
        """
        signals = [
            self.species is not None,
            self.population is not None,
            self.average_weight_g is not None or self.biomass_kg is not None,
            self.water.temperature_c is not None,
            self.water.ph is not None,
            self.water.dissolved_oxygen_mg_l is not None,
            self.feeding.daily_ration_g is not None,
            self.health.mortality_24h is not None,
        ]
        return round(sum(signals) / len(signals), 4)

    def is_empty(self) -> bool:
        """True when no usable farm signal was supplied at all."""
        return (
            self.species is None
            and self.population is None
            and self.biomass_kg is None
            and not self.water.available()
            and self.feeding.daily_ration_g is None
            and self.health.mortality_24h is None
        )
