# Athena Fund Assistant

Athena Fund Assistant is the first business scenario application built on top of the Athena Agent Runtime.

The product direction is an open, non-trading fund research and decision-support assistant. It helps users organize fund data, portfolio context, risk preferences, decision options, and review journals. It does not place trades or custody money.

## Positioning

- `Athena` remains the generic Agent Runtime foundation.
- `athena-fund-assistant` owns the fund research application, UI, domain data, scenario workflows, and product-specific governance.
- The application should integrate with Athena through stable API / SDK / tool contracts rather than importing Athena internal packages.

## MVP Scope

The first version focuses on funds and ETFs:

- investor profile
- portfolio holdings
- fund diagnosis
- conservative / balanced / aggressive decision matrix
- decision journal
- review tasks
- data-source and reasoning trace
- Athena runtime integration plan

The first version intentionally excludes:

- automatic trading
- custody or brokerage operations
- guaranteed returns
- single-path "must buy / must sell" conclusions
- paid regulated advisory service positioning

## Core Principle

The assistant should provide clear, actionable scenarios, not vague disclaimers. A valid answer can include position adjustment ranges such as 5% or 10%, but those ranges must be tied to user profile, portfolio constraints, data evidence, risk tradeoffs, and review conditions.

## Planning Documents

- `docs/api.zh-CN.md`
- `docs/api.en-US.md`
- `docs/product-boundary.md`
- `docs/architecture.md`
- `docs/mvp-plan.md`
- `docs/athena-integration.md`
- `docs/agent-workflows.md`
- `docs/data-governance.md`
- `docs/issue-plan.md`

## Repository Status

This repository now contains the first local Go API slice for the fund assistant MVP:

- fund analysis endpoint
- mock fund / ETF / US market data provider
- conservative / balanced / aggressive decision matrix
- in-memory decision journal and review task

The current implementation is still local-first and mock-data-backed. Docker Compose already starts the API, PostgreSQL, and Redis, but the API does not persist to PostgreSQL or cache through Redis yet. React UI, Athena runtime integration, and real data providers remain active MVP work.

## Local Run

```bash
ATHENA_FUND_API_ADDR=:8081 go run ./cmd/api
```

Docker Compose is also available for the API, PostgreSQL, and Redis:

```bash
cp .env.example .env
docker compose up --build
```
