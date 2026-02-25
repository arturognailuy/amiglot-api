# Amiglot API — Unit Test Plan

## 1. Purpose
Establish a unit testing baseline for the API repo and define coverage priorities.

## 2. Tooling
- **Test runner:** Go standard `testing` package
- **Assertions:** `testify` (`require` / `assert`)
- **Database:** real Postgres instance for DB-related tests (local dev + CI service container)

## 3. Scope & Priorities
### P0 (First wave)
- **DB connectivity** (`internal/db`): connection creation and ping behavior.
- **Config parsing** (`internal/config`): default values + env overrides.

### P1
- **Auth handlers**: magic link request + verify flow (unit tests with mocked dependencies).
- **Validation rules**: handle/locale validation and request payload guards.

### P2
- **Query logic**: sqlc query tests (seeded DB with minimal fixtures).

## 4. Test Environment Notes
- **Local DB:** Use the existing dev Postgres container (`amiglot-dev-db`). Set `DATABASE_URL` to that instance when running unit tests.
- **CI DB:** GitHub Actions runs a Postgres 16 service container; `DATABASE_URL` is set accordingly.

## 5. Current Status
- ✅ Test framework baseline in place (testify).
- ✅ Example DB connectivity test added (requires `DATABASE_URL`).

