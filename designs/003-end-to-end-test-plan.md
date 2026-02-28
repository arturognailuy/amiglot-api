# Amiglot API — End-to-End Test Plan

## 1. Scope
End-to-end coverage for the current API feature set, focusing on real request/response flows and persistence.

## 2. Test Environment
- API running locally on port 6176
- Postgres dev database
- Base URL: `https://test.gnailuy.com/api/v1`
- Localization via `Accept-Language`

## 3. Health & Readiness
1. `GET /healthz` returns `{ ok: true }` and build metadata.
2. `GET /readyz` returns `{ ok: true }` when DB is reachable.
3. `GET /metrics` returns Prometheus text (if enabled).

## 4. Authentication
1. `POST /auth/magic-link` returns `{ ok: true }` (dev mode returns `dev_login_url`).
2. `POST /auth/verify` with a valid token returns access token + user payload.
3. `POST /auth/verify` with invalid token returns localized error.
4. `POST /auth/logout` invalidates session/token.

## 5. Profile & Handle
1. `GET /profile` requires auth.
2. `PUT /profile` creates/updates profile with required fields.
3. `POST /profile/handle/check` returns availability.
4. Handle normalization accepts leading `@` and stores without `@`.

## 6. Languages
1. `PUT /profile/languages` replaces list.
2. Enforce at least one native language.
3. Validate level bounds and duplicates.

## 7. Availability
1. `PUT /profile/availability` replaces list.
2. Validate start < end and wrap-around handling.
3. Enforce timezone format and weekday bounds.

## 8. Discovery & Matching
1. `POST /search` returns paginated results with `next_cursor`.
2. Validate filters (min_level, age_range, country_code).
3. Confirm `availability_summary` is computed and localized.

## 9. Match Requests
1. `POST /match-requests` creates request.
2. `GET /match-requests` lists incoming/outgoing with pagination.
3. `POST /match-requests/{id}/messages` sends pre-accept message.
4. `POST /match-requests/{id}/accept` creates match and re-associates messages.
5. `POST /match-requests/{id}/decline` updates status.

## 10. Matches & Messaging
1. `GET /matches` lists matches with pagination.
2. `GET /matches/{id}/messages` lists conversation.
3. `POST /matches/{id}/messages` sends a message.
4. `POST /matches/{id}/close` closes match.

## 11. Safety
1. `POST /blocks` creates block entry.
2. `POST /reports` creates report entry.

## 12. Localization & Errors
1. All error responses conform to standard error shape.
2. `Accept-Language` yields localized error messages.
3. Unauthorized access returns standardized error code.
4. Validation errors are localized and include field details.
