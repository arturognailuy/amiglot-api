---
description: "End-to-end test plan for Amiglot API."
whenToUse: "Read when running or updating API E2E scenarios."
---

# Amiglot API — End-to-End Test Plan

## 1. Scope
End-to-end coverage for the current API feature set: health, authentication, profile, languages, availability, discovery & matching, and connection (handshake).

## 2. Test Environment
- API running locally on port 6176
- Postgres dev database
- Base URL: `https://app.example.com/api/v1`
- Localization via `Accept-Language` (run localization assertions in **English**, **Chinese**, and **Portuguese**)

## 2.1 Seed Data

**When setting up any new test environment, seed users must be created before running E2E tests.** Use the seed script to create all 12 test profiles via the API:

```bash
# Requires: pip install requests
# API must be running (DB + API container)
python3 scripts/seed-users.py --api-url http://localhost:6176/api/v1

# For full setup (blocks + discoverable overrides), also pass the DB DSN:
python3 scripts/seed-users.py --api-url http://localhost:6176/api/v1 \
  --db-dsn "postgresql://postgres@localhost:5432/amiglot_dev"
```

The script creates 12 users covering: basic mutual match, multi-language, bridge-only, no availability overlap, minimal overlap, base-language matching (zh vs zh-Hans), blocked pairs, non-discoverable users, and rare-language targets.

**Note:** Since test containers use ephemeral storage, re-run the seed script each time a test environment is recreated.

## 2.2 Seed Users by Test Group

Each test group below lists the seed users it requires. When setting up a new test environment, create all seed users via the seed script, then log in as the specified user to run each group.

### Group A: Fresh-Account Tests (no seed users)

| Tests | Description |
|-------|-------------|
| §3 Health, §4 Auth, §5 Profile & Handle, §6 Languages, §7 Availability, §8 Discoverable Flag | Each test creates a fresh account (`test+<timestamp>@example.com`). No seed data needed. |

### Group B: Basic Discovery & Matching

| Tests | Login As | Seed Users Needed | Purpose |
|-------|----------|-------------------|---------|
| M1 (happy path) | Alice (`test+seed1`) | Alice + Bob | Mutual English↔Chinese match, overlapping availability |
| M5 (base-language zh↔zh-Hans) | Alice (`test+seed1`) | Alice + Grace | Alice targets `zh`, Grace speaks `zh-Hans` native |
| M9 (pagination) | Alice (`test+seed1`) | Alice + Bob + Grace + Luna (+ others) | Multiple matches to verify cursor pagination |
| M10 (multiple mutual languages) | Kevin (`test+seed11`) | Kevin + Luna | Kevin targets `zh` + `pt`; Luna speaks both |

### Group C: Discovery Edge Cases & Errors

| Tests | Login As | Seed Users Needed | Purpose |
|-------|----------|-------------------|---------|
| M2 (no target languages, 422) | Fresh account | None (fresh account with only native lang) | Validation error |
| M3 (incomplete profile, 403) | Fresh account | None (no profile saved) | Auth without profile |
| M4 (unauthenticated, 401) | None | None | No auth header |
| M6 (no matches) | Hiro (`test+seed8`) | Hiro | Targets Korean — no teachers available |
| M7 (insufficient overlap) | Alice (`test+seed1`) | Alice + Eve | Same language match but zero time overlap |
| M8 (blocked user excluded) | Bob (`test+seed2`) | Bob + Ivan | Bob blocks Ivan; bidirectional exclusion |

### Group D: Discovery Localization

| Tests | Login As | Seed Users Needed | Purpose |
|-------|----------|-------------------|---------|
| M11 (localized errors) | Fresh account | None | `Accept-Language: pt-BR` and `zh-Hans` error messages |

### Group E: Connection Handshake — Happy Paths

| Tests | Login As | Seed Users Needed | Purpose |
|-------|----------|-------------------|---------|
| C1 (send request) | Alice (`test+seed1`) | Alice + Bob | Both discoverable with overlap |
| C7 (list incoming) | Bob (`test+seed2`) | Alice + Bob | Bob checks incoming from Alice |
| C8 (list outgoing) | Alice (`test+seed1`) | Alice + Bob | Alice checks outgoing to Bob |
| C9 (request detail) | Alice or Bob | Alice + Bob | Both participants view detail |
| C10 (accept) | Bob (`test+seed2`) | Alice + Bob | Bob accepts Alice's request |
| C13 (decline) | Bob (`test+seed2`) | Alice + Bob | Bob declines Alice's request |
| C14 (cancel) | Alice (`test+seed1`) | Alice + Bob | Alice cancels her request |
| C15 (send pre-accept message) | Bob (`test+seed2`) | Alice + Bob | Messaging on pending request |
| C16 (list messages) | Alice (`test+seed1`) | Alice + Bob | View conversation |
| C20 (accept re-associates messages) | Bob (`test+seed2`) | Alice + Bob | Messages migrate to match |
| C21 (pagination) | Bob (`test+seed2`) | Multiple requesters → Bob | Bob has many incoming requests |

