# Smart Aqua System Architecture

## 1. System Goal

Smart Aqua should evolve from a smart feeder into a modular precision-aquaculture platform.

The feeder remains an important actuator, but the platform becomes responsible for:

- farm management
- pond/tank management
- stock and production cycles
- sensor telemetry
- feeding control
- disease decision support
- AquaDoc Q&A
- recommendations
- expert escalation
- reporting
- alerts
- device lifecycle management

## 2. Target Architecture

```text
Farmer / Expert / Admin
          |
      Flutter App
          |
       HTTPS
          |
      Go Backend
          |
   +------+-------------------+------------------+
   |                          |                  |
Farm Domain              Device Domain      AquaDoc Gateway
   |                          |                  |
PostgreSQL                MQTT Broker       Python AquaDoc
   |                          |                  |
   |                        ESP32         +------+------+
   |                          |            |      |     |
   |                    Feeder/Sensors    RAG   Rules   LLM
   |
Redis / Cache / Events
```

## 3. Existing Components to Preserve

### Go Backend
Keep as the authoritative operational backend.

Responsibilities:

- authentication and authorization
- users
- devices
- MQTT bridge
- feeding schedules
- feeding history
- sensor ingestion
- command handling
- alerts
- farm/pond/stock APIs
- recommendation persistence
- expert consultation APIs
- payment status
- audit records
- access control

### Flutter App
Keep as the main farmer-facing interface.

Extend with:

- farms
- ponds/tanks
- water quality
- AquaDoc
- disease cases
- recommendations
- experts
- reports

### Firmware
Keep existing local control philosophy.

Responsibilities:

- read sensors
- execute local schedule
- calculate/consume deterministic feeding parameters
- actuate motor
- detect stalls/failures
- buffer unsent telemetry
- reconnect automatically
- reject unsafe commands
- support OTA updates eventually

## 4. New AquaDoc Service

Recommended technology:

- Python
- FastAPI
- PostgreSQL/pgvector
- external LLM API initially
- embeddings API or local embedding model
- optional Redis
- background workers for document ingestion

AquaDoc responsibilities:

- aquaculture question answering
- RAG retrieval
- farm-context reasoning
- disease assessment support
- recommendation generation
- confidence scoring
- explanation
- source attribution
- expert escalation decisions
- prediction-model integration

AquaDoc should not own:

- user authentication
- device authorization
- direct MQTT access
- device actuation
- financial transaction truth
- final command authorization

## 5. Three-Brain Model

### Edge Brain
ESP32.

Fast, deterministic, safe.

### Decision Brain
Go backend + deterministic rules/models.

Handles operational constraints, Q10, policy limits, recommendation/command lifecycle.

### Knowledge/Language Brain
AquaDoc.

Understands farmer language, retrieves knowledge, explains decisions, supports disease/farm reasoning.

## 6. Recommended Data Flow

### Telemetry

```text
Sensor
 -> ESP32
 -> MQTT
 -> Go Backend
 -> PostgreSQL
 -> farm/pond digital state
 -> AquaDoc context when requested
```

### Farmer Question

```text
Flutter
 -> Go Backend
 -> build farm context
 -> AquaDoc
 -> RAG + rules + LLM
 -> structured response
 -> Go Backend
 -> Flutter
```

### AI Recommendation

```text
AquaDoc
 -> Recommendation
 -> Go Backend
 -> Farmer
 -> Approve/Reject
 -> if approved: Command
 -> MQTT
 -> ESP32
 -> Execution Result
 -> Go Backend
 -> Outcome
```

## 7. Recommendation vs Command

Never merge these concepts.

### Recommendation
An intelligent proposal.

Fields should include:

- reason
- confidence
- evidence
- source/model provenance
- proposed action
- severity
- approval requirement
- expiry

### Command
A concrete device instruction.

Fields should include:

- device ID
- command ID
- command type
- payload
- requested by
- source recommendation
- created time
- expiry/timeout
- execution result

## 8. Digital Pond State

AquaDoc should consume a computed pond state, not random database rows.

Example:

```json
{
  "pond_id": "POND-001",
  "species": "Clarias gariepinus",
  "population": 500,
  "average_weight_g": 250,
  "biomass_kg": 125,
  "water": {
    "temperature_c": 29.8,
    "ph": null,
    "dissolved_oxygen_mg_l": null,
    "turbidity_ntu": null
  },
  "feeding": {
    "daily_ration_g": 3750,
    "last_feeding_g": 1800
  },
  "health": {
    "mortality_24h": 0,
    "active_disease_case": false
  }
}
```

`null` means unknown.

It must never be interpreted as safe/normal.
