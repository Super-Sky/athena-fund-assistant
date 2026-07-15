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
- 标准化 OpenTelemetry trace 投影、脱敏和可选观测后端导出。
- 通用执行预算、截止时间、停止条件和异步任务契约。

### athena-fund-assistant 负责

- InvestorProfile。
- Portfolio / PortfolioHolding。
- FundInstrument / FundSnapshot / MarketSnapshot。
- 中国基金/ETF 数据 provider。
- 美股股票/ETF/指数/汇率/交易日历数据 provider。
- 基金体检。
- 稳健 / 均衡 / 激进三档决策矩阵。
- 决策日志。
- 复盘任务。
- 基金业务 UI。
- 金融场景治理规则。
- 基于真实数据、用户授权和投资约束的领域评测集。

## 不允许跨层

- Athena core 不保存基金业务对象。
- Athena core 不写死基金、ETF、仓位、净值、收益率等业务语义。
- fund assistant 不直接导入 Athena 内部 Go 包。
- 两者通过 API / SDK / tool contract 对接。

## 演进原则

- **基金场景验证底座，而不是污染底座**：每项 Athena 能力都先由基金分析的明确需求推动，但必须以通用 goal、tool、context、trace 或 governance 契约交付。
- **Athena 的持久化 runtime records 是事实源**：PostgreSQL 内的 run、step、trace、usage、audit 供产品后台、授权审计和数据保留使用；外部观测系统仅接收脱敏投影。
- **先验证后编码**：市场数据、搜索、文件解析和外部工具必须先确认授权、字段、时区、限额、失败形态与可观测字段，再接入用户决策流程。
- **不以固定轮数终止 agent**：运行必须基于成功条件、预算、截止时间、等待外部数据、人工确认、不可恢复错误或治理拒绝等 stop reason 结束。
- **所有建议均可复盘**：结论必须关联数据来源、时间、工具调用、用户偏好、策略依据、风险和失效条件；不包含自动交易能力。

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
- 美股股票/ETF/指数/汇率/交易日历 provider。
- 数据 freshness / timezone / delay / license metadata。
- provider interface 与缓存层。
- 数据源失败 fallback。

验收：

- 至少一个中国数据路径可拉取真实基金或 ETF 数据。
- 至少一个美股数据路径可拉取真实个股、ETF 或指数数据，并能提供 USD/CNY 汇率或明确标记汇率缺失。
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
- run budget、deadline、stop reason 与异步恢复。

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

### Phase 5：基金驱动的 Agent Runtime 强化

本阶段以“用户询问组合是否需要调整”为统一验收场景。Agent 必须读取已授权账户摘要与偏好，取得带 freshness 的市场数据，选择只读业务 tools，生成三档方案，并留下可解释的全过程 trace。

交付：

- Athena 通用 agent loop：success criteria、budget、deadline、tool retry、waiting / terminal stop reason、checkpoint / resume。
- OpenTelemetry 分两段交付：先完成统一 `trace_id`、七类 runtime trace、allowlist / 递归脱敏与强制采样，再在执行状态稳定后接入 `OpenTelemetry Collector` 和可选 `Langfuse` 自托管 Docker profile。
- run / step / model / tool / memory / governance / remote callback 的统一 trace ID 与 correlation IDs。
- Redis 实际接入缓存、并发/速率限制、幂等锁和异步 job；长任务先使用 Go-native queue，复杂跨天工作流另行评估 Temporal。
- `pgvector` 知识与记忆检索切片，先复用 PostgreSQL，不新增独立向量数据库。
- 内置通用 tools：HTTP fetch、search provider adapter、calculator、time / market-calendar、file schema validation；基金领域 tool 继续由 fund assistant 远程注册。

验收：

- 同一个 `run_id` 可在 Athena 后台和可选观测后端按 trace ID 关联查看，且不泄露凭据、原始敏感持仓或未脱敏 prompt。
- 正常 run、工具错误、超时、治理拒绝、等待补数和 resume 都有稳定 stop reason 与 trace。
- Redis 不可用时，系统以明确降级或失败语义响应，不静默伪造缓存/异步成功。

### Phase 6：基金可信决策与持续评测

交付：

- 用户账户认证、token/session、read-only 数据授权、工具 scope、授权撤销和审计事件；券商同步只读且在单独授权后启用。
- `Promptfoo` 评测仓内配置与 CI 命令：先以确定性关键用例阻断发布，再接入 Athena trace 与可选模型评测；覆盖真实数据缺失、数据陈旧、工具失败、单一路径结论、保证收益措辞、缺风险/失效条件、百分比无依据、越权账户读取等案例。
- 文件解析、OCR 和网页搜索作为受限插件：大小、类型、外连域名、超时、来源和引用全受治理；不可信执行进入隔离 sandbox。
- 模型网关保持可选：先维持 Athena provider abstraction；多提供商路由、统一预算和虚拟 key 有明确需求后再接 LiteLLM profile。

