# 中美基金、ETF 与美股数据源策略

## 目标

MVP 需要真实基金/市场数据，但不能把授权不清晰的数据源默认为生产依赖。数据接入必须经过 provider interface，并且每条数据都保留来源、时间、授权和置信度 metadata。

美股数据必须作为一等数据域处理，而不是只作为“美股 ETF”的补充字段。国内用户持有的 QDII、纳指、标普、海外科技、美元债或美股主题基金，都需要底层美股个股、ETF、指数、汇率和交易日历数据参与分析。

## 必需 metadata

每条标准化数据必须保留：

- `source`
- `provider`
- `fetched_at`
- `market_time`
- `timezone`
- `delay`
- `license_terms`
- `confidence`
- `schema_version`
- `raw_payload_hash`

## 中国基金 / ETF 数据候选

### Tushare Pro

定位：优先候选，用于 A 股、指数、ETF、基金相关数据调研和 MVP provider。

依据：

- Tushare Pro 有公开文档和 token 机制。
- 官方文档说明 Pro 接口采用积分权限，不同接口和频次对应不同积分门槛。
- 文档也说明部分分钟数据和特色数据需要单独开权限，且分钟数据“只供策略研究和学习使用，不允许作为商业目的”。

风险：

- 不是完全无门槛免费 API。
- 具体接口的商用、展示、再分发边界需要逐项确认。
- MVP 可以支持用户自带 Tushare token，但不能默认宣称生产可商用。

建议：

- 第一版实现为 `tushare_provider`，要求用户提供 token。
- 默认用于研究、验证和个人本地运行。
- UI 和 trace 中记录 `license_terms=tushare_user_token_required`。

参考：

- https://tushare.pro/
- https://tushare.pro/document/1?doc_id=108
- https://tushare.pro/document/1?doc_id=290

### AKShare / Eastmoney / 天天基金路径

定位：开发验证候选，不作为默认生产数据源。

依据：

- AKShare 公募基金文档覆盖开放式基金实时数据、货币基金、历史净值等接口。
- AKShare 文档明确大量基金数据的目标地址来自东方财富 / 天天基金。

风险：

- AKShare 是开源数据接口库，不等于目标数据源授权清晰。
- 东方财富 / 天天基金页面或接口的抓取、展示、再分发条款需要单独确认。
- 生产 SaaS 默认使用该路径存在授权不确定性。

建议：

- 仅作为 `akshare_experimental_provider` 或本地开发 fallback。
- 任何由该路径取得的数据必须在 UI / trace 中标记为 experimental。
- 不把该路径作为商业部署默认 provider。

参考：

- https://akshare.akfamily.xyz/data/fund/fund_public.html
- https://quantapi.eastmoney.com/Manual?from=web

### 交易所 / 协会公开数据

定位：规则、统计、交易所基金数据机制参考；不作为第一版实时业务 API。

依据：

- 中国证券投资基金业协会提供公募基金市场统计数据。
- 深交所基金公司数据接口规范说明 ETF、LOF、分级基金等净值发布机制。
- 上证所信息网络历史数据接口说明书描述 Level-1、K 线和 CSV 历史数据形态。

风险：

- 这些资料更偏市场参与者、技术规范或统计报告，不一定适合作为普通 SaaS 的免费实时 API。
- 使用前需要确认服务申请、授权和数据展示规则。

建议：

- 用于 schema 设计、字段校验、市场统计引用。
- 不作为 MVP 的第一实时数据 provider。

参考：

- https://www.amac.org.cn/sjtj/
- https://docs.static.szse.cn/www/marketServices/technicalservice/notice/W020190408723859697048.pdf
- https://www.sseinfo.com/services/assortment/document/

## 美股股票 / ETF / 指数数据候选

### 美股数据范围

MVP 美股 provider 至少需要覆盖：

- 美股个股：例如大型科技股、行业龙头、ADR 或基金重仓股。
- 美股 ETF：例如标普、纳指、行业、债券、黄金、美元资产相关 ETF。
- 美股指数：例如 S&P 500、NASDAQ 100、Dow Jones、Russell 2000 等。
- 汇率：至少覆盖 USD/CNY，用于人民币持仓和美元资产的统一收益归因。
- 交易日历与时区：统一处理 `America/New_York`、美国节假日、半日交易、盘前盘后和数据延迟。

这些数据只服务于基金投研和组合分析，不用于自动交易、盘口撮合或券商交易指令。

### Alpha Vantage

定位：美股股票 / ETF / 指数第一优先 live provider 候选。

依据：

- 官方文档覆盖股票、ETF、共同基金、市场指数、外汇、技术指标等 API。
- Alpha Vantage 提供免费 API key 路径，适合 MVP 验证。
- 官方还提供面向 AI agent 的 MCP server 入口。

风险：

- 免费额度有限。
- 实时数据、高频数据、商用展示边界需要按条款确认。

建议：

