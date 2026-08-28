"""Prompt assembly.

Builds the user-turn payload from the pond state, deterministic rule findings,
and retrieved sources.

Two constraints shape this module:

1. **Untrusted content isolation** (07_SECURITY_ARCHITECTURE.md section 8).
   Retrieved documents and farmer-written text are treated as untrusted. They
   are placed inside labelled XML-ish blocks and stripped of any sequence that
   could close a block early and inject instructions at the prompt level.

2. **Missing data stays visible** (04_AQUADOC_RAG_LLM.md section 9). Unknown
   measurements are serialised as `null` *and* listed explicitly, so the model
   cannot quietly skip over them.
"""

from __future__ import annotations

import json
import re

from app.rag.citations import citation_id
from app.rag.reranking import Candidate
from app.schemas.chat import RuleFinding
from app.schemas.common import MEASUREMENT_LABELS
from app.schemas.farm_context import FarmContext

#: Anything that looks like a block delimiter is neutralised before it reaches
#: the prompt, so untrusted text cannot escape its container.
_TAG_LIKE = re.compile(r"<\s*/?\s*(?:source|sources|question|pond_state|rule_findings|"
                       r"missing_measurements|system|instructions?)\b[^>]*>", re.IGNORECASE)

_MAX_SOURCE_CHARS = 1200


def sanitize_untrusted(text: str) -> str:
    """Neutralise delimiter-like sequences in untrusted text.

    Defends the prompt structure itself. It is not a complete defence against
    prompt injection — the system prompt also instructs the model to ignore
    instructions found in retrieved content, and the safety layer re-checks the
    output afterwards.
    """
    return _TAG_LIKE.sub("[removed-markup]", text)


def build_sources_block(candidates: list[Candidate]) -> str:
    """Render retrieved passages with stable citation IDs.

    Attribution metadata (title, page, evidence level) is rendered as attributes
    so the model can cite accurately without being able to alter it.
    """
    if not candidates:
        return "<sources>\n(no approved sources matched this question)\n</sources>"

    parts = ["<sources>"]
    for index, candidate in enumerate(candidates):
        content = sanitize_untrusted(candidate.content)[:_MAX_SOURCE_CHARS]
        attributes = [
            f'id="{citation_id(index)}"',
            f'title="{_attribute(candidate.title)}"',
            f'evidence_level="{candidate.evidence_level.value}"',
        ]
        if candidate.page_number is not None:
            attributes.append(f'page="{candidate.page_number}"')
        if candidate.section:
            attributes.append(f'section="{_attribute(candidate.section)}"')
        if candidate.year is not None:
            attributes.append(f'year="{candidate.year}"')

        parts.append(f"<source {' '.join(attributes)}>")
        parts.append(content)
        parts.append("</source>")
    parts.append("</sources>")
    return "\n".join(parts)


def build_pond_state_block(context: FarmContext | None) -> str:
    """Serialise the computed pond state.

    Unknown measurements are emitted as JSON `null`, exactly as
    04_AQUADOC_RAG_LLM.md section 9 specifies. They are never omitted, because
    an absent key reads as "not applicable" rather than "not measured".
    """
    if context is None or context.is_empty():
        return "<pond_state>\n(no farm context was supplied with this question)\n</pond_state>"

    state = {
        "farm_id": context.farm_id,
        "pond_id": context.pond_id,
        "pond_name": context.pond_name,
        "species": context.species,
        "life_stage": context.life_stage,
        "population": context.population,
        "average_weight_g": context.average_weight_g,
        "biomass_kg": context.derived_biomass_kg(),
        "pond_volume_liters": context.pond_volume_liters,
        "water": {
            "temperature_c": context.water.temperature_c,
            "ph": context.water.ph,
            "dissolved_oxygen_mg_l": context.water.dissolved_oxygen_mg_l,
            "turbidity_ntu": context.water.turbidity_ntu,
            "ammonia_mg_l": context.water.ammonia_mg_l,
            "nitrite_mg_l": context.water.nitrite_mg_l,
            "measured_at": _iso(context.water.measured_at),
        },
        "feeding": {
            "daily_ration_g": context.feeding.daily_ration_g,
            "last_feeding_at": _iso(context.feeding.last_feeding_at),
            "last_feeding_g": context.feeding.last_feeding_g,
            "feeds_per_day": context.feeding.feeds_per_day,
            "feed_acceptance": context.feeding.feed_acceptance,
        },
        "health": {
            "mortality_24h": context.health.mortality_24h,
            "mortality_7d": context.health.mortality_7d,
            "active_disease_case": context.health.active_disease_case,
            "reported_symptoms": [
                sanitize_untrusted(symptom) for symptom in context.health.reported_symptoms
            ],
        },
    }
    body = json.dumps(state, indent=2, ensure_ascii=False)
    return f"<pond_state>\n{body}\n</pond_state>"


def build_missing_block(context: FarmContext | None) -> str:
    if context is None:
        return "<missing_measurements>all (no farm context supplied)</missing_measurements>"
    labels = context.water.missing_labels()
    if not labels:
        return "<missing_measurements>none</missing_measurements>"
    return f"<missing_measurements>{', '.join(labels)}</missing_measurements>"


def build_rule_findings_block(findings: list[RuleFinding]) -> str:
    """Render deterministic findings.

    These are computed by the platform. The prompt tells the model they take
    precedence over its own impression.
    """
    if not findings:
        return "<rule_findings>\n(no deterministic findings for this question)\n</rule_findings>"

    lines = ["<rule_findings>"]
    for finding in findings:
        lines.append(f"- [{finding.status}] {finding.rule_id}: {finding.summary}")
    lines.append("</rule_findings>")
    return "\n".join(lines)


def build_conversation_history_block(history: list[dict[str, str]] | None) -> str:
    """Render recent dialogue turns so the model maintains conversational continuity."""
    if not history:
        return ""
    lines = ["<conversation_history>"]
    for turn in history[-6:]:  # Keep up to last 6 dialogue turns for context memory
        role = "farmer" if turn.get("role") == "user" else "doctor"
        content = sanitize_untrusted(turn.get("content", ""))
        lines.append(f"<{role}>\n{content}\n</{role}>")
    lines.append("</conversation_history>")
    return "\n".join(lines)


def build_user_turn(
    *,
    question: str,
    context: FarmContext | None,
    findings: list[RuleFinding],
    candidates: list[Candidate],
    history: list[dict[str, str]] | None = None,
) -> str:
    """Assemble the complete user-turn payload with conversation memory."""
    blocks = []
    hist_block = build_conversation_history_block(history)
    if hist_block:
        blocks.append(hist_block)
    blocks.extend(
        [
            f"<question>\n{sanitize_untrusted(question)}\n</question>",
            build_pond_state_block(context),
            build_missing_block(context),
            build_rule_findings_block(findings),
            build_sources_block(candidates),
        ]
    )
    return "\n\n".join(blocks)


def missing_measurement_keys(context: FarmContext | None) -> tuple[list[str], list[str]]:
    """Return (keys, labels) for measurements that were not taken."""
    if context is None:
        keys = list(MEASUREMENT_LABELS.keys())
        return keys, [MEASUREMENT_LABELS[key] for key in keys]
    return context.water.missing(), context.water.missing_labels()


def _attribute(value: str) -> str:
    """Escape a value for use inside a quoted attribute."""
    return sanitize_untrusted(value).replace('"', "'").replace("\n", " ")[:200]


def _iso(value: object) -> str | None:
    return value.isoformat() if hasattr(value, "isoformat") else None
