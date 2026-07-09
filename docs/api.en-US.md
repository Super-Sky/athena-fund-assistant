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

Checks whether the service is alive.

Response example:

```json
{
  "status": "ok"
}
```

## `GET /api/accounts/{user_id}/overview`

Reads the user's account performance dashboard.

Current local demo user:

- `demo-user`

Response fields:

- `account`: local user account identity, display name, base currency, and auth mode.
- `holdings`: account holding snapshots with market, currency, units, cost, current price, `fx_to_base`, base-currency market value, unrealized return, allocation, and `data_authorization`.
- `total_market_value`
- `total_cost_value`
- `total_pnl`
- `total_pnl_pct`
- `recent_operation_pnl`
- `performance_trend`
- `recent_operations`
- `trace`

The current `trace` includes:

- `provider`
- `source`
- `fetched_at`
- `market_time`
- `timezone`
- `license_terms`
- `confidence`
- `schema_version`
- `mock_data_temporary`
- `read_only_sync_available`
- `warnings`

Without `DATABASE_URL`, local runs use the in-memory demo store. Docker / `DATABASE_URL` environments use the PostgreSQL store. Current market and return inputs are still demo/mock data and must not be represented as a real brokerage account or real return record.

## `POST /api/accounts/{user_id}/holdings`

Replaces the user's manually entered account holdings and recalculates the account overview.

Request fields:

- `holdings`: `AccountHoldingSnapshot[]`

Each holding must include:

- `instrument_code`
- `market`
- `currency`
- `units`
- `cost_basis`
- `current_price`
- `fx_to_base`
- `data_authorization`
- `metadata`

This endpoint only records manual data and local calculations. It does not trade, place brokerage orders, or connect to a brokerage order interface.

## `GET /api/conversations/skills`

Returns selectable Agent workspace skills.

Response fields:

- `items`: skill list. Each skill includes `id`, `name`, `description`, `tool_names`, and `enabled`.

Current built-in skills:

- `fund_research`
- `portfolio_review`
- `document_intake`

## `POST /api/conversations`

Creates a conversation session.

Request fields:

- `user_id`
- `skill_id`
- `title`

The response is `ConversationDetail` with `session`, `messages`, `attachments`, and `trace`.

## `GET /api/conversations/{conversation_id}`

Reads conversation detail, messages, attachment metadata, and trace timeline.

## `POST /api/conversations/{conversation_id}/attachments`

Uploads a file and returns attachment metadata.

Request type: `multipart/form-data`

Fields:

- `file`
- `user_id`

Upload boundaries:

- Per-file limit is `10 MiB`.
- Default retention window is `7 days`.
- `ATHENA_FUND_UPLOAD_DIR` configures the upload directory; if unset, the system temp directory is used.
- The current slice only generates metadata, SHA256, and `pending_parse` / `unsupported` status. It does not parse attachment content.
- Unparsed attachments must not be treated as confirmed facts, statements, or strategy knowledge.

## `POST /api/conversations/{conversation_id}/messages`

Appends one workspace message.

Request fields:

- `role`
- `content`
- `skill_id`
- `attachment_ids`

The response returns the updated `ConversationDetail`. The current slice writes local trace events; the real Athena agent run is still a pending contract and is not called in this slice.

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

## Current Boundaries

- The journal store is in-memory and is lost on service restart.
- The account overview uses PostgreSQL persistence when `DATABASE_URL` exists; otherwise it uses the in-memory demo store.
- The current data provider is a mock provider and must not be treated as production market data.
- The current API does not implement user authentication, custody, automatic trading, or brokerage order placement.
- Redis, Athena agent-run integration, attachment parsers/OCR/PDF/CSV parsing, persistent journal/review account links, and real providers are later implementation items.
