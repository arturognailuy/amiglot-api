---
description: "Design document for the Connection (Handshake) vertical slice — match requests, acceptance, and connection creation."
whenToUse: "Read when implementing match request endpoints, connection state machine, or pre-accept messaging."
---

# Connection (Handshake) — Backend Design

> Parent docs: `001-technical-specification.md` (DB schema, API contract notes), `000-architecture-guidelines.md` (coding standards).
> Shared UI ↔ API contract: `amiglot-ui/designs/003-technical-specification.md` §2.6.
> Prior slice: `004-discovery-matching-design.md` (discovery endpoint that feeds into this slice).

## 1. Overview

This slice implements the **connection handshake** — the state machine that takes two users from strangers to connected partners:

```
None → Pending → Accepted (creates a Match)
                → Declined
                → Canceled (by requester)
```

It covers:
- **Sending a connection request** (with optional initial message)
- **Inbox** — listing incoming/outgoing requests
- **Resolving a request** — accept (creates a `match`), decline, or cancel
- **Pre-accept messaging** — limited messages before acceptance

## 2. Endpoints

All endpoints require authentication (`Authorization: Bearer <token>`).

### 2.1 POST /api/v1/match-requests

Send a connection request to another user.

**Request:**
```json
{
  "recipient_id": "uuid",
  "initial_message": "Hi! I'd love to practice Spanish with you."
}
```

**Validation (Service layer):**
- `recipient_id` must exist and be discoverable.
- Requester ≠ recipient.
- No existing pending request between the pair (either direction).
- No existing active match between the pair.
- Neither user has blocked the other.
- `initial_message` is optional; max length 500 chars.

**Response (201):**
```json
{
  "id": "uuid",
  "requester_id": "uuid",
  "recipient_id": "uuid",
  "status": "pending",
  "initial_message": "Hi! I'd love to practice Spanish with you.",
  "created_at": "2026-03-28T12:00:00Z"
}
```

**Errors:**

| Status | Code | Condition |
|--------|------|-----------|
| 400 | `ERR_SELF_REQUEST` | Requester = recipient |
| 404 | `ERR_USER_NOT_FOUND` | Recipient doesn't exist or not discoverable |
| 409 | `ERR_REQUEST_EXISTS` | Pending request already exists between pair |
| 409 | `ERR_ALREADY_MATCHED` | Active match already exists |
| 403 | `ERR_USER_BLOCKED` | Either user has blocked the other |

### 2.2 GET /api/v1/match-requests

List connection requests (incoming and outgoing).

**Query Parameters:**

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `direction` | string | `incoming` | `incoming` or `outgoing` |
| `status` | string | `pending` | `pending`, `accepted`, `declined`, `canceled`, or `all` |
| `cursor` | string | null | Opaque pagination cursor |
| `limit` | int | 20 | Page size (max 50) |

**Response (200):**
```json
{
  "items": [
    {
      "id": "uuid",
      "requester": {
        "user_id": "uuid",
        "handle": "maria",
        "country_code": "MX",
        "age": 24
      },
      "recipient": {
        "user_id": "uuid",
        "handle": "arturo",
        "country_code": "CA",
        "age": 32
      },
      "status": "pending",
      "message_count": 1,
      "last_message_at": "2026-03-28T12:00:00Z",
      "created_at": "2026-03-28T12:00:00Z"
    }
  ],
  "next_cursor": "..."
}
```

### 2.3 GET /api/v1/match-requests/{id}

View a single request's details. Only accessible by requester or recipient.

**Response (200):** Same shape as a single item from the list endpoint, plus:
```json
{
  "mutual_teach": [...],
  "mutual_learn": [...],
  "bridge_languages": [...]
}
```

### 2.4 POST /api/v1/match-requests/{id}/accept

Accept a pending request. Only the **recipient** can accept.

**Business logic (transactional):**
1. Verify request status is `pending` and caller is the recipient.
2. Update `match_requests.status = 'accepted'`, set `responded_at`.
3. Insert into `matches(user_a, user_b)` — ordered as `(LEAST, GREATEST)` to match the unique index.
4. Re-associate pre-accept messages: `UPDATE messages SET match_id = <new>, match_request_id = NULL WHERE match_request_id = <id>`.
5. Return the new match ID.

