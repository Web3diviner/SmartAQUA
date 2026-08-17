# Production Readiness Checklist

## Backend

- [ ] All production secrets configured outside repo
- [ ] Debug disabled
- [ ] TLS enforced
- [ ] Database private
- [ ] Redis protected
- [ ] Auth tested
- [ ] Authorization tested
- [ ] Rate limiting enabled
- [ ] Audit logging enabled
- [ ] Backups enabled
- [ ] Restore tested

## MQTT / Device

- [ ] TLS enabled
- [ ] Per-device credentials
- [ ] Topic ACLs
- [ ] Replay protection
- [ ] Command expiry
- [ ] Command idempotency
- [ ] Local safety limits
- [ ] Offline mode tested
- [ ] Reconnect tested

## Firmware

- [ ] Motor stall protection
- [ ] Maximum feed protection
- [ ] Invalid sensor rejection
- [ ] Watchdog
- [ ] Buffer persistence
- [ ] Firmware version reporting
- [ ] Safe boot behavior

## AquaDoc

- [ ] LLM key server-side only
- [ ] RAG sources approved
- [ ] Source citations enabled
- [ ] Missing values handled explicitly
- [ ] Disease uncertainty wording
- [ ] Expert escalation rules
- [ ] Structured output validation
- [ ] Prompt-injection tests
- [ ] Model evaluation suite
- [ ] Cost limits

## Knowledge Base

- [ ] Document checksum
- [ ] Metadata
- [ ] Evidence level
- [ ] Review status
- [ ] Source page preserved
- [ ] Duplicate detection
- [ ] Access controls

## Mobile

- [ ] No API secrets
- [ ] Secure token storage
- [ ] Token refresh
- [ ] Logout
- [ ] Offline UI states
- [ ] Stale sensor states
- [ ] Recommendation approval confirmation
- [ ] Source display

## Operational

- [ ] Monitoring
- [ ] Error alerting
- [ ] LLM provider outage plan
- [ ] MQTT outage plan
- [ ] DB outage plan
- [ ] incident response contacts
- [ ] release rollback plan

## Release Gate

Do not release if any of the following are true:

- critical security vulnerability unresolved
- command authorization untested
- AI can directly issue unrestricted device commands
- missing sensor values are treated as normal
- production LLM key is present in mobile or firmware
- database backups are missing
- current feeder regression suite fails
