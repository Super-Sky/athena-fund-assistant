# Server Module

The server module maps MVP workflows to HTTP routes.

## File Index

- `server.go`
  - Exposes health, account overview, manual account holdings, Agent workspace, fund analysis, and journal creation endpoints using standard-library HTTP.
- `athena_runs.go`
  - Maps conversation messages into generic Athena Agent Run requests with context assets, OpenAI-compatible tools, constraints, and governance refs.
- `remote_tools.go`
  - Exposes the fund assistant's Athena `remote_tool_execution.v1` catalog and callback handler without importing Athena internal Go packages.
- `server_test.go`
  - Verifies account dashboard APIs, Agent workspace upload/message APIs, Athena remote tool callbacks, and the end-to-end local API workflow from analysis to journal and review task creation.
