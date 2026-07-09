# Account Performance Dashboard

## Scope

This feature moves the fund assistant from a one-off fund analysis page toward an account-based application. The first slice provides a local demo user, account-level holding snapshots, total return, recent operation return, performance trend, holding list, and provenance trace.

## Implemented

- `internal/domain/account.go`
  - Defines `UserAccount`, `AccountHoldingSnapshot`, `AccountOperationRecord`, `AccountPerformancePoint`, and `AccountOverview`.
- `internal/account/store.go`
  - Provides the account store interface and a local `MemoryStore` seeded with `demo-user`.
- `GET /api/accounts/{user_id}/overview`
  - Returns the account homepage read model.
- `POST /api/accounts/{user_id}/holdings`
  - Accepts manually entered holdings and recalculates account performance.
- `apps/web`
  - Shows total market value, total return, recent operation return, performance trend, and holding structure on the homepage.

## Boundaries

- Account data is still local memory plus mock/demo data, with `trace.mock_data_temporary=true`.
- The app does not store brokerage accounts, brokerage credentials, or order-placement capability.
- Account authorization sync remains a future read-only direction, so `read_only_sync_available=false`.
- CNY / USD holdings use `fx_to_base` to normalize into the account base currency instead of mixing US and China timelines without provenance.

## Follow-Up

- Add PostgreSQL schema for users, accounts, holding snapshots, operation records, and journal/review links.
- Move journal/review from memory storage to persistent storage.
- Connect account holdings to real data providers and replace mock/demo prices and FX.
- Integrate with Athena remote tools so the Agent can read account overview and write decision journals.

## Verification

- `go test ./...`
- `yarn build` in `apps/web`
