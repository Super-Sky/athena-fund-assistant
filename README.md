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
- local bearer session plus revocable read-only account consent
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
- `docs/features/feature-read-only-account-consent.zh-CN.md`
- `docs/features/feature-read-only-account-consent.en-US.md`
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
- React + TypeScript + Vite research console
- local account performance dashboard
- Agent conversation workspace with skill selection and attachment metadata
- Athena Agent Run client facade with mock and HTTP modes
- Athena remote business tool callbacks for read-only account and market data
- hashed bearer sessions, revisioned read-only consent grants, and redacted authorization audits
- mock fund / ETF / US market data provider
- conservative / balanced / aggressive decision matrix
- durable decision journal and review task when `DATABASE_URL` is configured

The current implementation is still local-first and mock-data-backed. Docker Compose starts the web console, API, PostgreSQL, and Redis. The account dashboard, decision journal, sessions, consent grants, and authorization audits use PostgreSQL when `DATABASE_URL` is configured and fall back to explicit in-memory demo stores otherwise. Account market data is still explicitly marked as temporary mock data. Athena integration now has an app-side Agent Run client and read-only remote business tools. Production-grade outbound service authentication for the full dual-service callback remains tracked in `Super-Sky/Athena#24`. Real data providers and persistent preference/knowledge storage remain active MVP work.

## Local Run

```bash
ATHENA_FUND_API_ADDR=:8081 go run ./cmd/api
```

Run the web console during development:

```bash
cd apps/web
yarn install
yarn dev
```

Docker Compose is also available for the web console, API, PostgreSQL, and Redis:

```bash
cp .env.example .env
docker compose up --build
```

Dual-service Athena smoke:

```bash
ATHENA_REPO=/Users/maxt/Desktop/maxt/Athena-remote-tools ./scripts/smoke_dual_service.sh
```
