# Server Module

The server module maps MVP workflows to HTTP routes.

## File Index

- `server.go`
  - Exposes health, protected account/user workflows, Agent workspace, fund analysis, and journal creation endpoints using standard-library HTTP.
- `authorization.go`
  - Maps bearer-session and consent-grant services into local auth, grant lifecycle, user-resource protection, and Athena service-auth boundaries.
- `athena_runs.go`
  - Maps conversation messages into generic Athena Agent Run requests with safe consent references, context assets, OpenAI-compatible tools, constraints, and governance refs.
- `remote_tools.go`
  - Exposes the Athena `remote_tool_execution.v1` catalog and validates service identity plus read-only grant scopes before returning account data.
- `authorization_test.go`
  - Verifies session/grant lifecycle, cross-user denial, remote service identity, required scopes, revocation, and credential redaction.
- `server_test.go`
  - Verifies account dashboard APIs, Agent workspace upload/message APIs, Athena remote tool callbacks, and the end-to-end local API workflow from analysis to journal and review task creation.
