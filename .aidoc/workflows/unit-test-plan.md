---
domain: Workflows
status: Active
entry_points:
  - internal/config/config.go
dependencies:
  - .aidoc/architecture/guidelines.md
---

# Unit Test Plan

Unit testing baseline for the Amiglot API with coverage priorities and environment setup.

## Related Docs

| Document | Relationship |
|----------|-------------|
| [Architecture Guidelines](../architecture/guidelines.md) | Layer separation that determines test boundaries |
| [API Contract](../designs/api-contract.md) | Endpoint behaviors being tested |

## Why This Plan Exists

Tests validate layer separation and catch regressions in auth, profile validation, and locale handling — the areas where bugs have the highest user impact.

## Tooling

- **Test runner:** Go standard `testing` package
- **Assertions:** `testify` (`require` / `assert`)
- **Database:** Real Postgres instance for DB-related tests (local dev + CI service container)

## Coverage Priorities

### P0 (First wave)
- **Auth handler flows** (`internal/http/auth.go`): magic link request normalization, dev-mode `dev_login_url`, verify flow
- **Profile validation** (`internal/http/profile.go`): handle rules, timezone validation, language validation (code, native/target rules, duplicate detection, `order` normalization), availability validation (weekday bounds, start < end, grouped `order` normalization)
- **Locale middleware** (`internal/http/router.go`): Accept-Language → context locale

### P1
- **Discoverable calculation** (`recalcDiscoverable`): requires handle, timezone, and at least one native language
- **Handle availability** (`/profile/handle/check`): available when unused or owned by current user
- **Profile load behavior**: empty profile defaults when no profile exists

### P2
- **Config parsing** (`internal/config`): defaults + env overrides
- **Token generation** (`generateToken`): length, uniqueness, hash size

## Test Environment

- **Local DB:** Dev Postgres container (`amiglot-dev-db`). Set `DATABASE_URL` to that instance.
- **CI DB:** GitHub Actions Postgres 16 service container with `DATABASE_URL` set.
- **CI coverage:** `go test -coverprofile=coverage.out ./...` with minimum 80% threshold.

## Current Status

- ✅ Test framework baseline in place (testify)
- ✅ DB connectivity test added (requires `DATABASE_URL`)
- ✅ Handler/unit coverage exists for auth/profile validation
