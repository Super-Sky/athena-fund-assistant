# 本地运行

## 目标

本文记录 athena-fund-assistant MVP 当前本地运行方式。第一版运行拓扑包含：

- React + TypeScript + Vite Web
- Go API
- PostgreSQL
- Redis

当前 API 使用内存 journal 和 mock provider；账户看板在 `DATABASE_URL` 存在时使用 PostgreSQL，在未设置时回退到内存 demo store。Redis 已进入 Docker 拓扑，供后续缓存、速率限制和异步刷新接入。

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
- 当前 mock 数据必须在 UI / trace 中继续标记为临时数据。
- 当前 Web 只调用 fund assistant API；Athena 双服务联调将在 Athena API 对接后补齐。
