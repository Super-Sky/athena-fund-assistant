# Athena MVP Gap List

## Background

Athena already has a runtime foundation layer: RuntimeContract, TaskTypeRegistration, HookBinding, System Truth lifecycle, projection boundary, runtime trace / usage / checkpoint readout, tool governance, Validation MCP, and Control Plane validation surfaces.

The fund assistant MVP needs stable runtime APIs that a business application can call. The current Athena surface is stronger on validation and control-plane readout than on external app-facing agent execution.

## Existing Capabilities

- Runtime persistence objects: run, step, lifecycle, trace, usage, projection, and checkpoint safe metadata.
- RuntimeContract foundation and registered task type validators.
- Tool governance policy / decision APIs.
- Validation MCP deterministic tool invocation.
- Context asset injection and direct respond rich delivery read model.
- System Validation page for runtime persistence readout.
- OpenAPI / Swagger for existing control-plane endpoints.

## Required MVP Additions

### 1. Goal-Driven Agent Run API

Athena needs an app-facing agent run API instead of relying only on chat/respond or control-plane validation runs.

Recommended endpoints:

- `POST /api/agent/runs`
- `GET /api/agent/runs/:runID`
- `POST /api/agent/runs/:runID/resume`
- `POST /api/agent/runs/:runID/cancel`
- `GET /api/agent/runs/:runID/trace`

Minimum request semantics:

- `goal`
- `success_criteria`
- `constraints`
- `budget`
- `context_assets`
- `tools`
- `tool_choice`
- `memory_scope`
- `governance_policy_refs`

### 2. OpenAI-Compatible Tools / Tool Calls

Athena needs external compatibility with OpenAI-style:

- `tools`
- `tool_choice`
- assistant-message `tool_calls`
- tool-result messages
- streaming tool-call delta

Internally, Athena should keep a canonical tool contract so it is not locked to one provider protocol.

### 3. Business Tool Registry / Execution Surface

The fund assistant needs to register its own business tools:

- fund snapshot tool
- market snapshot tool
- portfolio context tool
- decision journal tool
- review task tool

Athena needs:

- tool schema registry
- tool execution request/response contract
- tool timeout / retry / error contract
- tool trace
- governance pre-check
- business app callback or remote tool endpoint support

### 4. Memory / Context API

The fund assistant needs to inject investor profile, portfolio summaries, decision journals, and review results as governed context rather than storing fund business tables in Athena core.

Recommended endpoints:

- `POST /api/memory/query`
- `POST /api/memory/write`
- `POST /api/context-assets/resolve`
- `POST /api/context-assets/assemble`

Minimum capabilities:

- query scoped memory
- write decision summaries
- context compression
- trace memory reads/writes
- preserve app ownership

### 5. Trace Timeline API

Athena already has runtime trace readout, but the fund assistant needs an app-readable timeline.

The timeline should show:

- agent goal
- plan / loop step
- model calls
- tool calls
- data provider calls
- memory reads/writes
- context assembly
- governance decisions
- generated decision matrix
- final report delivery

### 6. Built-In Basic Tools

The MVP needs at least:

- HTTP fetch tool
- web/search tool or pluggable search provider
- calculator tool
- time / market-calendar helper
- file/CSV import helper
- JSON/schema validation helper

Financial business tools are registered by the fund assistant and must not enter Athena core.

### 7. Docker Integration Contract

Athena needs Docker Compose-friendly configuration for:

- PostgreSQL
- Redis
- Athena API
- Athena web control plane
- healthcheck
- env example

The fund assistant configures Athena base URL, auth token, PostgreSQL, Redis, and provider keys through environment variables.

## Content That Must Stay Outside Athena Core

- Fund / ETF business tables.
- User holding business tables.
- Fund NAV, drawdown, return, fund-manager business fields.
- Conservative / balanced / aggressive business templates.
- Financial data-provider business accounts and authorization config.
- Investment decision journal business tables.

These belong to `athena-fund-assistant`.

## Recommended Athena Issues

1. Agent Run API foundation.
2. OpenAI-compatible tool call contract.
3. Remote/business tool registry.
4. Memory/context external API.
5. Agent trace timeline read API.
6. Built-in basic tools pack.
7. Docker Compose runtime profile with PostgreSQL and Redis.

