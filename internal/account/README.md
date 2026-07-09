# Account Module

The account module owns the account dashboard store boundary and local MVP read model.

## Boundary

- Owns account overview aggregation for the fund assistant application.
- Does not place trades, store brokerage credentials, or synchronize brokerage accounts.
- Keeps future read-only account authorization sync behind explicit fields and contracts.

## File Index

- `README.md`
  - Describes module boundary and file map.
- `store.go`
  - Defines the account store interface, local in-memory demo store, holding normalization, total return calculation, recent operation return, and trend generation.
- `postgres_store.go`
  - Implements PostgreSQL schema bootstrap, demo seed, persisted account overview reads, manual holding replacement, operation records, and performance trend persistence.
- `postgres_store_integration_test.go`
  - Verifies the PostgreSQL store when `ATHENA_FUND_PG_TEST_DSN` is configured.
- `store_test.go`
  - Verifies account overview provenance, mock markers, manual holding replacement, and allocation recalculation.
