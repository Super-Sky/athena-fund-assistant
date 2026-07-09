# Conversation Module

The conversation module owns the local Agent workspace state for the fund assistant application.

## Boundary

- Owns conversation sessions, selectable skills, messages, attachment metadata, local upload storage, and safe trace events.
- Does not parse attachment content into facts.
- Does not call Athena directly in this slice; Athena run creation is represented as a pending trace event until the Athena client is wired.
- Does not store fund business objects in Athena core.

## File Index

- `README.md`
  - Describes module boundary and file map.
- `store.go`
  - Defines the conversation store interface, local memory implementation, built-in skills, upload persistence, SHA256 hashing, attachment size limits, retention metadata, and trace generation.
- `store_test.go`
  - Verifies conversation creation, attachment upload metadata, pending attachment trace, and oversized attachment rejection.
