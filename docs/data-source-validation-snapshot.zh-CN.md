# 数据源验证快照

## 范围

本快照记录 2026-07-10 对基金助手 MVP 真实数据源的验证状态。它不是法律意见，只用于工程决策：哪些数据源可以进入 validation probe，哪些只能作为实验路径，哪些可以进入默认业务链路。

## 结论

- 中国公募基金 / ETF 第一候选：`tushare_provider`，但必须由用户提供 `TUSHARE_TOKEN`，并且按接口逐项确认积分权限、频率和展示/商用边界。
- 美股 ETF / 个股 / 指数 / FX 第一候选：`alpha_vantage_provider`，但必须由用户提供 `ALPHA_VANTAGE_API_KEY`，并且遵守免费额度、缓存和展示条款。
- FMP、Tiingo、Nasdaq Data Link 适合作为可选用户 key provider，不作为默认免费 provider。
- AKShare / Eastmoney / 天天基金路径只作为实验和本地 fallback，不作为商业 SaaS 默认 provider。
- Stooq 历史 CSV 可作为候选 fallback，但本机验证遇到 TLS 连接失败；需要在目标部署网络重新验证后才能进入 probe。
- 当前默认业务 provider 仍必须是 `mock_provider`，或显式启用的 `csv_provider` 本地兜底；两者都必须继续明确标记临时或用户提供数据。

## 本机 probe 结果

命令：

```bash
go run ./cmd/providerprobe --provider alpha_vantage
go run ./cmd/providerprobe --provider tushare
```

结果：

- `alpha_vantage`：失败。Alpha Vantage 返回官方 demo-key 限制信息，提示需要申请 free API key；demo key 不能证明 live schema 可用，需要配置真实 API key 后重跑。
- `tushare`：失败。未设置 `TUSHARE_TOKEN`，probe 正确返回 `TUSHARE_TOKEN is required for live validation`。

这两个失败结果不是业务阻断，而是“未通过 live provider 准入”的证据。它们说明真实 provider 不能进入默认决策链路。

## MVP provider 决策

### 默认链路

- 保留 `mock_provider`。
- `csv_provider` 已作为无网络 fallback 接入 `internal/data.Provider`，通过 `ATHENA_FUND_PROVIDER=csv` 和 `ATHENA_FUND_CSV_PATH` 显式启用。
- 默认链路必须在 UI / API trace 中展示 `license_terms`、`provider`、`source`、`confidence` 和临时 / 用户提供数据标记。

### CSV fallback

- 覆盖：标准化 CSV 行可覆盖中国 ETF / 指数、美股 ETF / 个股 / 指数、USD/CNY 汇率和中美交易日历。
- 样例：`examples/market-data-sample.csv`。
- 验证：启动 API 前会用 `internal/data.ValidateProvider` 检查 `510300`、`QQQ`、`AAPL`、`000300`、`NDX`、`USD/CNY`、`CN` / `US` 交易日历。
- 边界：CSV 数据必须视为用户提供或本地演示数据，不是授权实时行情源；`license_terms` 不得为空，样例值为 `user_supplied_csv_for_local_mvp_not_licensed_live_feed`。

### 用户 key 链路

- `alpha_vantage_provider`
  - 覆盖：美股 ETF profile、quote、daily price、USD/CNY FX。
  - 准入：`cmd/providerprobe --provider alpha_vantage` 全部通过。
  - 缓存：必须接 Redis 缓存，避免免费额度压力。
- `tushare_provider`
  - 覆盖：中国公募基金基础信息、基金净值、A 股指数日线。
  - 准入：`cmd/providerprobe --provider tushare` 全部通过。
  - 限制：只在用户 token 和权限通过时启用。

### 实验链路

- `akshare_experimental_provider`
- `stooq_csv_provider`
- `fmp_provider`
- `tiingo_provider`
- `nasdaq_data_link_provider`

实验链路不得冒充生产数据，必须在 trace 中写入 experimental 或 terms-required 标记。

## 准入门槛

真实 provider 进入基金体检、决策矩阵、Athena remote tools 或 UI 默认展示前，必须满足：

- provider probe `passed=true`。
- 每条数据保留 `source`、`provider`、`fetched_at`、`market_time`、`timezone`、`delay`、`license_terms`、`confidence`、`schema_version`、`raw_payload_hash`。
- 美股数据使用 `America/New_York`，FX 使用源定义时区或 `UTC`，中国市场使用 `Asia/Shanghai`。
- 对限流、超时、字段缺失、授权不足、非交易日和延迟数据有明确失败策略。
- 不把任何真实 provider 输出用于自动交易。
