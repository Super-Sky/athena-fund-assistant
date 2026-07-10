# Provider 先验证后编码规则

## 目标

信息获取 tool、行情 provider、网页解析器、第三方 API adapter 都必须先验证后编码。也就是先稳定确认数据来源、字段结构、时区、更新频率、授权边界和错误形态，再把 provider 接入基金体检、决策矩阵或 Athena tool 调用。

这条规则适用于：

- 中国公募基金 / ETF provider
- 美股股票 / ETF / 指数 provider
- 汇率和交易日历 provider
- 新闻、公告、宏观数据和网页检索 tool
- 未来用户账号只读同步 adapter

## 接入门禁

任何真实 provider 进入业务链路前，必须完成：

1. 文档确认：记录官方文档、字段含义、免费额度、授权/展示/再分发限制。
2. 样本确认：保存至少一组成功样本、一组缺失/错误样本、一组非交易日或延迟样本。
3. 结构确认：把 raw payload 映射到标准结构，并确认必需 metadata 不为空。
4. 时区确认：美股必须明确 `America/New_York`，中国市场默认 `Asia/Shanghai`，汇率默认 `UTC` 或来源指定时区。
5. 授权确认：`license_terms` 必须写入 trace，不能用不清晰来源冒充生产数据。
6. 自动验证：必须通过 `internal/data.ValidateProvider` 或等价 provider validation 报告。
7. 失败策略：必须定义 provider 超时、限流、字段缺失、授权不足和非交易日返回的处理方式。

## 编码顺序

推荐顺序：

1. 写 provider validation probe。
2. 用真实 API key 或公开样本跑 probe。
3. 固化 normalized schema 和 metadata 映射。
4. 补 provider 单测 / contract test。
5. 再接入诊断、决策矩阵、UI 或 Athena tool。

禁止顺序：

1. 先写业务判断。
2. 临时猜测 API 字段。
3. 上线后再补 metadata / license / timezone。

## 当前代码落点

- `internal/data/provider.go`
  - provider 能力边界。
- `internal/data/validation.go`
  - provider validation 报告和探针。
- `internal/data/mock_provider.go`
  - 当前 mock provider，明确标记为临时数据。
- `internal/data/csv_provider.go`
  - 用户提供 CSV fallback provider，保留完整 metadata，并明确标记本地 MVP / 非授权实时行情边界。
- `cmd/providerprobe`
  - 对真实数据源执行验证探针，输出 JSON validation report。
- `internal/providerprobe`
  - Alpha Vantage / Tushare 等真实数据源的 validation-only 探针；不接入业务 provider。

## 验证命令

```bash
go run ./cmd/providerprobe --provider alpha_vantage
ALPHA_VANTAGE_API_KEY=... go run ./cmd/providerprobe --provider alpha_vantage
TUSHARE_TOKEN=... go run ./cmd/providerprobe --provider tushare
ATHENA_FUND_PROVIDER=csv ATHENA_FUND_CSV_PATH=examples/market-data-sample.csv go run ./cmd/api
```

命令输出 JSON 报告。任何必需探针失败时，命令返回非零退出码。

CSV provider 不替代真实 provider 准入；它只用于无 key、本地可验证、用户提供数据的 MVP 演示链路。CSV 行的 `license_terms`、`provider`、`source`、`market_time` 和 `timezone` 必须清晰，缺失 metadata 时加载失败。

当前验证快照见：

- `docs/data-source-validation-snapshot.zh-CN.md`
- `docs/data-source-validation-snapshot.en-US.md`

## 验收标准

- provider validation report 必须 `passed=true`。
- 每个被业务使用的数据点都必须保留 `source`、`fetched_at`、`market_time`、`timezone`、`delay`、`provider`、`license_terms`、`confidence`、`schema_version`。
- 真实 provider 未验证时，只能作为实验或手工研究路径，不能进入默认用户决策链。
