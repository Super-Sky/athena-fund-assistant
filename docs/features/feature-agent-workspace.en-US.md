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
  - Appends a message, writes local trace events, and starts an Agent Run through the Athena client. When `ATHENA_BASE_URL` is unset, the mock client keeps local demos runnable; when configured, the app calls external Athena `/api/agent/runs`.
- `GET /internal/tools/catalog`
  - Emits fund-assistant tool registrations that can be registered in Athena's remote tool registry.
- `POST /internal/tools/execute`
  - Executes `remote_tool_execution.v1` callbacks. Both current tools are read-only; `account_overview` also validates Athena service identity and user consent scopes.
- `apps/web`
  - Uses Agent conversation as the default home and shows core account context, skill selection, file upload, messages, attachment status, and trace timeline.
  - Separates account, holdings, performance, strategy analysis, preferences, knowledge, and data access into focused navigation views that retain only core data and configuration.
  - Uses grouped Lucide navigation and a conversation composer with an explicit attachment action plus an accessible icon send button. The auxiliary attachment/trace rail stays secondary to the conversation.

## Upload Boundaries

- Per-file limit: `10 MiB`.
- Default retention window: `7 days`.
- Upload directory: `ATHENA_FUND_UPLOAD_DIR`; if unset, the system temp directory is used.
- First supported types include image, PDF, CSV, TXT, and Excel MIME. Unknown types are marked with `unsupported=true`.
- Unparsed attachments can only be metadata / context candidates. They must not masquerade as parsed statements, facts, or strategy knowledge.

## Athena Boundary

- The UI and API now expose the contract shape for starting an Agent run, and the app-side Athena client writes run trace back to the conversation.
- The fund assistant exposes Athena remote tool callbacks. Complete account callbacks require Athena to inject a separate service identity and carry the safe `consent_grant_ref` from model context.
- Without `ATHENA_BASE_URL`, trace shows a mock run; with external Athena configured, trace records the real Athena run_id, status, and trace_available.
- Fund business objects, uploaded files, and business tool implementations stay in the fund assistant and are not written into Athena core.
- Current remote tools are read-only, use `side_effect_level=none`, and do not perform automatic trading or money movement.
- `Super-Sky/Athena#24` tracks remote-tool secret references and outbound header injection; historical dual-service smoke is not current service-authentication evidence until that capability lands.

## Verification

- `go test ./...`
- `yarn build` in `apps/web`
- Browser smoke at 1440px and 390px: conversation opens by default; skill selection, upload, composer, and trace are visible; desktop and mobile navigation do not create horizontal page overflow or browser console errors.
- Server test: remote tool catalog, service identity, allowed consent, missing scope, post-revocation denial, `fund_market_snapshot`, and the unknown-tool error envelope.
- Server test: conversation message starts an Athena mock run and writes an `athena_agent_run=ok` trace.
- Dual-service smoke: update and rerun after `Super-Sky/Athena#24` lands so the `account_overview` callback includes service authentication.
