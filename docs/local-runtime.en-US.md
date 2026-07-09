# Local Runtime

## Goal

This document records the current local runtime path for the athena-fund-assistant MVP. The first runtime topology includes:

- React + TypeScript + Vite web
- Go API
- PostgreSQL
- Redis

The API still uses the in-memory journal store and mock provider. The account dashboard uses PostgreSQL when `DATABASE_URL` exists and falls back to the in-memory demo store otherwise. Redis is already present in the Docker topology so later caching, rate limiting, and async refresh work can attach without changing deployment shape.

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
- Mock data must continue to be marked as temporary in UI / trace output.
- The current web app only calls the fund assistant API; dual-service Athena integration will be completed after Athena API wiring.
