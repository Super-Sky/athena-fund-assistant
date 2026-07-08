# 本地运行

## 目标

本文记录 athena-fund-assistant MVP 当前本地运行方式。第一版运行拓扑包含：

- Go API
- PostgreSQL
- Redis

当前 API 仍使用内存 journal 和 mock provider；PostgreSQL 与 Redis 已进入 Docker 拓扑，供后续持久化、缓存、速率限制和异步刷新接入。

API 启动前会先运行 provider validation。当前 mock provider 需要通过基金、个股、指数、USD/CNY 汇率和美股交易日历探针后才会开始监听端口。

## 直接运行 API

```bash
ATHENA_FUND_API_ADDR=:8081 go run ./cmd/api
```

健康检查：

```bash
curl http://127.0.0.1:8081/healthz
```

## Docker Compose

```bash
cp .env.example .env
docker compose up --build
```

默认端口：

- API: `8081`
- PostgreSQL: `5433`
- Redis: `6380`

## 当前边界

- API 容器会读取 `DATABASE_URL` 和 `REDIS_URL`，但当前代码尚未连接这两个服务。
- 当前 mock 数据必须在 UI / trace 中继续标记为临时数据。
- 当前 compose 只覆盖 fund assistant；Athena 双服务联调将在 Athena API 对接后补齐。
