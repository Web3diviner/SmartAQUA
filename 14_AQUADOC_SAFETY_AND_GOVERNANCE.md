# AquaDoc Safety, Governance and Explainability

## 1. Role

AquaDoc is decision support.

It is not:

- a veterinarian
- a laboratory
- an unrestricted device controller

## 2. Disease Output Policy

AquaDoc should provide:

- likely possibilities
- evidence
- uncertainty
- missing information
- recommended observations
- escalation advice

Avoid wording such as:

> Your fish definitely have disease X.

Prefer:

> The signs are consistent with X, but other causes remain possible.

## 3. Confidence

Confidence should not be arbitrary LLM self-confidence alone.

Combine:

- retrieval relevance
- evidence quality
- completeness of farm data
- rule agreement
- model confidence if applicable

## 4. Expert Escalation

Escalate when:

- mortality is high/rising
- severe symptoms
- low confidence
- conflicting evidence
- treatment decisions are high-risk
- laboratory confirmation may be needed

## 5. Recommendation Risk Tiers

### Tier 0: Informational
No approval.

Examples:
- explain FCR
- explain temperature trend

### Tier 1: Advisory
No physical action.

Examples:
- inspect pond
- check calibration

### Tier 2: Low-Risk Operational
Farmer approval in assisted mode.

Examples:
- reduce ration within configured limit
- delay feeding

### Tier 3: High-Risk
Always explicit human decision; expert confirmation where appropriate.

Examples:
- prolonged feed suspension
- major ration change
- treatment-related operational advice

## 6. Local Safety Override

ESP32 safety rules may stop an unsafe action without farmer approval.

Examples:

- motor stall
- overcurrent
- impossible command
- maximum feed violation
- hardware fault

## 7. Provenance

Every AquaDoc recommendation should record:

```text
input context
retrieved sources
rule versions
model versions
prompt version
timestamp
confidence
```

## 8. Outcome Learning

Record:

```text
Problem
 -> Recommendation
 -> Farmer Action
 -> Device Execution
 -> Outcome
```

Do not automatically retrain on all outcomes.

Use reviewed datasets.

## 9. Knowledge Governance

Every production knowledge document needs:

- source
- owner
- review status
- evidence level
- version/checksum
- ingest date

Documents can be:

- approved
- pending
- deprecated
- rejected

Deprecated/rejected documents should not be retrieved in production.

## 10. Prompt Governance

Prompts must be:

- version-controlled
- reviewed
- evaluated
- rollback-capable

Never modify production prompts without evaluation.
