# Agent Conversation Workspace

## Scope

This feature moves the fund assistant from fixed form workflows toward a daily Agent workspace. A user can select a skill, upload an image or file, send a natural-language request, and inspect attachment status plus a local trace timeline.

## Implemented

- `internal/domain/conversation.go`
  - Defines skills, conversation sessions, messages, attachment metadata, and trace events.
- `internal/conversation/store.go`
  - Provides the conversation store interface, local in-memory implementation, upload directory, SHA256, size limit, retention window, and pending/unsupported status.
- `GET /api/conversations/skills`
  - Returns selectable skills.
- `POST /api/conversations`
  - Creates a conversation session.
- `GET /api/conversations/{conversation_id}`
  - Reads conversation detail, messages, attachments, and trace.
- `POST /api/conversations/{conversation_id}/attachments`
  - Uploads a file and returns metadata. The current slice does not parse the attachment and does not treat it as fact.
- `POST /api/conversations/{conversation_id}/messages`
  - Appends a message and writes local trace events. Athena agent run is currently marked as `pending` until the real Athena client / remote tools are wired.
- `apps/web`
  - Shows Agent chat, skill selector, file upload, message list, attachment status, and trace timeline.

## Upload Boundaries

- Per-file limit: `10 MiB`.
- Default retention window: `7 days`.
- Upload directory: `ATHENA_FUND_UPLOAD_DIR`; if unset, the system temp directory is used.
- First supported types include image, PDF, CSV, TXT, and Excel MIME. Unknown types are marked with `unsupported=true`.
- Unparsed attachments can only be metadata / context candidates. They must not masquerade as parsed statements, facts, or strategy knowledge.

## Athena Boundary

- The UI and API now expose the local contract shape for starting an Agent run, but this slice does not call Athena yet.
- `athena_agent_run=pending` in trace means the next slice should wire Athena Agent Run API and remote business tools.
- Fund business objects, uploaded files, and business tool implementations stay in the fund assistant and are not written into Athena core.

## Verification

- `go test ./...`
- `yarn build` in `apps/web`
- Browser smoke: workspace, skill selector, upload entry, and trace timeline are visible.
