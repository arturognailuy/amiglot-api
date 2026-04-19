---
domain: Designs
status: Active
entry_points:
  - internal/handler/connection.go
  - internal/service/connection.go
  - internal/repository/connection.go
dependencies:
  - .aidoc/designs/database-schema.md
  - .aidoc/designs/discovery-matching.md
  - .aidoc/architecture/guidelines.md
---

# Connection (Handshake) — Design

State machine for connection requests: `None → Pending → Accepted/Declined/Canceled`. Covers sending requests, inbox listing, accept/decline/cancel actions, and pre-accept messaging.

## Related Docs

| Document | Relationship |
|----------|-------------|
| [Database Schema](database-schema.md) | match_requests, matches, messages tables |
| [Discovery Matching](discovery-matching.md) | Discovery feeds into connection requests |
| [Connection Handshake (UI)](https://github.com/gnailuy/amiglot-ui/blob/main/.aidoc/designs/connection-handshake.md) | UI flows and components for this state machine |
| [Architecture Guidelines](../architecture/guidelines.md) | Layer separation pattern |
| [API Contract](api-contract.md) | Shared endpoint conventions |

## Why This Design Exists

The handshake ensures mutual consent before connecting users. Pre-accept messaging lets users evaluate compatibility before committing, while message limits prevent spam.

## Endpoints

### POST /api/v1/match-requests
Send a connection request. Validates: recipient exists and is discoverable, not self, no pending request, no existing match, not blocked. Optional `initial_message` (max 500 chars).

### GET /api/v1/match-requests
List requests. Params: `direction` (incoming/outgoing), `status` (pending/accepted/declined/canceled/all), `cursor`, `limit`. Returns request cards with partner info, message count, last message time.

### GET /api/v1/match-requests/{id}
Single request detail. Includes `mutual_teach`, `mutual_learn`, `bridge_languages`. Accessible by requester or recipient only.

### POST /api/v1/match-requests/{id}/accept
Recipient-only. Transactional: update status → create match (LEAST/GREATEST ordering) → re-associate messages from `match_request_id` to `match_id`.

### POST /api/v1/match-requests/{id}/decline
Recipient-only. Sets status to `declined`.

### POST /api/v1/match-requests/{id}/cancel
Requester-only. Sets status to `canceled`.

### GET/POST /api/v1/match-requests/{id}/messages
List or send pre-accept messages. Per-user message limit enforced (`PRE_MATCH_MESSAGE_LIMIT`, default 5). Max body: 500 chars.

## Error Codes

| Code | Condition |
|------|-----------|
| `ERR_SELF_REQUEST` | Requester = recipient |
| `ERR_USER_NOT_FOUND` | Recipient missing or not discoverable |
| `ERR_REQUEST_EXISTS` | Pending request already exists |
| `ERR_ALREADY_MATCHED` | Active match exists |
| `ERR_USER_BLOCKED` | Either user blocked the other |
| `ERR_NOT_RECIPIENT` | Caller is not the recipient |
| `ERR_NOT_PENDING` | Request not in pending status |
| `ERR_NOT_PARTICIPANT` | Sender not part of this request |
| `ERR_MESSAGE_LIMIT` | Pre-accept message limit reached |

All error messages localized via `Accept-Language` across all 11 supported locales.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `PRE_MATCH_MESSAGE_LIMIT` | 5 | Max messages per user per request |
| `MATCH_REQUEST_MESSAGE_MAX_LENGTH` | 500 | Max body length |

## Architecture

- **Handler** (`internal/handler/connection.go`): route registration, input parsing
- **Service** (`internal/service/connection.go`): precondition validation, accept transaction, message limits
- **Repository** (`internal/repository/connection.go`): sqlc queries for CRUD, transactional accept

## Migration

Migration `00007_connection_handshake.sql` adds performance indexes for inbox queries and message count subqueries. No new tables — `match_requests`, `matches`, `messages` already exist.
