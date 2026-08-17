"""FastAPI dependencies: shared resources and per-request authentication.

Providers, the engine, and the orchestrator are built once at startup and held
on `app.state`. Rebuilding them per request would discard the HTTP connection
pool and the database pool.

Authentication itself lives in `app.security`; this module only wires it into
the dependency graph.
"""

from __future__ import annotations

from typing import Annotated

from fastapi import Depends, Request

from app.config import Settings
from app.db import Database
from app.embeddings.base import EmbeddingProvider
from app.llm.base import LLMProvider
from app.orchestration.orchestrator import Orchestrator
from app.security import Caller, require_dev_caller, require_service_caller

# `require_service_caller` / `require_dev_caller` are re-exported as dependency
# aliases so routes read declaratively: `caller: ServiceCallerDep`.
ServiceCallerDep = Annotated[Caller, Depends(require_service_caller)]
DevCallerDep = Annotated[Caller, Depends(require_dev_caller)]


def get_settings_dep(request: Request) -> Settings:
    return request.app.state.settings


def get_database(request: Request) -> Database:
    return request.app.state.database


def get_orchestrator(request: Request) -> Orchestrator:
    return request.app.state.orchestrator


def get_embeddings(request: Request) -> EmbeddingProvider:
    return request.app.state.embeddings


def get_llm(request: Request) -> LLMProvider:
    return request.app.state.llm


def get_ingestion_service(request: Request):
    return request.app.state.ingestion_service


SettingsDep = Annotated[Settings, Depends(get_settings_dep)]
DatabaseDep = Annotated[Database, Depends(get_database)]
OrchestratorDep = Annotated[Orchestrator, Depends(get_orchestrator)]
EmbeddingsDep = Annotated[EmbeddingProvider, Depends(get_embeddings)]
LLMDep = Annotated[LLMProvider, Depends(get_llm)]


__all__ = [
    "DatabaseDep",
    "DevCallerDep",
    "EmbeddingsDep",
    "LLMDep",
    "OrchestratorDep",
    "ServiceCallerDep",
    "SettingsDep",
    "get_database",
    "get_embeddings",
    "get_ingestion_service",
    "get_llm",
    "get_orchestrator",
    "get_settings_dep",
]
