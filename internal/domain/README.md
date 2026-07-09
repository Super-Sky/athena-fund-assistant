# Domain Module

The domain module owns versionable fund assistant business objects and validation rules.

## File Index

- `types.go`
  - Defines source metadata, investor profiles, portfolios, fund snapshots, market support snapshots, diagnoses, decision matrices, journal entries, and review tasks.
- `account.go`
  - Defines user accounts, account holding snapshots, operation records, performance trend points, and the account overview read model.
- `conversation.go`
  - Defines Agent workspace skills, sessions, messages, attachment metadata, and trace events.
- `types_test.go`
  - Verifies decision matrix governance shape, including rejection of single-path outputs.
