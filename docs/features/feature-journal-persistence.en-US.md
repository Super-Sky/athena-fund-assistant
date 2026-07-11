# Durable Decision Journal And Review

## Scope

This feature makes a user-selected decision option durable without moving financial business objects into Athena. It preserves the entire decision matrix as the evidence snapshot used at selection time and creates the first review task in the same operation.

## Implemented

- `internal/journal.Store` is a context-aware persistence boundary with in-memory and PostgreSQL implementations.
- PostgreSQL persists journal entries and review tasks atomically as JSONB snapshots, with idempotent embedded migrations.
- `POST /api/journals` creates the selected decision and its review task.
- `GET /api/journals/{journal_id}` and `GET /api/reviews/{review_id}` make the durable snapshots readable for later review work.
- `GET /readyz` checks the journal store for Compose readiness.

## Boundaries

- The user still selects an option; this feature never trades or sends broker instructions.
- `DATABASE_URL` enables PostgreSQL durability. Without it, the fallback store is explicitly non-durable and intended only for local development/tests.
- The snapshot preserves existing source, freshness, rule, governance, risk, invalidation, and review information. It does not claim that data is live or licensed.
- Account/journal relationship persistence and review comparison against later market observations remain follow-up work.

## Verification

- `go test ./internal/journal ./internal/server ./cmd/api`
- `go test -race ./internal/journal ./internal/server`
- `ATHENA_FUND_PG_TEST_DSN=... go test ./internal/journal -run TestPostgresStorePersistence -count=1`
- `docker compose config --quiet`

## Maintenance Entry

Feature skill intentionally deferred: the persistence boundary is compact and fully navigable through `internal/journal/README.md` plus this feature document; it does not yet justify a separate recurring-maintenance skill.
