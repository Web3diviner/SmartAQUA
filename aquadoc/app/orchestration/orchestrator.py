"""The orchestrator: intent -> rules -> retrieval -> prompt -> LLM -> safety -> response.

Layering per 13_CODING_AND_ENGINEERING_STANDARDS.md:

    api -> orchestrator -> retrieval/rules/models -> providers

The orchestrator coordinates; it owns no retrieval SQL, no provider calls, and
no safety decisions. Each stage hands the next a typed value, so the pipeline
stays inspectable and every stage stays independently testable.

One chat turn runs in one database transaction: the conversation, the user
message, the assistant message, and the retrieval trace either all persist or
none do. A stored answer with no trace would be unauditable
(14_AQUADOC_SAFETY_AND_GOVERNANCE.md section 7).
"""

from __future__ import annotations

import logging
import time
import uuid
from dataclasses import dataclass
from typing import Any

from pydantic import BaseModel, ConfigDict, Field, ValidationError
from sqlalchemy import select
from sqlalchemy.exc import SQLAlchemyError
from sqlalchemy.ext.asyncio import AsyncSession

from app.config import Settings
from app.db import Database
from app.embeddings.base import EmbeddingProvider
from app.errors import (
    ConversationNotFoundError,
    LLMProviderError,
    ResponseValidationError,
    RetrievalTraceNotFoundError,
)
from app.llm import (
    ANSWER_SCHEMA,
    RESPONSE_SCHEMA_VERSION,
    LLMMessage,
    LLMProvider,
    LLMRequest,
)
from app.models import AquaDocConversation, AquaDocMessage, AquaDocRetrievalTrace
from app.orchestration.confidence import ConfidenceBreakdown, score_confidence
from app.orchestration.context_builder import build_user_turn, missing_measurement_keys
from app.orchestration.intent import classify, is_health_related
from app.prompts import prompt_for_intent
from app.rag.citations import build_source_references, filter_reported_sources
from app.rag.filters import build_filters
from app.rag.retrieval import RetrievalResult, Retriever
from app.rules import RULES_VERSION, assess_feeding, enforce, evaluate_water_quality
from app.schemas.chat import (
    ChatRequest,
    ChatResponse,
    PossibleCause,
    Provenance,
    RecommendedAction,
    RetrievalTrace,
    SourceReference,
)
from app.schemas.common import Intent, RecommendationTier, RiskLevel, confidence_band
from app.schemas.farm_context import FarmContext

logger = logging.getLogger(__name__)


class _ModelPossibleCause(BaseModel):
    model_config = ConfigDict(extra="ignore")
    name: str = "Suspected Condition"
    confidence: float = 0.5
    explanation: str | None = None
    supporting_source_ids: list[str] = Field(default_factory=list)

    def to_schema(self) -> PossibleCause:
        return PossibleCause(
            name=self.name,
            confidence=max(0.0, min(1.0, float(self.confidence))),
            explanation=self.explanation,
            supporting_source_ids=self.supporting_source_ids,
        )


class _ModelRecommendedAction(BaseModel):
    model_config = ConfigDict(extra="ignore")
    action: str = ""
    tier: Any = RecommendationTier.TIER_1_ADVISORY
    reason: str = ""
    requires_approval: bool = False
    urgency: Any = RiskLevel.INFORMATIONAL

    def to_schema(self) -> RecommendedAction:
        resolved_tier = RecommendationTier.TIER_1_ADVISORY
        tier_str = str(self.tier).lower()
        if "0" in tier_str or "info" in tier_str:
            resolved_tier = RecommendationTier.TIER_0_INFORMATIONAL
        elif "2" in tier_str or "low" in tier_str or "operation" in tier_str:
            resolved_tier = RecommendationTier.TIER_2_LOW_RISK_OPERATIONAL
        elif "3" in tier_str or "high" in tier_str or "treat" in tier_str:
            resolved_tier = RecommendationTier.TIER_3_HIGH_RISK

        resolved_urgency = RiskLevel.INFORMATIONAL
        urg_str = str(self.urgency).lower()
        if "crit" in urg_str:
            resolved_urgency = RiskLevel.CRITICAL
        elif "warn" in urg_str:
            resolved_urgency = RiskLevel.WARNING
        elif "watch" in urg_str:
            resolved_urgency = RiskLevel.WATCH

        return RecommendedAction(
            action=self.action or "Observe fish and check water quality.",
            tier=resolved_tier,
            reason=self.reason or "Routine diagnostic confirmation.",
            requires_approval=self.requires_approval,
            urgency=resolved_urgency,
        )


