# 基金驱动的 Athena Agent 演进路线图

## 背景

`athena-fund-assistant` 是 Athena 的第一个真实业务验证场景。它需要读取用户授权的账户摘要和带 freshness 的市场数据，使用只读业务 tools，形成稳健、均衡、激进三档建议，并保留可复盘证据。这个场景用于验证 Athena 的通用 Agent 能力，不能把基金、持仓、净值或交易语义写入 Athena core。

## 已收敛的边界

- Athena 的 PostgreSQL runtime records 是 run、trace、usage、audit 和后台读取的事实源。
- OpenTelemetry、Langfuse、Prometheus 等外部系统只接收安全、脱敏、可替换的投影。
- 基金助手拥有账户、数据 provider、用户偏好、策略知识、授权与金融治理；通过 API、OpenAI-compatible tools 和 remote tool contract 接入 Athena。
- 产品只提供研究与决策支持，不提供自动交易、订单写入或资金操作。

## 阶段与依赖

1. 完成既有 Athena 接入链：`Athena#7`、`#8`、`#9`、`#14`、`#10`、`#11`、`#12`，并通过双服务 smoke。
2. 并行推进基金账户、对话、知识与真实数据准备：`fund#15`、`#16`、`#17`、`#10`、`#11`。真实数据必须使用用户自有 key/token 完成 live validation 后才可成为默认路径。
3. 以基金分析驱动新增通用能力：`Athena#21` 观测投影、`Athena#22` 目标驱动执行控制与 Redis job、`Athena#23` 受治理的 pgvector memory retrieval。
4. 在可读取真实账户数据前先完成 `fund#30` read-only consent；在发布演示前启用 `fund#31` Promptfoo 金融评测门禁。

完整交付项、验收和组件准入矩阵维护在 `docs/platform-mvp-plan.zh-CN.md` 与对应英文版本；开发队列维护在 `docs/issue-plan.md`。

## 组件准入结论

- 现在引入：OpenTelemetry Collector、可选 Langfuse profile、Redis、Go-native queue、PostgreSQL + pgvector、Promptfoo。
- 通过接口预留：search provider、文件解析/OCR、sandbox、LiteLLM gateway。
- 延后：Temporal、专用向量数据库、Keycloak、Vault。它们只在可测的规模、工作流或企业接入需求出现后再立项。

## 验证

- 计划中列出的 `Athena#21`、`#22`、`#23` 与 `fund#30`、`#31`、`#32` 均已在 GitHub 创建并核验为开放状态。
- 中英文计划文档包含相同的原则、Phase 5/6、组件矩阵和依赖顺序。
- 文档中的双服务启动路径使用仓库相对路径，可通过绝对路径检查。

## 维护 Skill

不新增专用 skill。本路线图只收敛已有 runtime、provider、Docker、金融治理和 GitHub Issue 工作流；后续实现各能力时沿用对应 feature 文档与仓库技能即可。
