# Read-Only Account Authorization And Consent Audit

## Goal

This feature establishes the fund assistant's user-session, read-only consent-grant, scope, expiry, revocation, and audit loop. It lets a user explicitly allow Athena to read an account summary and holding snapshots without adding order placement, money movement, brokerage writes, or automatic trading.

## Current Implementation

- `POST /api/auth/sessions` issues a local session only for the subject configured by `ATHENA_FUND_LOCAL_AUTH_SUBJECT`, which defaults to `demo-user`.
- The raw bearer token is returned once in the issuance response. Memory and PostgreSQL stores retain only its SHA-256 hash.
- A consent grant contains `subject`, `audience`, read-only `scopes`, `revision`, `expires_at`, and `revoked_at`.
- Current scopes cover account summary, holding snapshot, decision journal, and reserved future provider/broker read-only synchronization.
- The web workspace bootstraps the local session, while account-read consent must be explicitly created or revoked by the user.
- Agent Run requests carry only `user_id` and the non-secret `consent_grant_ref`; they never carry the user's bearer token.
- The `account_overview` remote tool checks Athena service identity, user subject, audience, grant state, account-summary scope, and holding-snapshot scope.
- Missing grants/scopes, expiry, revocation, and subject/audience mismatch return stable denial codes.
- Audit events contain only session/grant references, scope, allow/deny, and grant revision. They do not contain tokens, brokerage secrets, or account payloads.

## Persistence

The API uses the thread-safe in-memory store when `DATABASE_URL` is unset. When configured, it automatically creates and uses:

- `authorization_sessions`
- `authorization_consent_grants`
- `authorization_audit_events`

The migration contains no plaintext token, password, API-key, or brokerage-credential column.

## Cross-Service Boundary

The fund assistant validates Athena remote callbacks with the Bearer service identity in `ATHENA_FUND_REMOTE_TOOL_TOKEN`. This token comes only from the HTTP header and never enters tool arguments, model context, or trace data.

The remote tool catalog publishes only the `env://ATHENA_FUND_REMOTE_TOOL_TOKEN` reference. Athena resolves and injects the token at the outbound HTTP boundary, and the fund assistant validates the service identity before consent checks. Registration, trace, and smoke artifacts never store the token value. [Super-Sky/Athena#24](https://github.com/Super-Sky/Athena/issues/24) owns the platform implementation.

## Verification

- `go test ./internal/authorization ./internal/server`
- `go test -race -count=1 ./internal/authorization`
- `go vet ./internal/authorization`
- `yarn build` in `apps/web`
- `ATHENA_REPO=../Athena ./scripts/smoke_dual_service.sh`
- `ATHENA_REPO=../Athena ./scripts/smoke_dual_docker.sh`
- Browser smoke: local session bootstrap, grant create/revoke state changes, conversation send, and the mobile viewport layout passed with no console warning/error.

Server tests cover session issue/revocation, cross-user denial, grant create/list/revoke, service-identity denial, missing-scope denial, successful authorization, post-revocation denial, and raw-credential exclusion from audit output. Dual-service smoke additionally covers wrong-service-token denial, correct-token plus active-grant success, post-revocation denial, conversation trace writeback, and artifact no-leak checks.

## Non-Goals

- A production OAuth / OIDC identity provider
- Brokerage credential custody
- Brokerage writes, orders, automatic trading, or money movement
- Frontend access that bypasses application authorization checks
