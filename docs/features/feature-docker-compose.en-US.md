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
  - Verifies Athena Agent Run, the remote tool callback, fund conversation trace writeback, and CSV decision trace.

## Boundaries

- The dual-service overlay is for local MVP demos and contract verification, not cloud production deployment.
- The fake model is only for smoke tests and does not represent real model quality.
- The CSV provider is only a local fallback and must not masquerade as licensed real-time market data.
- Payment, subscriptions, brokerage account integration, and automatic trading are out of scope.

## Verification

- `docker compose -f docker-compose.yml -f docker-compose.dual.yml config`
- `bash -n scripts/smoke_dual_docker.sh`
- `git diff --check`
- Local base Compose runtime verification:
  - `docker compose up -d --build` completed successfully.
  - Web, API, PostgreSQL, and Redis reached healthy status.
  - `GET http://127.0.0.1:8081/readyz` returned `{"status":"ready"}` and the fund analysis endpoint returned a three-option matrix with a `passed` governance decision.
- `ATHENA_REPO=/Users/maxt/Desktop/maxt/Athena-remote-tools ./scripts/smoke_dual_docker.sh` passed:
  - Athena completed an Agent Run with the registered `account_overview` remote business tool.
  - A fund conversation recorded a completed Athena trace with one tool call and output present.
  - Fund analysis used `csv_provider`, marked the user-supplied local-data boundary and temporary-data state, and returned conservative, balanced, and aggressive options.
  - The first Athena image build can take several minutes with little BuildKit output; subsequent runs use cache and complete normally.
