# MVP Plan

## MVP Goal

Build a local-first fund research assistant that can guide a user from profile and holdings to a traceable decision matrix and decision journal.

## MVP Workflows

### 1. Investor Profile

Capture:

- risk preference
- investment horizon
- max acceptable drawdown
- single-fund max allocation
- cash preference
- default decision style

Output:

- profile summary
- default option weights
- decision-output preference

### 2. Portfolio Import

Capture:

- fund code / name
- holding amount
- cost basis
- holding percentage
- user thesis

Output:

- portfolio snapshot
- concentration warnings
- missing-data warnings

### 3. Fund Diagnosis

Input:

- fund instrument
- market / fund snapshots
- user profile

Output:

- performance and drawdown summary
- style and exposure summary
- risk factors
- data freshness
- evidence list

### 4. Decision Matrix

Output:

- conservative option
- balanced option
- aggressive option
- fallback option when the user profile is strongly biased
- option comparison table

### 5. Decision Journal

Record:

- selected option
- user notes
- evidence snapshot
- expected outcome
- invalidation condition
- review date

### 6. Review Task

Compare:

- original thesis
- actual fund / market movement
- whether invalidation conditions fired
- what should be adjusted next

## MVP Acceptance Criteria

- A user can enter one profile and one fund holding.
- The system can generate a three-option decision matrix from mock data.
- The user can select an option and create a decision journal entry.
- A review task can be generated from the journal.
- The output includes source/freshness metadata even when the first data provider is mock.
- The architecture can swap mock data for real providers without changing domain objects.

