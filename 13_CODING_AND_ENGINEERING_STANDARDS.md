# Coding and Engineering Standards

## General

- small focused modules
- clear interfaces
- explicit error handling
- no hidden global state where avoidable
- configuration through environment/config files
- no secrets in code
- deterministic logic must be testable without AI
- AI calls must be isolated behind interfaces

## Go Backend

Maintain layers:

```text
handler
 -> service
 -> repository
 -> database
```

Do not put business logic in HTTP handlers.

Use context propagation for request cancellation/timeouts.

All external calls require explicit timeouts.

## Python AquaDoc

Recommended layering:

```text
api
 -> orchestrator
 -> retrieval/rules/models
 -> providers
```

Pydantic schemas for:

- requests
- responses
- recommendation structures
- disease assessments
- provider payloads

Avoid passing unstructured dictionaries across the entire codebase.

## Flutter

Keep network calls in services/providers, not widgets.

UI should render:

- loading
- empty
- stale
- offline
- error
- partial-data states

## Firmware

Separate:

- sensor manager
- motor controller
- feeding controller
- network manager
- MQTT
- local storage
- safety controller

Avoid one giant loop containing all behavior.

## API Versioning

Backward-compatible changes stay in same version.

Breaking changes require versioned endpoint/contract.

## Migrations

Every schema change:

- migration file
- rollback strategy if practical
- staging test
- release note

## Dependency Policy

- pin major dependencies
- regular updates
- vulnerability scanning
- avoid unnecessary frameworks

## Definition of Done

A feature is not complete until it has:

- code
- tests
- error handling
- authorization
- logs
- documentation
- migration if needed
- monitoring where relevant
