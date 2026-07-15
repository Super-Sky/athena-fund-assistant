# Docs Index

## Language Policy

- `language-policy.zh-CN.md`
  - 中文文档与代码注释双语规范。
- `language-policy.en-US.md`
  - English documentation and bilingual code-comment policy.

Durable product and engineering docs should use paired files:

- `topic.zh-CN.md`
- `topic.en-US.md`

Existing single-language planning documents remain as seed drafts until they are converted into paired versions.

## Planning Docs

- `features/feature-read-only-account-consent.zh-CN.md`
  - 用户会话、只读 consent grant、scope、撤销、审计与 Athena 服务身份边界中文说明。
- `features/feature-read-only-account-consent.en-US.md`
  - English description of user sessions, read-only consent grants, scopes, revocation, audit, and the Athena service-identity boundary.
- `features/feature-fund-assistant-web.zh-CN.md`
  - 基金助手 Web MVP 当前实现的中文说明、边界和验收证据。
- `features/feature-fund-assistant-web.en-US.md`
  - English description, boundaries, and verification evidence for the current fund assistant web MVP.
- `features/feature-account-dashboard.zh-CN.md`
  - 账户收益看板第一片的中文说明、边界和后续持久化计划。
- `features/feature-account-dashboard.en-US.md`
  - English description, boundaries, and persistence follow-up plan for the first account dashboard slice.
- `features/feature-journal-persistence.zh-CN.md`
  - 决策日志与复盘任务 PostgreSQL 持久化、证据快照和运行边界中文说明。
- `features/feature-journal-persistence.en-US.md`
  - English description of PostgreSQL-backed decision journals, review tasks, evidence snapshots, and runtime boundaries.
- `features/feature-agent-workspace.zh-CN.md`
  - Agent 对话工作台、skill 选择、附件上传和 trace timeline 的中文说明。
- `features/feature-agent-workspace.en-US.md`
  - English description for the Agent workspace, skill selection, attachment upload, and trace timeline.
- `features/feature-csv-provider.zh-CN.md`
  - CSV 数据 provider、本地兜底边界、metadata 要求和验证证据中文说明。
- `features/feature-csv-provider.en-US.md`
  - English description of the CSV data provider, local fallback boundary, metadata requirements, and verification evidence.
- `features/feature-docker-compose.zh-CN.md`
  - Docker Compose MVP 运行方式、双服务 overlay 和验证状态中文说明。
- `features/feature-docker-compose.en-US.md`
  - English description of the Docker Compose MVP runtime, dual-service overlay, and verification status.
- `features/feature-preference-knowledge.zh-CN.md`
  - 用户偏好、agent.md、策略知识库、版本和治理边界中文说明。
- `features/feature-preference-knowledge.en-US.md`
  - English description of user preferences, agent.md, strategy knowledge base, revisions, and governance boundaries.
- `features/feature-financial-governance.zh-CN.md`
  - 金融建议输出治理门的中文规则、产品边界和验证证据。
- `features/feature-financial-governance.en-US.md`
  - English rules, product boundaries, and verification evidence for the financial-output governance gate.
- `api.zh-CN.md`
  - fund assistant MVP 后端当前已实现 API 的中文契约。
- `api.en-US.md`
  - English contract for the currently implemented fund assistant MVP backend API.
- `local-runtime.zh-CN.md`
  - fund assistant MVP 当前本地运行和 Docker Compose 中文说明。
- `local-runtime.en-US.md`
  - English local runtime and Docker Compose guide for the current fund assistant MVP.
- `provider-validation.zh-CN.md`
  - 信息获取 tool 和市场数据 provider 的先验证后编码中文规则。
- `provider-validation.en-US.md`
  - English validation-first rule for information tools and market-data providers.
- `platform-mvp-plan.zh-CN.md`
  - Athena + fund assistant 双服务 MVP 中文执行计划。
- `platform-mvp-plan.en-US.md`
  - English execution plan for the Athena + fund assistant dual-service MVP.
- `data-source-strategy.zh-CN.md`
  - 中国基金/ETF 与美股股票/ETF/指数/汇率/交易日历真实数据源中文方案。
- `data-source-strategy.en-US.md`
  - English real-data strategy for China funds/ETFs and US equities/ETFs/indices/FX/market calendars.
- `data-source-validation-snapshot.zh-CN.md`
  - 真实数据源本轮验证结果、provider 准入结论和下一步 provider 决策中文快照。
- `data-source-validation-snapshot.en-US.md`
  - English snapshot of real-data validation results, provider admission decisions, and next provider steps.
- `athena-mvp-gap.zh-CN.md`
  - Athena 底座为 fund assistant MVP 需要补齐的中文缺口清单。
- `athena-mvp-gap.en-US.md`
  - English gap list for Athena runtime capabilities required by the fund assistant MVP.
- `product-boundary.md`
  - Defines what the fund assistant is allowed to do and what it must avoid.
- `architecture.md`
  - Defines repository layering, domain modules, and Athena integration boundaries.
- `mvp-plan.md`
  - Defines the first product slice and acceptance criteria.
- `athena-integration.md`
  - Defines the expected Athena APIs, tool contracts, memory usage, and trace needs.
- `agent-workflows.md`
  - Defines the orchestrator and worker-agent workflow model.
- `data-governance.md`
  - Defines data-source, freshness, trace, and financial-output governance rules.
- `issue-plan.md`
  - Defines the first development issues and sequencing.
