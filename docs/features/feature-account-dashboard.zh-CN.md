# 账户收益看板

## 范围

本功能把 fund assistant 从一次性基金分析页推进到账户型应用。第一片实现提供本地 demo 用户、账户级持仓快照、总收益、近期操作收益、收益趋势、持仓列表和来源 trace。

## 已实现

- `internal/domain/account.go`
  - 定义 `UserAccount`、`AccountHoldingSnapshot`、`AccountOperationRecord`、`AccountPerformancePoint` 和 `AccountOverview`。
- `internal/account/store.go`
  - 提供账户 store interface 和本地 `MemoryStore`，内置 `demo-user`。
- `internal/account/postgres_store.go`
  - 提供 PostgreSQL store、schema bootstrap、demo seed、持仓替换和趋势持久化。
- `GET /api/accounts/{user_id}/overview`
  - 返回账户首页看板读模型。
- `POST /api/accounts/{user_id}/holdings`
  - 接收手动录入持仓并重新计算账户收益。
- `apps/web`
  - 首页展示账户总市值、总收益、近期操作收益、收益趋势和持仓结构。

## 边界

- 本地无 `DATABASE_URL` 时使用内存 demo store；Docker / DATABASE_URL 环境使用 PostgreSQL store。
- 当前账户行情仍为 mock/demo 数据，`trace.mock_data_temporary=true`。
- 当前不保存券商账号、券商凭证或下单能力。
- 用户 session 与可撤销账户读取 consent 已实现；真实券商/账户同步仍是后续只读方向，因此 `read_only_sync_available=false`。
- CNY / USD 持仓通过 `fx_to_base` 转换到账户本位币，避免把美股和 A 股未标注地混在一条时间线。

## 后续

- 将 journal/review 从内存 store 迁移到持久化 store，并关联账户/持仓。
- 将账户持仓与真实数据 provider 连接，替换 mock/demo 价格和汇率。
- 在 Athena #24 补齐服务身份 header 注入后，完成受 consent 保护的跨服务账户读取联调。

## 验证

- `go test ./...`
- `ATHENA_FUND_PG_TEST_DSN=... go test ./internal/account -run TestPostgresStoreOverviewAndReplaceHoldings -count=1`
- `yarn build` in `apps/web`
