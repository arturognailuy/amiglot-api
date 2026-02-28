# Amiglot API — Unit Test Plan

## 1. Purpose
Establish a unit testing baseline for the API repo and define coverage priorities.

## 2. Tooling
- **Test runner:** Go standard `testing` package
- **Assertions:** `testify` (`require` / `assert`)
- **Database:** real Postgres instance for DB-related tests (local dev + CI service container)

## 3. Scope & Priorities
### P0 (First wave)
- **Auth handler flows** (`internal/http/auth.go`):
  - Magic link request trims + lowercases email.
  - Dev-mode returns `dev_login_url`.
  - Verify flow consumes token and returns access token.
- **Profile validation** (`internal/http/profile.go`):
  - Handle rules (required, length, alphanumeric).
  - Timezone validation using `time.LoadLocation`.
  - Language validation (code, native/target rules, duplicate detection).
  - Availability validation (weekday bounds, start < end, timezone validity).
- **Locale middleware** (`internal/http/router.go`): Accept-Language → context locale.

### P1
- **Discoverable calculation** (`recalcDiscoverable`): requires handle, timezone, and at least one native language.
- **Handle availability** (`/profile/handle/check`): available when unused or owned by current user.
- **Profile load behavior**: empty profile defaults when no profile exists.

### P2
- **Config parsing** (`internal/config`): defaults + env overrides.
- **Token generation** (`generateToken`): length, uniqueness, hash size.

## 4. Test Environment Notes
- **Local DB:** Use the existing dev Postgres container (`amiglot-dev-db`). Set `DATABASE_URL` to that instance when running unit tests.
- **CI DB:** GitHub Actions runs a Postgres 16 service container; `DATABASE_URL` is set accordingly.
- **CI coverage:** `go test -coverprofile=coverage.out ./...` with a minimum total coverage threshold of 80%.

## 5. Current Status
- ✅ Test framework baseline in place (testify).
- ✅ Example DB connectivity test added (requires `DATABASE_URL`).
- ✅ Handler/unit coverage exists for auth/profile validation.
