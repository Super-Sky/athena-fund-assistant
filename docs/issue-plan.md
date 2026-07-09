# Initial Issue Plan

## Issue 1: Repository Scaffold And Product Boundary

Define repository structure, product boundary, MVP scope, and Athena integration assumptions.

Acceptance:

- README and docs index exist.
- Product boundary is explicit.
- MVP workflows are defined.
- Initial issue plan is written.

## Issue 2: Domain Model Foundation

Implement the first domain models:

- InvestorProfile
- Portfolio
- PortfolioHolding
- FundInstrument
- FundSnapshot
- DecisionMatrix
- DecisionOption
- DecisionJournal
- ReviewTask

Acceptance:

- Models are versioned.
- Validation rules exist.
- Unit tests cover basic invariants.

## Issue 3: Mock Data Provider

Implement mock fund / market data providers.

Acceptance:

- Fund snapshot can be loaded from mock data.
- Source and freshness metadata are preserved.
- Provider interface can support future real APIs.

## Issue 4: Decision Matrix Engine

Generate conservative / balanced / aggressive options.

Acceptance:

- Options include action, percentage/range, evidence, risk, invalidation, review time, and portfolio impact.
- Aggressive users can receive a compressed primary/fallback output.
- Percentages are traceable to profile, rule, template, or simulation basis.

## Issue 5: Decision Journal And Review Task

Persist user decisions and generate review tasks.

Acceptance:

- A selected option can create a journal entry.
- Evidence snapshot is preserved.
- Review task can compare original thesis with later data.

## Issue 6: Athena Client Contract

Create the app-side Athena API client facade.

Acceptance:

- Agent run request shape is defined.
- Tool registration / invocation shape is defined.
- Trace read expectations are documented in code.
- Mock Athena client supports local development.

## Issue 7: Fund Assistant UI MVP

Build the first user-facing workflow:

- profile form
- holding input
- fund diagnosis
- decision matrix cards
- journal creation

Acceptance:

- One mock fund can be evaluated end to end.
- User can select a decision option.
- Journal entry is visible.

## Issue 8: Financial Governance Gate

Implement product-specific output checks.

Acceptance:

- Guaranteed return language is blocked.
- Single absolute conclusion is blocked.
- Missing source / freshness is flagged.
- Missing risk / invalidation is flagged.

## Issue 9: Convert Planning Docs To zh-CN And en-US Pairs

Convert durable planning documents into paired Chinese and English versions.

Acceptance:

- Each durable document has `*.zh-CN.md` and `*.en-US.md` versions.
- `docs/README.md` links both versions.
- Product boundaries, governance rules, data assumptions, and MVP acceptance criteria stay aligned.

## Issue 10: China Fund And ETF Data Source Research

Confirm the first legal and technically viable data path for China public funds, ETFs, LOFs, and major indices.

Acceptance:

- At least one China live provider candidate is selected for MVP implementation.
- Tushare, AKShare/Eastmoney, exchange, and AMAC paths are documented with risks.
- Experimental providers are clearly marked as non-production defaults.

## Issue 11: US Equity, ETF, And Index Data Source Research

Confirm the first free or low-cost legal data path for US equities, ETFs, major US indices, USD/CNY FX rates, and US market-calendar handling.

Acceptance:

- At least one US live provider candidate is selected for MVP implementation.
- Alpha Vantage, FMP, Tiingo, Nasdaq Data Link, Stooq, and Yahoo/yfinance paths are documented with risks.
- US equities, ETFs, indices, FX, timezone, delay, and non-trading-day handling are part of the provider contract.

## Issue 12: Docker Compose MVP Runtime

Create a Docker / Docker Compose runtime profile for the fund assistant MVP.

Acceptance:

- Go API, React UI, PostgreSQL, and Redis can run locally.
- Athena base URL and auth token are configured by environment variables.
- Startup docs explain the local path.

## Issue 15: Account And Portfolio Performance Dashboard

Turn the product into an account-based assistant rather than a one-off analysis form.

Acceptance:

- User, account, holding snapshot, performance metric, and operation record domain models exist.
- API can return account overview, performance trend, and recent operation summary.
- UI homepage shows account performance and holdings.
- PostgreSQL persistence is added for account, holdings, operations, and trend points; journal/review account links remain follow-up work.
- Brokerage sync remains read-only future direction and no order interface is added.

## Issue 16: Agent Conversation Workspace With Skill And File Upload

Build the daily Agent workspace for natural-language requests, skill selection, and attachment upload.

Acceptance:

- UI has conversation workspace, skill selector, upload entry, and trace timeline.
- Backend has conversation/session/message/attachment metadata API.
- Attachments are clearly marked pending/unsupported until parsed.
- Athena remote tools are used for fund business actions.

## Issue 17: User Preference agent.md And Fund Strategy Knowledge Base

Create long-lived preference and strategy assets for account-aware decisions.

Acceptance:

- User preference / agent profile and strategy knowledge item models exist.
- Knowledge updates are versioned, traceable, governed, and rollbackable.
- Decision percentages can cite strategy templates, preferences, or data evidence.

## Athena Dependency Issues

The fund assistant depends on these Athena runtime foundation tasks:

- `Super-Sky/Athena#7`: Agent Run API foundation for business applications.
- `Super-Sky/Athena#8`: OpenAI-compatible tools and tool_calls contract.
- `Super-Sky/Athena#9`: Remote business tool registry and execution surface.
- `Super-Sky/Athena#10`: External memory and context asset APIs for business apps.
- `Super-Sky/Athena#11`: Agent trace timeline API and admin readout.
- `Super-Sky/Athena#12`: Docker Compose runtime profile with PostgreSQL and Redis.
