# Credential-Backed Live Market Providers

## Scope

This feature adds credential-backed market-data adapters to the fund-assistant application layer. It does not change Athena core, does not store brokerage credentials, and does not enable trading.

## Providers

- `alpha_vantage_provider`
  - Requires a user-owned `ALPHA_VANTAGE_API_KEY`.
  - Covers US ETF snapshots, US equity snapshots, USD/CNY FX, and explicitly labeled ETF proxies for S&P 500, Nasdaq 100, and Dow Jones.
  - Uses full daily history and the latest approximately 252 trading observations for one-year return, drawdown, and volatility calculations.
  - Does not claim exchange-calendar coverage. Calendar calls return an explicit unsupported-capability error.
- `tushare_provider`
  - Requires a user-owned `TUSHARE_TOKEN`.
  - Covers China public-fund NAV, CSI 300 index data, and SSE trading-calendar records.
  - Does not claim US equity, FX, or US-calendar coverage.

## Admission And Trace

- Both providers are opt-in through `ATHENA_FUND_PROVIDER`; `mock` remains the default.
- API startup runs `data.ValidateProvider` before listening. A missing credential, authorization failure, quota response, malformed field, or metadata failure prevents startup.
- Every normalized observation records source, provider, fetched time, market time, timezone, delay, license terms, confidence, schema version, and a raw-payload hash.
- Unsupported capability remains visible as an error. It is never replaced with a guessed value or mock observation.

## Verification

- `go test ./internal/data -run 'Test(AlphaVantageProvider|TushareProvider)' -count=1`
- `go test ./...`
- `yarn build` in `apps/web`
- `docker compose -f docker-compose.yml config --quiet`
- `docker compose -f docker-compose.yml -f docker-compose.dual.yml config --quiet`

Real provider admission is still pending a user-provided Alpha Vantage key and Tushare token. The test suite uses local HTTP fixtures and does not treat them as proof of live-provider authorization.

## Maintenance Skill

No dedicated maintenance skill was created. The provider contract remains small and is fully indexed by `internal/data/README.md`; create a dedicated skill when the future cross-market composition, Redis cache, provider-health monitoring, or credential lifecycle requires a repeatable operational workflow.
