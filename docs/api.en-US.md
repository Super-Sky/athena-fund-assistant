# API Contract

## Scope

This document records the local API currently implemented by the athena-fund-assistant MVP backend. The first API slice uses the Go standard-library HTTP server and does not import Athena internal Go packages.

This API belongs to the fund assistant business application layer, not Athena core. Athena should later call these business capabilities through APIs, SDKs, or tool contracts.

## General Conventions

- Default listen address: `:8081`
- Override with `ATHENA_FUND_API_ADDR`.
- Requests and responses are JSON.
- The API allows local web-development origins; current CORS support covers port-qualified `localhost` / `127.0.0.1` origins.
- Mock data must surface `mock_data_temporary=true` in trace output.
- Financial output must include multiple options, evidence, risks, invalidation conditions, and review timing.

## `GET /healthz`

Checks whether the process is alive without accessing the database.

Response example:

```json
{
  "status": "ok"
}
```

## `GET /readyz`

Checks whether the journal store is available. In Docker, this verifies the PostgreSQL connection and returns `503` when unavailable.

## `POST /api/analysis/fund`

Generates fund diagnosis and a three-option decision matrix from the investor profile, portfolio, and target instrument code.

Request fields:

- `instrument_code`: fund, ETF, or mock-provider-supported instrument code.
- `profile`: user risk profile.
- `portfolio`: user-entered holdings.

Current mock-provider support:

- `000001` / `CN-FUND-000001`
- `510300` / `CN-ETF-510300`
- `QQQ` / `US-ETF-QQQ`

Response fields:

- `profile`
- `portfolio`
- `fund_snapshot`
- `diagnosis`
- `decision_matrix`

`decision_matrix.trace` currently includes:

- `data_provider`
- `data_source`
- `data_fetched_at`
- `market_time`
- `timezone`
- `license_terms`
- `confidence`
- `rule_evaluations`
- `governance_checks`
- `mock_data_temporary`

## `POST /api/journals`

Stores the option selected by the user and creates a review task.

Request fields:

- `matrix`: the `decision_matrix` returned by `/api/analysis/fund`.
- `selected_option_id`: the option selected by the user.
- `user_notes`: user notes.

Response fields:

- `journal`
- `review`

## `GET /api/journals/{journalID}`

Reads a persisted decision journal. Returns `404` when it does not exist.

## `GET /api/reviews/{reviewID}`

Reads a persisted review task. Returns `404` when it does not exist.

## Current Boundaries

- When `DATABASE_URL` is configured, journals and review tasks are persisted in PostgreSQL. Without it, local development falls back to an in-memory store and logs that the state is non-durable.
- The current data provider is a mock provider and must not be treated as production market data.
- The current API does not implement user authentication, custody, automatic trading, or brokerage order placement.
- Redis caching, Athena agent-run integration, and real providers are later implementation items.
