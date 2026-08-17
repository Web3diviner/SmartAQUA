"""JSON Schema for the model's structured output.

04_AQUADOC_RAG_LLM.md section 12: do not rely on free-form text alone.

Note what the model is *not* asked for. It does not produce `confidence`
(computed from retrieval quality, evidence level, data completeness, and rule
agreement — 14_AQUADOC_SAFETY_AND_GOVERNANCE.md section 3), `sources`
(constructed from what retrieval actually returned, so citations cannot be
fabricated), or `missing_data` (derived from the farm context). The model
supplies only the parts that require language understanding.
"""

from __future__ import annotations

from typing import Any

#: Bump when the schema changes shape; recorded in provenance alongside the
#: prompt version so a stored response can be interpreted later.
RESPONSE_SCHEMA_VERSION = "v1"

ANSWER_SCHEMA: dict[str, Any] = {
    "type": "object",
    "additionalProperties": False,
    "required": [
        "answer",
        "possible_causes",
        "recommended_actions",
        "model_confidence",
        "risk_level",
        "expert_escalation",
        "escalation_reasons",
    ],
    "properties": {
        "answer": {
            "type": "string",
            "description": (
                "The answer for the farmer. Grounded in the supplied sources and farm "
                "context. State uncertainty plainly. Never assert a measurement that was "
                "not supplied."
            ),
        },
        "possible_causes": {
            "type": "array",
            "description": (
                "Candidate explanations, most likely first. Empty for purely educational "
                "questions."
            ),
            "items": {
                "type": "object",
                "additionalProperties": False,
                "required": ["name", "confidence", "explanation", "supporting_source_ids"],
                "properties": {
                    "name": {"type": "string"},
                    "confidence": {"type": "number"},
                    "explanation": {"type": "string"},
                    "supporting_source_ids": {
                        "type": "array",
                        "items": {"type": "string"},
                        "description": "IDs of supplied sources supporting this cause.",
                    },
                },
            },
        },
        "recommended_actions": {
            "type": "array",
            "description": (
                "Proposed actions. These are recommendations only — never device "
                "commands. Observation and inspection actions are preferred when data "
                "is incomplete."
            ),
            "items": {
                "type": "object",
                "additionalProperties": False,
                "required": ["action", "tier", "reason", "urgency"],
                "properties": {
                    "action": {"type": "string"},
                    "tier": {
                        "type": "string",
                        "enum": [
                            "tier_0_informational",
                            "tier_1_advisory",
                            "tier_2_low_risk_operational",
                            "tier_3_high_risk",
                        ],
                    },
                    "reason": {"type": "string"},
                    "urgency": {
                        "type": "string",
                        "enum": ["informational", "watch", "elevated", "high"],
                    },
                },
            },
        },
        "model_confidence": {
            "type": "number",
            "description": (
                "How well the supplied sources and context support this answer, 0-1. "
                "This is one input to the final confidence score, not the score itself."
            ),
        },
        "risk_level": {
            "type": "string",
            "enum": ["informational", "watch", "elevated", "high"],
        },
        "expert_escalation": {
            "type": "boolean",
            "description": "True when a qualified human should review this case.",
        },
        "escalation_reasons": {
            "type": "array",
            "items": {"type": "string"},
        },
    },
}
