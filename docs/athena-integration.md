# Athena Integration

## Integration Principle

This app should treat Athena as an external Agent Runtime. It should not import Athena internal packages.

## Expected Athena Capabilities

The fund assistant needs Athena to provide:

- goal-driven agent run API
- OpenAI-compatible tools / tool calls
- internal canonical tool contract
- memory query / write API
- context asset injection
- governance evaluation
- trace read API
- checkpoint / resume
- artifact / report output

## Fund App Responsibilities

The fund app registers or provides:

- fund data tools
- portfolio context tools
- decision-journal tools
- review-task tools
- financial governance policy
- fund-specific context assets
- UI and business persistence

## Example Goal

```json
{
  "goal": "Generate decision options for the user's holding in a fund",
  "constraints": {
    "no_auto_trading": true,
    "must_include_alternative_option": true,
    "must_include_risk_and_invalidation": true
  },
  "context_assets": [
    "investor_profile",
    "portfolio_snapshot",
    "fund_snapshot"
  ]
}
```

## Required Trace

Each run should expose:

- user goal
- selected workflow
- tools called
- data fetched
- data freshness
- model calls
- governance checks
- memory read/write
- generated options
- selected journal entry