### Group F: Connection Handshake — Error Cases

| Tests | Login As | Seed Users Needed | Purpose |
|-------|----------|-------------------|---------|
| C2 (self-request, 400) | Alice (`test+seed1`) | Alice | Send request to self |
| C3 (not found, 404) | Alice (`test+seed1`) | Alice | Non-existent recipient |
| C4 (duplicate, 409) | Alice (`test+seed1`) | Alice + Bob | Second pending request |
| C5 (already matched, 409) | Alice (`test+seed1`) | Alice + Bob (already matched) | Request after match exists |
| C6 (blocked, 403) | Bob (`test+seed2`) | Bob + Ivan | Bob blocked Ivan |
| C11 (accept not recipient, 403) | Alice (`test+seed1`) | Alice + Bob | Requester tries to accept |
| C12 (accept not pending, 409) | Bob (`test+seed2`) | Alice + Bob | Already resolved request |
| C17 (message limit, 429) | Alice (`test+seed1`) | Alice + Bob | Exhaust `PRE_MATCH_MESSAGE_LIMIT` |
| C18 (message not pending, 409) | Alice (`test+seed1`) | Alice + Bob | Message on accepted request |
| C19 (not participant, 403) | Carlos (`test+seed3`) | Alice + Bob + Carlos | Unrelated user messages |
| C23 (unauthenticated, 401) | None | None | No auth header |

### Group G: Connection Localization

| Tests | Login As | Seed Users Needed | Purpose |
|-------|----------|-------------------|---------|
| C22 (localized errors) | Alice (`test+seed1`) | Alice | Self-request with `zh-Hans` and `pt-BR` |

### Seed User Reference

| # | Handle | Email | Native | Targets | Key Trait |
|---|--------|-------|--------|---------|-----------|
| 1 | alice | test+seed1@example.com | en | zh | Primary test requester |
| 2 | bob | test+seed2@example.com | zh | en | Primary test recipient; blocks Ivan |
| 3 | carlos | test+seed3@example.com | pt-BR, es | en, zh | Multi-lang; bridge match test |
| 4 | diana | test+seed4@example.com | en | pt | No time overlap with others |
| 5 | eve | test+seed5@example.com | zh | en | No availability overlap with Alice |
| 6 | frank | test+seed6@example.com | en | zh | Minimal overlap (65 min) with Bob |
| 7 | grace | test+seed7@example.com | zh-Hans | en | Base-language matching test |
| 8 | hiro | test+seed8@example.com | ja | ko | Rare language — no matches |
| 9 | ivan | test+seed9@example.com | en | zh | Blocked by Bob |
| 10 | julia | test+seed10@example.com | zh | en | NOT discoverable |
| 11 | kevin | test+seed11@example.com | en | zh, pt | Multi-target language match |
| 12 | luna | test+seed12@example.com | pt-BR, zh-Hans (adv) | en | Multi-teach language match |

## 3. Health
1. `GET /healthz` returns `{ ok: true }` and build metadata.

## 4. Authentication
1. `POST /auth/magic-link` returns `{ ok: true }` (dev mode returns `dev_login_url`).
2. `POST /auth/verify` with a valid token returns access token + user payload.
3. `POST /auth/verify` with invalid token returns localized error.

## 5. Profile & Handle
1. `GET /profile` requires auth and returns empty profile defaults when none exist.
2. `PUT /profile` creates/updates profile with required fields.
3. `GET /profile/handle/check` returns availability (true when unused or owned by user).
4. Handle normalization accepts leading `@` and stores lowercased handle.
5. Handle format for E2E: **alphanumeric only** (letters/numbers; no underscores or symbols).

## 6. Languages
1. `PUT /profile/languages` replaces list.
2. Enforce at least one native language.
3. Validate level bounds, duplicates, and native/target constraints.
4. Persist `order` and return languages sorted by `order` ascending.

## 7. Availability
1. `PUT /profile/availability` replaces list.
2. Validate start < end and weekday bounds.
3. Timezone validation on slot and fallback to profile timezone.
4. Reject duplicate availability slots.
5. Persist `order` and return grouped slots sorted by shared `order` ascending (slots with identical start/end/timezone share the same order).

## 8. Discoverable Flag
1. After saving profile + native language, profile `discoverable` becomes true.
2. Removing native language flips `discoverable` to false.

## 9. Discovery & Matching