class _ModelAnswer(BaseModel):
    """The model's structured payload, validated before anything reads it.

    `extra="ignore"` rather than `forbid`: an unexpected key is not worth
    failing a farmer's question over, and every field consumed downstream is
    strictly typed here.
    """

    model_config = ConfigDict(extra="ignore")

    answer: str = ""
    possible_causes: list[Any] = Field(default_factory=list)
    recommended_actions: list[Any] = Field(default_factory=list)
    model_confidence: float = Field(default=0.5)
    risk_level: Any = RiskLevel.INFORMATIONAL
    expert_escalation: bool = False
    escalation_reasons: list[str] = Field(default_factory=list)

    def normalized_causes(self) -> list[PossibleCause]:
        results: list[PossibleCause] = []
        for item in self.possible_causes:
            if isinstance(item, PossibleCause):
                results.append(item)
            elif isinstance(item, dict):
                try:
                    results.append(_ModelPossibleCause.model_validate(item).to_schema())
                except Exception:
                    pass
            elif isinstance(item, str):
                results.append(PossibleCause(name=item, confidence=0.5, explanation=None))
        return results

    def normalized_actions(self) -> list[RecommendedAction]:
        results: list[RecommendedAction] = []
        for item in self.recommended_actions:
            if isinstance(item, RecommendedAction):
                results.append(item)
            elif isinstance(item, dict):
                try:
                    results.append(_ModelRecommendedAction.model_validate(item).to_schema())
                except Exception:
                    pass
            elif isinstance(item, str):
                results.append(
                    RecommendedAction(
                        action=item,
                        tier=RecommendationTier.TIER_1_ADVISORY,
                        reason="Clinical observation.",
                        requires_approval=False,
                        urgency=RiskLevel.INFORMATIONAL,
                    )
                )
        return results

    def normalized_risk_level(self) -> RiskLevel:
        if isinstance(self.risk_level, RiskLevel):
            return self.risk_level
        risk_str = str(self.risk_level).lower()
        if "crit" in risk_str:
            return RiskLevel.CRITICAL
        if "warn" in risk_str:
            return RiskLevel.WARNING
        if "watch" in risk_str:
            return RiskLevel.WATCH
        return RiskLevel.INFORMATIONAL


@dataclass(frozen=True)
class ChatOutcome:
    """A completed chat turn plus developer diagnostics."""

    response: ChatResponse
    trace: RetrievalTrace
    confidence_breakdown: ConfidenceBreakdown


