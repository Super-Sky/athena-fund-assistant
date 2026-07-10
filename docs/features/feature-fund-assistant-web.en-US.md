# Fund Assistant Web MVP

## Background

This feature moves the fund assistant from a local API slice to an interactive web MVP. The first goal is to let a real tester enter a risk profile and one holding, generate a fund diagnosis, compare three decision paths, inspect data trace, and save a decision journal entry.

## Rules

- The web app belongs to the `athena-fund-assistant` business application layer, not Athena core.
- The web app calls the Go API over HTTP and must not import backend internals.
- Current default data still comes from the mock provider that passes startup validation.
- The UI must show `mock_data_temporary`, provider, source, license, confidence, market time, and fetched time.
- Financial output must stay multi-option and avoid a single absolute conclusion.
- This feature does not implement automatic trading, brokerage order placement, custody, or brokerage credential storage.

## Implementation

- `apps/web/` adds the React + TypeScript + Vite research console.
- `internal/server` adds local-development CORS for port-qualified `localhost` / `127.0.0.1` origins.
- `Dockerfile.web` builds static assets from the Yarn lock and serves them through nginx, proxying `/api` and `/healthz` to the Compose `api` service.
- `docker-compose.yml` adds the `web` service on port `5173` by default.
- `docs/local-runtime.*.md` and `docs/api.*.md` record the web, local CORS, and Compose runtime paths.

## Interface Principles

- The page is ordered as account overview, Agent conversation, user preferences and knowledge, then strategy generation, so the most frequently reviewed and acted-on information appears first.
- A cool-gray neutral base and one blue primary action preserve focus. Source, governance, and temporary-data state stay visible through low-noise labels.
- Metrics and holdings use a continuous divided hierarchy instead of turning every item into a heavy card. Trace remains complete but reads as supporting information.
- Desktop retains a compact input-and-output split. Mobile collapses to one column; the strategy output must never be compressed sideways or create horizontal page scrolling.

## Interaction Flow

1. The user enters risk preference, drawdown constraints, single-instrument cap, and holding data in the web app.
2. The web app calls `POST /api/analysis/fund`.
3. The API uses a validated provider snapshot to generate diagnosis plus conservative / balanced / aggressive decision options.
4. The web app displays strategy cards, evidence, risks, invalidation conditions, review timing, and trace.
5. After the user selects one option, the web app calls `POST /api/journals`.
6. The API creates a journal entry and review task, and the web app displays the next review task.

## Risks

- The current mock provider must not be treated as production market data or real investment evidence.
- PostgreSQL and Redis are present in the Compose topology, but journal storage is still in memory.
- When the Docker daemon is not running, local validation can only run `docker compose config`, not image builds.
- Real providers must still pass `cmd/providerprobe` or an equivalent validation report before they enter the default UI decision flow.

## Verification

- `go test ./...`
- `mkdir -p build && go build -o build/athena-fund-api ./cmd/api`
- `cd apps/web && yarn build`
- `docker compose config`
- In-app browser smoke:
  - Open `http://127.0.0.1:5173/`
  - Click `生成三档策略`
  - Confirm three strategy cards render
  - Select the balanced option
  - Click `保存 journal`
  - Confirm the review task appears with no error banner
  - At 390px width, confirm account metrics, holdings, conversation, and strategy output use one column with no horizontal page scrolling

## Skill Decision

No dedicated feature skill is added yet. The current web MVP is still a first application-layer workflow, and maintenance can reuse the existing `frontend-design`, `webapp-testing`, and provider-validation docs. A dedicated maintenance skill should be considered after the UI workflow, real data providers, and Athena trace integration stabilize.
