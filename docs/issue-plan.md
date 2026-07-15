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
- `docs/data-source-validation-snapshot.*.md` records current probe evidence and states that Tushare cannot enter the default workflow until a user token passes validation.

## Issue 11: US Equity, ETF, And Index Data Source Research

Confirm the first free or low-cost legal data path for US equities, ETFs, major US indices, USD/CNY FX rates, and US market-calendar handling.

Acceptance:

- At least one US live provider candidate is selected for MVP implementation.
- Alpha Vantage, FMP, Tiingo, Nasdaq Data Link, Stooq, and Yahoo/yfinance paths are documented with risks.
- US equities, ETFs, indices, FX, timezone, delay, and non-trading-day handling are part of the provider contract.
- `docs/data-source-validation-snapshot.*.md` records current probe evidence and states that Alpha Vantage cannot enter the default workflow until a real API key passes validation.

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
- Athena run is represented as a pending contract in this slice; fund-owned read-only remote tools expose `remote_tool_execution.v1` callbacks for Athena registry integration.

## Issue 17: User Preference agent.md And Fund Strategy Knowledge Base

Create long-lived preference and strategy assets for account-aware decisions.

Acceptance:

- User preference / agent profile and strategy knowledge item models exist.
- Knowledge updates are versioned, traceable, governed, and rollbackable.
- Decision percentages can cite strategy templates, preferences, or data evidence.

## Issue 30: Read-Only Account Authorization And Consent Audit

Create the account-access boundary required before user-authorized portfolio or brokerage data can be exposed to an Athena remote tool.

Acceptance:

- A user identity, session, read-only data authorization, tool scope, expiration, and revocation are modeled and audited.
- A revoked or expired authorization makes the associated fund remote tool fail with a stable governed denial.
- No brokerage order, trade placement, or write permission is introduced.
- Decision-journal trace can identify the authorization and data-source metadata used for the analysis.

Current evidence: draft PR `#35` implements local sessions, revisioned read-only consent, revocation, scope enforcement, and redacted authorization audit. Production remote callbacks still depend on Athena `#24` service identity.

## Issue 31: Financial Agent Evaluation And Release Gate

Add repeatable quality and safety evaluation for fund-analysis workflows.

Acceptance:

- `#31A` runs deterministic fixed cases locally and in CI; `#31B` adds cross-service trace and optional model-assisted signals after Athena trace contracts stabilize.
- Cases cover stale/missing data, provider failure, unsupported source attribution, guaranteed-return language, single-path conclusions, missing risk/invalidation, unsupported percentages, and unauthorized account access.
- A failed mandatory case blocks the release path.
- The evaluation suite consumes Athena trace-safe outputs and fund decision evidence without requiring Athena to own fund business data.

Current evidence: draft PR `#36` implements `#31A`; all three GitHub CI gates pass. `#31B` remains open.

## Issue 37: Governed Attachment Ingestion And Evidence Extraction

Turn attachment metadata from Issue 16 into governed evidence that can safely enter fund analysis.

Acceptance:

- Local development storage and a replaceable S3-compatible adapter implement the same attachment contract.
- CSV and PDF text parsing preserve file hash, parser/version, page or row citations, authorization, retention, and failure status.
- OCR remains a governed adapter with explicit unsupported/degraded behavior when unavailable.
- Untrusted parsing uses an isolated boundary and fails closed without fabricating evidence.
- Athena receives only authorized context/evidence references and trace-safe summaries, never raw sensitive files in trace.

## Athena Dependency Issues

The fund assistant depends on these Athena runtime foundation tasks:

- `Super-Sky/Athena#7`: Agent Run API foundation for business applications.
- `Super-Sky/Athena#8`: OpenAI-compatible tools and tool_calls contract.
- `Super-Sky/Athena#9`: Remote business tool registry and execution surface.
- `Super-Sky/Athena#10`: External memory and context asset APIs for business apps.
- `Super-Sky/Athena#11`: Agent trace timeline API and admin readout.
- `Super-Sky/Athena#12`: Docker Compose runtime profile with PostgreSQL and Redis.
- `Super-Sky/Athena#14`: Built-in basic tools pack for agent runs.
- `Super-Sky/Athena#21`: OpenTelemetry trace projection and optional Langfuse runtime profile.
- `Super-Sky/Athena#22`: goal-driven execution controls, Redis-backed async-job contract, and stable stop reasons.
- `Super-Sky/Athena#23`: pgvector-backed governed memory retrieval for business-app context.
- `Super-Sky/Athena#24`: outbound remote-tool service authentication through secret references.
- `Super-Sky/Athena#25`: app-facing service authentication, tenant isolation, and run quotas.

## Follow-Up Sequencing

1. Finish and merge the existing Agent Run / tool / remote-tool stack, then run the dual-service smoke.
2. In parallel, complete fund `#30` consent / scopes / revocation, Athena `#24` outbound callback identity, Athena `#25` app / tenant identity and quotas, and Athena `#21A` trace ID, taxonomy, allowlisting, recursive redaction, and sampling.
3. Implement Athena `#22` goal evaluation, budgets, stop reasons, PostgreSQL state truth, and Redis dispatch while fund `#31A` enforces deterministic fixtures and a CI release block.
4. After the #22 state machine stabilizes, finish Athena `#21B` OTLP / optional Langfuse, then Athena `#23` consent-aware and tenant-aware pgvector retrieval and fund `#31B` cross-service trace evaluations.
5. After fund `#16` attachment metadata and Athena `#22` async states stabilize, implement fund `#37` storage, cited parsing, retention, and governed OCR adapters.
6. Continue real-provider admission with user-owned credentials in parallel; wire Redis cache and freshness status only around approved providers.
