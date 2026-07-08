# Decision Module

The decision module owns deterministic option generation.

## File Index

- `engine.go`
  - Generates conservative, balanced, and aggressive options from profile, portfolio, and snapshot rules.
- `engine_test.go`
  - Verifies traceable three-option output and prevents aggressive options from exceeding allocation caps.
