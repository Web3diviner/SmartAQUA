# DevOps, Deployment and Observability

## 1. Environments

Maintain:

- local
- development
- staging
- production

Never test new device-control logic directly in production first.

## 2. Containerization

Containerize:

- Go backend
- AquaDoc
- ingestion worker
- optional MQTT broker in self-hosted environments

PostgreSQL and Redis may be managed services.

## 3. CI Pipeline

On every PR:

```text
format
lint
unit tests
security scan
dependency scan
build
```

Recommended:

### Go
- `gofmt`
- `go vet`
- `go test`
- static analysis

### Python
- ruff
- mypy/pyright optional
- pytest
- dependency audit

### Flutter
- `dart format`
- `flutter analyze`
- `flutter test`

### Firmware
- PlatformIO build
- simulation tests

## 4. CD

Recommended workflow:

```text
main branch
 -> CI
 -> staging
 -> integration tests
 -> manual production approval
 -> production
```

Device firmware should use a separate staged release process.

## 5. Database Migrations

- version-controlled
- run before application rollout where compatible
- backward-compatible where possible
- tested against staging copy
- backups before destructive migrations

## 6. Monitoring

Track:

- API latency
- API errors
- database health
- Redis health
- MQTT connections
- device online rate
- device telemetry delay
- command success/failure
- AquaDoc latency
- LLM failure rate
- token usage/cost
- retrieval failures
- expert consultation failures

## 7. Logs

Structured JSON logs.

Every request should include:

```text
request_id
user_id if known
farm_id if relevant
device_id if relevant
service
severity
```

## 8. Tracing

Eventually use OpenTelemetry.

Trace:

```text
Flutter request
 -> Go
 -> AquaDoc
 -> vector search
 -> LLM
```

## 9. Alerts

Operational alerts:

- API unavailable
- database unavailable
- MQTT disconnected
- high command failure
- unusual LLM cost spike
- repeated login attack
- device fleet offline spike

## 10. Backups

Back up:

- PostgreSQL
- knowledge documents
- object storage
- critical configuration

Test restoration periodically.

A backup that has never been restored is not trusted.

## 11. Disaster Recovery

Document:

- database restore
- Redis loss handling
- MQTT broker recovery
- AquaDoc outage
- LLM provider outage
- object storage outage

Smart feeder local operation must remain safe during cloud outages.
