---
description: "Design document for the Discovery & Matching vertical slice (GET /matches endpoint, SQL query plan, availability overlap)."
whenToUse: "Read when implementing the GET /matches endpoint, matching query logic, or availability intersection."
---

# Discovery & Matching — Backend Design

> Parent docs: `001-technical-specification.md` (DB schema), `000-architecture-guidelines.md` (coding standards).
> Shared UI ↔ API contract: `amiglot-ui/designs/003-technical-specification.md`.

## 1. Overview

This document designs the **GET /matches** endpoint — the primary discovery surface that returns a paginated list of potential language exchange partners for the authenticated user. It applies the V1 matching rules (supply, demand, bridge checks) plus an **availability overlap filter** with a configurable minimum overlap.

## 2. Endpoint: GET /matches

### 2.1 Contract

```
GET /api/v1/matches/discover?cursor=<opaque>&limit=<int>
Authorization: Bearer <token>
Accept-Language: <locale>
```

> We use `/matches/discover` to avoid collision with the existing `GET /matches` endpoint (which lists accepted matches). This keeps the resource hierarchy clean.

**Query Parameters:**

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `cursor` | string | `null` | Opaque pagination cursor |
| `limit` | int | `20` | Page size (max 50) |

**Response (200):**

```json
{
  "items": [
    {
      "user_id": "uuid",
      "handle": "maria",
      "country_code": "MX",
      "age": 24,
      "mutual_teach": [
        { "language_code": "es", "level": 5, "is_native": true }
      ],
      "mutual_learn": [
        { "language_code": "en", "level": 4, "is_native": false }
      ],
      "bridge_languages": [
        { "language_code": "en", "level": 4 }
      ],
      "availability_overlap": [
        {
          "weekday": 1,
          "start_utc": "01:00",
          "end_utc": "03:00",
          "overlap_minutes": 120
        }
      ],
      "total_overlap_minutes": 120
    }
  ],
  "next_cursor": "..."
}
```

**Error Responses:**

| Status | Code | Condition |
|--------|------|-----------|
| 401 | `ERR_AUTH_REQUIRED` | Missing/invalid token |
| 403 | `ERR_PROFILE_INCOMPLETE` | User profile not discoverable (incomplete profile or missing languages) |
| 422 | `ERR_NO_TARGET_LANGUAGES` | User has no target languages set |

All error messages are localized via `Accept-Language`.

### 2.2 Matching Rules (recap from Production Definition)

All three checks must pass for a candidate to appear:

1. **Supply check** — Candidate has a language the user wants to learn at **level ≥ 4**.
2. **Demand check** — User has a language the candidate wants to learn at **level ≥ 4**.
3. **Bridge check** — Both share at least one language where **both** are **level ≥ 3**.

Additionally:
4. **Availability overlap** — At least `min_overlap_minutes` (default: 60, configurable) of shared weekly availability.
5. **Discoverable** — Candidate must have `profiles.discoverable = true`.
6. **Not blocked** — Neither user has blocked the other.
7. **Not self** — Exclude the requesting user.

### 2.3 Architecture (Layers)

Following `000-architecture-guidelines.md`:

- **Handler** (`internal/handler/discovery.go`): Parse query params, extract user ID from auth context, call service, serialize response.
- **Service** (`internal/service/discovery.go`): Orchestrate matching logic — call repository, compute availability overlaps, apply business rules, paginate.
- **Repository** (`internal/repository/discovery.go` / sqlc queries): Execute the SQL matching query.

## 3. SQL Query Execution Plan

The matching query is the primary latency bottleneck. This section designs the query strategy.

### 3.1 Strategy: Single-Pass CTE Query

Rather than fetching all users and filtering in Go, we push the matching logic into a single SQL query using CTEs. This minimizes round-trips and leverages PostgreSQL's query planner.

### 3.2 Query Design

