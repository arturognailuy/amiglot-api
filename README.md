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
- `PORT` (default `6176`)
- `DATABASE_URL`
- `ENV` (set to `dev` to enable magic-link dev behavior)
- `MAGIC_LINK_BASE_URL` (where the dev login link should point)

## Build

```bash
go build -o bin/amiglot-api ./cmd/server
```

## Database setup

You can use a local Postgres or a temporary Docker container. Example with Docker:

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

Set `DATABASE_URL` (for example):

```bash
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/amiglot_dev?sslmode=disable"
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

```bash
make run
```

Health check:

```bash
curl http://localhost:6176/healthz
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

### Testable APIs
- `GET /healthz`
- `POST /auth/magic-link`
- `POST /auth/verify`

### Test steps (dev mode)

1) Ensure `ENV=dev` and `MAGIC_LINK_BASE_URL` are set (see `.env.local`).
2) Start the API.
3) Request a magic link:

```bash
curl -i -X POST http://localhost:6176/auth/magic-link \
  -H 'Content-Type: application/json' \
  -d '{"email":"test2@example.com"}'
```

You should see a `DevLoginURL` header in the response. Copy the `token` value from that URL.

4) Verify the magic link:

```bash
curl -i -X POST http://localhost:6176/auth/verify \
  -H 'Content-Type: application/json' \
  -d '{"token":"<token-from-devloginurl>"}'
```

Expected: `204 No Content`, with headers including `AccessToken` and `User`.
