# Dual-Service MVP Execution Plan

## Goal

Bring `Athena` and `athena-fund-assistant` to a local Docker-based MVP that is runnable, demonstrable, and verifiable.

- `Athena`: the generic Agent Runtime foundation.
- `athena-fund-assistant`: the fund research assistant business application.

Both projects should keep a consistent stack:

- Backend: Go
- Frontend: React + TypeScript + Vite
- Primary database: PostgreSQL
- Cache / temporary state / async-job helper: Redis
- Deployment: Docker / Docker Compose

## Service Boundary

### Athena Owns

- Goal-driven agent loop.
- OpenAI-compatible `tools` / `tool_calls` input and output.
- Athena canonical tool contract.
- Tool registry.
- Trace timeline.
- Memory / context APIs.
- Governance gate.
- Checkpoint / resume.
- Built-in basic tools.
- Control Plane / System Validation / runtime readout.

### athena-fund-assistant Owns

- InvestorProfile.
- Portfolio / PortfolioHolding.
- FundInstrument / FundSnapshot / MarketSnapshot.
- China fund/ETF data providers.
- US ETF/index data providers.
- Fund diagnosis.
- Conservative / balanced / aggressive decision matrix.
- Decision journal.
- Review tasks.
- Fund business UI.
- Financial scenario governance rules.

## Layering Rules

- Athena core must not store fund business objects.
- Athena core must not hard-code fund, ETF, allocation, NAV, or return semantics.
- The fund assistant must not import Athena internal Go packages.
- The two services integrate through APIs, SDKs, or tool contracts.

## MVP Phases

### Phase 0: Planning And Boundary Freeze

Deliverables:

- Bilingual documentation policy.
- Product boundary.
- Data-source strategy.
- Dual-service architecture.
- Layered GitHub issues.

### Phase 1: Local Fund-App Business Loop

Deliverables:

- Go API skeleton.
- React + Vite UI skeleton.
- PostgreSQL schema.
- Redis configuration.
- InvestorProfile / Portfolio / DecisionMatrix / DecisionJournal models.
- Mock / CSV data provider.
- Three-option decision generation.
- Decision journal and review tasks.
- Local Docker Compose startup.

Acceptance:

- A user can enter a profile and holdings.
- The system can generate fund diagnosis from mock/CSV data.
- The system can generate conservative / balanced / aggressive options.
- The user can select one option and create a decision journal entry.
- The system can generate a review task.

### Phase 2: Real Data Providers

Deliverables:

- China fund/ETF provider.
- US ETF/index provider.
- Data freshness / timezone / delay / license metadata.
- Provider interface and cache layer.
- Data-source failure fallback.

Acceptance:

- At least one China data path can fetch real fund or ETF data.
- At least one US data path can fetch real ETF or index data.
- Every data item preserves `source`, `fetched_at`, `market_time`, `timezone`, `delay`, `provider`, `license_terms`, `confidence`, and `schema_version`.
- Temporary mock/CSV data must be clearly marked in the UI.

### Phase 3: Athena Integration

Deliverables:

- Fund-app Athena client.
- Fund data tool registration shape.
- Portfolio / decision-journal tools.
- Context asset injection.
- Memory read/write.
- Governance evaluation.
- Trace readout.

Acceptance:

- The fund app can start an Athena agent run.
- Athena can call tools registered by the fund app.
- Each fund analysis can trace tools, data, model calls, governance checks, and the final decision matrix.

### Phase 4: End-To-End Demo

Deliverables:

- Dual-service Docker Compose.
- One-command startup docs.
- Seed data.
- Local demo flow.
- Admin trace verification.

Acceptance:

- `docker compose up` starts Athena, fund assistant, PostgreSQL, and Redis.
- A user can complete one fund diagnosis and choose one decision option.
- The decision journal is visible.
- Athena / fund-app traces can be inspected.

## Multi-Agent / Parallel Development Split

Recommended parallel tracks:

- `Runtime Agent`: Athena agent loop, tool calls, trace, memory, and governance APIs.
- `Finance Domain Agent`: fund-app domain models and decision matrix.
- `Data Provider Agent`: China and US data providers.
- `UI Agent`: fund assistant frontend.
- `Docker Agent`: dual-service Docker / Compose.
- `Governance Agent`: financial output governance, data-license markers, and bilingual docs.

The first implementation should use one orchestrator plus deterministic workers. Autonomous multi-agent behavior is deferred.

## Key Risks

- A free data source is not automatically commercial-safe or redistribution-safe.
- China fund data has less clear licensing boundaries than US API providers.
- US data must handle the `America/New_York` timezone, non-trading days, and delayed feeds.
- The default output must not be a single-path buy/sell conclusion.
- Percentages must be derived from user profile, portfolio constraints, strategy templates, historical data, or explicit rules.