```sql
-- Inputs: $1 = user_id, $2 = min_overlap_minutes (default 60), $3 = limit, $4 = cursor_user_id (nullable)
WITH me_teach AS (
    -- Languages I can teach (level >= 4)
    SELECT language_code
    FROM user_languages
    WHERE user_id = $1 AND level >= 4
),
me_target AS (
    -- Languages I want to learn
    SELECT language_code
    FROM user_languages
    WHERE user_id = $1 AND is_target = true
),
me_bridge AS (
    -- Languages I can bridge (level >= 3)
    SELECT language_code, level
    FROM user_languages
    WHERE user_id = $1 AND level >= 3
),
me_slots AS (
    -- My availability converted to UTC minutes-since-week-start
    -- for a reference week (to handle DST: use next occurrence of each weekday)
    SELECT
        weekday,
        start_local_time,
        end_local_time,
        timezone,
        -- Convert to UTC offset minutes for overlap calculation
        EXTRACT(EPOCH FROM (
            (DATE '2026-03-16' + weekday * INTERVAL '1 day' + start_local_time)
            AT TIME ZONE timezone AT TIME ZONE 'UTC'
        ))::int / 60 AS start_utc_min,
        EXTRACT(EPOCH FROM (
            (DATE '2026-03-16' + weekday * INTERVAL '1 day' + end_local_time)
            AT TIME ZONE timezone AT TIME ZONE 'UTC'
        ))::int / 60 AS end_utc_min
    FROM availability_slots
    WHERE user_id = $1
),
candidates AS (
    -- Users who pass supply + demand + bridge checks
    SELECT DISTINCT p.user_id
    FROM profiles p
    WHERE p.discoverable = true
      AND p.user_id <> $1
      -- Not blocked (either direction)
      AND NOT EXISTS (
          SELECT 1 FROM user_blocks
          WHERE (blocker_id = $1 AND blocked_id = p.user_id)
             OR (blocker_id = p.user_id AND blocked_id = $1)
      )
      -- Supply check: candidate teaches what I want to learn
      AND EXISTS (
          SELECT 1 FROM user_languages ul
          WHERE ul.user_id = p.user_id
            AND ul.language_code IN (SELECT language_code FROM me_target)
            AND ul.level >= 4
      )
      -- Demand check: I teach what candidate wants to learn
      AND EXISTS (
          SELECT 1 FROM user_languages ul
          WHERE ul.user_id = p.user_id
            AND ul.is_target = true
            AND ul.language_code IN (SELECT language_code FROM me_teach)
      )
      -- Bridge check: shared language both >= 3
      AND EXISTS (
          SELECT 1 FROM user_languages ul
          JOIN me_bridge mb ON ul.language_code = mb.language_code
          WHERE ul.user_id = p.user_id
            AND ul.level >= 3
      )
      -- Cursor-based pagination
      AND ($4::uuid IS NULL OR p.user_id > $4)
),
candidate_slots AS (
    -- Candidate availability in UTC minutes
    SELECT
        c.user_id AS candidate_id,
        a.weekday,
        a.start_local_time,
        a.end_local_time,
        a.timezone,
        EXTRACT(EPOCH FROM (
            (DATE '2026-03-16' + a.weekday * INTERVAL '1 day' + a.start_local_time)
            AT TIME ZONE a.timezone AT TIME ZONE 'UTC'
        ))::int / 60 AS start_utc_min,
        EXTRACT(EPOCH FROM (
            (DATE '2026-03-16' + a.weekday * INTERVAL '1 day' + a.end_local_time)
            AT TIME ZONE a.timezone AT TIME ZONE 'UTC'
        ))::int / 60 AS end_utc_min
    FROM candidates c
    JOIN availability_slots a ON a.user_id = c.user_id
),
overlap AS (
    -- Compute per-slot overlaps between me and each candidate
    SELECT
        cs.candidate_id,
        ms.weekday,
        -- Overlap = min(end1, end2) - max(start1, start2)
        GREATEST(0,
            LEAST(ms.end_utc_min, cs.end_utc_min) -
            GREATEST(ms.start_utc_min, cs.start_utc_min)
        ) AS overlap_min,
        -- For response: overlap start/end in UTC
        TO_CHAR(
            INTERVAL '1 minute' * GREATEST(ms.start_utc_min, cs.start_utc_min) % (24 * 60),
            'HH24:MI'
        ) AS start_utc,
        TO_CHAR(
            INTERVAL '1 minute' * LEAST(ms.end_utc_min, cs.end_utc_min) % (24 * 60),
            'HH24:MI'
        ) AS end_utc
    FROM me_slots ms
    JOIN candidate_slots cs ON ms.weekday = cs.weekday
    WHERE LEAST(ms.end_utc_min, cs.end_utc_min) > GREATEST(ms.start_utc_min, cs.start_utc_min)
),
overlap_totals AS (
    SELECT
        candidate_id,
        SUM(overlap_min) AS total_overlap_minutes
    FROM overlap
    GROUP BY candidate_id
    HAVING SUM(overlap_min) >= $2  -- min_overlap_minutes filter
)
SELECT
    ot.candidate_id AS user_id,
    p.handle,
    p.country_code,
    p.birth_year,
    p.birth_month,
    ot.total_overlap_minutes
FROM overlap_totals ot
JOIN profiles p ON p.user_id = ot.candidate_id
ORDER BY ot.total_overlap_minutes DESC, ot.candidate_id ASC
LIMIT $3;
```

