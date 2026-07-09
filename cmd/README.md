# Command Entrypoints

This directory contains runnable process entrypoints.

## File Index

- `api/main.go`
  - Starts the local fund assistant API server on `ATHENA_FUND_API_ADDR` or `:8081`.
  - Wires the mock data provider, account store, conversation store, Athena mock/HTTP client, deterministic decision engine, in-memory journal store, and HTTP routes.
- `providerprobe/main.go`
  - Runs validation-first probes against real data sources and emits a JSON report.

The React web entrypoint lives under `../apps/web/`.
