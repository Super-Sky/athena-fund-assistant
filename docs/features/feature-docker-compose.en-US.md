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
  - Wires runtime dependencies through `DATABASE_URL`, `REDIS_URL`, `ATHENA_BASE_URL`, and optional `ATHENA_AUTH_TOKEN`.
- `docker-compose.dual.yml`
  - Adds Athena API, a fake OpenAI-compatible model, and dual-service network configuration.
  - Points the fund assistant API at container-local Athena: `http://athena-api:8080`.
  - Forwards optional `ATHENA_AUTH_TOKEN` when Athena enables authentication middleware.
  - Enables the CSV provider by default so the dual-service demo does not require third-party market-data keys.
- `scripts/fake_openai_tool_model.js`
  - Provides the OpenAI-compatible tool-call model double used by Docker smoke.
- `scripts/smoke_dual_docker.sh`
  - Builds and starts the dual-service Docker topology.
  - Registers the fake model and fund remote tools.
  - Verifies Athena Agent Run, the remote tool callback, fund conversation trace writeback, and CSV decision trace.

## Boundaries

- The dual-service overlay is for local MVP demos and contract verification, not cloud production deployment.
- The fake model is only for smoke tests and does not represent real model quality.
- The CSV provider is only a local fallback and must not masquerade as licensed real-time market data.
- Payment, subscriptions, brokerage account integration, and automatic trading are out of scope.
- Tokens do not belong in versioned content beyond the empty `.env.example` placeholder and never enter conversation traces.

## Verification

- `docker compose -f docker-compose.yml -f docker-compose.dual.yml config`
- `bash -n scripts/smoke_dual_docker.sh`
- `git diff --check`
- Attempted `ATHENA_REPO=../Athena-remote-tools ./scripts/smoke_dual_docker.sh`:
  - It completed base image pulls, Athena Dockerfile parsing, dependency download, and reached the Athena `go build` step.
  - Local Docker's first Athena build produced no output for an extended period during `go build`, so it was manually interrupted and the `athena-fund-dual-smoke` compose resources were cleaned up.
  - A later check showed that new `docker run --rm alpine:3.20 sh -lc 'echo ok'` and `docker run --rm golang:1.23-alpine ...` containers stayed in `Created` and never entered `Running`; this points to an unhealthy Docker Desktop new-container start path rather than deterministic fund assistant business-code failure.
  - The test containers left in `Created` were removed, and the hung Docker CLI processes were terminated.
  - A full smoke pass still needs to be rerun once Docker Desktop recovers, Docker cache is warm, or CI resources are more stable.
