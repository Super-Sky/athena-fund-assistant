# Data Module

The data module owns provider interfaces and data-source adapters.

## File Index

- `provider.go`
  - Defines the provider boundary for fund snapshots, US equities, indices, FX rates, and market calendars.
- `validation.go`
  - Runs validation-first probes before a provider is trusted by diagnosis or decision workflows.
- `mock_provider.go`
  - Provides deterministic mock data with explicit temporary metadata for local MVP development.
- `mock_provider_test.go`
  - Verifies mock coverage for US equity, index, FX, and market-calendar support data.
- `validation_test.go`
  - Verifies provider validation success and failure paths.
- `live_provider.go`
  - Defines the credential-backed provider extension and explicit unsupported-capability error.
- `live_http.go`
  - Centralizes bounded provider HTTP access and raw-response hashing.
- `alpha_vantage_provider.go`
  - Normalizes user-key-backed US ETF, equity, ETF-proxy benchmark, and FX responses.
- `alpha_vantage_provider_test.go`
  - Verifies normalized Alpha Vantage contracts against a local HTTP fixture.
- `tushare_provider.go`
  - Normalizes user-token-backed China public-fund NAV, index, and SSE calendar responses.
- `tushare_provider_test.go`
  - Verifies normalized Tushare contracts against a local HTTP fixture.
