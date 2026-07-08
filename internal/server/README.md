# Server Module

The server module maps MVP workflows to HTTP routes.

## File Index

- `server.go`
  - Exposes health, fund analysis, and journal creation endpoints using standard-library HTTP.
- `server_test.go`
  - Verifies the end-to-end local API workflow from analysis to journal and review task creation.
