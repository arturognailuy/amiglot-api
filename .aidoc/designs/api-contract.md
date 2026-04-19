---
domain: Designs
status: Active
entry_points:
  - internal/http/router.go
dependencies:
  - .aidoc/designs/database-schema.md
  - .aidoc/architecture/guidelines.md
---

# API Contract

API endpoint contract, authentication, validation rules, and operational concerns for Amiglot API V1.

## Related Docs

| Document | Relationship |
|----------|-------------|
| [Database Schema](database-schema.md) | Tables backing these endpoints |
| [Architecture Guidelines](../architecture/guidelines.md) | Layer separation and error handling patterns |
| [Discovery Matching](discovery-matching.md) | Discovery endpoint design |
| [Connection Handshake](connection-handshake.md) | Connection endpoint design |

## Why This Doc Exists

Captures implementation constraints and business rules that are not obvious from the endpoint signatures alone — validation nuances, authorization boundaries, and operational requirements.

## Authentication & Authorization

- Magic link auth issues access tokens. All non-public endpoints require auth.
- Authorization: resource ownership for profile/languages/availability; match membership for messaging.
- Email is returned only via `/me`, never exposed elsewhere.

## Validation & Business Rules

- Handle uniqueness is case-insensitive via `handle_norm`.
- At least one native language required on profile creation.
- Language ordering persisted via `sort_order` (API field: `order`); missing values normalized based on request list order.
- Availability ordering via `sort_order`; grouped slots sharing `(start_local_time, end_local_time, timezone)` must share the same order.
- `start_local_time < end_local_time` enforced; wrap-around slots split into two rows.
- `match_requests`: one pending request per user pair enforced.

## Rate Limits & Abuse Controls (V1)

| Endpoint | Limit |
|----------|-------|
| `/auth/magic-link` | Per-IP + per-email |
| `/matches/discover` | Per-user and per-IP |
| `/matches/{id}/messages` | Per-user/day (per product spec) |
| Pre-accept messages | Per-user limit + daily cap (configurable) |

## Monitoring & Metrics

- **Health:** `/healthz` (JSON + build metadata), `/readyz` (DB connectivity)
- **Metrics:** Prometheus `/metrics` — request count, latency, errors, auth failures, rate-limit hits, DB latency, mail sends, message sends
- **Logging:** Structured JSON with request_id, user_id, route, status, latency
- **Tracing:** OpenTelemetry spans (HTTP + DB)
- **Dashboards:** p50/p95 latency by route; error rate; auth failures; DAU/signups/discovery/match requests/accepts/messages; safety (blocks/reports, rate-limit hits)
