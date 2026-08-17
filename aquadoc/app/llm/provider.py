"""LLM provider selection.

Swapping providers is a configuration change, not a code change.
"""

from __future__ import annotations

from app.config import Settings
from app.llm.base import LLMProvider


def build_llm_provider(settings: Settings) -> LLMProvider:
    if settings.llm_provider == "claude":
        from app.llm.claude import ClaudeProvider

        return ClaudeProvider(
            api_key=settings.anthropic_api_key,
            model=settings.llm_model,
            enable_refusal_fallback=settings.llm_enable_refusal_fallback,
        )

    if settings.llm_provider == "echo":
        from app.llm.echo import EchoProvider

        return EchoProvider()

    raise ValueError(f"Unsupported LLM_PROVIDER: {settings.llm_provider}")
