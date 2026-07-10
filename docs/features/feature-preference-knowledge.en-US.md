# User Preference And Strategy Knowledge Base

## Scope

This feature adds user-level `agent.md`, durable preferences, and a fund strategy knowledge base to the fund assistant MVP. The business app owns these preference and strategy assets and can expose them to Athena as context or remote-tool data without writing fund business objects into Athena core.

## Implemented

- `internal/domain/knowledge.go`
  - Defines `PreferenceProfile`, `KnowledgeItem`, `KnowledgeRevision`, `KnowledgeAuditEvent`, and `KnowledgeWorkspace`.
- `internal/preference/store.go`
  - Provides the in-memory preference / knowledge store.
  - Seeds `demo-user` with agent.md, preferences, position-sizing rules, and audit events.
  - Supports preference drafts, preference activation, knowledge drafts, knowledge activation, and knowledge rollback.
- API:
  - `GET /api/users/{user_id}/knowledge`
  - `POST /api/users/{user_id}/preferences/drafts`
  - `POST /api/users/{user_id}/preferences/activate`
  - `POST /api/users/{user_id}/knowledge/drafts`
  - `POST /api/users/{user_id}/knowledge/{item_id}/activate`
  - `POST /api/users/{user_id}/knowledge/{item_id}/rollback`
- `apps/web`
  - Adds the "User preference · agent.md" and "Strategy knowledge base" panels.
  - Supports viewing current preferences, saving a knowledge draft, activating the latest knowledge draft, and inspecting audit events.

## Governance Boundaries

- Drafts can be saved by the UI, future function calls, or MCP.
- Formal use requires an explicit activation API call. The current MVP uses schema validation plus manual activation as the governance gate.
- Every item and revision preserves source, author, confidence, schema_version, governance_decision, and audit.
- The current store is in-memory and suitable for the local MVP. PostgreSQL persistence plus finer permission / approval flows remain follow-up work.
- Athena does not own fund business objects. Preferences and strategy knowledge remain managed by the fund assistant.

## Verification

- `go test ./internal/preference ./internal/server ./internal/domain`
- `go test ./...`
- `yarn build` in `apps/web`
- API smoke:
  - `GET /api/users/demo-user/knowledge` returns seeded preference, knowledge item, revision, and audit data.
  - `POST /api/users/demo-user/knowledge/drafts` returns a draft revision.
  - `POST /api/users/demo-user/knowledge/{item_id}/activate` promotes the draft to active.
