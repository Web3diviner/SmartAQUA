# IoT, MQTT and Device Protocol

## 1. Current Direction

Retain MQTT as the primary backend-device protocol.

## 2. Topic Convention

Recommended versioned topics:

```text
smartaqua/v1/devices/{device_id}/telemetry
smartaqua/v1/devices/{device_id}/feeding
smartaqua/v1/devices/{device_id}/alerts
smartaqua/v1/devices/{device_id}/status
smartaqua/v1/devices/{device_id}/commands
smartaqua/v1/devices/{device_id}/commands/result
smartaqua/v1/devices/{device_id}/config
smartaqua/v1/devices/{device_id}/diagnostics
```

If existing topics are already deployed, introduce aliases/versioning instead of breaking them suddenly.

## 3. Telemetry Contract

```json
{
  "message_id": "MSG-123",
  "device_id": "SA-FEEDER-001",
  "timestamp": 1786150000,
  "firmware_version": "1.2.0",
  "sensors": {
    "temperature_c": 29.4,
    "ph": null,
    "dissolved_oxygen_mg_l": null,
    "turbidity_ntu": null
  },
  "power": {
    "battery_percent": 86
  },
  "status": {
    "motor_ok": true
  }
}
```

## 4. Command Contract

```json
{
  "command_id": "CMD-123",
  "type": "FEED_NOW",
  "issued_at": 1786150050,
  "expires_at": 1786150110,
  "payload": {
    "quantity_g": 1000
  }
}
```

## 5. Command Validation on ESP32

ESP32 must validate:

- command ID
- expiry
- command type
- payload limits
- duplicate/replay detection
- local safety conditions
- maximum feed constraints
- motor health
- sensor validity where required

Cloud approval never overrides local safety.

## 6. QoS

Recommended:

- telemetry: QoS 0 or 1 depending on bandwidth/reliability
- alerts: QoS 1
- commands: QoS 1
- command result: QoS 1
- retained config: consider retained message carefully
- do not retain transient feed commands

## 7. Offline Mode

ESP32 must continue:

- approved schedules
- safe local Q10 logic
- motor protection
- local event logging

When reconnecting:

- send buffered telemetry
- send feeding results
- preserve original timestamps
- mark delayed messages

## 8. New Sensor Integration

For each new sensor:

1. hardware driver
2. calibration
3. range validation
4. quality flag
5. telemetry support
6. backend persistence
7. mobile display
8. AquaDoc context mapping
9. alert thresholds
10. tests

## 9. Sensor Failure

A failed sensor must produce:

```text
value = unknown
quality = invalid
```

Never substitute a "normal" value.

## 10. Device Identity

Each device needs:

- unique device ID
- unique credentials
- firmware identity
- provisioning state
- revocation capability

Avoid a shared global MQTT password for all devices.

## 11. OTA Firmware

Future production requirements:

- signed firmware
- versioned rollout
- rollback support
- staged deployment
- hardware compatibility checks
- firmware checksum
- update audit log
