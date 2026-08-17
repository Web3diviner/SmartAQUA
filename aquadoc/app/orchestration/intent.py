"""Intent classification.

04_AQUADOC_RAG_LLM.md section 7 puts intent classification at the head of the
retrieval pipeline. It is implemented with keyword rules rather than a model
call: it runs before retrieval on every request, so it must be fast, free, and
deterministic — and its behaviour must be inspectable in the Retrieval
Inspector. A learned classifier can replace `classify` without changing callers.
"""

from __future__ import annotations

import re

from app.schemas.common import Intent
from app.schemas.farm_context import FarmContext

#: Signals per intent. Weighted because a single strong term ("mortality")
#: should outrank several weak ones.
_SIGNALS: dict[Intent, tuple[tuple[str, float], ...]] = {
    Intent.DISEASE: (
        (r"\bdisease[sd]?\b", 3.0),
        (r"\bsick\b", 3.0),
        (r"\bdying\b|\bdeath[s]?\b|\bmortalit(?:y|ies)\b", 3.0),
        (r"\blesion[s]?\b|\bulcer[s]?\b|\bsore[s]?\b", 3.0),
        (r"\bparasit\w+\b|\bfungus\b|\bfungal\b|\bbacteri\w+\b", 2.5),
        (r"\binfect\w+\b", 2.5),
        (r"\bwhite spot[s]?\b|\bfin rot\b|\bgill\b", 2.0),
        (r"\bsymptom[s]?\b|\btreat\w*\b", 2.0),
        (r"\blethargic\b|\bgasping\b|\bfloating\b|\bswimming (?:oddly|strangely)\b", 2.0),
    ),
    Intent.WATER_QUALITY: (
        (r"\bwater quality\b", 3.0),
        (r"\bdissolved oxygen\b|\bDO\b", 3.0),
        (r"\bammonia\b|\bnitrite\b|\bnitrate\b", 3.0),
        (r"\bph\b", 2.5),
        (r"\bturbidity\b|\bcloudy\b|\bmurky\b|\bgreen water\b", 2.5),
        (r"\btemperature\b|\bwarm\b|\bcold\b", 1.5),
        (r"\baerat\w+\b|\bwater change\b", 2.0),
    ),
    Intent.FEEDING: (
        (r"\bfeed(?:ing|s)?\b", 2.5),
        (r"\bration\b", 3.0),
        (r"\bpellet[s]?\b", 2.5),
        (r"\bfcr\b|\bfeed conversion\b", 3.0),
        (r"\bappetite\b|\beating\b|\bnot eating\b|\brefus\w+\b", 2.5),
        (r"\bhow much (?:to )?feed\b", 3.0),
        (r"\bgrowth rate\b|\bweight gain\b", 1.5),
    ),
}

#: Words indicating the question is about *this* farm rather than aquaculture
#: in general. "Why are my fish eating less?" vs "What is FCR?".
_FARM_SPECIFIC = re.compile(
    r"\bmy\b|\bour\b|\bthis (?:pond|tank|farm|batch)\b|\btoday\b|\bright now\b|"
    r"\byesterday\b|\bthis (?:week|morning)\b|\bcurrently\b",
    re.IGNORECASE,
)

#: Purely educational phrasing.
_EDUCATIONAL = re.compile(
    r"^\s*(?:what (?:is|are|does)|how (?:do(?:es)?|can) (?:you|one|i)\b.*\bgenerally\b|"
    r"define|explain (?:what|the (?:concept|meaning)))",
    re.IGNORECASE,
)


def classify(question: str, context: FarmContext | None = None) -> Intent:
    """Classify a question into a retrieval/prompt intent.

    A supplied farm context raises farm-specific intents but does not force
    them: a farmer with a pond selected can still ask "what is FCR?".
    """
    text = question.lower()

    scores: dict[Intent, float] = {}
    for intent, patterns in _SIGNALS.items():
        score = sum(weight for pattern, weight in patterns if re.search(pattern, text, re.I))
        if score:
            scores[intent] = score

    has_context = context is not None and not context.is_empty()
    farm_specific = bool(_FARM_SPECIFIC.search(question))
    educational = bool(_EDUCATIONAL.match(question.strip()))

    if not scores:
        if educational or not has_context:
            return Intent.GENERAL_AQUACULTURE
        return Intent.FARM_ASSESSMENT if farm_specific else Intent.GENERAL_AQUACULTURE

    best_intent = max(scores.items(), key=lambda item: item[1])[0]

    # A topical question with no farm framing and no farm data is educational —
    # "what causes fin rot?" is general knowledge, not a case assessment.
    if educational and not farm_specific:
        return Intent.GENERAL_AQUACULTURE

    if not farm_specific and not has_context:
        return Intent.GENERAL_AQUACULTURE

    return best_intent


def is_health_related(intent: Intent, context: FarmContext | None) -> bool:
    """Whether health-case escalation rules apply.

    True for disease questions, and for any question where the farm context
    itself reports mortality, symptoms, or an open case — a feeding question
    asked during a mortality event is still a health situation.
    """
    if intent is Intent.DISEASE:
        return True
    if context is None:
        return False
    health = context.health
    return bool(
        health.reported_symptoms
        or health.active_disease_case
        or (health.mortality_24h is not None and health.mortality_24h > 0)
    )
