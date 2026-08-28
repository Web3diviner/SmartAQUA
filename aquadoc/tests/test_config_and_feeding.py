"""Production configuration gates and deterministic feeding maths."""

from __future__ import annotations

import pytest
from pydantic import ValidationError as PydanticValidationError

from app.config import Settings
from app.rules.feeding import (
    Q10_COEFFICIENT,
    REFERENCE_TEMPERATURE_C,
    assess_feeding,
    feed_rate_fraction,
    q10_factor,
)
from app.schemas.farm_context import FarmContext, FeedingContext, WaterQuality


class TestProductionGates:
    """07_SECURITY_ARCHITECTURE.md section 12 — refuse to boot, don't warn."""

    def _production(self, **overrides):
        base = dict(
            app_env="production",
            aquadoc_dev_token="",
            aquadoc_internal_service_secret="x" * 40,
            llm_provider="claude",
            anthropic_api_key="sk-test",
            embedding_provider="voyage",
            voyage_api_key="vk-test",
            _env_file=None,
        )
        return Settings(**{**base, **overrides})

    def test_valid_production_config_boots(self) -> None:
        settings = self._production()

        assert settings.is_production
        assert not settings.dev_routes_enabled

    def test_dev_token_in_production_is_refused(self) -> None:
        """The dev token must be "impossible to use in production"."""
        with pytest.raises(PydanticValidationError, match="AQUADOC_DEV_TOKEN"):
            self._production(aquadoc_dev_token="anything")

    def test_missing_service_secret_is_refused(self) -> None:
        with pytest.raises(PydanticValidationError, match="INTERNAL_SERVICE_SECRET"):
            self._production(aquadoc_internal_service_secret="")

    def test_weak_service_secret_is_refused(self) -> None:
        with pytest.raises(PydanticValidationError, match="at least 32"):
            self._production(aquadoc_internal_service_secret="short")

    def test_stub_llm_provider_cannot_reach_production(self) -> None:
        """The echo provider does not reason; shipping it would be silent harm."""
        with pytest.raises(PydanticValidationError, match="echo"):
            self._production(llm_provider="echo")

    def test_stub_embedding_provider_cannot_reach_production(self) -> None:
        with pytest.raises(PydanticValidationError, match="hashing"):
            self._production(embedding_provider="hashing")

    def test_claude_provider_requires_an_api_key(self) -> None:
        with pytest.raises(PydanticValidationError, match="ANTHROPIC_API_KEY"):
            self._production(anthropic_api_key="")

    def test_development_permits_the_offline_stubs(self) -> None:
        settings = Settings(
            app_env="development",
            aquadoc_dev_token="dev",
            llm_provider="echo",
            embedding_provider="hashing",
        )

        assert settings.dev_routes_enabled
        assert settings.enable_dev_api

    def test_dev_routes_need_a_token_even_in_development(self) -> None:
        """No token means no developer surface, not an open one."""
        settings = Settings(app_env="development", aquadoc_dev_token="")

        assert not settings.dev_routes_enabled

    def test_chunk_overlap_must_be_smaller_than_target(self) -> None:
        with pytest.raises(PydanticValidationError):
            Settings(chunk_target_tokens=500, chunk_overlap_tokens=500)

    def test_cors_origins_parse_to_a_list(self) -> None:
        settings = Settings(cors_allow_origins="http://a.test, http://b.test")

        assert settings.cors_origins == ["http://a.test", "http://b.test"]


class TestQ10:
    """Q10 is deterministic platform maths, not model output."""

    def test_factor_is_one_at_the_reference_temperature(self) -> None:
        assert q10_factor(REFERENCE_TEMPERATURE_C) == pytest.approx(1.0)

    def test_ten_degrees_above_reference_scales_by_q10(self) -> None:
        assert q10_factor(REFERENCE_TEMPERATURE_C + 10) == pytest.approx(Q10_COEFFICIENT)

    def test_ten_degrees_below_reference_halves_the_rate(self) -> None:
        assert q10_factor(REFERENCE_TEMPERATURE_C - 10) == pytest.approx(1 / Q10_COEFFICIENT)

    def test_factor_increases_monotonically_with_temperature(self) -> None:
        temperatures = [20.0, 24.0, 28.0, 32.0]
        factors = [q10_factor(t) for t in temperatures]

        assert factors == sorted(factors)

    def test_smaller_fish_eat_a_larger_fraction_of_biomass(self) -> None:
        assert feed_rate_fraction(3.0) > feed_rate_fraction(100.0) > feed_rate_fraction(500.0)


class TestFeedingAssessment:
    def test_missing_temperature_blocks_q10_rather_than_guessing(self) -> None:
        context = FarmContext(population=500, average_weight_g=250.0)
        assessment = assess_feeding(context)

        assert assessment.q10_factor is None
        assert assessment.temperature_adjusted_ration_g is None
        q10_finding = next(f for f in assessment.findings if f.rule_id == "feeding.q10")
        assert q10_finding.status == "unknown"

    def test_missing_weight_blocks_the_baseline_ration(self) -> None:
        context = FarmContext(population=500, water=WaterQuality(temperature_c=29.4))
        assessment = assess_feeding(context)

        assert assessment.baseline_ration_g is None

    def test_full_context_produces_a_comparison(self, current_pond_context: FarmContext) -> None:
        assessment = assess_feeding(current_pond_context)

        assert assessment.biomass_kg == pytest.approx(125.0)
        assert assessment.q10_factor is not None
        assert assessment.temperature_adjusted_ration_g is not None
        assert assessment.ration_ratio is not None

    def test_cold_water_flags_suppressed_appetite(self) -> None:
        """Reduced intake at low temperature is thermal, not necessarily disease."""
        context = FarmContext(
            population=500,
            average_weight_g=250.0,
            water=WaterQuality(temperature_c=18.0),
        )
        assessment = assess_feeding(context)
        appetite = next(
            f for f in assessment.findings if f.rule_id == "feeding.appetite_temperature"
        )

        assert appetite.status == "concern"
        assert "appetite" in appetite.summary.lower()

    def test_overfeeding_is_flagged_as_a_concern(self) -> None:
        context = FarmContext(
            population=500,
            average_weight_g=250.0,
            water=WaterQuality(temperature_c=29.4),
            feeding=FeedingContext(daily_ration_g=20000.0),
        )
        assessment = assess_feeding(context)
        comparison = next(
            f for f in assessment.findings if f.rule_id == "feeding.ration_comparison"
        )

        assert comparison.status == "concern"

    def test_reported_refusal_is_a_concern(self) -> None:
        context = FarmContext(feeding=FeedingContext(feed_acceptance="refused"))
        assessment = assess_feeding(context)
        acceptance = next(f for f in assessment.findings if f.rule_id == "feeding.acceptance")

        assert acceptance.status == "concern"
