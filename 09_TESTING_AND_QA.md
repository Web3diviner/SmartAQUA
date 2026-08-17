# Testing and Quality Assurance Strategy

## 1. Testing Pyramid

### Unit Tests
Fast and extensive.

### Integration Tests
Database, MQTT, API, RAG.

### End-to-End Tests
Flutter -> Go -> MQTT -> simulated ESP32.

### Field Tests
Real feeder and real farm conditions.

## 2. Backend Tests

Test:

- authentication
- authorization
- device ownership
- schedule CRUD
- manual feed
- feeding logs
- farm/pond CRUD
- production cycles
- recommendation lifecycle
- expert consultation
- audit logging

## 3. Firmware Tests

Test:

- sensor reads
- invalid sensor values
- offline feeding
- reconnect
- motor stall
- duplicate command
- expired command
- maximum feed limit
- power interruption
- buffer recovery
- corrupted message

## 4. MQTT Tests

Test:

- device connects
- telemetry ingestion
- alerts
- commands
- command result
- duplicate messages
- delayed messages
- malformed payloads
- unauthorized topic access

## 5. RAG Tests

Create a gold dataset.

Test:

- correct source retrieved
- correct species filter
- irrelevant source rejection
- missing-data handling
- citation accuracy
- page attribution
- prompt injection resistance

## 6. LLM Tests

Evaluate:

- groundedness
- uncertainty
- disease safety
- missing sensor disclosure
- source usage
- refusal to invent values
- expert escalation
- structured JSON validity

## 7. Disease Workflow Tests

Scenarios:

- mild symptoms
- severe mortality
- missing temperature
- no pH/DO available
- conflicting symptoms
- poor-confidence evidence
- expert required

## 8. Security Tests

- authentication bypass
- IDOR/cross-farm access
- SQL injection
- XSS in messages
- rate limit
- token reuse
- revoked refresh token
- MQTT ACL violation
- replay command
- malicious RAG document
- unsafe file upload

## 9. Mobile Tests

- login
- offline state
- stale sensor UI
- recommendation approval
- warning display
- AquaDoc chat
- source display
- token expiry
- device unavailable

## 10. Regression Suite

Every release must validate current feeder functionality.

Minimum:

- registration/login
- device binding
- manual feed
- schedules
- telemetry
- Q10
- logs
- alerts

## 11. AI Release Evaluation

Do not release a new prompt/model without running an evaluation set.

Record:

- model
- prompt version
- retrieval version
- pass rate
- hallucination rate
- safety pass rate
- cost
- latency
