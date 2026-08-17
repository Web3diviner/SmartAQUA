# Smart Aqua Security Architecture

## Security Objectives

Protect:

- farmer accounts
- farms and pond data
- device control
- feeding commands
- expert consultations
- payments
- AI knowledge base
- API keys
- telemetry
- source documents
- audit history

## 1. Trust Boundaries

```text
Flutter App
   |
Internet
   |
API Gateway / Go Backend
   |
Private Service Network
   +-- PostgreSQL
   +-- Redis
   +-- AquaDoc
   +-- MQTT
```

ESP32 is an untrusted remote edge device until authenticated.

## 2. Authentication

### Users

Use:

- strong password hashing: Argon2id preferred, bcrypt acceptable if configured strongly
- short-lived access tokens
- refresh-token rotation
- logout/revocation
- optional MFA for high-risk roles
- biometric unlock only as local app convenience, not server identity

### Services

Go -> AquaDoc:

- private network
- service token or mTLS
- key rotation
- request identity

### Devices

Each device should have unique credentials.

Preferred:

- client certificates where practical
- otherwise per-device secret with rotation

## 3. Authorization

Use RBAC + ownership checks.

Example permissions:

| Role | View Pond | Feed | Approve AI Action | Manage Users |
|---|---:|---:|---:|---:|
| Owner | yes | yes | yes | yes |
| Manager | yes | yes | configurable | limited |
| Worker | yes | configurable | no | no |
| Expert | case-limited | no | advisory only | no |

## 4. Secrets

Never store secrets in:

- Git repository
- Flutter source
- public `.env.example`
- firmware source
- logs

Use secret managers or environment variables.

Rotate immediately if any real secret is accidentally committed.

## 5. Transport Security

Require:

- HTTPS/TLS for APIs
- MQTT over TLS in production
- certificate validation
- modern cipher suites
- no plaintext production MQTT

## 6. Database Security

- least-privilege DB accounts
- private networking
- encrypted backups
- regular backups
- migration tracking
- no direct public database exposure
- query parameterization/ORM
- row ownership enforcement at service layer

## 7. Device Command Security

All commands must include:

- command ID
- issuing actor
- authorization result
- timestamp
- expiry
- payload
- target device
- optional source recommendation

Protect against:

- replay
- duplicate execution
- unauthorized feed
- stale commands

## 8. AI Security

Treat all retrieved text as untrusted input.

Defend against:

- prompt injection in uploaded documents
- malicious farmer text
- malicious expert notes
- poisoned knowledge documents

Rules:

- system policies cannot be overridden by retrieved content
- only approved knowledge documents should enter production RAG
- source document access should be permission-aware
- never expose system prompts or API keys
- sanitize model tool inputs
- strict JSON schemas for structured outputs

## 9. File Upload Security

For PDFs/images:

- restrict MIME types
- inspect actual content type
- file-size limits
- malware scanning where available
- generate random storage names
- no user-controlled filesystem paths
- strip dangerous metadata when necessary
- serve through controlled URLs

## 10. Logging

Never log:

- passwords
- access tokens
- refresh tokens
- API keys
- full payment card data
- private secrets

Use request IDs.

Security-relevant events must be logged.

## 11. Audit Logging

Audit:

- login failures
- permission changes
- device binding
- device unbinding
- manual feed
- schedule changes
- AI recommendation approval/rejection
- command execution
- expert diagnosis edits
- payment status changes
- knowledge-document approval

## 12. Production Security Gates

No release if:

- default credentials remain
- TLS is disabled
- secrets exist in repo
- high severity dependency vulnerabilities remain unresolved
- authorization tests fail
- device command replay protection is missing
- backups are untested
