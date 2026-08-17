"""Deterministic decision rules.

These run without the language model and are individually unit-testable
(13_CODING_AND_ENGINEERING_STANDARDS.md). AI output can never override them
(00_README.md, non-negotiable design rules).
"""

from app.rules.feeding import (
    FeedingAssessment,
    assess_feeding,
    feed_rate_fraction,
    q10_factor,
)
from app.rules.safety import (
    SafetyOutcome,
    classify_action_tier,
    enforce,
    requires_approval,
)
from app.rules.water_quality import evaluate_water_quality, worst_status

#: Composite version recorded in provenance. Bump the component version in the
#: module that changed; this string identifies the whole rule set.
RULES_VERSION = "water_quality/v1+feeding/v1+safety/v1"

__all__ = [
    "RULES_VERSION",
    "FeedingAssessment",
    "SafetyOutcome",
    "assess_feeding",
    "classify_action_tier",
    "enforce",
    "evaluate_water_quality",
    "feed_rate_fraction",
    "q10_factor",
    "requires_approval",
    "worst_status",
]