class Orchestrator:
    """Coordinates one chat turn end to end."""

    def __init__(
        self,
        *,
        settings: Settings,
        database: Database,
        llm: LLMProvider,
        embeddings: EmbeddingProvider,
    ) -> None:
        self._settings = settings
        self._database = database
        self._llm = llm
        self._embeddings = embeddings
        self._conversation_history: dict[str, list[dict[str, str]]] = {}

    @property
    def embedding_model_id(self) -> str:
        """Exposed so the developer config view can report the live model."""
        return self._embeddings.model_id

    async def chat(self, request: ChatRequest, *, include_debug: bool = False) -> ChatOutcome:
        started = time.perf_counter()
        request_id = request.request_id or f"REQ-{uuid.uuid4().hex[:12].upper()}"
        context = request.farm_context
        conv_history = (
            self._conversation_history.get(request.conversation_id, [])
            if request.conversation_id
            else []
        )

        # -- 1. intent -------------------------------------------------------
        intent = classify(request.question, context)

        # -- 2. deterministic rules ------------------------------------------
        # Run before the model, so its output can be constrained by them rather
        # than merged with them afterwards.
        rule_findings = []
        if context is not None:
            rule_findings = evaluate_water_quality(context.water) + assess_feeding(context).findings

        filters = build_filters(
            intent,
            request.filters,
            species=context.species if context else None,
        )

        # -- 3. retrieval & generation ---------------------------------------
        try:
            async with self._database.session() as session:
                retriever = Retriever(
                    session,
                    self._embeddings,
                    candidates=self._settings.retrieval_candidates,
                    top_k=self._settings.retrieval_top_k,
                    min_similarity=self._settings.retrieval_min_similarity,
                    enable_lexical=self._settings.retrieval_enable_lexical,
                )
                result: RetrievalResult = await retriever.retrieve(
                    request_id=request_id,
                    question=request.question,
                    intent=intent,
                    filters=filters,
                )
                selected = result.selected

                # -- 4. prompt assembly --------------------------------------
                prompt_version, system_prompt = prompt_for_intent(intent)
                user_turn = build_user_turn(
                    question=request.question,
                    context=context,
                    findings=rule_findings,
                    candidates=selected,
                    history=conv_history,
                )

                # -- 5. generation -------------------------------------------
                llm_response = await self._generate(
                    system_prompt, user_turn, intent, model_override=request.model
                )
                model_answer = self._validate_answer(llm_response.parsed, intent)

                # -- 6. citations --------------------------------------------
                sources = build_source_references(selected, include_full_text=include_debug)
                norm_causes = model_answer.normalized_causes()
                norm_actions = model_answer.normalized_actions()
                norm_risk = model_answer.normalized_risk_level()
                norm_confidence = max(0.0, min(1.0, float(model_answer.model_confidence or 0.5)))
                possible_causes = _ground_causes(norm_causes, sources)

                # -- 7. confidence -------------------------------------------
                missing_keys, missing_labels = missing_measurement_keys(context)
                completeness = context.completeness() if context else 0.0
                confidence = score_confidence(
                    intent=intent,
                    candidates=selected,
                    findings=rule_findings,
                    context_completeness=completeness,
                    model_confidence=norm_confidence,
                    has_farm_context=context is not None,
                )

                # -- 8. safety guardrails ------------------------------------
                safety = enforce(
                    answer=model_answer.answer,
                    model_risk_level=norm_risk,
                    model_actions=norm_actions,
                    model_escalation=model_answer.expert_escalation,
                    model_escalation_reasons=model_answer.escalation_reasons,
                    rule_findings=rule_findings,
                    confidence=confidence.final_score,
                    mortality_24h=context.health.mortality_24h if context else None,
                    has_health_signal=is_health_related(intent, context),
                )

                # -- 9. assemble the response --------------------------------
                conversation_id = await self._open_conversation(session, request, request_id, context)
                total_latency_ms = (time.perf_counter() - started) * 1000

                response = ChatResponse(
                    request_id=request_id,
                    conversation_id=str(conversation_id),
                    answer=model_answer.answer,
                    intent=intent,
                    risk_level=safety.risk_level,
                    confidence=confidence.final_score,
                    confidence_band=confidence_band(confidence.final_score),
                    possible_causes=possible_causes,
                    recommended_actions=safety.actions,
                    missing_data=missing_keys,
                    missing_data_labels=missing_labels,
                    expert_escalation=safety.expert_escalation,
                    escalation_reasons=safety.escalation_reasons,
                    sources=sources,
                    rule_findings=rule_findings,
                    warnings=safety.warnings,
                    provenance=Provenance(
                        prompt_version=f"{prompt_version}@{RESPONSE_SCHEMA_VERSION}",
                        llm_model=llm_response.model or self._llm.model_id,
                        llm_provider=self._llm.name,
                        embedding_model=self._embeddings.model_id,
                        embedding_provider=self._embeddings.name,
                        rules_version=RULES_VERSION,
                        retrieval_source_ids=[candidate.document_id for candidate in selected],
                        farm_context_supplied=context is not None,
                        farm_context_completeness=round(completeness, 4),
                        llm_latency_ms=round(llm_response.latency_ms, 2),
                        total_latency_ms=round(total_latency_ms, 2),
                        input_tokens=llm_response.usage.input_tokens,
                        output_tokens=llm_response.usage.output_tokens,
                    ),
                    retrieval_trace=result.trace if include_debug else None,
                    confidence_breakdown=confidence.as_dict() if include_debug else None,
                )

                session.add(
                    AquaDocMessage(
                        conversation_id=conversation_id,
                        role="assistant",
                        content=response.answer,
                        structured_payload_json=response.model_dump(mode="json"),
                        request_id=request_id,
                    )
                )
                session.add(
                    AquaDocRetrievalTrace(
                        request_id=request_id,
                        conversation_id=conversation_id,
                        question=request.question,
                        intent=intent.value,
                        trace_json=result.trace.model_dump(mode="json"),
                    )
                )
                conv_key = str(conversation_id)
                if conv_key not in self._conversation_history:
                    self._conversation_history[conv_key] = []
                self._conversation_history[conv_key].append({"role": "user", "content": request.question})
                self._conversation_history[conv_key].append({"role": "assistant", "content": model_answer.answer})
                if len(self._conversation_history[conv_key]) > 20:
                    self._conversation_history[conv_key] = self._conversation_history[conv_key][-20:]
        except (SQLAlchemyError, OSError) as db_err:
            if self._settings.is_production:
                raise

            logger.warning("database_unavailable_using_memory_retriever", extra={"error": str(db_err)})
            from app.rag.memory_retriever import MemoryRetriever

            mem_retriever = MemoryRetriever(
                self._embeddings,
                candidates=self._settings.retrieval_candidates,
                top_k=self._settings.retrieval_top_k,
                min_similarity=self._settings.retrieval_min_similarity,
                enable_lexical=self._settings.retrieval_enable_lexical,
            )
            result = await mem_retriever.retrieve(
                request_id=request_id,
                question=request.question,
                intent=intent,
                filters=filters,
            )
            selected = result.selected

            prompt_version, system_prompt = prompt_for_intent(intent)
            user_turn = build_user_turn(
                question=request.question,
                context=context,
                findings=rule_findings,
                candidates=selected,
                history=conv_history,
            )

            llm_response = await self._generate(
                system_prompt, user_turn, intent, model_override=request.model
            )
            model_answer = self._validate_answer(llm_response.parsed, intent)
            sources = build_source_references(selected, include_full_text=include_debug)
            norm_causes = model_answer.normalized_causes()
            norm_actions = model_answer.normalized_actions()
            norm_risk = model_answer.normalized_risk_level()
            norm_confidence = max(0.0, min(1.0, float(model_answer.model_confidence or 0.5)))
            possible_causes = _ground_causes(norm_causes, sources)

            missing_keys, missing_labels = missing_measurement_keys(context)
            completeness = context.completeness() if context else 0.0
            confidence = score_confidence(
                intent=intent,
                candidates=selected,
                findings=rule_findings,
                context_completeness=completeness,
                model_confidence=norm_confidence,
                has_farm_context=context is not None,
            )

            safety = enforce(
                answer=model_answer.answer,
                model_risk_level=norm_risk,
                model_actions=norm_actions,
                model_escalation=model_answer.expert_escalation,
                model_escalation_reasons=model_answer.escalation_reasons,
                rule_findings=rule_findings,
                confidence=confidence.final_score,
                mortality_24h=context.health.mortality_24h if context else None,
                has_health_signal=is_health_related(intent, context),
            )

            conv_id_str = request.conversation_id or f"dev-conv-{uuid.uuid4().hex[:8]}"
            conv_key = conv_id_str
            if conv_key not in self._conversation_history:
                self._conversation_history[conv_key] = []
            self._conversation_history[conv_key].append({"role": "user", "content": request.question})
            self._conversation_history[conv_key].append({"role": "assistant", "content": model_answer.answer})
            if len(self._conversation_history[conv_key]) > 20:
                self._conversation_history[conv_key] = self._conversation_history[conv_key][-20:]

            total_latency_ms = (time.perf_counter() - started) * 1000

            response = ChatResponse(
                request_id=request_id,
                conversation_id=conv_id_str,
                answer=model_answer.answer,
                intent=intent,
                risk_level=safety.risk_level,
                confidence=confidence.final_score,
                confidence_band=confidence_band(confidence.final_score),
                possible_causes=possible_causes,
                recommended_actions=safety.actions,
                missing_data=missing_keys,
                missing_data_labels=missing_labels,
                expert_escalation=safety.expert_escalation,
                escalation_reasons=safety.escalation_reasons,
                sources=sources,
                rule_findings=rule_findings,
                warnings=safety.warnings,
                provenance=Provenance(
                    prompt_version=f"{prompt_version}@{RESPONSE_SCHEMA_VERSION}",
                    llm_model=llm_response.model or self._llm.model_id,
                    llm_provider=self._llm.name,
                    embedding_model=self._embeddings.model_id,
                    embedding_provider=self._embeddings.name,
                    rules_version=RULES_VERSION,
                    retrieval_source_ids=[candidate.document_id for candidate in selected],
                    farm_context_supplied=context is not None,
                    farm_context_completeness=round(completeness, 4),
                    llm_latency_ms=round(llm_response.latency_ms, 2),
                    total_latency_ms=round(total_latency_ms, 2),
                    input_tokens=llm_response.usage.input_tokens,
                    output_tokens=llm_response.usage.output_tokens,
                ),
                retrieval_trace=result.trace if include_debug else None,
                confidence_breakdown=confidence.as_dict() if include_debug else None,
            )

        logger.info(
            "chat_completed",
            extra={
                "request_id": request_id,
                "intent": intent.value,
                "risk_level": safety.risk_level.value,
                "confidence": confidence.final_score,
                "source_count": len(sources),
                "escalated": safety.expert_escalation,
            },
        )
        return ChatOutcome(
            response=response,
            trace=result.trace,
            confidence_breakdown=confidence,
        )

    async def get_retrieval_trace(self, request_id: str) -> RetrievalTrace:
        """Replay a stored trace for the developer Retrieval Inspector."""
        async with self._database.session() as session:
            row = (
                await session.execute(
                    select(AquaDocRetrievalTrace).where(
                        AquaDocRetrievalTrace.request_id == request_id
                    )
                )
            ).scalar_one_or_none()

        if row is None:
            raise RetrievalTraceNotFoundError(f"No retrieval trace recorded for {request_id}.")
        return RetrievalTrace.model_validate(row.trace_json)

    async def search_knowledge(
        self,
        *,
        query: str,
        top_k: int,
        filters: dict[str, list[str]] | None = None,
        include_full_text: bool = False,
    ) -> tuple[list[SourceReference], RetrievalTrace]:
        """Retrieval without generation.

        Lets retrieval quality be evaluated independently of the model
        (04_AQUADOC_RAG_LLM.md section 15) and backs the knowledge-search
        endpoint the Go backend uses for non-conversational lookups.
        """
        request_id = f"SEARCH-{uuid.uuid4().hex[:12].upper()}"
        intent = classify(query, None)
        resolved_filters = build_filters(intent, filters)

        async with self._database.session() as session:
            retriever = Retriever(
                session,
                self._embeddings,
                candidates=self._settings.retrieval_candidates,
                top_k=top_k,
                min_similarity=self._settings.retrieval_min_similarity,
                enable_lexical=self._settings.retrieval_enable_lexical,
            )
            result = await retriever.retrieve(
                request_id=request_id,
                question=query,
                intent=intent,
                filters=resolved_filters,
                top_k=top_k,
            )

        references = build_source_references(result.selected, include_full_text=include_full_text)
        return references, result.trace

    # -- internals -----------------------------------------------------------

    async def _generate(
        self,
        system_prompt: str,
        user_turn: str,
        intent: Intent,
        model_override: str | None = None,
    ) -> LLMResponse:
        llm_request = LLMRequest(
            system=system_prompt,
            messages=[LLMMessage(role="user", content=user_turn)],
            json_schema=ANSWER_SCHEMA,
            max_tokens=self._settings.llm_max_tokens,
            effort=self._settings.llm_effort,
            timeout_seconds=self._settings.llm_timeout_seconds,
        )
        try:
            if model_override and hasattr(self._llm, "generate"):
                try:
                    return await self._llm.generate(llm_request, model_override=model_override)  # type: ignore[call-arg]
                except TypeError:
                    return await self._llm.generate(llm_request)
            return await self._llm.generate(llm_request)
        except LLMProviderError:
            raise  # already typed and already logged by the provider
        except Exception as exc:
            logger.exception("llm_generation_failed", extra={"intent": intent.value})
            raise LLMProviderError("Language model generation failed.") from exc

    @staticmethod
    def _validate_answer(parsed: dict | None, intent: Intent) -> _ModelAnswer:
        """Validate and normalize structured payload from model."""
        if parsed is None:
            raise ResponseValidationError(
                "The language model did not return a parseable structured response."
            )
        try:
            return _ModelAnswer.model_validate(parsed)
        except Exception:
            answer_text = str(
                parsed.get("answer")
                or parsed.get("text")
                or parsed.get("response")
                or parsed.get("message")
                or ""
            ).strip()
            if not answer_text:
                logger.warning("model_response_schema_violation", extra={"intent": intent.value})
                raise ResponseValidationError(
                    "The language model response did not match the required schema."
                )
            return _ModelAnswer(
                answer=answer_text,
                possible_causes=parsed.get("possible_causes") or [],
                recommended_actions=parsed.get("recommended_actions") or [],
                model_confidence=0.6,
                risk_level=RiskLevel.INFORMATIONAL,
            )

    @staticmethod
    async def _open_conversation(
        session: AsyncSession,
        request: ChatRequest,
        request_id: str,
        context: FarmContext | None,
    ) -> uuid.UUID:
        """Resolve or create the conversation and record the user's question."""
        conversation_id: uuid.UUID | None = None

        if request.conversation_id:
            try:
                candidate = uuid.UUID(request.conversation_id)
            except ValueError as exc:
                raise ConversationNotFoundError(
                    f"Conversation {request.conversation_id} is not a valid identifier."
                ) from exc

            existing = await session.get(AquaDocConversation, candidate)
            if existing is None:
                raise ConversationNotFoundError(
                    f"Conversation {request.conversation_id} does not exist."
                )
            # A conversation belongs to the user who created it. Without this
            # check, a known ID would expose another farmer's history.
            if existing.user_id != request.user_id:
                raise ConversationNotFoundError(
                    f"Conversation {request.conversation_id} does not exist."
                )
            conversation_id = candidate

        if conversation_id is None:
            conversation = AquaDocConversation(
                user_id=request.user_id,
                farm_id=context.farm_id if context else None,
                pond_id=context.pond_id if context else None,
                production_cycle_id=context.production_cycle_id if context else None,
                title=request.question[:80],
            )
            session.add(conversation)
            await session.flush()  # assigns the server-side UUID
            conversation_id = conversation.id

        session.add(
            AquaDocMessage(
                conversation_id=conversation_id,
                role="user",
                content=request.question,
                request_id=request_id,
            )
        )
        return conversation_id


def _ground_causes(
    causes: list[PossibleCause],
    sources: list[SourceReference],
) -> list[PossibleCause]:
    """Drop citation IDs the model invented, keeping the cause itself."""
    return [
        PossibleCause(
            name=cause.name,
            confidence=cause.confidence,
            explanation=cause.explanation,
            supporting_source_ids=filter_reported_sources(sources, cause.supporting_source_ids),
        )
        for cause in causes
    ]
