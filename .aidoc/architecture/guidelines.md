---
domain: Architecture
status: Active
entry_points:
  - internal/http/router.go
dependencies:
  - .aidoc/INDEX.md
---

# Architecture & Coding Guidelines

Backend architecture and coding standards for the Amiglot API. This is the single source of truth for all backend architectural decisions.

## Related Docs

| Document | Relationship |
|----------|-------------|
| [API Contract](../designs/api-contract.md) | Technical specification and DB schema |
| [Discovery Matching](../designs/discovery-matching.md) | Matching query design |
| [Connection Handshake](../designs/connection-handshake.md) | Connection state machine |

## Why These Standards Exist

Consistency across contributors and AI agents. Clear layer separation prevents business logic leaking into HTTP handlers, which has historically caused testing and maintenance issues.

## Technical Stack

- Go 1.24, Huma (HTTP framework), PostgreSQL with pgx + sqlc, migrations via goose
- API port: 6176, base path: `/api/v1`

## Data & Schema Conventions

- **Primary keys:** UUID (`gen_random_uuid()`)
- **Timestamps:** `timestamptz` in UTC
- **Handles:** stored without `@`, UI displays with `@`; letters/numbers only, case-insensitive via `handle_norm`
- **Timezone:** IANA name (e.g., `America/Vancouver`)
- **Languages:** BCP-47 code (e.g., `en`, `es-ES`)

## Architectural Layers

Strict separation: **Transport** (HTTP/Huma) → **Service** (Business Logic) → **Repository** (Database). No business logic in handlers.

- **Data over Code:** Translations and config live in external files (JSON/TOML), injected or embedded via `//go:embed`.
- **Strong Typing:** Always define strict Go structs for Huma `Input`/`Output` models. No `map[string]interface{}`.
- **Delegated Validation:** Let Huma handle schema validation via struct tags. Manual validation only for cross-field or DB-dependent rules.
- **Error Wrapping:** Override Huma's error formatter to localize schema validation errors.

## Internationalization (i18n)

- `golang.org/x/text` ecosystem with external locale files (`locales/*.json`), bundled via `//go:embed`.
- Extract `Accept-Language` in middleware, resolve to `language.Tag`, inject into `context.Context`.
- English is the absolute fallback. Never return empty strings or raw key IDs.

## Context & Concurrency

- `context.Context` must be the first parameter of any I/O-crossing function.
- Always respect context cancellation.
- Background goroutines must not use the HTTP context directly — extract values and create a detached context.

## Error Handling

- Return structured JSON errors: HTTP status, machine-readable code (e.g., `ERR_AUTH_INVALID`), localized message.
- Log original unwrapped errors server-side with stack traces. Never leak internal paths to clients.
