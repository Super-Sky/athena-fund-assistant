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
- App / service identity, tenant ownership, scopes, and run quotas.
- Trace timeline.
- Memory / context APIs.
- Governance gate.
- Checkpoint / resume.
- Built-in basic tools.
- Control Plane / System Validation / runtime readout.
- Standard OpenTelemetry trace projection, redaction, and optional observability-backend export.
- Generic execution budgets, deadlines, stop conditions, and asynchronous-job contracts.

### athena-fund-assistant Owns

- InvestorProfile.
- Portfolio / PortfolioHolding.
- FundInstrument / FundSnapshot / MarketSnapshot.
- China fund/ETF data providers.
- US equity/ETF/index/FX/market-calendar data providers.
- Fund diagnosis.
- Conservative / balanced / aggressive decision matrix.
- Decision journal.
- Review tasks.
- Fund business UI.
- Attachment storage, isolated parsing / OCR, and evidence citations.
- Financial scenario governance rules.
- Domain evaluation suites based on real data, user consent, and investment constraints.

## Layering Rules

- Athena core must not store fund business objects.
- Athena core must not hard-code fund, ETF, allocation, NAV, or return semantics.
- The fund assistant must not import Athena internal Go packages.
- The two services integrate through APIs, SDKs, or tool contracts.

## Evolution Principles

- **Validate the foundation through the fund scenario without contaminating it**: each Athena capability is driven by a clear fund-analysis need but ships as a generic goal, tool, context, trace, or governance contract.
- **Athena persisted runtime records are the source of truth**: PostgreSQL run, step, trace, usage, and audit data serve product admin, consent audit, and retention needs; external observability systems receive only redacted projections.
- **Validate before coding**: market data, search, document parsing, and external tools must confirm authorization, fields, timezone, quota, failures, and observability fields before entering the decision workflow.
- **Do not stop an agent at a fixed iteration count**: completion must follow success criteria, budget, deadline, external-data wait, human confirmation, unrecoverable error, or governance denial.
- **Every recommendation is reviewable**: it must link data source, timestamp, tool calls, user preferences, strategy basis, risks, and invalidation conditions; automatic trading is excluded.

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
- US equity/ETF/index/FX/market-calendar provider.
- Data freshness / timezone / delay / license metadata.
- Provider interface and cache layer.
- Data-source failure fallback.

Acceptance:

- At least one China data path can fetch real fund or ETF data.
- At least one US data path can fetch real equity, ETF, or index data and either provide USD/CNY FX rates or explicitly mark FX as unavailable.
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
- Run budget, deadline, stop reason, and asynchronous resumption.

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

### Phase 5: Fund-Driven Agent Runtime Reinforcement

This phase uses one acceptance scenario: a user asks whether their portfolio needs adjustment. The Agent must read authorized account summaries and preferences, obtain freshness-tagged market data, select read-only business tools, produce three options, and leave an explainable end-to-end trace.

Deliverables:

- Athena generic agent loop: success criteria, budget, deadline, tool retry, waiting / terminal stop reason, checkpoint / resume.
- Service identity, app / tenant / subject ownership, scopes, and quotas for app-facing APIs; Athena outbound remote-tool callbacks inject service credentials through secret references.
- Deliver OpenTelemetry in two slices: first add a unified `trace_id`, seven runtime-trace categories, allowlisting / recursive redaction, and forced sampling; after execution states stabilize, add the `OpenTelemetry Collector` and optional self-hosted `Langfuse` Docker profile.
- Unified trace and correlation IDs for run / step / model / tool / memory / governance / remote callbacks.
- Freeze an immutable manifest per run for model / provider, system prompt, skill, tool schema, governance policy, context-asset, and evaluator revisions so runs can be reproduced, evaluated, and audited.
- Actual Redis integration for cache, concurrency/rate limits, idempotency locks, and async jobs; start with a Go-native queue and evaluate Temporal separately for complex multi-day workflows.
- A `pgvector` knowledge and memory retrieval slice using PostgreSQL first, without a separate vector database.
- Generic built-in tools: HTTP fetch, search-provider adapter, calculator, time / market calendar, and file-schema validation. Fund-domain tools remain remotely registered by the fund assistant.

Acceptance:

- The same `run_id` can be correlated by trace ID in Athena admin and the optional observability backend without exporting credentials, raw sensitive holdings, or unredacted prompts.
- Normal runs, tool errors, timeouts, governance denials, waiting-for-data, and resumes have stable stop reasons and traces.
- If Redis is unavailable, the system explicitly degrades or fails rather than silently claiming cache or async success.

### Phase 6: Trusted Fund Decisions And Continuous Evaluation

Deliverables:

- User-account authentication, token/session, read-only data consent, tool scopes, consent revocation, and audit events. Brokerage sync remains read-only and requires separate consent.
- In-repository `Promptfoo` evaluation configuration and CI commands: block releases with deterministic critical cases first, then add Athena-trace and optional model-assisted evaluations. Cover missing or stale data, tool failure, single-path conclusions, guaranteed-return language, missing risk/invalidation, unsupported percentages, and unauthorized account reads.
- Attachment pipeline: local development storage plus a replaceable S3-compatible adapter, CSV / PDF text parsing first, and OCR as a governed adapter. File hashes, parser versions, page / row citations, retention, and authorization state must remain traceable.
- Restricted document parsing, OCR, and web search plugins with governed size, file type, outbound domains, timeout, source, and citations; untrusted execution goes through an isolated sandbox.
- Keep the model gateway optional: retain the Athena provider abstraction first, then add a LiteLLM profile only when multi-provider routing, virtual keys, or central budgets are justified.

Acceptance:

