"""Versioned prompt loading.

04_AQUADOC_RAG_LLM.md section 13 and 14_AQUADOC_SAFETY_AND_GOVERNANCE.md
section 10: prompts are version-controlled, reviewed, and rollback-capable, and
every production response records which prompt version produced it.

Prompts live as files rather than string literals so they diff cleanly in review
and so changing one is a visible, reviewable event.
"""

from __future__ import annotations

from functools import lru_cache
from pathlib import Path

from app.schemas.common import Intent

_PROMPT_DIR = Path(__file__).parent

#: Intent -> prompt file. Adding a version means adding a file and changing this
#: map, so the previous version stays available for rollback.
PROMPT_VERSIONS: dict[Intent, str] = {
    Intent.GENERAL_AQUACULTURE: "general_v1",
    Intent.UNKNOWN: "general_v1",
    Intent.FARM_ASSESSMENT: "farm_assessment_v1",
    Intent.WATER_QUALITY: "farm_assessment_v1",
    Intent.DISEASE: "farm_assessment_v1",
    Intent.FEEDING: "feeding_explanation_v1",
}


@lru_cache(maxsize=16)
def load_prompt(version: str) -> str:
    path = _PROMPT_DIR / f"{version}.md"
    if not path.is_file():
        raise FileNotFoundError(f"Prompt version not found: {version}")
    return path.read_text(encoding="utf-8").strip()


def prompt_for_intent(intent: Intent) -> tuple[str, str]:
    """Return (version, text) for an intent."""
    version = PROMPT_VERSIONS.get(intent, PROMPT_VERSIONS[Intent.GENERAL_AQUACULTURE])
    return version, load_prompt(version)


__all__ = ["PROMPT_VERSIONS", "load_prompt", "prompt_for_intent"]
