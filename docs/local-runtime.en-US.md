# Local Runtime

## Goal

This document records the current local runtime path for the athena-fund-assistant MVP. The first runtime topology includes:

- React + TypeScript + Vite web
- Go API
- PostgreSQL
- Redis

The API still uses the in-memory journal store and mock provider. The account dashboard uses PostgreSQL when `DATABASE_URL` exists and falls back to the in-memory demo store otherwise. Redis is already present in the Docker topology so later caching, rate limiting, and async refresh work can attach without changing deployment shape. The Athena client uses a local mock when `ATHENA_BASE_URL` is unset and calls the external Athena Agent Run API when configured.

The API runs provider validation before listening. The current mock provider must pass fund, equity, index, USD/CNY FX, and US market-calendar probes before the server starts.

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

## Current Boundaries

- The API container reads `DATABASE_URL` for account dashboard persistence. `REDIS_URL` remains reserved for later caching and async work.
- The API reads `ATHENA_FUND_UPLOAD_DIR` as the attachment upload directory. If unset, the system temp directory is used.
- The API reads `ATHENA_BASE_URL` and optional `ATHENA_AUTH_TOKEN`; when unset, it uses the mock Athena client for single-service demos.
- Mock data must continue to be marked as temporary in UI / trace output.
- The current web app still calls only the fund assistant API. The fund assistant API starts an Agent Run through the Athena client after user messages and exposes read-only remote business tools through `/internal/tools/execute` for Athena callbacks.
