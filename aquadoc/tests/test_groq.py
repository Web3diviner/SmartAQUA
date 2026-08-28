"""Unit tests for Groq LLM and Audio provider."""

from __future__ import annotations

import json
from unittest.mock import AsyncMock, patch

import httpx
import pytest

from app.config import Settings
from app.errors import LLMProviderError, LLMRefusalError
from app.llm.base import LLMMessage, LLMRequest
from app.llm.groq import GroqProvider
from app.llm.provider import build_llm_provider


@pytest.mark.asyncio
async def test_groq_provider_generation_success() -> None:
    mock_response = {
        "choices": [
            {
                "message": {
                    "role": "assistant",
                    "content": json.dumps({"answer": "Oxygen levels are optimal for catfish."}),
                },
                "finish_reason": "stop",
            }
        ],
        "usage": {
            "prompt_tokens": 120,
            "completion_tokens": 45,
        },
    }

    provider = GroqProvider(api_key="gsk_test_key", model="openai/gpt-oss-120b")

    with patch.object(provider._client, "post", new_callable=AsyncMock) as mock_post:
        mock_http_response = httpx.Response(
            status_code=200,
            json=mock_response,
            request=httpx.Request("POST", "https://api.groq.com/openai/v1/chat/completions"),
        )
        mock_post.return_value = mock_http_response

        request = LLMRequest(
            system="You are AquaDoc.",
            messages=[LLMMessage(role="user", content="Check my pond DO.")],
            json_schema={"type": "object", "properties": {"answer": {"type": "string"}}},
        )

        response = await provider.generate(request)

        assert response.provider == "groq"
        assert response.model == "openai/gpt-oss-120b"
        assert response.parsed == {"answer": "Oxygen levels are optimal for catfish."}
        assert response.usage.input_tokens == 120
        assert response.usage.output_tokens == 45


@pytest.mark.asyncio
async def test_groq_provider_model_override() -> None:
    mock_response = {
        "choices": [
            {
                "message": {
                    "role": "assistant",
                    "content": "Qwen reasoning output",
                },
                "finish_reason": "stop",
            }
        ],
        "usage": {"prompt_tokens": 80, "completion_tokens": 20},
    }

    provider = GroqProvider(api_key="gsk_test_key", model="openai/gpt-oss-120b")

    with patch.object(provider._client, "post", new_callable=AsyncMock) as mock_post:
        mock_http_response = httpx.Response(
            status_code=200,
            json=mock_response,
            request=httpx.Request("POST", "https://api.groq.com/openai/v1/chat/completions"),
        )
        mock_post.return_value = mock_http_response

        request = LLMRequest(
            system="You are AquaDoc.",
            messages=[LLMMessage(role="user", content="Test prompt.")],
        )

        response = await provider.generate(request, model_override="qwen/qwen3.8-27b")

        assert response.model == "qwen/qwen3.8-27b"
        assert response.text == "Qwen reasoning output"


@pytest.mark.asyncio
async def test_groq_provider_handles_refusal() -> None:
    mock_response = {
        "choices": [
            {
                "message": {"role": "assistant", "content": ""},
                "finish_reason": "content_filter",
            }
        ],
    }

    provider = GroqProvider(api_key="gsk_test_key")

    with patch.object(provider._client, "post", new_callable=AsyncMock) as mock_post:
        mock_http_response = httpx.Response(
            status_code=200,
            json=mock_response,
            request=httpx.Request("POST", "https://api.groq.com/openai/v1/chat/completions"),
        )
        mock_post.return_value = mock_http_response

        request = LLMRequest(
            system="System",
            messages=[LLMMessage(role="user", content="Harmful prompt")],
        )

        with pytest.raises(LLMRefusalError):
            await provider.generate(request)


@pytest.mark.asyncio
async def test_groq_whisper_transcription() -> None:
    mock_response = {"text": "What is the feeding rate for fingerlings?"}

    provider = GroqProvider(api_key="gsk_test_key")

    with patch.object(provider._client, "post", new_callable=AsyncMock) as mock_post:
        mock_http_response = httpx.Response(
            status_code=200,
            json=mock_response,
            request=httpx.Request("POST", "https://api.groq.com/openai/v1/audio/transcriptions"),
        )
        mock_post.return_value = mock_http_response

        text = await provider.transcribe_audio(
            b"fake_wav_data",
            filename="voice.wav",
            model="whisper-large-v3-turbo",
        )

        assert text == "What is the feeding rate for fingerlings?"


def test_build_groq_provider_in_dev_falls_back_without_key() -> None:
    settings = Settings(
        app_env="development",
        llm_provider="groq",
        groq_api_key="",
    )
    provider = build_llm_provider(settings)
    assert provider.name == "echo"
