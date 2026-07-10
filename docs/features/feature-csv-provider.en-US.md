# CSV Data Provider

## Scope

This feature wires a no-key, locally verifiable fallback data path into `internal/data.Provider`. It lets users provide normalized CSV files for fund diagnosis, three-option decision matrices, Athena remote tools, and local demos while making clear that the data is not a licensed real-time market feed.

## Implemented

- `internal/data/csv_provider.go`
  - Loads one CSV file or every CSV file in a directory.
  - Supports normalized `fund` / `ETF` / `LOF`, `equity`, `index`, `fx`, and `calendar` rows.
  - Requires every row to include `source`, `provider`, `fetched_at`, `market_time`, `timezone`, `delay`, `license_terms`, `confidence`, and `schema_version`.
  - Generates a SHA256 `raw_payload_hash` from the raw CSV row when the field is omitted.
- `examples/market-data-sample.csv`
  - Covers sample China ETF / index, US ETF / equity / index, USD/CNY FX, and China plus US market calendars.
- `cmd/api`
  - Enables the CSV provider when `ATHENA_FUND_PROVIDER=csv`.
  - Reads a CSV file or directory from `ATHENA_FUND_CSV_PATH`.
  - Validates `510300`, `QQQ`, `AAPL`, `000300`, `NDX`, `USD/CNY`, and `CN` / `US` market calendars before the API listens.
- `internal/domain.TraceSummary`
  - Adds `data_boundary` and `temporary_data` while keeping the older `mock_data_temporary` compatibility field.
- `apps/web`
  - Shows `data_boundary` and generic `temporary_data` in the Trace panel.

## Boundaries

- The CSV provider is only for local MVP runs, demos, user-supplied samples, or temporary fallback.
- The CSV provider does not replace real-provider admission for licensing, quota, schema, and timezone validation.
- The sample data uses `user_supplied_csv_for_local_mvp_not_licensed_live_feed` and must not be described by the UI, API, or docs as real-time market data.
- The CSV provider does not trade, connect to brokers, or store third-party account credentials.

## Verification

- `go test ./internal/data`
- `go test ./...`
- `yarn build` in `apps/web`
- `git diff --check`
- CSV API smoke:

```bash
ATHENA_FUND_PROVIDER=csv \
ATHENA_FUND_CSV_PATH=examples/market-data-sample.csv \
ATHENA_FUND_API_ADDR=:18084 \
go run ./cmd/api
```

Observed response checks:

- `/healthz` returns `{"status":"ok"}`.
- `/api/analysis/fund` returns `csv_provider`, `temporary_data=true`, `data_boundary=user_supplied_csv_for_local_mvp_not_licensed_live_feed`, and the `conservative` / `balanced` / `aggressive` options.
