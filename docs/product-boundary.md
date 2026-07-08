# Product Boundary

## Product Position

Athena Fund Assistant is a fund research and decision-support product. It helps users improve investment decision quality through data organization, structured options, risk review, and decision journaling.

It should be useful enough to support concrete actions such as "reduce by 5%" or "keep holding and review later", but the system must present these as decision options tied to user-defined profile, constraints, evidence, and risks.

## Allowed Outputs

- Fund diagnosis and comparison.
- Portfolio concentration and risk review.
- Conservative / balanced / aggressive decision options.
- Position-adjustment ranges when derived from user profile, portfolio constraints, or explicit strategy rules.
- Watch conditions and review reminders.
- Strategy simulation and historical backtest summaries.
- Decision journal entries and follow-up reviews.

## Disallowed Outputs

- Automatic trade execution.
- Custody or brokerage operations.
- Guaranteed return claims.
- Single-path absolute commands such as "must buy" or "must sell".
- Unattributed market rumors as evidence.
- Hidden paid advisory positioning.
- Claims that the assistant is a licensed investment adviser unless the business actually has the required status.

## Default Decision Shape

The default answer should contain three options:

1. Conservative
2. Balanced
3. Aggressive

If a user explicitly identifies as aggressive, the UI may compress the output to:

1. Aggressive primary option
2. Conservative fallback option

Every option must include:

- action
- percentage or range when applicable
- applicable conditions
- evidence
- risk
- invalidation condition
- next review time
- portfolio impact