### M1. Discover matches (happy path)
**Setup:** Two accounts — User A (teaches English native, targets Chinese, has availability Mon 09:00–12:00 UTC) and User B (teaches Chinese native, targets English, has availability Mon 10:00–13:00 UTC). Both profiles complete and discoverable.
**Steps:**
1. `GET /api/v1/matches/discover` as User A.
**Expected:** Response includes User B with `mutual_teach` (Chinese), `mutual_learn` (English), `total_overlap_minutes >= 60`, and `availability_overlap` containing Mon 10:00–12:00 UTC.

### M2. No target languages (422)
**Setup:** Fresh account with profile saved but **no target languages** (only native).
**Steps:**
1. `GET /api/v1/matches/discover`.
**Expected:** 422 with `ERR_NO_TARGET_LANGUAGES`; message localized per `Accept-Language`.

### M3. Incomplete profile (403)
**Setup:** Fresh account with auth token but **no profile saved** (discoverable = false).
**Steps:**
1. `GET /api/v1/matches/discover`.
**Expected:** 403 with `ERR_PROFILE_INCOMPLETE`; message localized.

### M4. Unauthenticated (401)
**Steps:**
1. `GET /api/v1/matches/discover` with no/invalid auth header.
**Expected:** 401 with `ERR_AUTH_REQUIRED`.

### M5. Base-language matching (zh matches zh-Hans)
**Setup:** User A targets `zh`; User B speaks `zh-Hans` (native). Both have bridge language and availability overlap.
**Steps:**
1. `GET /api/v1/matches/discover` as User A.
**Expected:** User B appears in results; `mutual_teach` includes the `zh-Hans` entry.

### M6. No matches (empty result)
**Setup:** User A targets a rare language that no other user teaches at level ≥ 4.
**Steps:**
1. `GET /api/v1/matches/discover` as User A.
**Expected:** 200 with `items: []` and `next_cursor: null`.

### M7. Insufficient availability overlap
**Setup:** User A and User B qualify on language checks but have only 30 min of overlapping availability (below the 60-min default).
**Steps:**
1. `GET /api/v1/matches/discover` as User A.
**Expected:** User B does **not** appear in results.

### M8. Blocked user excluded
**Setup:** User A blocks User B (both otherwise match on language + availability).
**Steps:**
1. `GET /api/v1/matches/discover` as User A.
**Expected:** User B does not appear in results.
2. `GET /api/v1/matches/discover` as User B.
**Expected:** User A does not appear in results (block is bidirectional).

### M9. Pagination (cursor)
**Setup:** Create enough matching users (> default page size or use `limit=2`) for User A.
**Steps:**
1. `GET /api/v1/matches/discover?limit=2` as User A.
**Expected:** `items` has ≤ 2 entries; `next_cursor` is non-null if more exist.
2. `GET /api/v1/matches/discover?limit=2&cursor=<next_cursor>`.
**Expected:** Next page of results; no duplicates from page 1.

### M10. Multiple mutual languages listed
**Setup:** User A targets `zh` and `pt`; User B speaks `zh-Hans` (native) and `pt-BR` (level 5), and targets English. Both have bridge + overlap.
**Steps:**
1. `GET /api/v1/matches/discover` as User A.
**Expected:** User B's `mutual_teach` array includes both `zh-Hans` and `pt-BR` entries.

### M11. Localized error messages
**Steps:**
1. `GET /api/v1/matches/discover` (no target langs) with `Accept-Language: pt-BR`.
2. Same with `Accept-Language: zh-Hans`.
**Expected:** Error messages in Portuguese and Chinese respectively.

## 10. Connection (Handshake)

### C1. Send connection request (happy path)
**Setup:** Two accounts — User A and User B — both discoverable with matching languages and availability overlap.
**Steps:**
1. `POST /api/v1/match-requests` as User A with `{ "recipient_id": "<User B>", "initial_message": "Hi!" }`.
**Expected:** 201 with `status: "pending"`, `requester_id` = User A, `recipient_id` = User B, `initial_message` present.

### C2. Self-request (400)
**Steps:**
1. `POST /api/v1/match-requests` as User A with `recipient_id` = User A.
**Expected:** 400 with `ERR_SELF_REQUEST`.

### C3. Recipient not found / not discoverable (404)
**Steps:**
1. `POST /api/v1/match-requests` with a non-existent `recipient_id`.
**Expected:** 404 with `ERR_USER_NOT_FOUND`.

### C4. Duplicate request (409)
**Setup:** User A already has a pending request to User B.
**Steps:**
1. `POST /api/v1/match-requests` as User A to User B again.
**Expected:** 409 with `ERR_REQUEST_EXISTS`.

### C5. Already matched (409)
**Setup:** User A and User B are already matched.
**Steps:**
1. `POST /api/v1/match-requests` as User A to User B.
**Expected:** 409 with `ERR_ALREADY_MATCHED`.