验收：

- 每次基金建议在进入 UI 前通过金融治理与回归评测；失败用例阻断发布而非仅记录。
- 用户能查看并撤销数据授权，撤销后所有对应 remote tools 被治理层拒绝。
- 任何数据、检索、附件或策略结论均可从决策日志追溯到授权和来源 metadata。

## 组件准入矩阵

| 能力 | 当前选择 | 准入阶段 | 不选择或延后原因 |
| --- | --- | --- | --- |
| Agent trace / LLM eval | OpenTelemetry Collector + 可选 Langfuse | Phase 5 | Athena trace 仍为事实源，避免业务后台依赖第三方数据模型。 |
| Cache / queue | Redis + Go-native queue | Phase 5 | 已有 Docker Redis；Temporal 仅在跨天、人工审批和复杂恢复出现后评估。 |
| Knowledge retrieval | PostgreSQL + pgvector | Phase 5 | 先避免引入 Qdrant / Milvus / Weaviate 的复制与运维面。 |
| LLM gateway | Athena provider abstraction；LiteLLM 可选 profile | Phase 6 后 | 未出现多租户模型路由、虚拟 key 或统一成本结算前不增加 Python 服务。 |
| Continuous eval | Promptfoo | Phase 6 | 用例属于 fund app，但结果要反哺 Athena runtime contract。 |
| Auth / secrets | 应用 JWT/OAuth、Docker secrets；后续外部 secrets manager | Phase 6 | Keycloak / Vault 在企业 SSO 或多环境密钥轮转明确后再引入。 |
| Sandbox | 受限 Docker executor | Phase 6 | 仅在需要执行不可信脚本或复杂文件处理时启用。 |
| Dedicated vector DB | 暂不引入 | 后续 | 只有 pgvector 容量、延迟或召回能力成为可测瓶颈时再立项。 |

## Issue 依赖与执行顺序

1. **现有 Athena 接入链**：`Athena#7` → `#8` → `#9` → `#14` → `#10` → `#11` → `#12`。先完成通用 run、tool、remote callback、built-in tools、memory、trace、Docker 的合并与双服务 smoke。
2. **现有基金业务链**：`fund#15`、`#16`、`#17` 与 `fund#10`、`#11` 并行推进，但真实 provider 仍需用户自有 key/token 的 live validation 才能进入默认路径。
3. **安全基础并行切片**：`fund#30` 先完成身份、read-only consent、scope、撤销和 remote tool 拒绝；同时 `Athena#21A` 只完成统一 `trace_id`、trace taxonomy、allowlist / 递归脱敏和采样，不接 Langfuse 业务能力。
4. **执行与评测切片**：`Athena#22` 在安全 trace 基础上完成目标评估、预算、稳定 stop reason、PostgreSQL 状态真相和 Redis dispatch；`fund#31A` 并行增加确定性 fixture 与 CI 阻断。
5. **观测与记忆收口**：`Athena#21B` 在 #22 状态机稳定后接 OTLP Collector 与可选 Langfuse profile；`Athena#23` 复用 `fund#30` 的 ownership / consent 契约实现 pgvector retrieval；`fund#31B` 再加入跨服务 trace 与可选模型评测。

上述 Athena 能力只能依赖通用 runtime 契约，不可读取基金表；基金可信能力通过 remote tools 调用 Athena，不改写 Athena core。

每项任务在开始实现前都要在对应仓创建或更新 canonical GitHub Issue；跨仓依赖在 Issue 中使用 `Refs`，不在未完成时使用 `Closes`。

## 多子 Agent / 并行开发切分

建议按以下工作流并行：

- `Runtime Agent`：Athena agent loop、tool call、trace、memory、governance API。
- `Finance Domain Agent`：fund app 领域模型与决策矩阵。
- `Data Provider Agent`：中国基金 / ETF 与美股股票 / ETF / 指数 / 汇率 / 交易日历 provider。
- `UI Agent`：fund assistant 前端。
- `Docker Agent`：双服务 Docker / Compose。
- `Governance Agent`：金融输出治理、数据授权标记、文档双语同步。

第一版工程实现采用 orchestrator + deterministic workers，暂不做多 agent 自治。

## 关键风险

- 免费数据源不等于可商用或可再分发。
- 中国基金数据的授权边界比美股 API 更不清晰。
- 美股数据必须处理 `America/New_York` 时区、非交易日、半日交易、延迟和 USD/CNY 汇率。
- 不能把单一路径买卖结论作为默认输出。
- 百分比必须来自用户画像、组合约束、策略模板、历史数据或明确规则。