**Response (200):**
```json
{
  "ok": true,
  "match_id": "uuid"
}
```

**Errors:**

| Status | Code | Condition |
|--------|------|-----------|
| 403 | `ERR_NOT_RECIPIENT` | Caller is not the recipient |
| 409 | `ERR_NOT_PENDING` | Request is not in pending status |
| 409 | `ERR_ALREADY_MATCHED` | Match already exists (race condition guard) |

### 2.5 POST /api/v1/match-requests/{id}/decline

Decline a pending request. Only the **recipient** can decline.

**Response (200):**
```json
{ "ok": true }
```

### 2.6 POST /api/v1/match-requests/{id}/cancel

Cancel a pending request. Only the **requester** can cancel.

**Response (200):**
```json
{ "ok": true }
```

### 2.7 GET /api/v1/match-requests/{id}/messages

List pre-accept messages for a request. Accessible by requester or recipient.

**Query Parameters:**

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `cursor` | string | null | Opaque pagination cursor |
| `limit` | int | 20 | Page size (max 50) |

**Response (200):**
```json
{
  "items": [
    {
      "id": "uuid",
      "sender_id": "uuid",
      "body": "Hi! I'd love to practice Spanish with you.",
      "created_at": "2026-03-28T12:00:00Z"
    }
  ],
  "next_cursor": "..."
}
```

### 2.8 POST /api/v1/match-requests/{id}/messages

Send a pre-accept message. Accessible by requester or recipient while request is pending.

**Request:**
```json
{ "body": "Hello! When are you usually free?" }
```

**Validation:**
- Request must be in `pending` status.
- Sender must be requester or recipient.
- Per-user message limit enforced (configurable `PRE_MATCH_MESSAGE_LIMIT`, default 5).
- Max body length: 500 chars.

**Response (201):**
```json
{
  "id": "uuid",
  "sender_id": "uuid",
  "body": "Hello! When are you usually free?",
  "created_at": "2026-03-28T12:01:00Z"
}
```

**Errors:**

| Status | Code | Condition |
|--------|------|-----------|
| 409 | `ERR_NOT_PENDING` | Request not in pending status |
| 403 | `ERR_NOT_PARTICIPANT` | Sender not part of this request |
| 429 | `ERR_MESSAGE_LIMIT` | Pre-accept message limit reached |

## 3. Architecture (Layers)

Following `000-architecture-guidelines.md`:

### 3.1 Handler

`internal/handler/connection.go` — registers all routes under `/api/v1/match-requests`. Parses inputs, extracts auth user, delegates to service, serializes responses.

### 3.2 Service

`internal/service/connection.go` — orchestrates business logic:
- Validates preconditions (no duplicate request, no block, no existing match).
- Manages the accept transaction (update request → create match → re-associate messages).
- Enforces message limits.

### 3.3 Repository

`internal/repository/connection.go` (sqlc queries):
- `CreateMatchRequest`
- `ListMatchRequests` (with direction/status filters + cursor pagination)
- `GetMatchRequest`
- `AcceptMatchRequest` (transactional: update + insert match + update messages)
- `DeclineMatchRequest`
- `CancelMatchRequest`
- `CreatePreAcceptMessage`
- `ListPreAcceptMessages`
- `CountPreAcceptMessages` (for limit enforcement)

## 4. SQL Queries

### 4.1 Create Match Request

```sql
-- Precondition checks run as separate queries in the service or as CTEs
INSERT INTO match_requests (requester_id, recipient_id, status)
VALUES ($1, $2, 'pending')
RETURNING id, requester_id, recipient_id, status, created_at;
```

The initial message (if provided) is inserted into `messages` with `match_request_id` set:

```sql
INSERT INTO messages (match_request_id, sender_id, body)
VALUES ($1, $2, $3)
RETURNING id, created_at;
```

### 4.2 Accept (Transaction)

