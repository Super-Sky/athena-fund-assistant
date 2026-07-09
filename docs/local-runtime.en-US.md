# Local Runtime

## Goal

This document records the current local runtime path for the athena-fund-assistant MVP. The first runtime topology includes:

- React + TypeScript + Vite web
- Go API
- PostgreSQL
- Redis

When `DATABASE_URL` is configured, the API uses PostgreSQL for decision journals and review tasks. Without it, direct local runs and tests fall back to the in-memory store. Redis is already present in the Docker topology for later provider caching, rate limiting, and async refresh work.

The API runs provider validation before listening. The current mock provider must pass fund, equity, index, USD/CNY FX, and US market-calendar probes before the server starts.

## Run The API Directly

```bash
ATHENA_FUND_API_ADDR=:8081 go run ./cmd/api
```

Health check:

```bash
curl http://127.0.0.1:8081/healthz
curl http://127.0.0.1:8081/readyz
```

When the API runs directly without `DATABASE_URL`, it logs that the current store is non-durable. To connect directly to the local PostgreSQL service:

```bash
DATABASE_URL='postgres://athena_fund:athena_fund@127.0.0.1:5433/athena_fund?sslmode=disable' \
  ATHENA_FUND_API_ADDR=:8081 \
  go run ./cmd/api
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

- The API container connects to PostgreSQL through `DATABASE_URL` and applies an idempotent schema migration on startup. Startup fails when the database is unavailable.
- `/healthz` is process liveness; `/readyz` checks the journal store. Compose uses `/readyz` for API health.
- The API container reads `REDIS_URL`, but cache wiring is not implemented yet.
- Mock data must continue to be marked as temporary in UI / trace output.
- The current web app only calls the fund assistant API; dual-service Athena integration will be completed after Athena API wiring.