### C6. Blocked user (403)
**Setup:** User A has blocked User B (or vice versa).
**Steps:**
1. `POST /api/v1/match-requests` as User A to User B.
**Expected:** 403 with `ERR_USER_BLOCKED`.

### C7. List incoming requests
**Setup:** User B has a pending request from User A.
**Steps:**
1. `GET /api/v1/match-requests?direction=incoming&status=pending` as User B.
**Expected:** 200 with items containing the request from User A, including `requester.handle`, `message_count`, `created_at`.

### C8. List outgoing requests
**Setup:** User A has sent a pending request to User B.
**Steps:**
1. `GET /api/v1/match-requests?direction=outgoing&status=pending` as User A.
**Expected:** 200 with items containing the request to User B.

### C9. Get request detail
**Setup:** Pending request from User A to User B.
**Steps:**
1. `GET /api/v1/match-requests/{id}` as User A.
2. `GET /api/v1/match-requests/{id}` as User B.
**Expected:** 200 with request details including `mutual_teach`, `mutual_learn`, `bridge_languages`.

### C10. Accept request (happy path)
**Setup:** Pending request from User A to User B.
**Steps:**
1. `POST /api/v1/match-requests/{id}/accept` as User B.
**Expected:** 200 with `ok: true` and a `match_id`. Request status becomes `accepted`.

### C11. Accept — not recipient (403)
**Setup:** Pending request from User A to User B.
**Steps:**
1. `POST /api/v1/match-requests/{id}/accept` as User A (the requester).
**Expected:** 403 with `ERR_NOT_RECIPIENT`.

### C12. Accept — not pending (409)
**Setup:** Request already accepted/declined/canceled.
**Steps:**
1. `POST /api/v1/match-requests/{id}/accept` as User B.
**Expected:** 409 with `ERR_NOT_PENDING`.

### C13. Decline request
**Setup:** Pending request from User A to User B.
**Steps:**
1. `POST /api/v1/match-requests/{id}/decline` as User B.
**Expected:** 200 with `ok: true`. Request status becomes `declined`.

### C14. Cancel request
**Setup:** Pending request from User A to User B.
**Steps:**
1. `POST /api/v1/match-requests/{id}/cancel` as User A.
**Expected:** 200 with `ok: true`. Request status becomes `canceled`.

### C15. Pre-accept messaging — send message
**Setup:** Pending request from User A to User B.
**Steps:**
1. `POST /api/v1/match-requests/{id}/messages` as User B with `{ "body": "Hello!" }`.
**Expected:** 201 with message details.

### C16. Pre-accept messaging — list messages
**Setup:** Pending request with at least one message.
**Steps:**
1. `GET /api/v1/match-requests/{id}/messages` as User A.
**Expected:** 200 with items containing messages in chronological order.

### C17. Pre-accept messaging — message limit (429)
**Setup:** User A has sent `PRE_MATCH_MESSAGE_LIMIT` messages on a pending request.
**Steps:**
1. `POST /api/v1/match-requests/{id}/messages` as User A with another message.
**Expected:** 429 with `ERR_MESSAGE_LIMIT`.

### C18. Pre-accept messaging — not pending (409)
**Setup:** Request already accepted.
**Steps:**
1. `POST /api/v1/match-requests/{id}/messages` as User A.
**Expected:** 409 with `ERR_NOT_PENDING`.

### C19. Pre-accept messaging — not participant (403)
**Setup:** Pending request between User A and User B.
**Steps:**
1. `POST /api/v1/match-requests/{id}/messages` as User C (unrelated user).
**Expected:** 403 with `ERR_NOT_PARTICIPANT`.

### C20. Accept re-associates messages to match
**Setup:** Pending request with pre-accept messages.
**Steps:**
1. `POST /api/v1/match-requests/{id}/accept` as User B.
2. Verify messages are now associated with the new match (query DB or future match messages endpoint).
**Expected:** Messages have `match_id` set and `match_request_id` cleared.

### C21. Request pagination
**Setup:** User B has multiple incoming pending requests.
**Steps:**
1. `GET /api/v1/match-requests?direction=incoming&status=pending&limit=2` as User B.
**Expected:** `items` ≤ 2; `next_cursor` non-null if more exist. Second page has no duplicates.

### C22. Connection error localization
**Steps:**
1. `POST /api/v1/match-requests` (self-request) with `Accept-Language: zh-Hans`.
2. Same with `Accept-Language: pt-BR`.
**Expected:** Error messages localized in Chinese and Portuguese respectively.

### C23. Unauthenticated access (401)
**Steps:**
1. `POST /api/v1/match-requests` with no auth header.
**Expected:** 401 with `ERR_AUTH_REQUIRED`.

## 11. Localization & Errors
1. All error responses conform to standard error shape.
2. `Accept-Language` yields localized error messages.
3. Unauthorized access returns standardized error code.
4. Validation errors are localized and include field details.
