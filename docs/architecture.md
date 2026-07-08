# Architecture

## Repository Split

Athena stays generic. This repository owns fund-specific application behavior.

```text
Athena
  agent runtime
  tool-call orchestration
  trace / usage / persistence
  memory / context compression
  governance
  system truth
  control-plane

athena-fund-assistant
  investor profile
  portfolio and holding domain
  fund data adapters
  decision matrix
  decision journal
  fund research UI
  product-specific governance
```

## Planned Local Modules

```text
apps/web
  user-facing fund research workspace

apps/api
  product API, auth, portfolio storage, Athena client facade

packages/domain
  InvestorProfile, Portfolio, FundInstrument, DecisionMatrix, DecisionJournal

packages/data-providers
  fund data, market data, index data, news, announcement adapters

packages/decision-engine
  conservative / balanced / aggressive option generation

packages/athena-client
  Athena run, tool, memory, trace, governance API client
```

## Domain Objects

- `InvestorProfile`
- `Portfolio`
- `PortfolioHolding`
- `FundInstrument`
- `FundSnapshot`
- `MarketSnapshot`
- `WatchCondition`
- `DecisionMatrix`
- `DecisionOption`
- `DecisionJournal`
- `ReviewTask`
- `ResearchReport`

## Core Rule

Fund business objects live in this repository. Athena only receives generic goals, tools, context assets, memory entries, and trace metadata.

