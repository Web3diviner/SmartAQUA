"""The missing-data policy.

04_AQUADOC_RAG_LLM.md section 9 and 15_AQUADOC_FRONTEND.md section 14: an
unmeasured value stays unknown. It is never inferred, defaulted, or rendered
as 0.

This is the rule most likely to be broken by a well-meaning refactor — a
`or 0` here, a `getattr(..., 0.0)` there — so it is tested at every layer that
touches a measurement.
"""

from __future__ import annotations

import json

import pytest

from app.orchestration.context_builder import (
    build_missing_block,
    build_pond_state_block,
    missing_measurement_keys,
)
from app.rules.water_quality import STATUS_OK, STATUS_UNKNOWN, evaluate_water_quality
from app.schemas.common import MeasurementKey
from app.schemas.farm_context import FarmContext, WaterQuality


class TestWaterQualityModel:
    def test_unmeasured_parameters_are_none_not_zero(self) -> None:
        water = WaterQuality(temperature_c=29.4)

        assert water.temperature_c == 29.4
        assert water.ph is None
        assert water.dissolved_oxygen_mg_l is None
        assert water.turbidity_ntu is None

    def test_missing_lists_every_unmeasured_key(self, current_pond_context: FarmContext) -> None:
        missing = current_pond_context.water.missing()

        assert MeasurementKey.TEMPERATURE_C not in missing
        assert MeasurementKey.PH in missing
        assert MeasurementKey.DISSOLVED_OXYGEN_MG_L in missing
        assert MeasurementKey.TURBIDITY_NTU in missing

    def test_available_excludes_unmeasured(self, current_pond_context: FarmContext) -> None:
        available = current_pond_context.water.available()

        assert available == {"temperature_c": 29.4}

    def test_zero_is_a_real_measurement_not_a_missing_one(self) -> None:
        """0.0 mg/L dissolved oxygen is a catastrophic reading, not an absent one.

        Any truthiness check (`if value:`) would silently reclassify it as
        missing and hide a pond-killing event.
        """
        water = WaterQuality(dissolved_oxygen_mg_l=0.0)

        assert water.dissolved_oxygen_mg_l == 0.0
        assert MeasurementKey.DISSOLVED_OXYGEN_MG_L not in water.missing()
        assert water.available()["dissolved_oxygen_mg_l"] == 0.0


class TestDerivedValues:
    def test_biomass_is_none_when_inputs_unknown(self) -> None:
        assert FarmContext(population=500).derived_biomass_kg() is None
        assert FarmContext(average_weight_g=250.0).derived_biomass_kg() is None

    def test_biomass_computed_when_both_known(self) -> None:
        context = FarmContext(population=500, average_weight_g=250.0)

        assert context.derived_biomass_kg() == pytest.approx(125.0)


class TestWaterQualityRules:
    def test_unmeasured_parameter_yields_unknown_never_ok(
        self, current_pond_context: FarmContext
    ) -> None:
        findings = {
            finding.measurement: finding
            for finding in evaluate_water_quality(current_pond_context.water)
        }

        assert findings["temperature_c"].status == STATUS_OK
        for key in ("ph", "dissolved_oxygen_mg_l", "turbidity_ntu"):
            assert findings[key].status == STATUS_UNKNOWN, f"{key} must not be assessed"
            assert findings[key].observed is None

    def test_every_parameter_reported_when_nothing_measured(
        self, empty_context: FarmContext
    ) -> None:
        """Silence about a parameter would read as "fine". Every one is named."""
        findings = evaluate_water_quality(empty_context.water)

        assert len(findings) == len(MeasurementKey)
        assert all(finding.status == STATUS_UNKNOWN for finding in findings)


class TestPromptSerialisation:
    def test_unknown_measurements_serialise_as_null(
        self, current_pond_context: FarmContext
    ) -> None:
        """The model must see explicit nulls, not absent keys.

        An omitted key reads as "not applicable"; `null` reads as "not measured".
        """
        block = build_pond_state_block(current_pond_context)
        payload = json.loads(block.split(">", 1)[1].rsplit("<", 1)[0])

        assert payload["water"]["temperature_c"] == 29.4
        assert payload["water"]["ph"] is None
        assert "ph" in payload["water"]
        assert payload["water"]["dissolved_oxygen_mg_l"] is None

    def test_missing_block_names_parameters_in_plain_language(
        self, current_pond_context: FarmContext
    ) -> None:
        block = build_missing_block(current_pond_context)

        assert "pH" in block
        assert "Dissolved Oxygen" in block
        assert "Temperature" not in block

    def test_missing_block_when_no_context_supplied(self) -> None:
        assert "all" in build_missing_block(None)

    def test_missing_keys_reported_for_absent_context(self) -> None:
        keys, labels = missing_measurement_keys(None)

        assert len(keys) == len(MeasurementKey)
        assert "Dissolved Oxygen" in labels
