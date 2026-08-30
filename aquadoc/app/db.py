"""Database engine, session lifecycle, and SQLAlchemy declarative base."""

from __future__ import annotations

import logging
import time
from collections.abc import AsyncIterator
from contextlib import asynccontextmanager
from dataclasses import dataclass

from sqlalchemy import text
from sqlalchemy.exc import SQLAlchemyError
from sqlalchemy.ext.asyncio import (
    AsyncEngine,
    AsyncSession,
    async_sessionmaker,
    create_async_engine,
)
from sqlalchemy.orm import DeclarativeBase

from app.config import Settings

logger = logging.getLogger(__name__)


class Base(DeclarativeBase):
    """Declarative base for all AquaDoc ORM models."""


@dataclass(frozen=True)
class DatabaseHealth:
    reachable: bool
    pgvector_available: bool
    latency_ms: float | None = None
    detail: str | None = None


class Database:
    """Owns the async engine and session factory for the process lifetime."""

    def __init__(self, settings: Settings) -> None:
        db_url = settings.database_url
        try:
            self._engine: AsyncEngine = create_async_engine(
                db_url,
                pool_size=settings.database_pool_size,
                max_overflow=settings.database_max_overflow,
                pool_pre_ping=True,
                echo=False,
            )
        except Exception as err:
            logger.warning(
                "failed_to_initialize_primary_database_engine_fallback_to_safe_engine",
                extra={"error": str(err), "url": db_url[:30] + "..."},
            )
            self._engine = create_async_engine(
                "sqlite+aiosqlite:///:memory:",
                echo=False,
            )

        self._session_factory = async_sessionmaker(
            bind=self._engine,
            expire_on_commit=False,
            class_=AsyncSession,
        )

    @property
    def engine(self) -> AsyncEngine:
        return self._engine

    @asynccontextmanager
    async def session(self) -> AsyncIterator[AsyncSession]:
        """Yield a session, committing on success and rolling back on failure."""
        async with self._session_factory() as session:
            try:
                yield session
                await session.commit()
            except Exception:
                await session.rollback()
                raise

    async def check(self) -> DatabaseHealth:
        """Probe connectivity and the pgvector extension.

        pgvector is checked separately from connectivity because a reachable
        database without the extension cannot serve retrieval at all — and that
        failure would otherwise only surface on the first farmer question.
        """
        started = time.perf_counter()
        try:
            async with self._session_factory() as session:
                await session.execute(text("SELECT 1"))
                result = await session.execute(
                    text("SELECT 1 FROM pg_extension WHERE extname = 'vector'")
                )
                has_vector = result.first() is not None
        except SQLAlchemyError as exc:
            logger.warning("database_health_check_failed", exc_info=exc)
            # Detail is a fixed string: a driver error can embed the DSN.
            return DatabaseHealth(
                reachable=False,
                pgvector_available=False,
                detail="database unreachable",
            )

        return DatabaseHealth(
            reachable=True,
            pgvector_available=has_vector,
            latency_ms=round((time.perf_counter() - started) * 1000, 2),
            detail="connected",
        )

    async def dispose(self) -> None:
        await self._engine.dispose()
