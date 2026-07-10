# CSV 数据 Provider

## 范围

本功能把无 key、本地可验证的数据兜底接入 `internal/data.Provider`。它允许用户提供标准化 CSV 文件，支撑基金体检、三档决策矩阵、Athena remote tool 和本地演示，同时明确标记这不是授权实时行情源。

## 已实现

- `internal/data/csv_provider.go`
  - 支持从单个 CSV 文件或目录加载数据。
  - 支持 `fund` / `ETF` / `LOF`、`equity`、`index`、`fx`、`calendar` 五类标准化行。
  - 所有行必须包含 `source`、`provider`、`fetched_at`、`market_time`、`timezone`、`delay`、`license_terms`、`confidence`、`schema_version`。
  - 未提供 `raw_payload_hash` 时基于 CSV 原始行生成 SHA256 hash。
- `examples/market-data-sample.csv`
  - 覆盖中国 ETF / 指数、美股 ETF / 个股 / 指数、USD/CNY 汇率和中美交易日历样例。
- `cmd/api`
  - `ATHENA_FUND_PROVIDER=csv` 时启用 CSV provider。
  - `ATHENA_FUND_CSV_PATH` 指向 CSV 文件或目录。
  - API 启动前会验证 `510300`、`QQQ`、`AAPL`、`000300`、`NDX`、`USD/CNY`、`CN` / `US` 交易日历。
- `internal/domain.TraceSummary`
  - 增加 `data_boundary` 和 `temporary_data`，保留旧 `mock_data_temporary` 兼容字段。
- `apps/web`
  - Trace 区域显示 `data_boundary` 和通用 `temporary_data`。

## 边界

- CSV provider 只用于本地 MVP、演示、用户自备样本或临时兜底。
- CSV provider 不替代真实 provider 的授权、额度、字段和时区准入验证。
- 样例数据使用 `user_supplied_csv_for_local_mvp_not_licensed_live_feed`，不得被 UI、API 或文档描述成实时行情。
- CSV provider 不执行交易、不连接券商、不保存第三方账号凭证。

## 验证

- `go test ./internal/data`
- `go test ./...`
- `yarn build` in `apps/web`
- `git diff --check`
- CSV API smoke:

```bash
ATHENA_FUND_PROVIDER=csv \
ATHENA_FUND_CSV_PATH=examples/market-data-sample.csv \
ATHENA_FUND_API_ADDR=:18084 \
go run ./cmd/api
```

实际响应验证：

- `/healthz` 返回 `{"status":"ok"}`。
- `/api/analysis/fund` 返回 `csv_provider`、`temporary_data=true`、`data_boundary=user_supplied_csv_for_local_mvp_not_licensed_live_feed` 和 `conservative` / `balanced` / `aggressive` 三档方案。
