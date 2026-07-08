# Agent Workflows

## First Implementation Shape

Use one orchestrator plus deterministic workers before introducing autonomous multi-agent behavior.

## Orchestrator

`InvestmentOrchestrator` owns:

- user goal interpretation
- workflow selection
- tool planning
- worker invocation
- final answer assembly
- trace handoff to Athena

## Workers

### User Profile Worker

Maintains:

- risk preference
- horizon
- max drawdown
- allocation constraints
- output preference

### Fund Research Worker

Produces:

- fund diagnosis
- risk factors
- evidence list
- data freshness summary

### Portfolio Risk Worker

Produces:

- concentration risk
- exposure risk
- cash / fund allocation impact

### Decision Matrix Worker

Produces:

- conservative option
- balanced option
- aggressive option
- fallback option

### Governance Worker

Checks:

- no guaranteed returns
- no automatic trading
- no single absolute conclusion
- source attribution present
- risk and invalidation present

### Report Writer Worker

Assembles:

- user-facing summary
- option comparison
- decision journal draft

## Future Multi-Agent Extension

After the deterministic workers are stable, workers can become Athena sub-agent runs with independent traces and checkpoint state.