- Fund recommendations pass financial governance and regression evaluations before reaching the UI; failed cases block release rather than merely being logged.
- A user can view and revoke data consent; affected remote tools are denied by the governance layer after revocation.
- Every data point, retrieval, attachment, and strategy conclusion in a decision journal can be traced to consent and source metadata.

## Component Admission Matrix

| Capability | Current choice | Admission phase | Reason to defer or avoid |
| --- | --- | --- | --- |
| Agent trace / LLM eval | OpenTelemetry Collector + optional Langfuse | Phase 5 | Athena trace remains the source of truth, preventing product admin from depending on a third-party data model. |
| Cache / queue | Redis + Go-native queue | Phase 5 | Docker Redis already exists; assess Temporal only after multi-day workflows, approvals, and complex recovery appear. |
| Knowledge retrieval | PostgreSQL + pgvector | Phase 5 | Avoid Qdrant / Milvus / Weaviate replication and operations initially. |
| LLM gateway | Athena provider abstraction; optional LiteLLM profile | After Phase 6 | Do not add a Python service before multi-tenant routing, virtual keys, or central cost settlement are needed. |
| Continuous eval | Promptfoo | Phase 6 | Cases live in the fund app but must feed back into Athena runtime contracts. |
| Runtime identity / tenant | Provider-neutral service verifier + secret references + scoped ownership | Phase 5 | Add Athena inbound app identity, outbound callback identity, and cross-tenant denial first; evaluate Keycloak only for concrete enterprise SSO. |
| User auth / consent | Fund session + read-only consent / scopes / revocation | Phase 6 | User and brokerage authorization stays in the business app; Athena receives only safe subject and scope data. |
| Usage / quotas | Athena usage records + Redis counters / locks | Phase 5 | Implement concurrency, rate, token, and cost budgets first; billing and plans are outside the MVP. |
| Prompt / skill reproducibility | Athena revisioned assets + immutable run manifest | Phase 5 | Reuse the existing skill / system-resource revisions instead of adding a separate Prompt CMS. |
| Human approval | Eino interrupt / checkpoint + Athena governance | Phase 5 | Cover high-risk tools and missing-input confirmation first; evaluate Temporal only for complex multi-day approvals. |
| Attachments / artifacts | Fund local storage + S3-compatible adapter + cited extraction | Phase 6 | Do not require MinIO for local MVP; raw business files never enter Athena traces. |
| Sandbox | Restricted Docker executor | Phase 6 | Enable only for untrusted scripts or complex file processing. |
| Dedicated vector DB | Do not introduce yet | Later | Create a task only after pgvector capacity, latency, or retrieval quality becomes a measurable bottleneck. |

## Issue Dependencies And Execution Order

1. **Existing Athena integration chain**: `Athena#7` → `#8` → `#9` → `#14` → `#10` → `#11` → `#12`. First complete the generic run, tool, remote callback, built-in tools, memory, trace, and Docker merges plus the dual-service smoke.
2. **Existing fund business chain**: progress `fund#15`, `#16`, `#17` alongside `fund#10` and `#11`; live providers still require user-owned key/token validation before becoming the default path.
3. **Parallel safety foundations**: complete user identity, read-only consent, scopes, revocation, and remote-tool denial in `fund#30`; add outbound callback service identity in `Athena#24`, app-facing inbound identity, tenant ownership, and quotas in `Athena#25`; in parallel, limit `Athena#21A` to unified `trace_id`, trace taxonomy, allowlisting / recursive redaction, and sampling.
4. **Execution and evaluation slice**: build goal evaluation, budgets, stable stop reasons, PostgreSQL state truth, and Redis dispatch on the safe trace and identity contracts in `Athena#22`; use deterministic fixtures and a CI release block in `fund#31A` to validate financial-safety rules first.
5. **Observability and memory closure**: after the #22 state machine stabilizes, connect the OTLP Collector and optional Langfuse profile in `Athena#21B`; make `Athena#23` reuse the `fund#30` / `Athena#25` ownership and consent contracts for pgvector retrieval; then add cross-service trace and optional model-assisted evaluations in `fund#31B`.
6. **Attachment evidence chain**: after `fund#16` attachment metadata and the `Athena#22` async contract stabilize, `fund#37` delivers storage, parsing, citations, and retention. Complex OCR / sandbox adapters may proceed in parallel but must not block the CSV / PDF text MVP.

These Athena capabilities depend only on generic runtime contracts and cannot read fund tables. Trusted-fund capabilities call Athena through remote tools and do not modify Athena core.

Before implementation, each task must create or update its canonical GitHub Issue in the owning repository. Cross-repository dependencies use `Refs`; unfinished work must not use `Closes`.

## Multi-Agent / Parallel Development Split

Recommended parallel tracks:

- `Runtime Agent`: Athena agent loop, tool calls, trace, memory, and governance APIs.
- `Finance Domain Agent`: fund-app domain models and decision matrix.
- `Data Provider Agent`: China fund / ETF and US equity / ETF / index / FX / market-calendar providers.
- `UI Agent`: fund assistant frontend.
- `Docker Agent`: dual-service Docker / Compose.
- `Security / Governance Agent`: service identity, tenant isolation, user consent, financial-output governance, trace redaction, and evaluation gates.
- `Artifact Agent`: attachment storage, parsing / OCR, citations, and retention.

The first implementation should use one orchestrator plus deterministic workers. Autonomous multi-agent behavior is deferred.

## Key Risks

- A free data source is not automatically commercial-safe or redistribution-safe.
- China fund data has less clear licensing boundaries than US API providers.
- US data must handle the `America/New_York` timezone, non-trading days, half trading days, delayed feeds, and USD/CNY FX rates.
- The default output must not be a single-path buy/sell conclusion.
- Percentages must be derived from user profile, portfolio constraints, strategy templates, historical data, or explicit rules.
