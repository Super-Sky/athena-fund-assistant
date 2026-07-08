# Data And Governance

## Data Requirements

Every normalized data point must preserve:

- source
- fetched_at
- market_time
- delay
- provider
- license / terms marker
- confidence
- raw payload hash when available
- normalized schema version

## First Data Strategy

Start with mock and manually curated sample data. Then add real providers behind the same interface.

Recommended provider order:

1. mock provider
2. CSV / manual import provider
3. public or user-key data provider
4. paid data provider, only after license review

## Governance Rules

The assistant must:

- include at least one alternative option
- include risks for each option
- include invalidation conditions
- include review timing
- attribute data sources
- avoid guaranteed return language
- avoid auto-trading behavior
- mark stale or missing data clearly

## Position Adjustment Rule

Percentages such as 5% or 10% may be generated only when tied to at least one of:

- user risk profile
- portfolio allocation constraint
- user-authored strategy rule
- predefined decision template
- backtest / simulation result

