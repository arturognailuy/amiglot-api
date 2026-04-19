---
domain: Designs
status: Active
entry_points:
  - internal/repository/discovery.go
dependencies:
  - .aidoc/designs/discovery-matching.md
  - .aidoc/designs/database-schema.md
---

# Discovery Matching — SQL Query & Indexes

The single-pass CTE query and index strategy for the discovery matching endpoint. This is the primary latency bottleneck.

## Related Docs

| Document | Relationship |
|----------|-------------|
| [Discovery Matching](discovery-matching.md) | Parent design — endpoint contract and matching rules |
| [Database Schema](database-schema.md) | Tables and constraints referenced by the query |

## Why a Single-Pass CTE

Pushing matching logic into SQL minimizes round-trips and leverages PostgreSQL's query planner. The alternative (fetch all users, filter in Go) does not scale.

## Query Strategy

The CTE flows through these stages:
1. **`me_teach`/`me_target`/`me_bridge`/`me_slots`** — Extract the requesting user's language and availability data
2. **`candidates`** — Filter users passing supply + demand + bridge checks + not blocked + discoverable + cursor pagination
3. **`candidate_slots`** — Convert candidate availability to UTC minutes
4. **`overlap`** — Compute per-slot overlaps between user and each candidate
5. **`overlap_totals`** — Sum overlaps, filter by `min_overlap_minutes`
6. **Final SELECT** — Join with profiles, order by `total_overlap_minutes DESC, user_id ASC`

Availability conversion uses `AT TIME ZONE` with IANA names for automatic DST handling. The reference date should be "now" in production (not hardcoded).

## Index Strategy

```sql
-- Supply/demand/bridge subqueries
CREATE INDEX IF NOT EXISTS user_languages_target_idx
    ON user_languages(user_id, language_code, level) WHERE is_target = true;

-- Block lookups (both directions)
CREATE INDEX IF NOT EXISTS user_blocks_pair_idx
    ON user_blocks(blocker_id, blocked_id);
CREATE INDEX IF NOT EXISTS user_blocks_reverse_idx
    ON user_blocks(blocked_id, blocker_id);
```

## Performance Considerations

| Concern | Mitigation |
|---------|-----------|
| UTC conversion per slot | PG handles TZ conversion efficiently; fine for V1 scale |
| Large candidate set | `discoverable` + language checks narrow early; pagination limits output |
| DST correctness | `AT TIME ZONE` with IANA names handles DST automatically |
| Sorting stability | `ORDER BY total_overlap_minutes DESC, user_id ASC` for deterministic cursor pagination |

## Migration

Migration `00006_discovery_indexes.sql` adds the indexes above. No schema changes required.

<!-- TODO: verify the full CTE query is in internal/repository/discovery.go — check sqlc generated code vs raw SQL -->
