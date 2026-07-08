# 双服务 MVP 执行计划

## 目标

把 `Athena` 和 `athena-fund-assistant` 推进到本地 Docker 可运行、可演示、可验证的完整 MVP。

- `Athena`：通用 Agent Runtime 底座。
- `athena-fund-assistant`：基金投研辅助业务应用。

两个项目保持技术栈一致：

- 后端：Go
- 前端：React + TypeScript + Vite
- 主数据库：PostgreSQL
- 缓存 / 临时状态 / 异步任务辅助：Redis
- 部署：Docker / Docker Compose

## 服务边界

### Athena 负责

- 目标驱动 agent loop。
- OpenAI-compatible `tools` / `tool_calls` 输入输出。
- Athena canonical tool contract。
- tool registry。
- trace timeline。
- memory / context API。
- governance gate。
- checkpoint / resume。
- 基础内置 tools。
- Control Plane / System Validation / runtime readout。

### athena-fund-assistant 负责

- InvestorProfile。
- Portfolio / PortfolioHolding。
- FundInstrument / FundSnapshot / MarketSnapshot。
- 中国基金/ETF 数据 provider。
- 美股 ETF/指数数据 provider。
- 基金体检。
- 稳健 / 均衡 / 激进三档决策矩阵。
- 决策日志。
- 复盘任务。
- 基金业务 UI。
- 金融场景治理规则。

## 不允许跨层

- Athena core 不保存基金业务对象。
- Athena core 不写死基金、ETF、仓位、净值、收益率等业务语义。
- fund assistant 不直接导入 Athena 内部 Go 包。
- 两者通过 API / SDK / tool contract 对接。

## MVP 阶段

### Phase 0：规划与边界冻结

交付：

- 双语文档规则。
- 产品边界。
- 数据源策略。
- 双服务架构。
- GitHub issue 分层。

### Phase 1：fund assistant 本地业务闭环

交付：

- Go API skeleton。
- React + Vite UI skeleton。
- PostgreSQL schema。
- Redis 配置。
- InvestorProfile / Portfolio / DecisionMatrix / DecisionJournal 模型。
- mock / CSV data provider。
- 三档方案生成。
- 决策日志与复盘任务。
- Docker Compose 本地启动。

验收：

- 用户可以录入画像与持仓。
- 系统可以用 mock/CSV 数据生成基金体检。
- 系统可以生成稳健 / 均衡 / 激进方案。
- 用户可以选择方案并写入决策日志。
- 系统可以生成复盘任务。

### Phase 2：真实数据 provider

交付：

- 中国基金/ETF provider。
- 美股 ETF/指数 provider。
- 数据 freshness / timezone / delay / license metadata。
- provider interface 与缓存层。
- 数据源失败 fallback。

验收：

- 至少一个中国数据路径可拉取真实基金或 ETF 数据。
- 至少一个美股数据路径可拉取真实 ETF 或指数数据。
- 每条数据保留 `source`、`fetched_at`、`market_time`、`timezone`、`delay`、`provider`、`license_terms`、`confidence`、`schema_version`。
- 如果使用临时 mock/CSV，UI 必须明确标记。

### Phase 3：Athena 对接

交付：

- fund app Athena client。
- fund data tools 注册形态。
- portfolio / decision journal tools。
- context asset 注入。
- memory read/write。
- governance evaluation。
- trace readout。

验收：

- fund app 可以发起 Athena agent run。
- Athena 可以调用 fund app 注册的 tools。
- 每次基金分析可以追踪 tools、数据、模型、治理检查和最终决策矩阵。

### Phase 4：端到端演示

交付：

- 双服务 Docker Compose。
- 一键启动文档。
- Seed data。
- 本地 demo 流程。
- 后台 trace 验收。

验收：

- `docker compose up` 后可启动 Athena、fund assistant、PostgreSQL、Redis。
- 用户完成一次基金体检和三档方案选择。
- 决策日志可见。
- Athena / fund app trace 可回看。

## 多子 Agent / 并行开发切分

建议按以下工作流并行：

- `Runtime Agent`：Athena agent loop、tool call、trace、memory、governance API。
- `Finance Domain Agent`：fund app 领域模型与决策矩阵。
- `Data Provider Agent`：中国与美股数据源 provider。
- `UI Agent`：fund assistant 前端。
- `Docker Agent`：双服务 Docker / Compose。
- `Governance Agent`：金融输出治理、数据授权标记、文档双语同步。

第一版工程实现采用 orchestrator + deterministic workers，暂不做多 agent 自治。

## 关键风险

- 免费数据源不等于可商用或可再分发。
- 中国基金数据的授权边界比美股 API 更不清晰。
- 美股数据必须处理 `America/New_York` 时区、非交易日和延迟。
- 不能把单一路径买卖结论作为默认输出。
- 百分比必须来自用户画像、组合约束、策略模板、历史数据或明确规则。

