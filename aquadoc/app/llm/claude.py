"""Claude implementation of `LLMProvider`.

Uses the official Anthropic SDK. Everything vendor-specific stays inside this
module — the orchestrator only ever sees `LLMRequest` / `LLMResponse`.
"""

from __future__ import annotations

import json
import logging
import time
from typing import Any

from app.errors import LLMProviderError, LLMRefusalError
from app.llm.base import LLMProvider, LLMRequest, LLMResponse, LLMUsage

logger = logging.getLogger(__name__)

#: Routes a safety refusal to Anthropic's recommended fallback model inside the
#: same call, so a false-positive classifier hit on legitimate aquaculture
#: health content does not become a dead end for the farmer.
_FALLBACK_BETA = "server-side-fallback-2026-07-01"


class ClaudeProvider(LLMProvider):
    name = "anthropic"

    def __init__(
        self,
        *,
        api_key: str,
        model: str = "claude-opus-5",
        enable_refusal_fallback: bool = True,
    ) -> None:
        if not api_key:
            raise LLMProviderError("ANTHROPIC_API_KEY is not configured.")
        try:
            from anthropic import AsyncAnthropic
        except ImportError as exc:  # pragma: no cover - dependency guard
            raise LLMProviderError("The 'anthropic' package is not installed.") from exc

        self._client = AsyncAnthropic(api_key=api_key)
        self._model = model
        self._enable_refusal_fallback = enable_refusal_fallback

    @property
    def model_id(self) -> str:
        return self._model

    async def generate(self, request: LLMRequest) -> LLMResponse:
        payload: dict[str, Any] = {
            "model": self._model,
            "max_tokens": request.max_tokens,
            "system": request.system,
            "messages": [{"role": m.role, "content": m.content} for m in request.messages],
            "output_config": {"effort": request.effort},
        }
        if request.json_schema is not None:
            payload["output_config"]["format"] = {
                "type": "json_schema",
                "schema": request.json_schema,
            }
        if self._enable_refusal_fallback:
            payload["betas"] = [_FALLBACK_BETA]
            payload["fallbacks"] = "default"

        started = time.perf_counter()
        try:
            client = self._client.with_options(timeout=request.timeout_seconds)
            response = await client.beta.messages.create(**payload)
        except Exception as exc:
            # Never surface the provider's raw exception text to a caller: it can
            # carry request fragments (07_SECURITY_ARCHITECTURE.md section 10).
            logger.exception("claude_request_failed", extra={"model": self._model})
            raise LLMProviderError("The language model provider request failed.") from exc
        latency_ms = (time.perf_counter() - started) * 1000

        # Check stop_reason before reading content — a refused response has an
        # empty or partial content array.
        if getattr(response, "stop_reason", None) == "refusal":
            category = getattr(getattr(response, "stop_details", None), "category", None)
            logger.warning("claude_refusal", extra={"category": category})
            raise LLMRefusalError(
                "The language model declined to answer this request.",
                details={"category": category},
            )

        text = "".join(
            block.text for block in response.content if getattr(block, "type", None) == "text"
        )
        parsed = self._parse_json(text) if request.json_schema is not None else None

        usage = getattr(response, "usage", None)
        return LLMResponse(
            text=text,
            parsed=parsed,
            model=getattr(response, "model", self._model),
            provider=self.name,
            stop_reason=getattr(response, "stop_reason", None),
            usage=LLMUsage(
                input_tokens=getattr(usage, "input_tokens", None),
                output_tokens=getattr(usage, "output_tokens", None),
            ),
            latency_ms=latency_ms,
        )

    @staticmethod
    def _parse_json(text: str) -> dict[str, Any] | None:
        """Parse the structured payload.

        `output_config.format` guarantees valid JSON, but the caller validates
        the parsed object against the Pydantic model regardless — a malformed
        payload must fail closed, not be rendered as an answer.
        """
        try:
            value = json.loads(text)
        except json.JSONDecodeError:
            return None
        return value if isinstance(value, dict) else None

    async def aclose(self) -> None:
        await self._client.close()
