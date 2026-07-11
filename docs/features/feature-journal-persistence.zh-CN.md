# 决策日志与复盘持久化

## 范围

本功能让用户选定的决策方案可持久保存，同时不把金融业务对象写入 Athena。它会保留用户选择时的完整决策矩阵作为证据快照，并在同一操作中创建第一条复盘任务。

## 已实现

- `internal/journal.Store` 是带 context 的持久化边界，包含内存与 PostgreSQL 实现。
- PostgreSQL 以原子方式将 journal 和 review task 保存为 JSONB 快照，并使用内嵌的幂等 migration。
- `POST /api/journals` 创建用户选择的决策及其复盘任务。
- `GET /api/journals/{journal_id}` 与 `GET /api/reviews/{review_id}` 可供后续复盘读取持久化快照。
- `GET /readyz` 检查 journal store，供 Compose 判断就绪状态。

## 边界

- 用户仍需自行选择方案；本功能不交易，也不发送券商指令。
- 设置 `DATABASE_URL` 后使用 PostgreSQL 持久化；未设置时明确使用非持久化 fallback，仅用于本地开发和测试。
- 快照会保留已有的数据来源、新鲜度、规则、治理、风险、反证条件和复盘信息，不会把数据冒充为实时或已授权行情。
- journal 与账户的持久化关联、以及复盘时和后续市场观察数据的对比，仍是后续工作。

## 验证

- `go test ./internal/journal ./internal/server ./cmd/api`
- `go test -race ./internal/journal ./internal/server`
- `ATHENA_FUND_PG_TEST_DSN=... go test ./internal/journal -run TestPostgresStorePersistence -count=1`
- `docker compose config --quiet`

## 维护入口

本功能暂不新增 skill：持久化边界规模较小，可通过 `internal/journal/README.md` 和本文完整定位，尚不需要独立的高频维护流程 skill。
