"""Deterministic offline `LLMProvider` stub.

Purpose: let the full pipeline — ingestion, retrieval, rules, orchestration,
API, frontend — run and be tested end to end with no API key and no network.

This is NOT a language model. It performs extractive assembly of the retrieved
passages and nothing else: it does not reason, paraphrase, or infer. The
settings validator refuses to start a production instance configured with it.
"""

from __future__ import annotations

import re
import time
from typing import Any

from app.llm.base import LLMProvider, LLMRequest, LLMResponse, LLMUsage

#: The prompt builder emits retrieved passages inside these delimiters.
_SOURCE_BLOCK = re.compile(
    r"<source\s+id=\"(?P<id>[^\"]+)\"[^>]*>(?P<body>.*?)</source>",
    re.DOTALL,
)
_QUESTION_BLOCK = re.compile(r"<question>(?P<body>.*?)</question>", re.DOTALL)
_MISSING_BLOCK = re.compile(r"<missing_measurements>(?P<body>.*?)</missing_measurements>", re.DOTALL)


class EchoProvider(LLMProvider):
    """Extractive, deterministic, offline."""

    name = "echo"

    def __init__(self, model: str = "echo-extractive-v1") -> None:
        self._model = model

    @property
    def model_id(self) -> str:
        return self._model

    async def generate(self, request: LLMRequest) -> LLMResponse:
        started = time.perf_counter()
        prompt = "\n\n".join(message.content for message in request.messages)

        question = self._extract(_QUESTION_BLOCK, prompt) or "your question"
        sources = _SOURCE_BLOCK.findall(prompt)
        missing_raw = self._extract(_MISSING_BLOCK, prompt) or ""
        missing = [item.strip() for item in missing_raw.split(",") if item.strip()]

        answer = self._compose_answer(question, sources, missing)
        payload: dict[str, Any] = {
            "answer": answer,
            "possible_causes": [],
            "recommended_actions": [],
            "model_confidence": 0.4 if sources else 0.05,
            "risk_level": "informational",
            "expert_escalation": False,
            "escalation_reasons": [],
        }

        text = self._to_json(payload)
        return LLMResponse(
            text=text,
            parsed=payload if request.json_schema is not None else None,
            model=self._model,
            provider=self.name,
            stop_reason="end_turn",
            usage=LLMUsage(input_tokens=len(prompt) // 4, output_tokens=len(text) // 4),
            latency_ms=(time.perf_counter() - started) * 1000,
        )

    @staticmethod
    def _extract(pattern: re.Pattern[str], text: str) -> str | None:
        match = pattern.search(text)
        return match.group("body").strip() if match else None

    @staticmethod
    def _compose_answer(question: str, sources: list[tuple[str, str]], missing: list[str]) -> str:
        if not sources:
            return (
                "No approved knowledge sources matched this question, so there is nothing "
                "to ground an answer on. (Offline development provider — no language model "
                "was used.)"
            )

        lines = [
            f"Regarding: {question}",
            "",
            "The following approved sources are relevant:",
        ]
        for source_id, body in sources[:3]:
            excerpt = " ".join(body.split())[:400]
            lines.append(f"- [{source_id}] {excerpt}")

        if missing:
            lines += [
                "",
                f"The following measurements are not currently available and could not be "
                f"evaluated: {', '.join(missing)}.",
            ]
        lines += [
            "",
            "(Offline development provider — passages are quoted verbatim, not interpreted.)",
        ]
        return "\n".join(lines)

    @staticmethod
    def _to_json(payload: dict[str, Any]) -> str:
        import json

        return json.dumps(payload, ensure_ascii=False)
