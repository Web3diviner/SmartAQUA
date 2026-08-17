"""Health and readiness.

Split deliberately:

- `/health` is a liveness probe. It answers "is this process up?" and never
  touches a dependency, so a database blip cannot make an orchestrator restart
  itself into a loop.
- `/internal/v1/aquadoc/health` is readiness. It checks the database and the
  pgvector extension, because a service that cannot retrieve cannot answer.

Neither endpoint requires authentication (07_SECURITY_ARCHITECTURE.md keeps
probes reachable), so neither reveals configuration: no connection strings, no
API keys, no dependency hostnames.
"""

from __future__ import annotations

import logging

from fastapi import APIRouter, Response, status

from app.api.deps import DatabaseDep, SettingsDep
from app.schemas.system import ComponentHealth, HealthResponse, ReadinessResponse

logger = logging.getLogger(__name__)

router = APIRouter(tags=["health"])


@router.get("/health", response_model=HealthResponse, summary="Liveness probe")
async def health(settings: SettingsDep) -> HealthResponse:
    return HealthResponse(
        status="ok",
        service=settings.service_name,
        version=settings.service_version,
        environment=settings.app_env,
    )


@router.get(
    "/internal/v1/aquadoc/health",
    response_model=ReadinessResponse,
    summary="Readiness probe",
)
async def readiness(
    settings: SettingsDep,
    database: DatabaseDep,
    response: Response,
) -> ReadinessResponse:
    """Check the dependencies required to serve a grounded answer."""
    components: list[ComponentHealth] = []

    db_health = await database.check()
    components.append(
        ComponentHealth(
            name="postgres",
            status="ok" if db_health.reachable else "error",
            detail=db_health.detail,
            latency_ms=db_health.latency_ms,
        )
    )
    components.append(
        ComponentHealth(
            name="pgvector",
            status="ok" if db_health.pgvector_available else "error",
            detail=(
                "extension available"
                if db_health.pgvector_available
                else "extension missing; run the migrations"
            ),
        )
    )

    # Providers are reported, not called: a readiness probe that consumed model
    # tokens on every poll would be its own cost problem.
    components.append(
        ComponentHealth(
            name="llm_provider",
            status="ok",
            detail=f"{settings.llm_provider}:{settings.llm_model}",
        )
    )
    components.append(
        ComponentHealth(
            name="embedding_provider",
            status="ok",
            detail=f"{settings.embedding_provider} ({settings.embedding_dimensions}d)",
        )
    )

    ready = all(component.status == "ok" for component in components)
    if not ready:
        response.status_code = status.HTTP_503_SERVICE_UNAVAILABLE

    return ReadinessResponse(
        status="ok" if ready else "degraded",
        service=settings.service_name,
        version=settings.service_version,
        environment=settings.app_env,
        ready=ready,
        components=components,
    )
