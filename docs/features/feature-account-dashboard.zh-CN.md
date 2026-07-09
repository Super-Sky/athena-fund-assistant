# 账户收益看板

## 范围

本功能把 fund assistant 从一次性基金分析页推进到账户型应用。第一片实现提供本地 demo 用户、账户级持仓快照、总收益、近期操作收益、收益趋势、持仓列表和来源 trace。

## 已实现

- `internal/domain/account.go`
  - 定义 `UserAccount`、`AccountHoldingSnapshot`、`AccountOperationRecord`、`AccountPerformancePoint` 和 `AccountOverview`。
- `internal/account/store.go`
  - 提供账户 store interface 和本地 `MemoryStore`，内置 `demo-user`。
- `GET /api/accounts/{user_id}/overview`
  - 返回账户首页看板读模型。
- `POST /api/accounts/{user_id}/holdings`
  - 接收手动录入持仓并重新计算账户收益。
- `apps/web`
  - 首页展示账户总市值、总收益、近期操作收益、收益趋势和持仓结构。

## 边界

- 当前账户数据仍为本地内存 + mock/demo 数据，`trace.mock_data_temporary=true`。
- 当前不保存券商账号、券商凭证或下单能力。
- 账号授权同步只作为后续只读方向，当前 `read_only_sync_available=false`。
- CNY / USD 持仓通过 `fx_to_base` 转换到账户本位币，避免把美股和 A 股未标注地混在一条时间线。

## 后续

- 接入 PostgreSQL schema，覆盖用户、账户、持仓快照、操作记录、journal/review 关联。
- 将 journal/review 从内存 store 迁移到持久化 store。
- 将账户持仓与真实数据 provider 连接，替换 mock/demo 价格和汇率。
- 与 Athena remote tools 对接，让 Agent 可读取账户概览和写入决策日志。

## 验证

- `go test ./...`
- `yarn build` in `apps/web`
