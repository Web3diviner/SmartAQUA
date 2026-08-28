"""LLM provider selection.

Swapping providers is a configuration change, not a code change.
"""

from __future__ import annotations

from app.config import Settings
from app.llm.base import LLMProvider


def build_llm_provider(settings: Settings) -> LLMProvider:
    if settings.llm_provider == "groq":
        from app.llm.groq import GroqProvider

        # In dev mode without API key, fall back gracefully to EchoProvider
        if not settings.groq_api_key and not settings.is_production:
            from app.llm.echo import EchoProvider

            return EchoProvider()

        return GroqProvider(
            api_key=settings.groq_api_key,
            model=settings.llm_model,
            base_url=settings.groq_base_url,
        )

    if settings.llm_provider == "claude":
        from app.llm.claude import ClaudeProvider

        if not settings.anthropic_api_key and not settings.is_production:
            from app.llm.echo import EchoProvider

            return EchoProvider()

        return ClaudeProvider(
            api_key=settings.anthropic_api_key,
            model=settings.llm_model,
            enable_refusal_fallback=settings.llm_enable_refusal_fallback,
        )

    if settings.llm_provider == "echo":
        from app.llm.echo import EchoProvider

        return EchoProvider()

    raise ValueError(f"Unsupported LLM_PROVIDER: {settings.llm_provider}")