```sql
-- Step 1: Lock and update the request
UPDATE match_requests
SET status = 'accepted', responded_at = now()
WHERE id = $1 AND recipient_id = $2 AND status = 'pending'
RETURNING requester_id, recipient_id;

-- Step 2: Create match (LEAST/GREATEST for unique index)
INSERT INTO matches (user_a, user_b)
VALUES (LEAST($1, $2), GREATEST($1, $2))
ON CONFLICT DO NOTHING
RETURNING id;

-- Step 3: Re-associate messages
UPDATE messages
SET match_id = $1, match_request_id = NULL
WHERE match_request_id = $2;
```

All three steps run inside a single DB transaction.

### 4.3 List Requests (Incoming, Pending)

```sql
SELECT
    mr.id, mr.status, mr.created_at,
    mr.requester_id, mr.recipient_id,
    rp.handle AS requester_handle,
    rp.country_code AS requester_country,
    rp.birth_year AS requester_birth_year,
    rp.birth_month AS requester_birth_month,
    pp.handle AS recipient_handle,
    pp.country_code AS recipient_country,
    pp.birth_year AS recipient_birth_year,
    pp.birth_month AS recipient_birth_month,
    (SELECT COUNT(*) FROM messages m WHERE m.match_request_id = mr.id) AS message_count,
    (SELECT MAX(m.created_at) FROM messages m WHERE m.match_request_id = mr.id) AS last_message_at
FROM match_requests mr
JOIN profiles rp ON rp.user_id = mr.requester_id
JOIN profiles pp ON pp.user_id = mr.recipient_id
WHERE mr.recipient_id = $1   -- incoming; swap to requester_id for outgoing
  AND mr.status = $2          -- filter by status
  AND ($3::uuid IS NULL OR mr.id < $3)  -- cursor
ORDER BY mr.created_at DESC
LIMIT $4;
```

## 5. Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `PRE_MATCH_MESSAGE_LIMIT` | 5 | Max messages per user per request before acceptance |
| `MATCH_REQUEST_MESSAGE_MAX_LENGTH` | 500 | Max body length for pre-accept messages |

## 6. Error Handling & i18n

All error codes must have localized messages in all supported locale files:

| Code | English Message |
|------|----------------|
| `ERR_SELF_REQUEST` | "You cannot send a connection request to yourself." |
| `ERR_USER_NOT_FOUND` | "User not found." |
| `ERR_REQUEST_EXISTS` | "A pending connection request already exists." |
| `ERR_ALREADY_MATCHED` | "You are already connected with this user." |
| `ERR_USER_BLOCKED` | "This action is not available." |
| `ERR_NOT_RECIPIENT` | "Only the recipient can perform this action." |
| `ERR_NOT_PENDING` | "This request is no longer pending." |
| `ERR_NOT_PARTICIPANT` | "You are not part of this connection request." |
| `ERR_MESSAGE_LIMIT` | "You have reached the message limit for this request." |

## 7. Migration

New migration `00007_connection_handshake.sql`:

No new tables needed — `match_requests`, `matches`, and `messages` already exist from `001-technical-specification.md`. This migration adds performance indexes:

```sql
-- Speed up inbox queries (incoming requests by status)
CREATE INDEX IF NOT EXISTS match_requests_recipient_status_idx
    ON match_requests(recipient_id, status, created_at DESC);

-- Speed up outgoing requests list
CREATE INDEX IF NOT EXISTS match_requests_requester_status_idx
    ON match_requests(requester_id, status, created_at DESC);

-- Speed up message count/last_message subqueries
CREATE INDEX IF NOT EXISTS messages_match_request_created_idx
    ON messages(match_request_id, created_at DESC)
    WHERE match_request_id IS NOT NULL;
```

## 8. Implementation Checklist

- [ ] Migration: `00007_connection_handshake.sql` (indexes)
- [ ] Repository: `internal/repository/connection.go` (sqlc queries)
- [ ] Service: `internal/service/connection.go` (business logic + transaction)
- [ ] Handler: `internal/handler/connection.go` (Huma route registration)
- [ ] Config: `PRE_MATCH_MESSAGE_LIMIT`, `MATCH_REQUEST_MESSAGE_MAX_LENGTH` env vars
- [ ] i18n: Add all error codes to locale files (all 11 locales)
- [ ] Tests: Unit tests for service logic (precondition checks, accept flow, message limits)
- [ ] Tests: Integration test for the accept transaction (request → match → message re-association)
