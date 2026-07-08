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

- `docs/product-boundary.md`
- `docs/architecture.md`
- `docs/mvp-plan.md`
- `docs/athena-integration.md`
- `docs/agent-workflows.md`
- `docs/data-governance.md`
- `docs/issue-plan.md`

## Repository Status

This repository currently contains planning and boundary documents. Implementation should begin after the Athena-side agent loop, tool-call, trace, memory, and governance integration points are confirmed.
