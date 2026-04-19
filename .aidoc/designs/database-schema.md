---
domain: Designs
status: Active
entry_points:
  - internal/repository/queries.sql
dependencies:
  - .aidoc/architecture/guidelines.md
---

# Database Schema

Database schema for Amiglot API V1 — tables, constraints, and migration notes.

## Related Docs

| Document | Relationship |
|----------|-------------|
| [Architecture Guidelines](../architecture/guidelines.md) | Data conventions (UUIDs, timestamps, handles) |
| [API Contract](api-contract.md) | Endpoint behaviors that use this schema |
| [Discovery Matching](discovery-matching.md) | Matching query that queries these tables |
| [Connection Handshake](connection-handshake.md) | Connection state machine tables |

## Why This Schema Exists

Supports the V1 feature set: auth, profiles with languages/availability, discovery matching, connection handshake, and basic safety (blocks/reports). Schema design decisions prioritize DST-safe availability storage and order-preserving lists.

## Core Tables

- **`users`** — Auth + identity (email only in V1). See `migrations/00001_*.sql`.
- **`profiles`** — One row per user: handle, handle_norm, birth_year/month, country_code, timezone, discoverable flag. Handle constraint: `^[a-zA-Z0-9]+$`.
- **`user_languages`** — Languages per user with level (0–5), is_native, is_target, description, sort_order. Unique on `(user_id, language_code)`. At least one `is_native = true` enforced by app.
- **`availability_slots`** — Weekly availability in **local time + timezone** (not UTC). Matching converts to UTC at query time for DST safety. Grouped slots sharing `(start_local_time, end_local_time, timezone)` must share `sort_order`.

## Matching & Messaging Tables

- **`match_requests`** — State machine: `pending` → `accepted`/`declined`/`canceled`. Unique pending constraint per `(requester_id, recipient_id)`.
- **`matches`** — Accepted connections. Unique pair via `LEAST/GREATEST` index.
- **`messages`** — Single table for both pre-accept and match messages. Pre-accept messages reference `match_request_id`; on accept, re-associated to `match_id` (no copy). CHECK constraint ensures exactly one FK is set.

## Safety Tables

- **`user_blocks`** — Bidirectional block lookup. Unique on `(blocker_id, blocked_id)`.
- **`user_reports`** — Reporter + reported + optional message.

## Key Design Decisions

- **Availability in local time:** Stored as `(weekday, start_local_time, end_local_time, timezone)`. Matching converts to UTC for specific dates, handling DST shifts without rewriting rows.
- **Handle normalization:** `handle_norm` stores lowercase for case-insensitive uniqueness. API accepts optional leading `@` and strips it.
- **Sort order:** `sort_order` on languages and availability preserves user-defined ordering. API field name: `order`.
- **Message re-association:** On accept, messages move from `match_request_id` to `match_id` via UPDATE (not copy), preserving conversation continuity.

## Migration Notes

- Add new tables via sequential goose migrations.
- When backfilling `sort_order`, use existing `created_at ASC` row order.
- For availability, assign same order to slots sharing `(start_local_time, end_local_time, timezone)`.

<!-- TODO: verify — are all migration files in `migrations/` or `db/migrations/`? Check actual directory structure. -->
