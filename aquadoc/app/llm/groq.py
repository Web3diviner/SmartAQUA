"""Groq LLM and Audio transcription provider implementation.

Uses httpx for high-performance asynchronous HTTP calls to Groq's
OpenAI-compatible endpoints:
- https://api.groq.com/openai/v1/chat/completions
- https://api.groq.com/openai/v1/audio/transcriptions
"""

from __future__ import annotations

import asyncio
import json
import logging
import re
import time
from typing import Any

import httpx

from app.errors import LLMProviderError, LLMRefusalError
from app.llm.base import LLMProvider, LLMRequest, LLMResponse, LLMUsage

logger = logging.getLogger(__name__)

GROQ_BASE_URL = "https://api.groq.com/openai/v1"

# Verified active Groq model mappings
DEFAULT_GROQ_CHAT_MODEL = "openai/gpt-oss-120b"
DEFAULT_GROQ_WHISPER_MODEL = "whisper-large-v3-turbo"

GROQ_MODEL_ALIASES: dict[str, str] = {
    "qwen/qwen3.8-27b": "qwen/qwen3.6-27b",
    "llama-3.3-70b-versatile": "openai/gpt-oss-120b",
    "llama-3.1-8b-instant": "openai/gpt-oss-20b",
}


def resolve_groq_model(model_name: str | None) -> str:
    if not model_name:
        return DEFAULT_GROQ_CHAT_MODEL
    cleaned = model_name.strip()
    return GROQ_MODEL_ALIASES.get(cleaned, cleaned)


# Verified active Groq models in failover priority order
AVAILABLE_GROQ_CHAT_MODELS: list[str] = [
    "openai/gpt-oss-120b",
    "openai/gpt-oss-20b",
    "qwen/qwen3.6-27b",
    "groq/compound-mini",
]


