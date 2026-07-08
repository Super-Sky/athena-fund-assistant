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

## Current Local Modules

```text
apps/web
  React + TypeScript fund research workspace

cmd/api
  local HTTP API process

cmd/providerprobe
  validation-first real data-source probe command

internal/domain
  InvestorProfile, Portfolio, FundInstrument, DecisionMatrix, DecisionJournal

internal/data
  provider interface, validation report, mock fund and market data

internal/providerprobe
  validation-only probes for real data-source response shapes

internal/decision
  conservative / balanced / aggressive option generation

internal/journal
  decision journal storage boundary

internal/server
  HTTP route mapping and local CORS boundary
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
