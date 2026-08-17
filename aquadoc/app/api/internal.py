"""Internal API — called by the Go backend, never by farmer devices.

05_API_AND_SERVICE_CONTRACTS.md, "AquaDoc Service Endpoints":

    POST /internal/v1/aquadoc/chat
    POST /internal/v1/knowledge/search
    GET  /internal/v1/aquadoc/health

The response model is the stable contract. 15_AQUADOC_FRONTEND.md section 19
depends on it staying unchanged when the temporary React client is replaced by
the Flutter integration, so fields are added here, never renamed or removed.
"""

from __future__ import annotations

import logging

from fastapi import APIRouter, status

from app.api.deps import OrchestratorDep, ServiceCallerDep
from app.schemas.chat import (
    ChatRequest,
    ChatResponse,
    KnowledgeSearchRequest,
    KnowledgeSearchResponse,
)

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/internal/v1", tags=["internal"])


@router.post(
    "/aquadoc/chat",
    response_model=ChatResponse,
    response_model_exclude_none=True,
    status_code=status.HTTP_200_OK,
    summary="Ask AquaDoc a grounded question",
)
async def chat(
    payload: ChatRequest,
    caller: ServiceCallerDep,
    orchestrator: OrchestratorDep,
) -> ChatResponse:
    """Run one chat turn: retrieve, reason, guard, respond.

    The retrieval trace and confidence breakdown are attached only for developer
    callers. A farmer-facing client gets the answer, its citations, and its
    provenance — never the prompt internals (15_AQUADOC_FRONTEND.md section 5).
    """
    outcome = await orchestrator.chat(payload, include_debug=caller.is_developer)
    return outcome.response


@router.post(
    "/knowledge/search",
    response_model=KnowledgeSearchResponse,
    response_model_exclude_none=True,
    summary="Search approved knowledge without generating an answer",
)
async def knowledge_search(
    payload: KnowledgeSearchRequest,
    caller: ServiceCallerDep,
    orchestrator: OrchestratorDep,
) -> KnowledgeSearchResponse:
    """Retrieval only.

    Lets retrieval quality be measured independently of the model
    (04_AQUADOC_RAG_LLM.md section 15), and serves backend lookups that need
    sources rather than prose.
    """
    references, trace = await orchestrator.search_knowledge(
        query=payload.query,
        top_k=payload.top_k,
        filters=payload.filters,
        include_full_text=caller.is_developer,
    )
    return KnowledgeSearchResponse(
        request_id=trace.request_id,
        query=payload.query,
        results=references,
        embedding_model=trace.embedding_model,
    )
