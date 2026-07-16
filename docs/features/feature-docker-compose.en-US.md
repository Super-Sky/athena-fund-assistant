# Docker Compose MVP Runtime

## Scope

This feature provides the local Docker runtime path for the fund assistant MVP and adds an optional dual-service overlay so Athena and athena-fund-assistant can be demonstrated and verified in one Docker Compose project.

## Implemented

- `Dockerfile.api`
  - Builds the Go API container.
  - Copies `examples/` into the runtime image so the CSV provider can read sample data in containers.
- `Dockerfile.web`
  - Builds the React + TypeScript + Vite frontend and serves it through Nginx with API proxying.
- `docker-compose.yml`
  - Starts the fund assistant web app, API, PostgreSQL, and Redis.
  - Wires runtime dependencies through `DATABASE_URL`, `REDIS_URL`, and `ATHENA_BASE_URL`.
- `docker-compose.dual.yml`
  - Adds Athena API, a fake OpenAI-compatible model, and dual-service network configuration.
  - Points the fund assistant API at container-local Athena: `http://athena-api:8080`.
  - Enables the CSV provider by default so the dual-service demo does not require third-party market-data keys.
- `scripts/fake_openai_tool_model.js`
  - Provides the OpenAI-compatible tool-call model double used by Docker smoke.
- `scripts/smoke_dual_docker.sh`
  - Builds and starts the dual-service Docker topology.
  - Registers the fake model and fund remote tools.
  - Verifies wrong-service-token denial, correct-token plus consent success, post-revocation denial, fund conversation trace writeback, artifact no-leak checks, and CSV decision trace.

## Boundaries

- The dual-service overlay is for local MVP demos and contract verification, not cloud production deployment.
- The fake model is only for smoke tests and does not represent real model quality.
- The CSV provider is only a local fallback and must not masquerade as licensed real-time market data.
- Payment, subscriptions, brokerage account integration, and automatic trading are out of scope.
- `ATHENA_FUND_REMOTE_TOOL_TOKEN` must come from local `.env` or a production secret source and no real value may be committed. The catalog and Athena registration retain only the `env://ATHENA_FUND_REMOTE_TOOL_TOKEN` reference.
- The dual-service overlay makes the Athena and fund API health checks fail closed when the token is empty without blocking lifecycle commands such as `docker compose config/down/ps/logs`.
- Athena enables debug observability during dual-service smoke. The script exports container logs and control-plane JSON and scans them together with host artifacts for credential leaks.

## Verification

- `docker compose -f docker-compose.yml -f docker-compose.dual.yml config`
- `bash -n scripts/smoke_dual_docker.sh`
- `git diff --check`
- `ATHENA_REPO=../Athena ./scripts/smoke_dual_docker.sh` passed:
  - Compose started Athena, the fund API/web app, PostgreSQL, Redis, and the fake model.
  - A wrong token returned `service_auth_denied`; the correct token plus an active grant completed `account_overview`.
  - The fund conversation wrote back a completed Athena trace; a revoked grant returned `authorization_denied`.
  - Smoke artifacts, container logs, Athena remote trace, and control-plane JSON did not contain service-token or user-session-token values.
  - The CSV provider continued to report `temporary_data=true` and conservative/balanced/aggressive options.
