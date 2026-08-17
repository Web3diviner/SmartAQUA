# API and Service Contracts

## Principle

The Go backend remains the authoritative public API.

AquaDoc should normally be an internal service.

## Public API

Recommended namespace:

```text
/api/v1
```

Use `/api/v2` only for breaking changes.

## Farm APIs

```text
POST   /api/v1/farms
GET    /api/v1/farms
GET    /api/v1/farms/{farm_id}
PATCH  /api/v1/farms/{farm_id}
```

## Pond APIs

```text
POST   /api/v1/farms/{farm_id}/ponds
GET    /api/v1/farms/{farm_id}/ponds
GET    /api/v1/ponds/{pond_id}
PATCH  /api/v1/ponds/{pond_id}
```

## Production Cycle APIs

```text
POST   /api/v1/ponds/{pond_id}/cycles
GET    /api/v1/ponds/{pond_id}/cycles
GET    /api/v1/cycles/{cycle_id}
PATCH  /api/v1/cycles/{cycle_id}
```

## Sampling / Mortality

```text
POST /api/v1/cycles/{cycle_id}/sampling
GET  /api/v1/cycles/{cycle_id}/sampling

POST /api/v1/cycles/{cycle_id}/mortality
GET  /api/v1/cycles/{cycle_id}/mortality
```

## Sensor APIs

```text
GET /api/v1/ponds/{pond_id}/sensors
GET /api/v1/ponds/{pond_id}/readings
GET /api/v1/ponds/{pond_id}/water-quality
```

## AquaDoc Public APIs

The Flutter app should call the Go backend.

```text
POST /api/v1/aquadoc/chat
POST /api/v1/aquadoc/disease-assessments
GET  /api/v1/aquadoc/conversations/{id}
```

## Go -> AquaDoc Internal APIs

```text
POST /internal/v1/chat
POST /internal/v1/disease/assess
POST /internal/v1/recommendations/evaluate
POST /internal/v1/knowledge/search
GET  /internal/v1/health
```

Authenticate with service credentials/mTLS/private networking.

## Chat Request

```json
{
  "request_id": "REQ-123",
  "user_id": "USER-1",
  "conversation_id": "CONV-1",
  "question": "Why are my fish eating less?",
  "farm_context": {
    "farm_id": "FARM-1",
    "pond_id": "POND-1",
    "production_cycle_id": "CYCLE-1"
  }
}
```

## Context Contract

The Go backend should build context.

```json
{
  "pond_id": "POND-1",
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
    "last_feeding_at": "2026-08-08T09:00:00+01:00",
    "last_feeding_g": 1800
  },
  "health": {
    "mortality_24h": 0
  }
}
```

## Recommendation API

```text
POST /api/v1/recommendations/{id}/approve
POST /api/v1/recommendations/{id}/reject
GET  /api/v1/ponds/{pond_id}/recommendations
```

Approval must require:

- authenticated user
- correct farm permission
- recommendation not expired
- action within user authority
- device available if command needed

## Error Format

Use a consistent format.

```json
{
  "error": {
    "code": "POND_NOT_FOUND",
    "message": "Pond was not found.",
    "request_id": "REQ-123"
  }
}
```

## Idempotency

Use idempotency keys for:

- feed commands
- payments
- recommendation approvals
- command creation
- consultation creation

## Rate Limiting

Apply stronger limits to:

- login
- registration
- AquaDoc chat
- document upload
- disease image upload
- manual feed
- commands
