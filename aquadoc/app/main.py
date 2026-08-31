"""AquaDoc service entrypoint.

01_SYSTEM_ARCHITECTURE.md keeps AquaDoc a separate service from the Go backend,
reachable only through it. This module owns process lifecycle: build the
providers, database pool, and orchestrator once at startup, share them via
`app.state`, and dispose of them on shutdown.

The developer router is mounted only outside production. It is the surface that
exposes prompts and retrieval traces, so its absence in production is
structural rather than a matter of remembering to check a flag per route.
"""

from __future__ import annotations

import logging
from collections.abc import AsyncIterator
from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from app.api import dev, health, internal
from app.config import Settings, get_settings
from app.db import Database
from app.embeddings import build_embedding_provider
from app.errors import register_exception_handlers
from app.llm import build_llm_provider
from app.logging_config import configure_logging
from app.middleware import RequestContextMiddleware
from app.orchestration.orchestrator import Orchestrator
from ingestion.service import IngestionConfig, IngestionService

logger = logging.getLogger(__name__)


@asynccontextmanager
async def lifespan(app: FastAPI) -> AsyncIterator[None]:
    settings: Settings = app.state.settings

    database = Database(settings)
    llm = build_llm_provider(settings)
    embeddings = build_embedding_provider(settings)

    app.state.database = database
    app.state.llm = llm
    app.state.embeddings = embeddings
    app.state.orchestrator = Orchestrator(
        settings=settings,
        database=database,
        llm=llm,
        embeddings=embeddings,
    )
    app.state.ingestion_service = IngestionService(
        embeddings=embeddings,
        config=IngestionConfig(
            chunk_target_tokens=settings.chunk_target_tokens,
            chunk_overlap_tokens=settings.chunk_overlap_tokens,
        ),
    )

    logger.info(
        "aquadoc_started",
        extra={
            "environment": settings.app_env,
            "llm_provider": settings.llm_provider,
            "llm_model": settings.llm_model,
            "embedding_provider": settings.embedding_provider,
            "embedding_model": embeddings.model_id,
            "developer_api": settings.enable_dev_api,
        },
    )

    try:
        yield
    finally:
        # Close in reverse order of construction. Provider clients hold HTTP
        # pools that must be released before the loop closes.
        await llm.aclose()
        await embeddings.aclose()
        await database.dispose()
        logger.info("aquadoc_stopped")


def create_app(settings: Settings | None = None) -> FastAPI:
    settings = settings or get_settings()
    configure_logging(settings.log_level, json_logs=settings.is_production)

    app = FastAPI(
        title="AquaDoc",
        description=(
            "Grounded aquaculture advisory service for SmartAQUA. Internal "
            "service — called by the SmartAQUA backend, not by farmer devices."
        ),
        version=settings.service_version,
        lifespan=lifespan,
        # Interactive docs are a developer tool; production exposes no schema.
        docs_url=None if settings.is_production else "/docs",
        redoc_url=None,
        openapi_url=None if settings.is_production else "/openapi.json",
    )
    app.state.settings = settings

    app.add_middleware(RequestContextMiddleware)

    # Allow CORS for development frontend clients and production cloud hosts
    app.add_middleware(
        CORSMiddleware,
        allow_origins=["*"],
        allow_credentials=False,
        allow_methods=["*"],
        allow_headers=["*"],
        expose_headers=["*"],
        max_age=86400,
    )

    register_exception_handlers(app)

    app.include_router(health.router)
    app.include_router(internal.router)

    if settings.enable_dev_api:
        app.include_router(dev.router)
    else:
        logger.info("developer_api_disabled", extra={"environment": settings.app_env})

    return app


def get_app() -> FastAPI:
    """ASGI factory — run with `uvicorn app.main:get_app --factory`.

    A factory rather than a module-level instance, so importing this module has
    no side effects: tests and the ingestion CLI can import from `app.*` without
    constructing an application or reading the environment.
    """
    return create_app()


# Standard ASGI instance for `uvicorn app.main:app`
app = get_app()
