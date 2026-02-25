# Amiglot API

Backend service for Amiglot — find language learning partners.

## Stack
- HTTP framework: [Huma](https://github.com/danielgtaylor/huma)
- DB driver: [pgx](https://github.com/jackc/pgx)
- Query layer: [sqlc](https://sqlc.dev)
- Migrations: [goose](https://github.com/pressly/goose)

## Environment
Copy `.env.example` to `.env.local` and adjust as needed:

```bash
cp .env.example .env.local
```

Key variables:
- `PORT` (default `6176` — set by the Dockerfile; override in `.env.local` as needed)
- `DATABASE_URL`
- `ENV` (set to `dev` to enable magic-link dev behavior)
- `MAGIC_LINK_BASE_URL` (where the dev login link should point)

## Build

### Local

```bash
go build -o bin/amiglot-api ./cmd/server
```

### Docker

Build args (optional; defaults shown in Dockerfile):
- `GIT_SHA` (default `dev`)
- `GIT_BRANCH` (default `dev`)
- `BUILD_TIME_UTC` (default `unknown`)

```bash
docker build -t amiglot-api:dev \
  --build-arg GIT_SHA="$(git rev-parse HEAD)" \
  --build-arg GIT_BRANCH="$(git rev-parse --abbrev-ref HEAD)" \
  --build-arg BUILD_TIME_UTC="$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  .
```

## Database setup

Use a temporary Docker container for Postgres (same network as the API container).

```bash
docker network create amiglot-dev-net

docker run -d --name amiglot-dev-db --rm \
  --network amiglot-dev-net \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_DB=amiglot_dev \
  -p 5432:5432 \
  postgres:16
```

Set `DATABASE_URL` (from another container on the same network):

```bash
export DATABASE_URL="postgres://postgres:postgres@amiglot-dev-db:5432/amiglot_dev?sslmode=disable"
```

Install goose (if needed):

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
```

Run migrations:

```bash
make migrate-up
```

## Run

### Local

```bash
make run
```

### Docker

```bash
docker run --rm -d --name amiglot-dev-api \
  --network amiglot-dev-net \
  -p 6176:6176 \
  --env-file .env.local \
  amiglot-api:dev
```

Health check:

```bash
curl http://localhost:6176/api/v1/healthz
```

Example response:

```json
{
  "ok": true,
  "git_sha": "<git-sha>",
  "git_branch": "<git-branch>",
  "build_time_utc": "2026-02-25T17:30:00Z"
}
```

## Stop

- Stop the API: `Ctrl+C` (foreground) or stop the process if running in the background.
- Stop the DB container (if using Docker):

```bash
docker stop amiglot-dev-db
```

## Cleanup

If you used the Docker DB:

```bash
docker network rm amiglot-dev-net
```

(Any container started with `--rm` will be removed automatically after stop.)

## Tests

### Unit tests

```bash
make test
```

CI runs `go test -coverprofile=coverage.out ./...` and enforces a minimum total coverage of 80%.

### Testable APIs
- `GET /api/v1/healthz`
- `POST /api/v1/auth/magic-link`
- `POST /api/v1/auth/verify`

### Test steps (dev mode)

1) Ensure `ENV=dev` and `MAGIC_LINK_BASE_URL` are set (see `.env.local`).
2) Start the API.
3) Request a magic link:

```bash
curl -i -X POST http://localhost:6176/api/v1/auth/magic-link \
  -H 'Content-Type: application/json' \
  -d '{"email":"test2@example.com"}'
```

You should see a `DevLoginURL` header in the response. Copy the `token` value from that URL.

4) Verify the magic link:

```bash
curl -i -X POST http://localhost:6176/api/v1/auth/verify \
  -H 'Content-Type: application/json' \
  -d '{"token":"<token-from-devloginurl>"}'
```

Expected: `204 No Content`, with headers including `AccessToken` and `User`.
