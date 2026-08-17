"""Shared test fixtures."""

from __future__ import annotations

import pytest

from app.config import Settings
from app.schemas.farm_context import FarmContext, FeedingContext, HealthContext, WaterQuality


@pytest.fixture
def settings() -> Settings:
    """Development settings with the offline providers.

    Never reads the ambient environment, so a developer's local .env cannot
    change test outcomes.
    """
    return Settings(
        app_env="development",
        aquadoc_dev_token="test-dev-token",
        aquadoc_internal_service_secret="test-service-secret",
        llm_provider="echo",
        embedding_provider="hashing",
        embedding_dimensions=256,
    )


@pytest.fixture
def current_pond_context() -> FarmContext:
    """The pond as the deployment actually measures it today.

    Temperature is the only live water measurement; pH, dissolved oxygen and
    turbidity sensors are not installed yet (15_AQUADOC_FRONTEND.md section 14).
    Keeping the real shape in the fixtures means the missing-data paths are
    exercised by default rather than as a special case.
    """
    return FarmContext(
        farm_id="FARM-1",
        pond_id="POND-1",
        pond_name="Concrete Tank 1",
        species="Clarias gariepinus",
        population=500,
        average_weight_g=250.0,
        water=WaterQuality(temperature_c=29.4),
        feeding=FeedingContext(daily_ration_g=3750.0, last_feeding_g=1800.0),
        health=HealthContext(mortality_24h=0),
    )


@pytest.fixture
def empty_context() -> FarmContext:
    """No measurements at all — every water parameter unknown."""
    return FarmContext()
