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
        q_lower = question.strip().lower()

        # 1. Handle Greetings
        if any(re.search(p, q_lower) for p in [r"^(?:hi|hello|hey|greetings|howdy)\b", r"\bgood (?:morning|afternoon|evening|day)\b", r"^dr\.?\s*(?:fish|aquadoc)"]):
            return (
                "### 🩺 Hello there! I'm Dr. Fish (AquaDoc)\n\n"
                "I am your dedicated **aquatic veterinary doctor and aquaculture consultant** for the Smart Aqua platform.\n\n"
                "I'm here to assist you with:\n"
                "- **Fish Health & Disease Triage** (diagnosing symptoms, lesions, surface piping, mortality)\n"
                "- **Water Quality Assessment** (dissolved oxygen, pH, ammonia, nitrite, temperature)\n"
                "- **Feeding Optimization & FCR Calculations**\n"
                "- **Biosecurity & Pond Husbandry**\n\n"
                "**How is your fish stock doing today?** Please let me know what you are observing in your ponds or tanks!"
            )

        # 2. Handle Identity / Capability inquiries
        if any(re.search(p, q_lower) for p in [r"who are you", r"what can you do", r"tell me about yourself", r"can you help me"]):
            return (
                "### 🩺 Welcome! I am Dr. AquaDoc (or simply Dr. Fish)\n\n"
                "As an aquatic health physician, I provide clinical decision support and husbandry guidance for fish farmers.\n\n"
                "Whether you're managing **African catfish (*Clarias gariepinus*)**, **Nile tilapia (*Oreochromis niloticus*)**, "
                "or operating a recirculating aquaculture system (RAS), you can describe any symptoms, water parameters, or feeding questions, "
                "and I will provide an evidence-grounded diagnosis and step-by-step triage plan.\n\n"
                "What would you like to consult on today?"
            )

        # 3. Handle Thank You & Gratitude
        if any(re.search(p, q_lower) for p in [r"thank(?:s|\s+you)", r"appreciate it", r"great doctor"]):
            return (
                "### 🩺 You're very welcome!\n\n"
                "As **Dr. Fish**, it's my mission to keep your aquatic stock healthy, thriving, and productive. \n\n"
                "Continue monitoring your morning dissolved oxygen and feeding response closely. "
                "Don't hesitate to consult me anytime if you notice behavioral changes or unusual water conditions!"
            )

        # 4. Handle "How are you"
        if "how are you" in q_lower or "how are you doing" in q_lower:
            return (
                "### 🩺 I'm doing well, thank you for asking!\n\n"
                "All systems are online, and I'm ready to review your pond readings, diagnostic symptoms, or feeding plans. \n\n"
                "How are your fish and water quality holding up today?"
            )

        # 5. Handle General Aquaculture / Diagnostic consultation
        if not sources:
            return (
                f"### 🩺 Dr. Fish (AquaDoc) Clinical Assessment\n\n"
                f"Regarding your consultation on **\"{question}\"**:\n\n"
                f"#### 🔬 Clinical Veterinary Guidance\n"
                f"- **Symptom & Husbandry Triage:** Behavioral changes and health anomalies in fish typically indicate acute environmental stressors (dissolved oxygen drop, toxic free ammonia/nitrite, or sudden temperature swings) before progressing to bacterial (*Aeromonas*, *Columnaris*) or parasitic infections.\n"
                f"- **Target Benchmarks:** Ensure dissolved oxygen is maintained ≥ 5.0 mg/L, water temperature is within 26–30°C, and un-ionized ammonia ($NH_3$) remains below 0.05 mg/L.\n\n"
                f"#### 📋 Prescribed Next Steps\n"
                f"1. **Immediate Water Verification:** Test dissolved oxygen at dawn and verify pH and Ammonia levels.\n"
                f"2. **Feeding Adjustment:** Withhold or reduce feeding if fish are sluggish or piping at the surface.\n"
                f"3. **Biosecurity & Quarantine:** Disinfect sampling equipment and isolate affected fish with open sores or dropsy."
            )

        lines = [
            "### 🩺 Dr. AquaDoc (Dr. Fish) Clinical Assessment",
            f"Based on your clinical inquiry regarding **\"{question}\"**, here is the evidence-grounded diagnosis and triage plan:\n",
            "#### 🔬 Clinical Findings & Approved Evidence",
        ]
        for source_id, body in sources[:3]:
            excerpt = " ".join(body.split())[:350]
            lines.append(f"- **[{source_id}]**: {excerpt}")

        if missing:
            lines += [
                "",
                "#### ⚠️ Diagnostic Gaps (Unmeasured Parameters)",
                f"The following critical parameters are currently unknown: **{', '.join(missing)}**. Low dissolved oxygen or toxic ammonia cannot be ruled out until verified.",
            ]

        lines += [
            "",
            "#### 📋 Prescribed Next Steps",
            "1. **Immediate Inspection:** Check surface respiration and verify dissolved oxygen (> 4.0 mg/L).",
            "2. **Feeding Management:** Withhold or adjust feeding if fish exhibit lethargy or temperature falls outside 26–30°C.",
            "3. **Diagnostic Confirmation:** If symptoms worsen or mortality exceeds 2%, isolate sample specimens for veterinary gill/skin scrape.",
        ]
        return "\n".join(lines)

    @staticmethod
    def _to_json(payload: dict[str, Any]) -> str:
        import json

        return json.dumps(payload, ensure_ascii=False)
