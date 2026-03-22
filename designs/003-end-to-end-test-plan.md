---
description: "End-to-end test plan for Amiglot API."
whenToUse: "Read when running or updating API E2E scenarios."
---

# Amiglot API — End-to-End Test Plan

## 1. Scope
End-to-end coverage for the current API feature set: health, authentication, profile, languages, and availability.

## 2. Test Environment
- API running locally on port 6176
- Postgres dev database
- Base URL: `https://test.gnailuy.com/api/v1`
- Localization via `Accept-Language` (run localization assertions in **English**, **Chinese**, and **Portuguese**)

## 2.1 Seed Data

For discovery & matching tests (M1, M5, M9, M10, etc.), use the seed script to prefill test profiles:

```bash
psql -f db/seeds/seed_test_profiles.sql
```

This script is idempotent (cleans previous seed data first) and creates 12 users covering: basic mutual match, multi-language, bridge-only, no availability overlap, minimal overlap, base-language matching (zh vs zh-Hans), blocked pairs, non-discoverable users, and rare-language targets. See comments at the end of the script for the expected match matrix.

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

## 10. Localization & Errors
1. All error responses conform to standard error shape.
2. `Accept-Language` yields localized error messages.
3. Unauthorized access returns standardized error code.
4. Validation errors are localized and include field details.
