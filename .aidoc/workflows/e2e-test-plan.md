---
domain: Workflows
status: Active
entry_points:
  - scripts/e2e-test.py
  - scripts/seed-users.py
dependencies:
  - .aidoc/designs/api-contract.md
  - .aidoc/designs/discovery-matching.md
  - .aidoc/designs/connection-handshake.md
---

# End-to-End Test Plan

E2E test coverage for the API: health, auth, profile, languages, availability, discovery, and connection handshake.

## Related Docs

| Document | Relationship |
|----------|-------------|
| [API Contract](../designs/api-contract.md) | Endpoints under test |
| [Discovery Matching](../designs/discovery-matching.md) | Matching rules tested in §10 |
| [Connection Handshake](../designs/connection-handshake.md) | Connection scenarios tested in §11 |
| [Database Schema](../designs/database-schema.md) | Schema backing test assertions |

## Why This Plan Exists

Validates the full API stack end-to-end including DB interactions, auth flows, matching algorithm correctness, and connection state machine transitions. Catches integration issues that unit tests miss.

## Test Environment

- API on port 6176, Postgres dev database
- Base URL: `https://app.example.com/api/v1`
- Localization assertions in English, Chinese, and Portuguese via `Accept-Language`

## Test Data

- Fresh accounts use `test+<timestamp>@example.com`
- Discovery/matching/connection tests require 12 seed users created via `scripts/seed-users.py`
- Seed script is idempotent; re-run after recreating test containers

### Seed User Summary

| Handle | Native | Targets | Key Trait |
|--------|--------|---------|-----------|
| alice | en | zh | Primary requester |
| bob | zh | en | Primary recipient; blocks Ivan |
| carlos | pt-BR, es | en, zh | Multi-lang; bridge match |
| diana | en | pt | No time overlap |
| eve | zh | en | No availability overlap with Alice |
| frank | en | zh | Minimal overlap (65 min) with Bob |
| grace | zh-Hans | en | Base-language matching test |
| hiro | ja | ko | Rare language — no matches |
| ivan | en | zh | Blocked by Bob |
| julia | zh | en | NOT discoverable |
| kevin | en | zh, pt | Multi-target match |
| luna | pt-BR, zh-Hans | en | Multi-teach match |

## Test Groups

### Group A: Fresh-Account Tests
Health, auth, profile, handle, languages, availability, discoverable flag. No seed data needed.

### Group B: Discovery Happy Paths
Login as Alice/Kevin. Tests: mutual match, base-language matching, pagination, multiple mutual languages.

### Group C: Discovery Edge Cases
Tests: no target languages (422), incomplete profile (403), unauthenticated (401), no matches, insufficient overlap, blocked user excluded.

### Group D: Discovery Localization
Error messages in pt-BR and zh-Hans.

### Group E: Connection Happy Paths
Tests: send request, list incoming/outgoing, request detail, accept, decline, cancel, pre-accept messaging, message re-association, pagination.

### Group F: Connection Error Cases
Tests: self-request (400), not found (404), duplicate (409), already matched (409), blocked (403), not recipient (403), not pending (409), message limit (429), not participant (403), unauthenticated (401).

### Group G: Connection Localization
Error messages in zh-Hans and pt-BR.

## Current Status

- ✅ `scripts/e2e-test.py`: 45 test scenarios (Python + requests)
- ✅ `scripts/seed-users.py`: creates 12 seed profiles via API
- ✅ SQL seed alternative: `db/seeds/seed_test_profiles.sql`
