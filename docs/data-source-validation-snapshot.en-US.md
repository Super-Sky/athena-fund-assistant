# Data Source Validation Snapshot

## Scope

This snapshot records the 2026-07-10 validation status for real data sources in the fund assistant MVP. It is not legal advice; it is an engineering decision record for which sources can enter validation probes, which sources are experimental only, and which sources can enter the default business workflow.

## Conclusion

- First China public fund / ETF candidate: `tushare_provider`, but users must provide `TUSHARE_TOKEN`, and endpoint permissions, frequency, display, and commercial boundaries must be checked per API.
- First US ETF / equity / index / FX candidate: `alpha_vantage_provider`, but users must provide `ALPHA_VANTAGE_API_KEY`, and free quotas, caching, and display terms must be respected.
- FMP, Tiingo, and Nasdaq Data Link are better suited as optional user-key providers, not default free providers.
- AKShare / Eastmoney / Tiantian Fund should remain experimental and local fallback only, not the default commercial SaaS provider.
- Stooq historical CSV can remain a fallback candidate, but local validation hit a TLS connection failure; it must be revalidated in the target deployment network before it enters probes.
- The default business provider must remain `mock_provider`, or the explicitly enabled local `csv_provider` fallback; both must keep temporary or user-supplied data markers visible.

## Local Probe Results

Commands:

```bash
go run ./cmd/providerprobe --provider alpha_vantage
go run ./cmd/providerprobe --provider tushare
```

Results:

- `alpha_vantage`: failed. Alpha Vantage returned the official demo-key limitation message and asked for a free API key; the demo key does not prove live schema availability, so configure a real API key and rerun.
- `tushare`: failed. `TUSHARE_TOKEN` was not set, and the probe correctly returned `TUSHARE_TOKEN is required for live validation`.

These failures are not product blockers; they are evidence that live providers have not passed admission and must not enter the default decision workflow.

## MVP Provider Decision

### Default Workflow

- Keep `mock_provider`.
- `csv_provider` is now wired into `internal/data.Provider` as a no-network fallback and is enabled explicitly through `ATHENA_FUND_PROVIDER=csv` plus `ATHENA_FUND_CSV_PATH`.
- The default workflow must keep showing `license_terms`, `provider`, `source`, `confidence`, and temporary / user-supplied data markers in UI / API trace.

### CSV Fallback

- Coverage: normalized CSV rows can cover China ETF / index, US ETF / equity / index, USD/CNY FX, and China plus US market calendars.
- Sample: `examples/market-data-sample.csv`.
- Validation: before the API starts, `internal/data.ValidateProvider` checks `510300`, `QQQ`, `AAPL`, `000300`, `NDX`, `USD/CNY`, and `CN` / `US` market calendars.
- Boundary: CSV data must be treated as user-supplied or local demo data, not as a licensed real-time market-data feed. `license_terms` must not be empty; the sample uses `user_supplied_csv_for_local_mvp_not_licensed_live_feed`.

### User-Key Workflow

- `alpha_vantage_provider`
  - Coverage: US ETF profile, quote, daily price, and USD/CNY FX.
  - Admission: `cmd/providerprobe --provider alpha_vantage` passes all checks.
  - Cache: Redis caching is required to reduce free-quota pressure.
- `tushare_provider`
  - Coverage: China public fund basics, fund NAV, and A-share index daily data.
  - Admission: `cmd/providerprobe --provider tushare` passes all checks.
  - Limit: enabled only when the user token and permissions pass validation.

### Experimental Workflow

- `akshare_experimental_provider`
- `stooq_csv_provider`
- `fmp_provider`
- `tiingo_provider`
- `nasdaq_data_link_provider`

Experimental providers must not masquerade as production data. Their trace output must include experimental or terms-required markers.

## Admission Gate

Before a real provider can feed fund diagnosis, decision matrices, Athena remote tools, or default UI display, it must satisfy:

- provider probe `passed=true`.
- Every data item preserves `source`, `provider`, `fetched_at`, `market_time`, `timezone`, `delay`, `license_terms`, `confidence`, `schema_version`, and `raw_payload_hash`.
- US market data uses `America/New_York`, FX uses the source-defined timezone or `UTC`, and China market data uses `Asia/Shanghai`.
- Rate limit, timeout, missing fields, insufficient permission, non-trading day, and delayed-data behavior have explicit failure policy.
- No real provider output is used for automatic trading.
