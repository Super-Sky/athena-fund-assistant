# Local Runtime

## Goal

This document records the current local runtime path for the athena-fund-assistant MVP. The first runtime topology includes:

- React + TypeScript + Vite web
- Go API
- PostgreSQL
- Redis

The API still uses the in-memory journal store and mock provider. PostgreSQL and Redis are already present in the Docker topology so later persistence, caching, rate limiting, and async refresh work can attach without changing deployment shape.

The API runs provider validation before listening. The current mock provider must pass fund, equity, index, USD/CNY FX, and US market-calendar probes before the server starts.

## Run The API Directly

```bash
ATHENA_FUND_API_ADDR=:8081 go run ./cmd/api
```

Health check:

```bash
curl http://127.0.0.1:8081/healthz
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

- The API container reads `DATABASE_URL` and `REDIS_URL`, but the current code does not connect to these services yet.
- Mock data must continue to be marked as temporary in UI / trace output.
- The current web app only calls the fund assistant API; dual-service Athena integration will be completed after Athena API wiring.
