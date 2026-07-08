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

