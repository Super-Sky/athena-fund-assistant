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
