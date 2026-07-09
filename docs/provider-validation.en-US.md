# Provider Validation-First Rule

## Goal

Information tools, market-data providers, web parsers, and third-party API adapters must be validated before business coding. The team must first confirm source provenance, field structure, timezone, update frequency, licensing boundary, and error behavior before a provider feeds fund diagnosis, decision matrices, or Athena tool calls.

This rule applies to:

- China public fund / ETF providers
- US equity / ETF / index providers
- FX and market-calendar providers
- News, announcement, macro-data, and web-search tools
- Future read-only user-account sync adapters

## Entry Gate

Before a real provider enters a business workflow, it must complete:

1. Documentation check: record official docs, field meanings, free quota, and display / redistribution limits.
2. Sample check: keep at least one successful sample, one missing/error sample, and one non-trading-day or delayed sample.
3. Structure check: map raw payloads into normalized structures and confirm required metadata is not empty.
4. Timezone check: US data must use `America/New_York`, China market data defaults to `Asia/Shanghai`, and FX data uses `UTC` or the source-defined timezone.
5. License check: `license_terms` must be written into trace output; unclear sources must not masquerade as production data.
6. Automated validation: pass `internal/data.ValidateProvider` or an equivalent provider-validation report.
7. Failure strategy: define timeout, rate-limit, missing-field, authorization, and non-trading-day behavior.

## Coding Order

Recommended order:

1. Write provider validation probes.
2. Run probes with a real API key or public sample.
3. Freeze the normalized schema and metadata mapping.
4. Add provider unit tests or contract tests.
5. Then connect the provider to diagnosis, decision matrix, UI, or Athena tools.

Forbidden order:

1. Write business judgment logic first.
2. Guess API fields temporarily.
3. Add metadata / license / timezone handling after release.

## Current Code Anchors

- `internal/data/provider.go`
  - Provider capability boundary.
- `internal/data/validation.go`
  - Provider validation report and probes.
- `internal/data/mock_provider.go`
  - Current mock provider, explicitly marked as temporary data.
- `cmd/providerprobe`
  - Runs validation probes against real data sources and emits JSON validation reports.
- `internal/providerprobe`
  - Validation-only probes for real data sources such as Alpha Vantage and Tushare; these do not connect business providers.

## Validation Commands

```bash
go run ./cmd/providerprobe --provider alpha_vantage
ALPHA_VANTAGE_API_KEY=... go run ./cmd/providerprobe --provider alpha_vantage
TUSHARE_TOKEN=... go run ./cmd/providerprobe --provider tushare
```

The command emits a JSON report. It exits non-zero when any required probe fails.

Current validation snapshots:

- `docs/data-source-validation-snapshot.zh-CN.md`
- `docs/data-source-validation-snapshot.en-US.md`

## Acceptance Criteria

- Provider validation report must have `passed=true`.
- Every business-used data point must preserve `source`, `fetched_at`, `market_time`, `timezone`, `delay`, `provider`, `license_terms`, `confidence`, and `schema_version`.
- Real providers that have not passed validation may only be used as experimental or manual research paths; they must not enter the default user decision workflow.
