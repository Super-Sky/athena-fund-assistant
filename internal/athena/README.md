# Athena Client Module

The Athena client module owns the app-side HTTP facade for Athena Agent Run integration.

## Boundary

- Calls Athena through external HTTP APIs only.
- Does not import Athena internal Go packages.
- Converts fund-assistant conversation state into generic Agent Run requests with context assets, OpenAI-compatible function tools, governance refs, and memory scope.
- Provides a mock client so local development can run without a live Athena service.

## File Index

- `client.go`
  - Defines the `Client` interface, HTTP client, mock client, Agent Run request DTOs, context assets, tool declarations, and response read model.
- `client_test.go`
  - Verifies HTTP request mapping, authorization header behavior, and mock client responses.
