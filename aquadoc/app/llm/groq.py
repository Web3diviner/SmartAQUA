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
        """Generate structured response from Groq with resilient fallbacks."""
        if not self._api_key:
            raise LLMProviderError("GROQ_API_KEY is not configured.")

        requested_model = model_override or self._model
        active_model = resolve_groq_model(requested_model)

        # Build messages payload
        messages: list[dict[str, str]] = []
        if request.system:
            # When requesting JSON schema, append explicit formatting instructions
            system_content = request.system
            if request.json_schema is not None:
                system_content += (
                    "\n\nIMPORTANT: You must reply strictly with a valid JSON object matching this schema:\n"
                    + json.dumps(request.json_schema)
                )
            messages.append({"role": "system", "content": system_content})

        for msg in request.messages:
            messages.append({"role": msg.role, "content": msg.content})

        # Groq TPM budgets: clamp max_tokens to 2000 to prevent 413 rate limit errors
        clamped_tokens = min(request.max_tokens or 2000, 2000)

        payload: dict[str, Any] = {
            "model": active_model,
            "messages": messages,
            "max_tokens": clamped_tokens,
            "temperature": 0.2,
        }

        # Request JSON mode if a schema is present
        if request.json_schema is not None:
            payload["response_format"] = {"type": "json_object"}

        started = time.perf_counter()
        data: dict[str, Any] | None = None

        try:
            response = await self._client.post(
                "/chat/completions",
                json=payload,
                timeout=request.timeout_seconds,
            )
            response.raise_for_status()
            data = response.json()
        except httpx.HTTPStatusError as exc:
            status_code = exc.response.status_code
            error_body = exc.response.text
            logger.warning(
                "groq_http_error",
                extra={"status": status_code, "model": active_model, "body": error_body},
            )

            # Handle 404/400 model not found by falling back to primary high-availability models
            if (status_code in (400, 404)) and ("model" in error_body.lower() or "not found" in error_body.lower()):
                fallback_models = ["openai/gpt-oss-120b", "openai/gpt-oss-20b", "groq/compound-mini"]
                for fb_model in fallback_models:
                    if fb_model == active_model:
                        continue
                    try:
                        fb_payload = dict(payload)
                        fb_payload["model"] = fb_model
                        fb_resp = await self._client.post(
                            "/chat/completions",
                            json=fb_payload,
                            timeout=request.timeout_seconds,
                        )
                        fb_resp.raise_for_status()
                        data = fb_resp.json()
                        active_model = fb_model
                        break
                    except Exception:
                        continue

            if data is None and status_code == 400 and "json_validate_failed" in error_body:
                try:
                    res_err = exc.response.json()
                    failed_gen = res_err.get("error", {}).get("failed_generation")
                    if failed_gen:
                        data = {
                            "choices": [
                                {
                                    "message": {"content": json.dumps({"answer": failed_gen})},
                                    "finish_reason": "stop",
                                }
                            ]
                        }
                    else:
                        retry_payload = dict(payload)
                        retry_payload.pop("response_format", None)
                        retry_resp = await self._client.post(
                            "/chat/completions",
                            json=retry_payload,
                            timeout=request.timeout_seconds,
                        )
                        retry_resp.raise_for_status()
                        data = retry_resp.json()
                except Exception as retry_exc:
                    logger.warning("groq_json_retry_failed", extra={"error": str(retry_exc)})
            elif data is None and status_code == 429:
                retry_seconds = 1.0
                if "try again in" in error_body:
                    try:
                        match = re.search(r"try again in ([\d\.]+)s", error_body)
                        if match:
                            retry_seconds = min(float(match.group(1)) + 0.3, 3.0)
                        else:
                            ms_match = re.search(r"try again in ([\d\.]+)ms", error_body)
                            if ms_match:
                                retry_seconds = min((float(ms_match.group(1)) / 1000.0) + 0.2, 3.0)
                    except Exception:
                        retry_seconds = 1.0
                await asyncio.sleep(retry_seconds)
                try:
                    retry_payload = dict(payload)
                    retry_payload["model"] = "openai/gpt-oss-20b"
                    retry_resp = await self._client.post(
                        "/chat/completions",
                        json=retry_payload,
                        timeout=request.timeout_seconds,
                    )
                    retry_resp.raise_for_status()
                    data = retry_resp.json()
                    active_model = "openai/gpt-oss-20b"
                except Exception as rate_exc:
                    logger.warning("groq_rate_limit_retry_failed", extra={"error": str(rate_exc)})
            elif data is None and status_code == 400 and "refusal" in error_body.lower():
                raise LLMRefusalError("Groq declined this request due to safety policies.") from exc

            if data is None:
                # Re-raise to allow upstream diagnostics or retry rather than generic fallback
                raise LLMProviderError(f"Groq API error (status {status_code}): {error_body}") from exc
        except (LLMProviderError, LLMRefusalError):
            raise
        except Exception as exc:
            logger.exception("groq_request_failed", extra={"model": active_model})
            raise LLMProviderError(f"Groq API call failed: {exc}") from exc

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
            model=requested_model,
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
