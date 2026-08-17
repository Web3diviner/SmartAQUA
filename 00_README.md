# Smart Aqua Implementation Documentation

## Purpose

This documentation pack defines the technical, security, AI, IoT, data, deployment, testing, and implementation requirements for evolving the existing Smart Fish Feeder platform into **Smart Aqua**, an AI-powered aquaculture management and decision-support platform.

It is written for the current codebase architecture:

- `backend/` — Go backend
- `firmware/` — ESP32/LILYGO firmware
- `mobile/` — Flutter application
- `docs/` — project documentation
- `aquadoc/` — proposed new Python AI/RAG service

The existing system should be preserved and extended rather than rewritten.

## Core Architecture Principle

Smart Aqua is separated into three major intelligence/control layers:

1. **Edge Control Layer**
   - ESP32/LILYGO
   - sensor acquisition
   - motor/auger control
   - local schedules
   - local safety interlocks
   - offline operation

2. **Platform/Operational Layer**
   - existing Go backend
   - authentication
   - farm/device/pond management
   - telemetry ingestion
   - MQTT
   - commands
   - schedules
   - feeding history
   - alerts
   - expert consultation
   - recommendation approval
   - audit logging

3. **AI/Knowledge Layer**
   - AquaDoc Python service
   - RAG
   - LLM orchestration
   - disease decision support
   - farm-context reasoning
   - recommendation generation
   - prediction models
   - explainability

## Documentation Index

| File | Purpose |
|---|---|
| `01_SYSTEM_ARCHITECTURE.md` | Complete target architecture and service boundaries |
| `02_CURRENT_TO_TARGET_MIGRATION.md` | How to evolve the current live feeder safely |
| `03_DATA_MODEL.md` | Production database/domain model |
| `04_AQUADOC_RAG_LLM.md` | Complete RAG + LLM implementation blueprint |
| `05_API_AND_SERVICE_CONTRACTS.md` | REST/internal APIs and service boundaries |
| `06_IOT_MQTT_AND_DEVICE_PROTOCOL.md` | ESP32/MQTT/device communication rules |
| `07_SECURITY_ARCHITECTURE.md` | Security model and hardening requirements |
| `08_THREAT_MODEL.md` | Threats, abuse cases, risks, mitigations |
| `09_TESTING_AND_QA.md` | Testing strategy for backend, firmware, AI and mobile |
| `10_DEVOPS_DEPLOYMENT_OBSERVABILITY.md` | Deployment, CI/CD, monitoring, backups |
| `11_IMPLEMENTATION_ROADMAP.md` | Recommended development sequence |
| `12_PRODUCTION_READINESS_CHECKLIST.md` | Final release checklist |
| `13_CODING_AND_ENGINEERING_STANDARDS.md` | Engineering rules for maintainability |
| `14_AQUADOC_SAFETY_AND_GOVERNANCE.md` | AI safety, provenance, expert escalation and audit |
| `15_AQUADOC_FRONTEND.md` | Temporary React/Vite frontend, debugging UI, security boundaries and Flutter migration |

## Non-Negotiable Design Rules

- Do not allow the LLM to directly control the feeder.
- AquaDoc produces recommendations; the platform produces commands.
- ESP32 must remain capable of safe offline operation.
- Missing sensor data must remain `unknown`, never silently assumed normal.
- AI output must never overwrite deterministic safety rules.
- All device commands must be authenticated, authorized, auditable, and traceable.
- Health/disease outputs must be framed as decision support, not guaranteed diagnosis.
- High-risk or uncertain health cases should support expert escalation.
- RAG sources must be curated and traceable.
- Secrets must never be embedded in Flutter APKs or firmware source.
- Production telemetry must be encrypted in transit.
- Existing feeder functionality must not be broken during AquaDoc development.

## Recommended Repository Structure

```text
Project/
├── backend/
├── firmware/
├── mobile/
├── aquadoc/
│   ├── app/
│   ├── ingestion/
│   ├── tests/
│   ├── migrations/
│   └── Dockerfile
├── infrastructure/
│   ├── docker/
│   ├── scripts/
│   └── environments/
└── docs/
    └── architecture/
```

## Recommended First Milestone

The first complete AquaDoc milestone should be:

> A farmer asks an aquaculture question, AquaDoc retrieves supporting information from approved aquaculture sources, produces a grounded answer with source references, and can optionally include current Smart Aqua farm context supplied by the Go backend.

Do not begin autonomous feeding before this works reliably.
