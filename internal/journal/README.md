# Journal Module

The journal module owns durable user-selected decision options and their follow-up review tasks.

## File Index

- `store.go`
  - Defines the context-aware store contract, sentinel missing-record errors, and immutable evidence/review record construction.
- `memory.go`
  - Provides a non-durable local and test fallback.
- `postgres.go`
  - Persists journal and review snapshots atomically through PostgreSQL with embedded idempotent migrations.
- `migrations/`
  - Defines the `journal_entries` and `review_tasks` tables.
- `store_test.go`
  - Covers the in-memory contract, generated review timing, context cancellation, and missing-record behavior.
- `postgres_test.go`
  - Covers optional PostgreSQL restart persistence when `ATHENA_FUND_PG_TEST_DSN` is configured.
