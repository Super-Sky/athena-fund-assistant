# 本地运行

## 目标

本文记录 athena-fund-assistant MVP 当前本地运行方式。第一版运行拓扑包含：

- React + TypeScript + Vite Web
- Go API
- PostgreSQL
- Redis

当前 API 默认使用 mock provider；账户看板、决策日志、session、consent grant 和授权审计在 `DATABASE_URL` 存在时使用 PostgreSQL，在未设置时回退到明确的内存 demo store。Redis 已进入 Docker 拓扑，供后续缓存、速率限制和异步刷新接入。Athena client 未配置 `ATHENA_BASE_URL` 时使用本地 mock，配置后调用外部 Athena Agent Run API。

API 启动前会先运行 provider validation。当前 mock provider 需要通过基金、个股、指数、USD/CNY 汇率和美股交易日历探针后才会开始监听端口。CSV provider 需要同时通过中国 ETF / 指数、美股 ETF / 个股 / 指数、USD/CNY 汇率和中美交易日历探针。

## 直接运行 API

```bash
ATHENA_FUND_API_ADDR=:8081 go run ./cmd/api
```

健康检查：

```bash
curl http://127.0.0.1:8081/healthz
```

持久化就绪检查：

```bash
curl http://127.0.0.1:8081/readyz
```

账户看板检查：

```bash
SESSION_TOKEN="$(
  curl -fsS -X POST http://127.0.0.1:8081/api/auth/sessions \
    -H 'Content-Type: application/json' \
    -d '{"user_id":"demo-user"}' \
    | node -pe 'JSON.parse(require("fs").readFileSync(0, "utf8")).token'
)"

curl -H "Authorization: Bearer ${SESSION_TOKEN}" \
  http://127.0.0.1:8081/api/accounts/demo-user/overview
```

创建账户只读授权：

```bash
curl -fsS -X POST http://127.0.0.1:8081/api/consents \
  -H "Authorization: Bearer ${SESSION_TOKEN}" \
  -H 'Content-Type: application/json' \
  -d '{"audience":"athena-runtime","scopes":["fund.account.summary.read","fund.holding.snapshot.read"]}'
```

Agent 工作台 skill 检查：

```bash
curl http://127.0.0.1:8081/api/conversations/skills
```

Athena remote tools catalog 检查：

```bash
curl 'http://127.0.0.1:8081/internal/tools/catalog?base_url=http://127.0.0.1:8081'
```

连接真实 Athena：

```bash
ATHENA_BASE_URL=http://127.0.0.1:8080 \
ATHENA_AUTH_TOKEN=optional-token \
ATHENA_FUND_REMOTE_TOOL_TOKEN=separate-service-token \
go run ./cmd/api
```

使用用户提供 CSV 数据兜底：

```bash
ATHENA_FUND_PROVIDER=csv \
ATHENA_FUND_CSV_PATH=examples/market-data-sample.csv \
ATHENA_FUND_API_ADDR=:8081 \
go run ./cmd/api
```

CSV provider 是本地 MVP / 演示兜底，不是授权实时行情源。CSV 每行必须保留 `source`、`provider`、`fetched_at`、`market_time`、`timezone`、`delay`、`license_terms`、`confidence`、`schema_version`，样例文件使用 `user_supplied_csv_for_local_mvp_not_licensed_live_feed` 明确标记授权边界。

## 双服务 smoke（等待 Athena #24）

本仓历史脚本可以启动 Athena、fund assistant 和 fake OpenAI-compatible 模型；只读授权上线后，脚本需要等待 Athena #24 的服务身份 header 注入能力后再恢复完整通过：

```bash
ATHENA_REPO=/Users/maxt/Desktop/maxt/Athena-remote-tools ./scripts/smoke_dual_service.sh
```

脚本的目标验证项：

- Athena `/healthz` 和 fund assistant `/healthz` 可访问。
- fund assistant 的 `/internal/tools/catalog` 能生成 `account_overview` / `fund_market_snapshot` remote tool 注册。
- Athena `/api/control-plane/remote-tools/:name` 接受两个只读工具。
- fake model 触发 `account_overview` tool call。
- Athena 通过 `remote_tool_execution.v1` 回调 fund assistant `/internal/tools/execute`。
- fund conversation message 通过 Athena client 得到 `athena_agent_run=ok` trace，并记录 `run_status=completed`、`tool_call_count=1`、`output_present=true`。