class GroqProvider(LLMProvider):
    name = "groq"

    def __init__(
        self,
        *,
        api_key: str,
        model: str = DEFAULT_GROQ_CHAT_MODEL,
        base_url: str = GROQ_BASE_URL,
    ) -> None:
        self._api_key = api_key
        self._model = model
        self._base_url = base_url.rstrip("/")
        self._client = httpx.AsyncClient(
            base_url=self._base_url,
            headers={
                "Authorization": f"Bearer {api_key}",
                "User-Agent": "SmartAqua-AquaDoc/0.1.0",
            },
            timeout=120.0,
        )

    @property
    def model_id(self) -> str:
        return self._model

    def with_model(self, model: str) -> GroqProvider:
        """Create a clone targeting a specific model for a single request."""
        return GroqProvider(
            api_key=self._api_key,
            model=model,
            base_url=self._base_url,
        )

    async def generate(self, request: LLMRequest, model_override: str | None = None) -> LLMResponse:
        """Generate structured response from Groq with automatic multi-model failover."""
        if not self._api_key:
            raise LLMProviderError("GROQ_API_KEY is not configured.")

        requested_model = resolve_groq_model(model_override or self._model)

        # Construct candidate fallback chain with requested model first
        candidate_models: list[str] = [requested_model]
        for m in AVAILABLE_GROQ_CHAT_MODELS:
            if m not in candidate_models:
                candidate_models.append(m)

        # Build messages payload
        messages: list[dict[str, str]] = []
        if request.system:
            system_content = request.system
            if request.json_schema is not None:
                system_content += (
                    "\n\nIMPORTANT: You must reply strictly with a valid JSON object matching this schema:\n"
                    + json.dumps(request.json_schema)
                )
            messages.append({"role": "system", "content": system_content})

        for msg in request.messages:
            messages.append({"role": msg.role, "content": msg.content})

        started = time.perf_counter()
        data: dict[str, Any] | None = None
        active_model = requested_model
        last_error_msg = ""

        for candidate in candidate_models:
            active_model = candidate
            # Safe token clamp: 2000 for primary, 1500 for fallback models
            clamped_tokens = min(request.max_tokens or 2000, 2000 if candidate == requested_model else 1500)

            payload: dict[str, Any] = {
                "model": active_model,
                "messages": messages,
                "max_tokens": clamped_tokens,
                "temperature": 0.2,
            }
            if request.json_schema is not None:
                payload["response_format"] = {"type": "json_object"}

            try:
                response = await self._client.post(
                    "/chat/completions",
                    json=payload,
                    timeout=request.timeout_seconds,
                )
                response.raise_for_status()
                data = response.json()
                # Successfully received model output
                break
            except httpx.HTTPStatusError as exc:
                status_code = exc.response.status_code
                error_body = exc.response.text
                last_error_msg = f"status {status_code}: {error_body}"
                logger.warning(
                    "groq_model_call_failed_switching_model",
                    extra={"failed_model": active_model, "status": status_code, "body": error_body},
                )

                if status_code == 400 and "refusal" in error_body.lower():
                    raise LLMRefusalError("Groq declined this request due to safety policies.") from exc

                # If JSON validate failed, try once without strict response_format on this model
                if status_code == 400 and "json_validate_failed" in error_body:
                    try:
                        retry_payload = dict(payload)
                        retry_payload.pop("response_format", None)
                        retry_resp = await self._client.post(
                            "/chat/completions",
                            json=retry_payload,
                            timeout=request.timeout_seconds,
                        )
                        retry_resp.raise_for_status()
                        data = retry_resp.json()
                        break
                    except Exception as retry_exc:
                        logger.warning("groq_json_retry_failed", extra={"error": str(retry_exc)})

                # If rate limited (429), brief pause before testing next candidate model
                if status_code == 429:
                    await asyncio.sleep(0.5)

                # Continue loop to switch automatically to the next model
                continue
            except (LLMProviderError, LLMRefusalError):
                raise
            except Exception as exc:
                last_error_msg = str(exc)
                logger.warning(
                    "groq_request_exception_switching_model",
                    extra={"failed_model": active_model, "error": str(exc)},
                )
                continue

        if data is None:
            raise LLMProviderError(f"All Groq fallback models exhausted. Last error: {last_error_msg}")

        latency_ms = (time.perf_counter() - started) * 1000

        choices = data.get("choices", [])
        if not choices:
            raise LLMProviderError("Groq returned an empty choice set.")

        choice = choices[0]
        message = choice.get("message", {})
        text = message.get("content", "")
        stop_reason = choice.get("finish_reason")

        if stop_reason == "content_filter":
            raise LLMRefusalError("Groq content filter triggered.")

        parsed = self._parse_json(text) if request.json_schema is not None else None

        usage_data = data.get("usage", {})
        usage = LLMUsage(
            input_tokens=usage_data.get("prompt_tokens"),
            output_tokens=usage_data.get("completion_tokens"),
        )

        return LLMResponse(
            text=text,
            parsed=parsed,
            model=active_model,
            provider=self.name,
            stop_reason=stop_reason,
            usage=usage,
            latency_ms=latency_ms,
        )

    async def transcribe_audio(
        self,
        file_bytes: bytes,
        filename: str = "audio.wav",
        model: str = DEFAULT_GROQ_WHISPER_MODEL,
        language: str | None = None,
    ) -> str:
        """Transcribe audio recording using Groq Whisper."""
        if not self._api_key:
            raise LLMProviderError("GROQ_API_KEY is not configured for Whisper transcription.")

        files = {"file": (filename, file_bytes, "audio/wav")}
        data: dict[str, Any] = {
            "model": model,
            "response_format": "json",
            "prompt": "Aquaculture farmer questions with accurate punctuation, commas, periods, and question marks: FCR, dissolved oxygen, pH, water temperature, ammonia, feeding rate, and disease diagnosis.",
        }
        if language:
            data["language"] = language

        try:
            response = await self._client.post(
                "/audio/transcriptions",
                files=files,
                data=data,
                timeout=60.0,
            )
            response.raise_for_status()
            res_json = response.json()
            return str(res_json.get("text", "")).strip()
        except Exception as exc:
            logger.exception("groq_whisper_transcription_failed", extra={"model": model})
            raise LLMProviderError("Whisper audio transcription failed.") from exc

    @staticmethod
    def _parse_json(text: str) -> dict[str, Any] | None:
        """Extract and parse structured JSON from model output."""
        try:
            return json.loads(text)  # type: ignore[no-any-return]
        except json.JSONDecodeError:
            # Fallback: extract JSON between ```json ... ``` or first { ... }
            clean = text.strip()
            if clean.startswith("```json"):
                clean = clean[7:]
            if clean.endswith("```"):
                clean = clean[:-3]
            clean = clean.strip()
            try:
                val = json.loads(clean)
                return val if isinstance(val, dict) else None
            except json.JSONDecodeError:
                start = clean.find("{")
                end = clean.rfind("}")
                if start != -1 and end != -1 and end > start:
                    try:
                        val = json.loads(clean[start : end + 1])
                        return val if isinstance(val, dict) else None
                    except json.JSONDecodeError:
                        return None
                return None

    async def aclose(self) -> None:
        await self._client.aclose()
