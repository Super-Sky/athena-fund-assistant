# Fund-Driven Athena Agent Evolution Roadmap

## Background

`athena-fund-assistant` is the first real business-validation scenario for Athena. It needs to read user-authorized account summaries and freshness-tagged market data, use read-only business tools, produce conservative, balanced, and aggressive options, and preserve reviewable evidence. The scenario validates generic Athena Agent capabilities and must not write fund, holding, NAV, or trading semantics into Athena core.

## Settled Boundaries

- Athena PostgreSQL runtime records are the source of truth for runs, traces, usage, audit, and admin readout.
- OpenTelemetry, Langfuse, Prometheus, and other external systems receive only safe, redacted, replaceable projections.
- The fund assistant owns accounts, data providers, user preferences, strategy knowledge, consent, and financial governance; it integrates with Athena through APIs, OpenAI-compatible tools, and the remote-tool contract.
- The product provides research and decision support only. It does not provide automatic trading, order writes, or money movement.

## Phases And Dependencies

1. Complete the existing Athena integration chain: `Athena#7`, `#8`, `#9`, `#14`, `#10`, `#11`, and `#12`, then pass the dual-service smoke.
2. Prepare fund accounts, conversation, knowledge, and real-data work in parallel: `fund#15`, `#16`, `#17`, `#10`, and `#11`. A real provider becomes the default only after live validation with a user-owned key or token.
3. Complete two safety foundations in parallel: `fund#30` read-only consent / scopes / revocation, and `Athena#21A` unified `trace_id`, trace taxonomy, allowlisting / recursive redaction, and sampling.
4. Then use `Athena#22` for goal-driven execution controls, stable stop reasons, and Redis jobs while `fund#31A` adds deterministic financial fixtures and a CI release block.
5. After the state machine stabilizes, use `Athena#21B` for the OTLP Collector and optional Langfuse profile, make `Athena#23` reuse the consent contract for pgvector memory retrieval, and finish cross-service trace plus optional model-assisted evaluations in `fund#31B`.

The full deliverables, acceptance criteria, and component-admission matrix live in `docs/platform-mvp-plan.en-US.md` and its Chinese counterpart. The delivery queue lives in `docs/issue-plan.md`.

## Component Admission Decisions

- Introduce now: OpenTelemetry Collector, optional Langfuse profile, Redis, a Go-native queue, PostgreSQL + pgvector, and Promptfoo.
- Reserve through interfaces: search providers, document parsing/OCR, sandboxing, and the LiteLLM gateway.
- Defer: Temporal, dedicated vector databases, Keycloak, and Vault. Create work only after a measurable scale, workflow, or enterprise-integration need exists.

## Verification

- GitHub issues `Athena#21`, `#22`, `#23` and `fund#30`, `#31`, `#32` were created and verified as open.
- The Chinese and English plans contain the same principles, Phase 5/6, component matrix, and dependency order.
- Dual-service launch examples use repository-relative paths and pass the absolute-path check.

## Maintenance Skill

No dedicated skill was created. This roadmap consolidates existing runtime, provider, Docker, financial-governance, and GitHub-issue workflows; implementation work should continue through the relevant feature documents and repository skills.