该 smoke 不需要真实模型 API key。只读授权引入后，完整 remote callback 还要求 Athena 安全注入独立服务身份；该能力由 [Super-Sky/Athena#24](https://github.com/Super-Sky/Athena/issues/24) 跟踪。在 #24 合入并同步 smoke 脚本前，不能把历史 smoke 结果当作当前生产级服务身份验证证据。

PostgreSQL store 集成测试：

```bash
ATHENA_FUND_PG_TEST_DSN='postgres://athena_fund:athena_fund@127.0.0.1:5433/athena_fund?sslmode=disable' \
  go test ./internal/account ./internal/journal ./internal/authorization -count=1
```

## 直接运行 Web

```bash
cd apps/web
yarn install
yarn dev
```

Vite 默认监听 `http://127.0.0.1:5173`，并把 `/api` 与 `/healthz` 代理到 `http://127.0.0.1:8081`。

## Docker Compose

```bash
cp .env.example .env
docker compose up --build
```

默认端口：

- Web: `5173`
- API: `8081`
- PostgreSQL: `5433`
- Redis: `6380`

## 双服务 Docker Compose

使用 overlay 可以在同一个 Docker Compose 项目中启动 Athena、fund assistant、PostgreSQL、Redis、Web 和一个 fake OpenAI-compatible 模型：

```bash
ATHENA_REPO=../Athena-remote-tools \
docker compose -f docker-compose.yml -f docker-compose.dual.yml up --build
```

默认端口：

- Athena API: `8080`
- fund assistant Web: `5173`
- fund assistant API: `8081`
- fake OpenAI-compatible 模型: `18083`

双服务 overlay 会把 fund assistant API 的 `ATHENA_BASE_URL` 设置为 `http://athena-api:8080`，并默认启用 CSV provider：`ATHENA_FUND_PROVIDER=csv`、`ATHENA_FUND_CSV_PATH=/app/examples/market-data-sample.csv`。CSV 数据仍是本地 MVP / 演示兜底，不是授权实时行情源。

端到端 Docker smoke（在 Athena #24 合入并同步脚本后运行）：

```bash
ATHENA_REPO=../Athena-remote-tools ./scripts/smoke_dual_docker.sh
```

该脚本的目标是构建并启动双服务 Docker 拓扑，注册 fake 模型和 fund remote tools，验证带服务身份与 consent 的 Athena Agent Run、remote tool callback、fund conversation trace 回写，以及 CSV provider 决策 trace。首次构建 Athena 镜像可能较慢，后续会复用 Docker cache。

## 当前边界

- API 容器会读取 `DATABASE_URL` 并用于账户看板、journal/review 持久化；`REDIS_URL` 当前仍预留给后续缓存和异步任务。
- API 会读取 `ATHENA_FUND_UPLOAD_DIR` 作为附件上传目录；未设置时使用系统临时目录。
- API 会读取 `ATHENA_FUND_PROVIDER`；未设置或为 `mock` 时使用 `mock_provider`，为 `csv` 时读取 `ATHENA_FUND_CSV_PATH`。
- API 会读取 `ATHENA_BASE_URL` 和可选 `ATHENA_AUTH_TOKEN`；未设置时使用 mock Athena client，便于单服务演示。
- API 会读取 `ATHENA_FUND_LOCAL_AUTH_SUBJECT` 作为本地 session issuer 唯一允许的主体；默认是 `demo-user`。
- API 会读取 `ATHENA_FUND_REMOTE_TOOL_TOKEN` 校验 Athena callback 的 Bearer 服务身份。生产环境必须从 secret manager / Docker secret 注入，不能提交真实值。
- 双服务 Docker overlay 会额外读取 `ATHENA_REPO`、`ATHENA_DUAL_API_PORT`、`ATHENA_FAKE_MODEL_PORT`、`ATHENA_FUND_PROVIDER` 和 `ATHENA_FUND_CSV_PATH`。
- 当前 mock / CSV 数据必须在 UI / trace 中继续标记为临时或用户提供数据。
- 当前 Web 仍只调用 fund assistant API；fund assistant API 会在用户消息后通过 Athena client 发起 Agent Run，并通过 `/internal/tools/execute` 暴露只读 remote business tools 供 Athena 回调。
- Web 仅在内存中持有当前本地 session token；Agent Run、工具参数和 trace 只接收非秘密 `consent_grant_ref`。
