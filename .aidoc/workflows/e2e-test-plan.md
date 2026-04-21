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

| Test | Login As | Description |
|------|----------|-------------|
| M1 | Alice | Mutual match happy path (English↔Chinese, overlapping availability) |
| M5 | Alice | Base-language matching (Alice targets `zh`, Grace speaks `zh-Hans`) |
| M9 | Alice | Cursor pagination with `limit=2` |
| M10 | Kevin | Multiple mutual languages (Kevin targets `zh`+`pt`, Luna speaks both) |

### Group C: Discovery Edge Cases

| Test | Login As | Description |
|------|----------|-------------|
| M2 | Fresh account | No target languages → 422 `ERR_NO_TARGET_LANGUAGES` |
| M3 | Fresh account | Incomplete profile → 403 `ERR_PROFILE_INCOMPLETE` |
| M4 | None | Unauthenticated → 401 `ERR_AUTH_REQUIRED` |
| M6 | Hiro | Rare language (ja→ko) — empty results |
| M7 | Alice | Zero availability overlap with Eve — not matched |
| M8 | Bob | Blocked user (Ivan) excluded bidirectionally |

### Group D: Discovery Localization

| Test | Login As | Description |
|------|----------|-------------|
| M11 | Fresh account | Error messages in pt-BR and zh-Hans |

### Group E: Connection Happy Paths

| Test | Login As | Description |
|------|----------|-------------|
| C1 | Alice | Send connection request to Bob |
| C7 | Bob | List incoming requests |
| C8 | Alice | List outgoing requests |
| C9 | Alice / Bob | View request detail (both participants) |
| C10 | Bob | Accept request → creates match |
| C13 | Bob | Decline request |
| C14 | Alice | Cancel own request |
| C15 | Bob | Send pre-accept message |
| C16 | Alice | List pre-accept messages |
| C20 | Bob | Accept re-associates messages to match |
| C21 | Bob | Request pagination with multiple incoming |

### Group F: Connection Error Cases

| Test | Login As | Description |
|------|----------|-------------|
| C2 | Alice | Self-request → 400 `ERR_SELF_REQUEST` |
| C3 | Alice | Non-existent recipient → 404 `ERR_USER_NOT_FOUND` |
| C4 | Alice | Duplicate pending request → 409 `ERR_REQUEST_EXISTS` |
| C5 | Alice | Already matched → 409 `ERR_ALREADY_MATCHED` |
| C6 | Bob | Blocked user (Ivan) → 403 `ERR_USER_BLOCKED` |
| C11 | Alice | Requester tries accept → 403 `ERR_NOT_RECIPIENT` |
| C12 | Bob | Accept non-pending → 409 `ERR_NOT_PENDING` |
| C17 | Alice | Exceed message limit → 429 `ERR_MESSAGE_LIMIT` |
| C18 | Alice | Message on accepted request → 409 `ERR_NOT_PENDING` |
| C19 | Carlos | Unrelated user messages → 403 `ERR_NOT_PARTICIPANT` |
| C23 | None | Unauthenticated → 401 `ERR_AUTH_REQUIRED` |

### Group G: Connection Localization

| Test | Login As | Description |
|------|----------|-------------|
| C22 | Alice | Self-request errors in zh-Hans and pt-BR |

## Current Status

- ✅ `scripts/e2e-test.py`: 45 test scenarios (Python + requests)
- ✅ `scripts/seed-users.py`: creates 12 seed profiles via API
- ✅ SQL seed alternative: `db/seeds/seed_test_profiles.sql`
