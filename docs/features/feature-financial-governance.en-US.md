# Financial Governance Gate

## Scope

The fund-analysis endpoint evaluates the generated decision matrix with a deterministic, product-specific governance gate before returning it. The gate keeps the product useful for concrete decision support while preserving the boundary that users, not the system, choose and execute an action.

## Rules

- Block guaranteed-return claims, automatic order execution, and absolute commands such as `must buy` or `must sell`.
- Block output with fewer than two options.
- Flag an option missing risk, invalidation, or review timing.
- Block a non-zero allocation change when it has no profile, portfolio, template, rule, or simulation basis.
- Flag missing or malformed source, provider, fetched-time, market-time, or timezone metadata.
- Return a machine-readable result with `passed`, `flagged`, or `blocked` checks. `blocked` output is not delivered by `POST /api/analysis/fund`; `flagged` output remains deliverable but must retain the disclosure.

## Boundaries

- This is a deterministic output guardrail, not investment suitability review or licensed financial advice.
- It does not execute trades, store brokerage credentials, or accept broker order instructions.
- It does not certify data quality, licensing, or freshness; it requires those attributes to be explicit so the UI and user can assess them.

## Verification

- `go test ./internal/governance ./internal/decision ./internal/server`
- Governance tests cover allowed output, missing metadata and risk/invalidation flags, prohibited language, and percentage-basis blocks.
