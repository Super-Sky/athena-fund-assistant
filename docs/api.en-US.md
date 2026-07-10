# API Contract

## Scope

This document records the local API currently implemented by the athena-fund-assistant MVP backend. The first API slice uses the Go standard-library HTTP server and does not import Athena internal Go packages.

This API belongs to the fund assistant business application layer, not Athena core. Athena should later call these business capabilities through APIs, SDKs, or tool contracts.

## General Conventions

- Default listen address: `:8081`
- Override with `ATHENA_FUND_API_ADDR`.
- Requests and responses are JSON.
- The API allows local web-development origins; current CORS support covers port-qualified `localhost` / `127.0.0.1` origins.
- When `ATHENA_BASE_URL` is unset, the service uses a local mock Athena client; when configured, it calls external Athena through `POST /api/agent/runs`.
- `ATHENA_AUTH_TOKEN` is optional and is sent to Athena as a Bearer token.
- Mock data must surface `mock_data_temporary=true` in trace output. Mock / CSV fallback data must surface `temporary_data=true` and an explicit `data_boundary` in decision traces.
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

The response returns the updated `ConversationDetail`. The service saves the message, starts one generic Agent Run through the Athena client, and writes run status, run_id, trace_available, and stop_reason back to the conversation trace. When `ATHENA_BASE_URL` is unset, this call is handled by the mock client for local demos.

The Agent Run request maps business semantics into generic Athena input:

- `goal`: the user message.
- `context_assets`: conversation ID, skill ID, and attachment IDs; attachments remain metadata-only.
- `tools` / `enabled_tools`: OpenAI-compatible function tools, currently `account_overview` and `fund_market_snapshot`.
- `governance_refs`: no automatic trading, no guaranteed-return claims, and data-source metadata required.
- `constraints`: no automatic trading, no brokerage order placement, risk and invalidation required, and no single absolute path conclusion.

The current `athena_agent_run` trace metadata includes:

- `run_id`
- `run_status`
- `trace_available`
- `stop_reason`
- `tool_call_count`
- `output_present`

## `GET /internal/tools/catalog`

Returns suggested tool registrations exposed by the fund assistant for Athena's remote tool registry.

Query parameters:

- `base_url`: optional. When set, the response generates a full `endpoint`, for example `http://127.0.0.1:8081/internal/tools/execute`; when omitted, the response returns the relative path `/internal/tools/execute`.

Response fields:

- `contract_version`: currently `remote_tool_execution.v1`.
- `app_id`: currently `athena-fund-assistant`.
- `items`: remote tools that can be registered in Athena.

Current read-only tools:

- `account_overview`: reads the user's account overview, holdings, recent operations, and performance trend.
- `fund_market_snapshot`: reads a fund / ETF snapshot while preserving source, provider, fetched_at, market_time, timezone, delay, license, confidence, and schema_version.

All current tools use `side_effect_level=none`. They do not trade, connect to brokerage order placement, or move money.

## `POST /internal/tools/execute`

Executes Athena `remote_tool_execution.v1` callbacks. This endpoint is for the Athena remote adapter, not for direct frontend user calls.

Request fields:

- `contract_version`
- `request_id`
- `tool_call_id`
- `registration_id`
- `app_id`
- `tool_name`
- `arguments`
- `attempt`
- `metadata`

A successful response returns the same `request_id` and `tool_call_id` plus:

- `status=ok`
- `content`: JSON string.

Error responses use the same envelope and include:

- `status=error`
- `error.code`
- `error.message`
- `error.retryable`

Supported arguments:

- `account_overview`: `{"user_id":"demo-user"}`; `user_id` defaults to `demo-user` when omitted.
- `fund_market_snapshot`: `{"instrument_code":"QQQ"}`; `instrument_code` is required.

Boundaries:

- This endpoint exposes fund-assistant business tool implementations only and does not import Athena internal Go packages.
- Returned content still must be interpreted through its metadata to distinguish real and mock data.
- Unknown tools such as order-placement `place_order` return `unknown_tool` and never execute a money-moving action.

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
- `data_boundary`
- `temporary_data`
- `rule_evaluations`
- `governance_checks`
- `mock_data_temporary`

## `GET /api/users/{user_id}/knowledge`

Reads durable user preferences, `agent.md`, the strategy knowledge base, revision history, and audit events.

Response fields:

- `preference`: user risk preference, communication preference, default strategy level, preferred / blocked assets, review frequency, `agent_md`, active revision, and governance fields.
- `items`: strategy knowledge items with title, category, content, tags, status, active revision, source, author, confidence, schema_version, and governance decision.
- `revisions`: immutable preference or knowledge item revisions.
- `audit`: audit events for draft saves, activations, rollbacks, and related actions.

## `POST /api/users/{user_id}/preferences/drafts`

Saves a user preference / `agent.md` draft without activating it.

Request fields:

- `risk_preference`
- `communication_style`
- `default_strategy_level`
- `preferred_assets`
- `blocked_assets`
- `review_frequency_days`
- `agent_md`
- `source`
- `author`
- `confidence`
- `summary`

## `POST /api/users/{user_id}/preferences/activate`

Explicitly activates one preference revision.

Request fields:

- `revision_id`

## `POST /api/users/{user_id}/knowledge/drafts`

Saves one strategy knowledge draft without activating it.

Request fields:

- `item_id`: optional; when omitted the API creates a new item.
- `title`
- `category`
- `content`
- `tags`
- `source`
- `author`
- `confidence`
- `summary`

## `POST /api/users/{user_id}/knowledge/{item_id}/activate`

Explicitly activates one revision for a knowledge item.

Request fields:

- `revision_id`

## `POST /api/users/{user_id}/knowledge/{item_id}/rollback`

Rolls back a knowledge item to a historical revision and records an audit event.

Request fields:

- `revision_id`

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
- The preference / knowledge store is in-memory and returns to the demo seed on service restart. PostgreSQL persistence and permissioned approval are follow-up work.
- The account overview uses PostgreSQL persistence when `DATABASE_URL` exists; otherwise it uses the in-memory demo store.
- The current data provider is a mock provider and must not be treated as production market data.
- The current API does not implement user authentication, custody, automatic trading, or brokerage order placement.
- Redis, Athena agent-run integration, attachment parsers/OCR/PDF/CSV parsing, persistent journal/review account links, and real providers are later implementation items.
