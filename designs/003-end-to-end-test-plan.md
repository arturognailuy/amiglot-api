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

## 7. Availability
1. `PUT /profile/availability` replaces list.
2. Validate start < end and weekday bounds.
3. Timezone validation on slot and fallback to profile timezone.
4. Reject duplicate availability slots.

## 8. Discoverable Flag
1. After saving profile + native language, profile `discoverable` becomes true.
2. Removing native language flips `discoverable` to false.

## 9. Localization & Errors
1. All error responses conform to standard error shape.
2. `Accept-Language` yields localized error messages.
3. Unauthorized access returns standardized error code.
4. Validation errors are localized and include field details.
