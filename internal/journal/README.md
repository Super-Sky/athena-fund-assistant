# Journal Module

The journal module owns the persistence boundary for user-selected decision options and follow-up review tasks.

## File Index

- `store.go`
  - Defines the context-aware `Store` contract, not-found sentinel errors, and shared journal record construction.
- `memory.go`
  - Provides the in-memory implementation used by local and test workflows.
- `postgres.go`
  - Provides the `pgxpool` PostgreSQL implementation and applies embedded migrations at startup.
- `migrations/*.sql`
  - Defines idempotent journal-entry and review-task tables with complete JSONB snapshots.
- `store_test.go`
  - Covers memory storage, review retrieval, context cancellation, and sentinel errors.
- `postgres_test.go`
  - Covers migration idempotency and PostgreSQL snapshot round trips when `ATHENA_FUND_PG_TEST_DSN` is configured.
