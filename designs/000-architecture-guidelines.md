---
description: "Backend architecture and coding standards for Amiglot API."
whenToUse: "Read when designing handlers/services/repos, i18n, or error handling for the API."
---

# Amiglot API — Architecture & Coding Guidelines

## 0. Scope
This document is the single source of truth for backend architecture and coding standards. All backend guidelines live here.

## 1. Technical Standards
- Go 1.24
- Huma (HTTP framework)
- PostgreSQL with pgx + sqlc, migrations via goose
- API port: 6176
- Base URL path prefix: `/api/v1` (dev via Caddy: `https://test.example.com/api/v1`)

## 2. Data & Schema Conventions
- **Primary keys:** UUID (`gen_random_uuid()`)
- **Timestamps:** `timestamptz` in UTC
- **Handles:** stored **without** `@`, UI displays with `@` (letters/numbers; API accepts an optional leading `@` and normalizes it away)
- **Timezone:** IANA name (e.g., `America/Vancouver`)
- **Languages:** BCP-47 language code (e.g., `en`, `es-ES`)

---

## 3. Architectural Philosophy
* **Separation of Concerns:** Do not write business logic inside HTTP handlers. Keep a strict boundary between Transport (HTTP/Huma), Service (Business Logic), and Repository (Database/Storage) layers.
* **Architectural Depth:** Value clear orchestration, explicit context propagation, and predictable latency over building unnecessary surface-level abstractions.
* **Data over Code:** Configuration and static data (like translations) must live outside the compiled execution path (e.g., in JSON/TOML files) and be injected or embedded, never hardcoded into logic files.

---

## 4. API & Routing (Huma)
* **Standard:** Use **Huma** for declarative routing and automatic OpenAPI generation.
* **Implementation:**
  * **Strong Typing:** Always define strict Go structs for Request (`Input`) and Response (`Output`) models. Do not use `map[string]interface{}`.
  * **Delegated Validation:** Let Huma handle schema validation (required fields, lengths, regex) via struct tags. Do not write manual validation blocks inside handlers unless it involves cross-field or database-dependent business rules.
  * **Error Wrapping:** Override Huma's default error formatter to intercept schema validation errors, ensuring they are localized before returning to the client.

---

## 5. Internationalization (i18n)
* **Standard:** `golang.org/x/text` ecosystem combined with external locale files.
* **Implementation:**
  * **Context Injection:** Extract the `Accept-Language` header in a top-level middleware, resolve it against the supported languages array, and inject the resulting `language.Tag` into the `context.Context`.
  * **External Data:** Translation strings must be stored in standard external files (e.g., `locales/en.json`, `locales/pt-BR.json`) and bundled into the binary using the `//go:embed` directive.
  * **English Fallback:** The locale resolver must always be configured with English as the absolute fallback for any missing individual translation keys. Never return an empty string or a raw key ID to the client.

---

## 6. Context & Concurrency
* **Standard:** `context.Context` must be the first parameter of any function that crosses an I/O boundary (DB, external API, background task).
* **Implementation:**
  * Always respect context cancellation to free up resources when a client disconnects early.
  * **Background Tasks:** When spinning off a goroutine (e.g., sending a Magic Link email) that outlives the HTTP request, **do not** pass the HTTP context directly. You must extract necessary values (like the locale tag) and create a new, detached context, or use `context.WithoutCancel`.

---

## 7. Error Handling
* **Standard:** Errors should be treated as values, wrapped for internal tracing, but mapped to clean, standardized codes for the client.
* **Implementation:**
  * **API Responses:** Return structured JSON errors containing a standard HTTP status code, a machine-readable Error Code (e.g., `ERR_AUTH_INVALID`), and a human-readable, localized message.
  * **Logging:** Log the original, unwrapped error with a stack trace on the server side, but never leak internal database or system paths to the client response.