### 3.3 Index Strategy

The existing indexes should cover most of the query. Recommended additions:

```sql
-- Speed up the supply/demand/bridge subqueries
CREATE INDEX IF NOT EXISTS user_languages_target_idx
    ON user_languages(user_id, language_code, level) WHERE is_target = true;

-- Speed up block lookups
CREATE INDEX IF NOT EXISTS user_blocks_pair_idx
    ON user_blocks(blocker_id, blocked_id);
CREATE INDEX IF NOT EXISTS user_blocks_reverse_idx
    ON user_blocks(blocked_id, blocker_id);
```

### 3.4 Performance Considerations

| Concern | Mitigation |
|---------|-----------|
| UTC conversion per slot | Use a reference week date; PG handles TZ conversion efficiently. For V1 scale this is fine. |
| Large candidate set | The `discoverable` + language checks narrow candidates early. Pagination limits output. |
| DST correctness | Using `AT TIME ZONE` with IANA names handles DST automatically. The reference date should be "now" in production (not hardcoded). |
| Sorting stability | `ORDER BY total_overlap_minutes DESC, user_id ASC` ensures deterministic cursor pagination. |

### 3.5 Configurable Minimum Overlap

The `min_overlap_minutes` parameter defaults to **60** (1 hour) and is passed as a query parameter to the SQL. It should be configurable via:

- Application config (environment variable `MATCH_MIN_OVERLAP_MINUTES`)
- Future: per-user preference (out of scope for V1)

## 4. Response Assembly (Service Layer)

After the SQL query returns candidate IDs + overlap totals, the service layer:

1. Batch-fetches candidate languages (`user_languages` for each candidate).
2. Computes `mutual_teach`, `mutual_learn`, and `bridge_languages` arrays by intersecting with the requesting user's language data.
3. Fetches per-slot overlap details (from a secondary query or in-memory from the CTE results).
4. Computes age from `birth_year`/`birth_month`.
5. Assembles the response objects.

## 5. Error Handling

- If the user has no profile or `discoverable = false` → 403 `ERR_PROFILE_INCOMPLETE` (localized).
- If the user has no target languages → 422 `ERR_NO_TARGET_LANGUAGES` (localized).
- DB errors → 500 with structured error, logged server-side.

## 6. Migration Plan

New migration `00006_discovery_indexes.sql`:
- Add the indexes from §3.3.
- No schema changes required (all tables already exist from prior migrations).

## 7. Implementation Checklist

- [ ] Migration: `00006_discovery_indexes.sql`
- [ ] Repository: `internal/repository/discovery.go` (sqlc query for matching)
- [ ] Service: `internal/service/discovery.go` (orchestration, response assembly)
- [ ] Handler: `internal/handler/discovery.go` (Huma route registration)
- [ ] Config: `MATCH_MIN_OVERLAP_MINUTES` env var
- [ ] i18n: Add `ERR_PROFILE_INCOMPLETE`, `ERR_NO_TARGET_LANGUAGES` to all locale files
- [ ] Tests: Unit tests for service logic; integration test for the SQL query
