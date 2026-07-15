# Authorization Module

This module owns local bearer sessions, revisioned read-only consent grants, service-side authorization decisions, and minimal redacted audit events.

## Boundary

- Stores only SHA-256 hashes of bearer session tokens.
- Supports read-only scopes ending in `.read`; no write or trading scope is accepted.
- Keeps user session authentication separate from Athena service authentication.
- Persists only safe session/grant references, scope, decision, and grant revision in audit events.
- Does not store brokerage credentials, account payloads, or model secrets.

## File Index

- `README.md`
  - Describes module ownership and file map.
- `types.go`
  - Defines sessions, grants, scopes, denial codes, decisions, audit events, and the store contract.
- `service.go`
  - Issues sessions, hashes tokens, manages grants, and evaluates user-backed or service-backed authorization requests.
- `memory_store.go`
  - Implements the thread-safe, non-durable local/test store.
- `postgres_store.go`
  - Implements PostgreSQL persistence and embedded migration execution.
- `migrations/001_authorization.sql`
  - Creates session, consent-grant, and minimal audit tables.
- `service_test.go`
  - Verifies lifecycle, denial codes, redaction, stable ordering, and concurrency behavior.
- `postgres_store_test.go`
  - Verifies migration safety and optional PostgreSQL persistence when `ATHENA_FUND_PG_TEST_DSN` is configured.