- 第一版实现 `alpha_vantage_provider`。
- 默认用于美股个股 / ETF 日线、基础 quote、指数、USD/CNY 汇率和技术指标。
- 缓存到 Redis，避免超额度。

参考：

- https://www.alphavantage.co/documentation/
- https://www.alphavantage.co/

### Financial Modeling Prep

定位：美股股票 / ETF / mutual fund information 与补充行情 provider 候选。

依据：

- 官方文档提供 ETF & Mutual Fund Information API，并覆盖股票 quote / historical price 等市场数据接口。
- Pricing 文档说明 free plan 存在带宽限制。
- Pricing 文档明确 FMP 数据展示或再分发需要单独 Data Display and Licensing Agreement。

风险：

- 展示 / 再分发要求需要重点确认。
- 适合作为用户自带 API key 的可选 provider，不适合作为默认免费生产数据。

建议：

- 第二优先实现 `fmp_provider`。
- 在配置中要求用户明确接受 FMP terms。
- UI / trace 记录 `license_terms=fmp_terms_required`。

参考：

- https://site.financialmodelingprep.com/developer/docs
- https://site.financialmodelingprep.com/developer/docs/pricing
- https://site.financialmodelingprep.com/developer/docs/stable/information

### Tiingo

定位：美股 EOD、ETF、共同基金、新闻等候选 provider。

依据：

- 官方文档说明 Tiingo APIs 支持市场数据。
- Tiingo 产品页覆盖 EOD 股票、ETF、共同基金等。

风险：

- 免费账号、API 限额和商业展示条款需要确认。

建议：

- 作为可选 provider，不作为第一默认 provider。
- 重点用于 EOD 历史价格和新闻扩展。

参考：

- https://www.tiingo.com/documentation/
- https://www.tiingo.com/products/end-of-day-stock-price-data

### Nasdaq Data Link

定位：补充型数据集 provider。

依据：

- 官方文档说明 Nasdaq Data Link 提供 free 和 premium data，并支持 Streaming API、REST API、Tables API。

风险：

- 具体 dataset 是否免费、是否覆盖目标股票 / ETF / 指数、是否允许展示，需要逐个 dataset 确认。

建议：

- 用于补充宏观、指数、专题数据。
- 不作为第一默认行情 provider。

参考：

- https://docs.data.nasdaq.com/docs/getting-started
- https://www.nasdaq.com/solutions/data/nasdaq-data-link/api

### Stooq

定位：历史数据 fallback 候选。

依据：

- Stooq 提供免费历史数据下载页面。

风险：

- 自动化调用、商用展示、再分发条款不清晰。
- 不适合作为生产默认 provider。

建议：

- 仅作为本地研究 fallback 或 CSV 导入来源。

参考：

- https://stooq.com/db/h/

### Yahoo / yfinance

定位：不建议作为正式 provider。

依据：

- `yfinance` 文档明确其不隶属、不背书、未经 Yahoo 审核，只是使用公开 API，主要用于研究和教育。
- Yahoo 开发者页面并不提供当前可用的官方 Yahoo Finance 市场数据 API。

建议：

- 不作为生产 provider。
- 如用户本地实验使用，必须标记为 unofficial / research-only。

参考：

- https://ranaroussi.github.io/yfinance/
- https://developer.yahoo.com/api/

## MVP Provider 优先级

### 第一批

- `csv_provider`
- `mock_provider`
- `alpha_vantage_provider`
- `tushare_provider`

实现状态：`alpha_vantage_provider` 与 `tushare_provider` 已通过显式的用户 Key / 用户 Token 启动模式接入，但两者都不是默认 provider。每次启动服务前都会重新执行结构和 metadata validation。Alpha Vantage 的基准值使用 ETF 代理并明确标注；交易所日历能力有意保持不可用。Tushare 当前仅覆盖中国基金净值、沪深 300 和上交所日历。后续跨市场组合 provider 必须保留底层 provider metadata，不能把它隐藏掉。

### 第二批

- `fmp_provider`
- `tiingo_provider`
- `nasdaq_data_link_provider`

### 仅实验

- `akshare_experimental_provider`
- `stooq_csv_provider`
- `yfinance_unofficial_provider`

## Provider Interface 要求

Provider 必须实现：

- `GetFundSnapshot`
- `GetEquitySnapshot`
- `GetIndexSnapshot`
- `GetFXRate`
- `GetMarketCalendar`

需要凭据的 adapter 还必须实现 `ProviderName` 与 `ValidateCredentials`。不支持的能力必须返回明确错误，不能伪造交易日历、价格或汇率观测值。

Provider 返回值必须包含：

- normalized data
- raw source metadata
- freshness metadata
- provider confidence
- license marker
- cache key

## 缓存策略

- Redis 用于缓存 provider 响应、速率限制状态和异步 refresh 状态。
- PostgreSQL 保存标准化后的 snapshot、journal evidence snapshot 和用户选择。
- 不把未经授权的 raw payload 作为可再分发资产长期保存。
