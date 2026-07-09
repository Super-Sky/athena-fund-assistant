# Decision Journal PostgreSQL Persistence

## Goal

Persist the selected decision option and its review task in PostgreSQL so the fund assistant can still read decision evidence, user notes, and review triggers after API or container restarts.

## Current Capability

- `journal.Store` separates in-memory and PostgreSQL implementations.
- With `DATABASE_URL`, API startup connects to PostgreSQL and applies an idempotent schema migration.
- Journals and review tasks use separate tables while preserving complete JSON business snapshots.
- `POST /api/journals` atomically writes the journal and its first review task.
- `GET /api/journals/{journalID}` and `GET /api/reviews/{reviewID}` expose read paths.
- `/readyz` checks the active journal store and is used by Docker Compose as the API health gate.
- Without `DATABASE_URL`, only direct development and tests fall back to the in-memory store.

## Boundaries

- This slice does not store authentication data, brokerage credentials, or trade instructions.
- Redis is not wired in this slice; it remains reserved for provider caching and temporary task state.
- PostgreSQL stores fund-assistant business objects only. No fund business table is added to Athena.

## Verification

- Journal package unit tests.
- Optional PostgreSQL integration test: set `ATHENA_FUND_PG_TEST_DSN` and run `go test ./internal/journal`.
- Docker acceptance: create a journal, restart the API container, and confirm the record remains readable.

