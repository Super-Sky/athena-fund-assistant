# 本地运行

## 目标

本文记录 athena-fund-assistant MVP 当前本地运行方式。第一版运行拓扑包含：

- React + TypeScript + Vite Web
- Go API
- PostgreSQL
- Redis

当前 API 使用内存 journal 和 mock provider；账户看板在 `DATABASE_URL` 存在时使用 PostgreSQL，在未设置时回退到内存 demo store。Redis 已进入 Docker 拓扑，供后续缓存、速率限制和异步刷新接入。Athena client 未配置 `ATHENA_BASE_URL` 时使用本地 mock，配置后调用外部 Athena Agent Run API。

API 启动前会先运行 provider validation。当前 mock provider 需要通过基金、个股、指数、USD/CNY 汇率和美股交易日历探针后才会开始监听端口。

## 直接运行 API

```bash
ATHENA_FUND_API_ADDR=:8081 go run ./cmd/api
```

健康检查：

```bash
curl http://127.0.0.1:8081/healthz
```

账户看板检查：

```bash
curl http://127.0.0.1:8081/api/accounts/demo-user/overview
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
ATHENA_BASE_URL=http://127.0.0.1:8080 ATHENA_AUTH_TOKEN=optional-token go run ./cmd/api
```

## 双服务 smoke

使用本仓脚本可以在本机启动 Athena、fund assistant 和一个 fake OpenAI-compatible 模型，验证完整本地链路：

```bash
ATHENA_REPO=/Users/maxt/Desktop/maxt/Athena-remote-tools ./scripts/smoke_dual_service.sh
```

脚本会验证：

- Athena `/healthz` 和 fund assistant `/healthz` 可访问。
- fund assistant 的 `/internal/tools/catalog` 能生成 `account_overview` / `fund_market_snapshot` remote tool 注册。
- Athena `/api/control-plane/remote-tools/:name` 接受两个只读工具。
- fake model 触发 `account_overview` tool call。
- Athena 通过 `remote_tool_execution.v1` 回调 fund assistant `/internal/tools/execute`。
- fund conversation message 通过 Athena client 得到 `athena_agent_run=ok` trace，并记录 `run_status=completed`、`tool_call_count=1`、`output_present=true`。

该 smoke 不需要真实模型 API key；它验证的是双服务 contract、tool registry、tool callback 和 trace 回写。真实模型供应商仍应通过 Athena 模型管理配置。

PostgreSQL store 集成测试：

```bash
ATHENA_FUND_PG_TEST_DSN='postgres://athena_fund:athena_fund@127.0.0.1:5433/athena_fund?sslmode=disable' \
  go test ./internal/account -run TestPostgresStoreOverviewAndReplaceHoldings -count=1
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

## 当前边界

- API 容器会读取 `DATABASE_URL` 并用于账户看板持久化；`REDIS_URL` 当前仍预留给后续缓存和异步任务。
- API 会读取 `ATHENA_FUND_UPLOAD_DIR` 作为附件上传目录；未设置时使用系统临时目录。
- API 会读取 `ATHENA_BASE_URL` 和可选 `ATHENA_AUTH_TOKEN`；未设置时使用 mock Athena client，便于单服务演示。
- 当前 mock 数据必须在 UI / trace 中继续标记为临时数据。
- 当前 Web 仍只调用 fund assistant API；fund assistant API 会在用户消息后通过 Athena client 发起 Agent Run，并通过 `/internal/tools/execute` 暴露只读 remote business tools 供 Athena 回调。
