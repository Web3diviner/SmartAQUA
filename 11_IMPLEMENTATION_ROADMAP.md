# Smart Aqua Implementation Roadmap

## Stage 1: Stabilize Current Product

Deliverables:

- production tag
- architecture inventory
- environment inventory
- regression tests
- secrets audit
- DB backup
- firmware version tracking

Exit criteria:
Current feeder functions reliably.

## Stage 2: Farm/Pond Domain

Build:

- Farm
- FarmMember
- Pond
- ProductionCycle
- SamplingRecord
- MortalityRecord

Update Flutter with basic farm/pond setup.

Exit criteria:
A device can be linked to a pond without breaking existing users.

## Stage 3: Sensor Normalization

Build:

- Sensor
- SensorReading
- SensorCalibration
- quality flags

Continue supporting legacy SensorData.

Exit criteria:
Temperature can flow through new generic sensor model.

## Stage 4: AquaDoc Standalone MVP

Build:

- FastAPI service
- LLM provider abstraction
- embedding provider abstraction
- PostgreSQL + pgvector
- RAG ingestion
- semantic retrieval
- source citations
- structured answers

Exit criteria:
Approved-document Q&A works with good grounding.

## Stage 5: Farm-Aware AquaDoc

Connect simulated farm context.

Then Go read-only integration.

Exit criteria:
AquaDoc can answer:
"Why are my fish eating less?"
using actual Smart Aqua context.

## Stage 6: Disease Decision Support

Build:

- DiseaseCase
- symptom workflow
- RAG disease retrieval
- triage
- confidence
- expert escalation

Exit criteria:
Safe differential-style support with uncertainty.

## Stage 7: Flutter AquaDoc UI

Add:

- chat
- source references
- pond context
- disease assessment
- recommendation cards

Exit criteria:
End-to-end farmer experience works.

## Stage 8: New Sensors

Integrate pH and turbidity.

Then DO later.

Each sensor must pass:

- calibration
- firmware
- MQTT
- backend
- mobile
- AquaDoc
- alert
- field tests

## Stage 9: Recommendation Service

Build:

- Recommendation
- evidence
- confidence
- approval/rejection
- audit

Exit criteria:
Recommendations exist independently from commands.

## Stage 10: Command Linking

Approved recommendation -> command.

Exit criteria:
Trace:

```text
Observation
 -> Analysis
 -> Recommendation
 -> Approval
 -> Command
 -> Execution
 -> Outcome
```

## Stage 11: Experts

Build:

- expert profiles
- consultation
- case summary
- messages
- payment integration
- expert outcome

## Stage 12: Production Intelligence

Later:

- growth prediction
- mortality prediction
- advanced water-quality models
- camera behavior
- feed acceptance
- digital twin
- bounded automation
