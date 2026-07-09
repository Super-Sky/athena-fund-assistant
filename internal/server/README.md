# Server Module

The server module maps MVP workflows to HTTP routes.

## File Index

- `server.go`
  - Exposes health, account overview, manual account holdings, Agent workspace, fund analysis, and journal creation endpoints using standard-library HTTP.
- `server_test.go`
  - Verifies account dashboard APIs, Agent workspace upload/message APIs, and the end-to-end local API workflow from analysis to journal and review task creation.
