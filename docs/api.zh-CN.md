# API 契约

## 范围

本文记录 athena-fund-assistant MVP 后端当前已经实现的本地 API。第一版 API 使用 Go 标准库 HTTP 服务，不依赖 Athena 内部 Go 包。

当前 API 属于 fund assistant 业务应用层，不属于 Athena core。Athena 未来通过 API / SDK / tool contract 调用这些业务能力。

## 通用约定

- 默认监听地址：`:8081`
- 可通过环境变量 `ATHENA_FUND_API_ADDR` 覆盖。
- 请求和响应均为 JSON。
- API 允许本地 Web 开发源访问，当前 CORS 覆盖 `localhost` / `127.0.0.1` 的端口化 origin。
- mock 数据必须在 trace 中显示 `mock_data_temporary=true`。
- 金融输出必须包含多方案、依据、风险、反证条件和复盘时间。

## `GET /healthz`

检查服务是否存活。

响应示例：

```json
{
  "status": "ok"
}
```

## `POST /api/analysis/fund`

根据用户画像、持仓和标的代码生成基金体检与三档决策矩阵。

请求字段：

- `instrument_code`：基金、ETF 或 mock provider 支持的标的代码。
- `profile`：用户风险画像。
- `portfolio`：用户手动录入持仓。

当前 mock provider 支持：

- `000001` / `CN-FUND-000001`
- `510300` / `CN-ETF-510300`
- `QQQ` / `US-ETF-QQQ`

响应字段：

- `profile`
- `portfolio`
- `fund_snapshot`
- `diagnosis`
- `decision_matrix`

`decision_matrix.trace` 当前包含：

- `data_provider`
- `data_source`
- `data_fetched_at`
- `market_time`
- `timezone`
- `license_terms`
- `confidence`
- `rule_evaluations`
- `governance_checks`
- `mock_data_temporary`

## `POST /api/journals`

保存用户选择的一个方案，并生成复盘任务。

请求字段：

- `matrix`：来自 `/api/analysis/fund` 的 `decision_matrix`。
- `selected_option_id`：用户选择的方案 ID。
- `user_notes`：用户备注。

响应字段：

- `journal`
- `review`

## 当前边界

- 当前 journal 使用内存存储，服务重启后会丢失。
- 当前 data provider 是 mock provider，不能作为生产行情。
- 当前 API 不做用户认证、资金托管、自动交易或券商下单。
- PostgreSQL、Redis、Athena agent run 对接和真实 provider 是后续实现项。
