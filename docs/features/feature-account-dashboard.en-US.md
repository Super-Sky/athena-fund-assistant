# Account Performance Dashboard

## Scope

This feature moves the fund assistant from a one-off fund analysis page toward an account-based application. The first slice provides a local demo user, account-level holding snapshots, total return, recent operation return, performance trend, holding list, and provenance trace.

## Implemented

- `internal/domain/account.go`
  - Defines `UserAccount`, `AccountHoldingSnapshot`, `AccountOperationRecord`, `AccountPerformancePoint`, and `AccountOverview`.
- `internal/account/store.go`
  - Provides the account store interface and a local `MemoryStore` seeded with `demo-user`.
- `internal/account/postgres_store.go`
  - Provides the PostgreSQL store, schema bootstrap, demo seed, holding replacement, and persisted trend points.
- `GET /api/accounts/{user_id}/overview`
  - Returns the account homepage read model.
- `POST /api/accounts/{user_id}/holdings`
  - Accepts manually entered holdings and recalculates account performance.
- `apps/web`
  - Shows total market value, total return, recent operation return, performance trend, and holding structure on the homepage.

## Boundaries

- Without `DATABASE_URL`, local runs use the in-memory demo store. Docker / DATABASE_URL environments use the PostgreSQL store.
- Current account market data is still mock/demo data, with `trace.mock_data_temporary=true`.
- The app does not store brokerage accounts, brokerage credentials, or order-placement capability.
- User sessions and revocable account-read consent are implemented. Real brokerage/account sync remains a future read-only direction, so `read_only_sync_available=false`.
- CNY / USD holdings use `fx_to_base` to normalize into the account base currency instead of mixing US and China timelines without provenance.

## Follow-Up

- Link the already persistent journal/review records to accounts and holdings.
- Connect account holdings to real data providers and replace mock/demo prices and FX.
- Connect the verified service-identity plus consent account-read path to real account data providers; account prices and FX are still mock/demo data.

## Verification

- `go test ./...`
- `ATHENA_FUND_PG_TEST_DSN=... go test ./internal/account -run TestPostgresStoreOverviewAndReplaceHoldings -count=1`
- `yarn build` in `apps/web`
- `ATHENA_REPO=../Athena ./scripts/smoke_dual_docker.sh`
