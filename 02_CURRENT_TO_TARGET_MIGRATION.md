# Current-to-Target Migration Plan

## Objective

Extend the working Smart Fish Feeder without destabilizing it.

## Current Strengths

The current repository already provides:

- Flutter mobile app
- Go backend
- PostgreSQL
- Redis
- JWT authentication
- MQTT integration
- device registration
- feeding schedules
- manual feeding
- telemetry
- alerts
- Q10-related logic
- device diagnostics
- ESP32/LILYGO firmware
- offline capability concepts

Therefore Smart Aqua V2 should be an extension, not a rewrite.

## Phase 0: Freeze Stable Feeder Behavior

Before major development:

- tag a known-good production release
- document current backend URL/environment
- document firmware version
- document MQTT broker configuration
- create database backup
- record current mobile release version
- create regression tests for:
  - login
  - device binding
  - manual feed
  - schedules
  - telemetry
  - feeding logs
  - Q10 flow

## Phase 1: Add Farm Domain

Introduce:

- Farm
- Pond
- ProductionCycle
- FishStock
- SamplingRecord
- MortalityRecord

Do not remove existing `Device` relationships.

Add optional links first.

Example:

```text
Device
  pond_id nullable
```

Existing devices continue to work.

## Phase 2: Normalize Sensor Domain

Current sensor storage is device-oriented.

Add:

- Sensor
- SensorReading
- SensorCalibration

Keep legacy `SensorData` during migration.

New sensor readings should support:

- temperature
- pH
- dissolved oxygen
- turbidity
- ammonia
- nitrite
- salinity
- conductivity
- ORP

Even if only temperature is currently installed.

## Phase 3: Build AquaDoc Standalone

AquaDoc should initially use:

- test user
- test farm
- test pond
- manually entered temperature
- manually entered feeding data
- manual mortality/symptom entries

No device control.

## Phase 4: Read-Only Smart Aqua Integration

Go backend exposes farm context to AquaDoc.

AquaDoc can read:

- stock
- biomass
- temperature
- feeding history
- Q10 outputs
- mortality
- farmer symptoms

AquaDoc still cannot execute commands.

## Phase 5: Add AquaDoc UI

Flutter adds:

- AquaDoc chat
- disease assessment
- recommendation cards
- source/evidence view
- missing-data notices

## Phase 6: Add New Physical Sensors

Add sensors one by one.

Recommended order:

1. pH
2. turbidity
3. dissolved oxygen
4. other sensors later

Each sensor requires:

- hardware validation
- calibration method
- firmware driver
- MQTT field
- backend persistence
- health checks
- mobile display
- AquaDoc context support

## Phase 7: Recommendation Approval

AquaDoc can create recommendations.

Example:

```text
Reduce evening ration by 8%
```

Farmer can:

- approve
- reject
- request explanation

Only approved recommendations may produce commands.

## Phase 8: Bounded Automation

Only after extensive validation.

Allow pre-authorized low-risk actions such as:

- delay feeding
- reduce ration within configured percentage
- skip feeding under hard safety condition

Never allow unrestricted AI control.

## Backward Compatibility Rules

- Existing mobile versions should fail gracefully.
- New fields should be nullable or defaulted.
- Old ESP32 firmware should continue to communicate.
- MQTT topic changes should be versioned.
- Database migrations must be reversible when practical.
- New APIs should use `/api/v2` only where breaking changes are unavoidable.
