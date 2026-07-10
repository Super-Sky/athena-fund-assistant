# Local Runtime

## Goal

This document records the current local runtime path for the athena-fund-assistant MVP. The first runtime topology includes:

- React + TypeScript + Vite web
- Go API
- PostgreSQL
- Redis

The API still uses the in-memory journal store and defaults to the mock provider. The account dashboard uses PostgreSQL when `DATABASE_URL` exists and falls back to the in-memory demo store otherwise. Redis is already present in the Docker topology so later caching, rate limiting, and async refresh work can attach without changing deployment shape. The Athena client uses a local mock when `ATHENA_BASE_URL` is unset and calls the external Athena Agent Run API when configured.

The API runs provider validation before listening. The current mock provider must pass fund, equity, index, USD/CNY FX, and US market-calendar probes before the server starts. The CSV provider must pass China ETF / index, US ETF / equity / index, USD/CNY FX, and China plus US market-calendar probes.

## Run The API Directly

```bash
ATHENA_FUND_API_ADDR=:8081 go run ./cmd/api
```

Health check:

```bash
curl http://127.0.0.1:8081/healthz
```

Account dashboard check:

```bash
curl http://127.0.0.1:8081/api/accounts/demo-user/overview
```

Agent workspace skill check:

```bash
curl http://127.0.0.1:8081/api/conversations/skills
```

Athena remote tools catalog check:

```bash
curl 'http://127.0.0.1:8081/internal/tools/catalog?base_url=http://127.0.0.1:8081'
```

Connect real Athena:

```bash
ATHENA_BASE_URL=http://127.0.0.1:8080 ATHENA_AUTH_TOKEN=optional-token go run ./cmd/api
```

Use user-supplied CSV data as the fallback provider:

```bash
ATHENA_FUND_PROVIDER=csv \
ATHENA_FUND_CSV_PATH=examples/market-data-sample.csv \
ATHENA_FUND_API_ADDR=:8081 \
go run ./cmd/api
```

The CSV provider is a local MVP / demo fallback, not a licensed real-time market-data feed. Every CSV row must preserve `source`, `provider`, `fetched_at`, `market_time`, `timezone`, `delay`, `license_terms`, `confidence`, and `schema_version`. The sample file uses `user_supplied_csv_for_local_mvp_not_licensed_live_feed` to make the license boundary explicit.

## Dual-Service Smoke

This repository includes a local smoke script that starts Athena, the fund assistant, and a fake OpenAI-compatible model to verify the full local contract:

```bash
ATHENA_REPO=/Users/maxt/Desktop/maxt/Athena-remote-tools ./scripts/smoke_dual_service.sh
```

The script verifies:

- Athena `/healthz` and fund assistant `/healthz` are reachable.
- The fund assistant `/internal/tools/catalog` emits `account_overview` / `fund_market_snapshot` remote tool registrations.
- Athena `/api/control-plane/remote-tools/:name` accepts both read-only tools.
- The fake model triggers an `account_overview` tool call.
- Athena calls back into the fund assistant `/internal/tools/execute` through `remote_tool_execution.v1`.
- A fund conversation message gets an `athena_agent_run=ok` trace with `run_status=completed`, `tool_call_count=1`, and `output_present=true`.

This smoke does not require a real model API key. It validates the dual-service contract, tool registry, tool callback, and trace writeback. Real model providers should still be configured through Athena model management.

PostgreSQL store integration test:

```bash
ATHENA_FUND_PG_TEST_DSN='postgres://athena_fund:athena_fund@127.0.0.1:5433/athena_fund?sslmode=disable' \
  go test ./internal/account -run TestPostgresStoreOverviewAndReplaceHoldings -count=1
```

## Run The Web App Directly

```bash
cd apps/web
yarn install
yarn dev
```

Vite listens on `http://127.0.0.1:5173` by default and proxies `/api` plus `/healthz` to `http://127.0.0.1:8081`.

## Docker Compose

```bash
cp .env.example .env
docker compose up --build
```

Default ports:

- Web: `5173`
- API: `8081`
- PostgreSQL: `5433`
- Redis: `6380`

## Dual-Service Docker Compose

Use the overlay to start Athena, the fund assistant, PostgreSQL, Redis, the web app, and a fake OpenAI-compatible model in one Docker Compose project:

```bash
ATHENA_REPO=../Athena-remote-tools \
docker compose -f docker-compose.yml -f docker-compose.dual.yml up --build
```

Default ports:

- Athena API: `8080`
- fund assistant Web: `5173`
- fund assistant API: `8081`
- fake OpenAI-compatible model: `18083`

The dual-service overlay points the fund assistant API at `ATHENA_BASE_URL=http://athena-api:8080` and enables the CSV provider by default with `ATHENA_FUND_PROVIDER=csv` plus `ATHENA_FUND_CSV_PATH=/app/examples/market-data-sample.csv`. CSV data remains a local MVP / demo fallback, not a licensed real-time market-data feed.

End-to-end Docker smoke:

```bash
ATHENA_REPO=../Athena-remote-tools ./scripts/smoke_dual_docker.sh
```

The script builds and starts the dual-service Docker topology, registers the fake model and fund remote tools, then verifies Athena Agent Run, the remote tool callback, fund conversation trace writeback, and CSV provider decision trace. The first Athena image build can be slow; later runs reuse the Docker cache.

## Current Boundaries

- The API container reads `DATABASE_URL` for account dashboard persistence. `REDIS_URL` remains reserved for later caching and async work.
- The API reads `ATHENA_FUND_UPLOAD_DIR` as the attachment upload directory. If unset, the system temp directory is used.
- The API reads `ATHENA_FUND_PROVIDER`; unset or `mock` uses `mock_provider`, while `csv` reads `ATHENA_FUND_CSV_PATH`.
- The API reads `ATHENA_BASE_URL` and optional `ATHENA_AUTH_TOKEN`; when unset, it uses the mock Athena client for single-service demos.
- The dual-service Docker overlay also reads `ATHENA_REPO`, `ATHENA_DUAL_API_PORT`, `ATHENA_FAKE_MODEL_PORT`, `ATHENA_FUND_PROVIDER`, and `ATHENA_FUND_CSV_PATH`.
- Mock / CSV data must continue to be marked as temporary or user-supplied in UI / trace output.
- The current web app still calls only the fund assistant API. The fund assistant API starts an Agent Run through the Athena client after user messages and exposes read-only remote business tools through `/internal/tools/execute` for Athena callbacks.
