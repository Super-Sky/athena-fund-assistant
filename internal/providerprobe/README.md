# Provider Probe Module

The provider probe module validates real data-source response shape, error behavior, and minimum fields before any real provider is connected to business workflows.

## File Index

- `report.go`
  - Defines the JSON validation report, governance admission metadata, and check shape.
- `http.go`
  - Provides a small bounded-timeout JSON HTTP client and field helpers.
- `alpha_vantage.go`
  - Validates Alpha Vantage ETF profile, quote, daily price, and FX daily response shapes.
- `tushare.go`
  - Validates Tushare Pro envelope structure for fund basics, fund NAV, and index daily probes.
- `alpha_vantage_test.go`
  - Verifies Alpha Vantage probe success and provider-message failure handling.
- `tushare_test.go`
  - Verifies Tushare token gating and response envelope validation.
