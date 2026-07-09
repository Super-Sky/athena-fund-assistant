# 决策日志 PostgreSQL 持久化

## 目标

把用户选中的决策方案及其复盘任务保存到 PostgreSQL，使基金助手在 API 或容器重启后仍能读取决策依据、用户备注和复盘触发条件。

## 当前能力

- `journal.Store` 隔离内存与 PostgreSQL 实现。
- 配置 `DATABASE_URL` 时，API 启动阶段连接 PostgreSQL 并执行幂等 schema migration。
- journal 与 review task 使用独立表，并保留完整 JSON 业务快照。
- `POST /api/journals` 原子写入 journal 与首个 review task。
- `GET /api/journals/{journalID}` 和 `GET /api/reviews/{reviewID}` 提供读取接口。
- `/readyz` 检查当前 journal store，Docker Compose 用它作为 API 健康门禁。
- 未配置 `DATABASE_URL` 时，仅为直接开发和测试回退到内存存储。

## 边界

- 本切片不保存用户认证信息、券商凭据或交易指令。
- 本切片不接入 Redis；Redis 后续用于数据 provider 缓存和临时任务状态。
- PostgreSQL 只保存 fund assistant 业务对象，不把基金业务表写入 Athena。

## 验证

- journal 包单元测试。
- 可选 PostgreSQL 集成测试：设置 `ATHENA_FUND_PG_TEST_DSN` 后运行 `go test ./internal/journal`。
- Docker 验收：创建 journal，重启 API 容器，再通过读取接口确认数据仍存在。

