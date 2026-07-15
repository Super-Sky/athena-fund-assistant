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
3. 并行完成安全基础：`fund#30` read-only consent / scope / 撤销，`Athena#24` 出站 remote callback 身份，`Athena#25` 入站 app identity / tenant ownership / quota，以及 `Athena#21A` 的统一 `trace_id`、trace taxonomy、allowlist / 递归脱敏和采样。
4. 再由 `Athena#22` 完成目标驱动执行控制、稳定 stop reason 与 Redis job；`fund#31A` 以确定性金融 fixture 和 CI 阻断先验证关键规则。
5. 状态机稳定后由 `Athena#21B` 接入 OTLP Collector 与可选 Langfuse profile；`Athena#23` 复用授权和 tenant contract 完成 pgvector memory retrieval；`fund#31B` 补跨服务 trace 与可选模型评测。
6. `fund#37` 在 `fund#16` attachment metadata 和 `Athena#22` async contract 稳定后补附件存储、隔离解析 / OCR、citation 与保留策略。

完整交付项、验收和组件准入矩阵维护在 `docs/platform-mvp-plan.zh-CN.md` 与对应英文版本；开发队列维护在 `docs/issue-plan.md`。

## 组件准入结论

- 现在引入：OpenTelemetry Collector、可选 Langfuse profile、Redis、Go-native queue、PostgreSQL + pgvector、Promptfoo、provider-neutral service identity 与 quota contract。
- 通过接口预留：search provider、S3-compatible attachment storage、OCR、sandbox、LiteLLM gateway。
- 延后：Temporal、专用向量数据库、Keycloak、Vault。它们只在可测的规模、工作流或企业接入需求出现后再立项。

## 当前推进状态

- `fund#30` 已进入草稿 PR #35；真实账户读取仍需先完成 Athena `#24` 服务身份和双服务拒绝 / 通过 smoke。
- `fund#31A` 已进入草稿 PR #36，固定 fixture 的本地评测与三条 GitHub CI 门禁均通过；`#31B` 仍等待 Athena `#21`–`#23`。
- Athena `#25` 与 fund `#37` 已登记为开放 issue，尚未进入实现。

## 验证

- 计划中列出的 `Athena#21`–`#25` 与 `fund#30`、`#31`、`#32`、`#37` 均已在 GitHub 创建并核验为开放状态。
- 中英文计划文档包含相同的原则、Phase 5/6、组件矩阵和依赖顺序。
- 文档中的双服务启动路径使用仓库相对路径，可通过绝对路径检查。

## 维护 Skill

不新增专用 skill。本路线图只收敛已有 runtime、provider、Docker、金融治理和 GitHub Issue 工作流；后续实现各能力时沿用对应 feature 文档与仓库技能即可。
