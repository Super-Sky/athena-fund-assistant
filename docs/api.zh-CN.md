# API 契约

## 范围

本文记录 athena-fund-assistant MVP 后端当前已经实现的本地 API。第一版 API 使用 Go 标准库 HTTP 服务，不依赖 Athena 内部 Go 包。

当前 API 属于 fund assistant 业务应用层，不属于 Athena core。Athena 未来通过 API / SDK / tool contract 调用这些业务能力。

## 通用约定

- 默认监听地址：`:8081`
- 可通过环境变量 `ATHENA_FUND_API_ADDR` 覆盖。
- 请求和响应均为 JSON。
- API 允许本地 Web 开发源访问，当前 CORS 覆盖 `localhost` / `127.0.0.1` 的端口化 origin。
- 未配置 `ATHENA_BASE_URL` 时使用本地 mock Athena client；配置后通过 `POST /api/agent/runs` 调用外部 Athena。
- `ATHENA_AUTH_TOKEN` 可选，会作为 Bearer token 发给 Athena。
- mock 数据必须在 trace 中显示 `mock_data_temporary=true`；mock / CSV 兜底数据必须在决策 trace 中显示 `temporary_data=true` 和明确的 `data_boundary`。
- 金融输出必须包含多方案、依据、风险、反证条件和复盘时间。
- 基金分析会返回确定性的 `governance` 结果。`blocked` 输出会被拒绝；`flagged` 输出仍会返回，但会保留来源/新鲜度披露。
- 除健康检查、本地 session 签发、skill 列表和 remote tool catalog 外，用户业务 API 需要 `Authorization: Bearer <session_token>`。
- 当前本地 session issuer 只接受 `ATHENA_FUND_LOCAL_AUTH_SUBJECT` 指定的主体，默认是 `demo-user`。生产身份提供方不在本期范围内。
- 原始 session token 只在签发时返回一次，服务端只保存 SHA-256 hash。

## `GET /healthz`

检查服务是否存活。

响应示例：

```json
{
  "status": "ok"
}
```

## `GET /readyz`

检查 journal 持久化边界是否可接受请求。配置的 store 不可用时返回 `503`；Docker Compose 使用它判断 API 是否就绪。

## `POST /api/auth/sessions`

为配置的本地 demo 主体签发 Bearer session。

请求字段：

- `user_id`：必须等于 `ATHENA_FUND_LOCAL_AUTH_SUBJECT`。
- `ttl_seconds`：可选，默认 24 小时，最大 7 天。

响应包含一次性的 `token` 和不含 token 的 `session` metadata。该接口是本地 MVP bootstrap，不是生产登录接口。

## `GET /api/auth/session`

验证 Bearer token 并返回当前 session metadata。响应不会返回原始 token。

## `DELETE /api/auth/sessions/current`

立即撤销当前 session。成功返回 `204`。

## `GET /api/consents`

返回当前用户的 consent grants，按最新优先稳定排序。grant 只包含安全引用、scope、revision、有效期和撤销状态。

## `POST /api/consents`

创建面向 `athena-runtime` 的只读 consent grant。

请求示例：

```json
{
  "audience": "athena-runtime",
  "scopes": [
    "fund.account.summary.read",
    "fund.holding.snapshot.read"
  ]
}
```

`ttl_seconds` 可选，默认 30 天，最大 90 天。scope 必须以 `.read` 结尾。

## `POST /api/consents/{grant_ref}/revoke`

撤销属于当前用户的 consent grant。首次撤销将 grant revision 加一；撤销后 remote tool 会返回 `grant_revoked`。

## `GET /api/accounts/{user_id}/overview`

读取用户账户收益看板。

当前本地 demo 用户：

- `demo-user`

Bearer session 的 subject 必须与路径中的 `user_id` 一致，否则返回 `subject_mismatch`。

响应字段：

- `account`：本地用户账户身份、展示名、本位币和认证模式。
- `holdings`：账户持仓快照，包含市场、币种、份额、成本、现价、`fx_to_base`、本位币市值、未实现收益、占比和 `data_authorization`。
- `total_market_value`
- `total_cost_value`
- `total_pnl`
- `total_pnl_pct`
- `recent_operation_pnl`
- `performance_trend`
- `recent_operations`
- `trace`

`trace` 当前包含：

- `provider`
- `source`
- `fetched_at`
- `market_time`
- `timezone`
- `license_terms`
- `confidence`
- `schema_version`
- `mock_data_temporary`
- `read_only_sync_available`
- `warnings`

本地未设置 `DATABASE_URL` 时使用内存 demo store；Docker / `DATABASE_URL` 环境会使用 PostgreSQL store。当前行情和收益输入仍为 demo/mock 数据，不能冒充真实券商账户或真实收益。

## `POST /api/accounts/{user_id}/holdings`

替换用户手动录入的账户持仓，并重新计算账户概览。

请求字段：

- `holdings`：`AccountHoldingSnapshot[]`

每条持仓必须包含：

- `instrument_code`
- `market`
- `currency`
- `units`
- `cost_basis`
- `current_price`
- `fx_to_base`
- `data_authorization`
- `metadata`

当前接口只表示手动记录和本地计算，不执行交易，不连接券商下单。

## `GET /api/conversations/skills`

返回 Agent 工作台可选 skill。

响应字段：

- `items`：skill 列表，每个 skill 包含 `id`、`name`、`description`、`tool_names` 和 `enabled`。

当前内置 skill：

- `fund_research`
- `portfolio_review`
- `document_intake`

## `POST /api/conversations`

创建一条对话 session。

请求字段：

- `user_id`
- `skill_id`
- `title`

响应为 `ConversationDetail`，包含 `session`、`messages`、`attachments` 和 `trace`。

`messages`、`attachments` 和 `trace` 始终返回 JSON 数组；没有记录时返回 `[]`，不会返回 `null`。

## `GET /api/conversations/{conversation_id}`

读取对话详情、消息、附件 metadata 和 trace timeline。

## `POST /api/conversations/{conversation_id}/attachments`

上传文件并返回附件 metadata。

请求类型：`multipart/form-data`

字段：

- `file`
- `user_id`

上传边界：

- 单文件最大 `10 MiB`。
- 默认保留期 `7 天`。
- `ATHENA_FUND_UPLOAD_DIR` 可配置上传目录；未设置时使用系统临时目录。
- 当前只生成 metadata、SHA256、`pending_parse` / `unsupported` 状态，不解析附件内容。
- 未解析附件不能作为已确认事实、账单或策略知识。

## `POST /api/conversations/{conversation_id}/messages`

追加一条工作台消息。

请求字段：

- `role`
- `content`
- `skill_id`
- `attachment_ids`
- `consent_grant_ref`：可选的非秘密只读授权引用。需要账户工具时由 Web 工作台传入。

响应返回更新后的 `ConversationDetail`。服务会保存消息，随后通过 Athena client 发起一次通用 Agent Run，并把 run status、run_id、trace_available 和 stop_reason 写入对话 trace。未配置 `ATHENA_BASE_URL` 时该调用由 mock client 完成，方便本地演示。

Agent Run 请求会把业务语义转换为通用 Athena 输入：

- `goal`：用户消息。
- `context_assets`：conversation ID、skill ID、attachment IDs、`user_id` 和安全的 `consent_grant_ref`；附件仍是 metadata-only。
- `tools` / `enabled_tools`：OpenAI-compatible function tools，目前包含 `account_overview` 和 `fund_market_snapshot`。
- `governance_refs`：无自动交易、无收益承诺、数据来源 metadata 必填。
- `constraints`：禁止自动交易、禁止券商下单、必须包含风险和反证条件、禁止单一路径绝对结论。

`athena_agent_run` trace metadata 当前包含：

- `run_id`
- `run_status`
- `trace_available`
- `stop_reason`
- `tool_call_count`
- `output_present`
- `consent_contract`
- `consent_grant_ref`
- `authorization_subject`

## `GET /internal/tools/catalog`

返回 fund assistant 暴露给 Athena remote tool registry 的工具注册建议。

查询参数：

- `base_url`：可选。传入后会生成完整 `endpoint`，例如 `http://127.0.0.1:8081/internal/tools/execute`；不传时返回相对路径 `/internal/tools/execute`。

响应字段：

- `contract_version`：当前为 `remote_tool_execution.v1`。
- `app_id`：当前为 `athena-fund-assistant`。
- `items`：可注册到 Athena 的 remote tool 列表。

每个 `items[]` 都包含 `auth.type=bearer` 与 `auth.secret_ref=env://ATHENA_FUND_REMOTE_TOOL_TOKEN`。catalog 只发布 secret reference；真实 token 由 Athena 和 fund assistant 各自的运行时环境注入，不进入注册 JSON。

当前只读工具：

- `account_overview`：读取用户账户概览、持仓、近期操作和收益趋势。
- `fund_market_snapshot`：读取基金 / ETF 快照，并保留 source、provider、fetched_at、market_time、timezone、delay、license、confidence 和 schema_version。

`account_overview` 声明要求 `user_id` 和 `consent_grant_ref`，并要求：

- `fund.account.summary.read`
- `fund.holding.snapshot.read`

所有当前工具均为 `side_effect_level=none`，不执行交易、不连接券商下单、不移动资金。

## `POST /internal/tools/execute`

执行 Athena `remote_tool_execution.v1` callback。该接口面向 Athena remote adapter，不是前端用户 API。

当授权服务启用时，请求必须携带 `Authorization: Bearer <service_token>`，且与 `ATHENA_FUND_REMOTE_TOOL_TOKEN` 一致。该服务 token 不能出现在工具参数、模型上下文或 trace 中。

请求字段：

- `contract_version`
- `request_id`
- `tool_call_id`
- `registration_id`
- `app_id`
- `tool_name`
- `arguments`
- `attempt`
- `metadata`

成功响应会回传相同 `request_id` 和 `tool_call_id`，并返回：

- `status=ok`
- `content`：JSON 字符串。

错误响应使用同一 envelope，包含：

- `status=error`
- `error.code`
- `error.message`
- `error.retryable`

支持参数：

- `account_overview`：`{"user_id":"demo-user","consent_grant_ref":"grant_..."}`；两个字段均必填。
- `fund_market_snapshot`：`{"instrument_code":"QQQ"}`；`instrument_code` 必填。

边界：

- 该接口只暴露 fund assistant 的业务工具实现，不导入 Athena 内部 Go 包。
- `account_overview` 会校验服务身份、subject、audience、grant 有效期/撤销状态以及账户摘要和持仓快照两个 scope。
- 授权拒绝使用 `error.code=authorization_denied`，并在 metadata 返回安全的 `authorization_code`、`consent_grant_ref`、revision 和 required scope。
- 返回内容仍需按 metadata 判断真实 / mock 数据来源。
- 未知工具例如下单类 `place_order` 会返回 `unknown_tool`，不会执行任何资金动作。

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
- `data_boundary`
- `temporary_data`
- `rule_evaluations`
- `governance_checks`
- `mock_data_temporary`

响应还包含 `governance`：

- `decision`：`passed`、`flagged` 或 `blocked`。
- `checks`：按规则给出的状态和安全说明。

当生成输出为 `blocked` 时，接口会返回 `422`。阻断范围包括收益承诺措辞、自动交易措辞、单一路径绝对指令、方案数量不足，或无推导依据的仓位调整。风险、反证条件、复盘时间、来源或新鲜度缺失会以 `flagged` 披露返回。

## `GET /api/users/{user_id}/knowledge`

读取用户长期偏好、`agent.md`、策略知识库、版本记录和审计事件。

响应字段：

- `preference`：用户风险偏好、表达偏好、默认策略层级、偏好/禁用资产、复盘频率、`agent_md`、active revision 和治理信息。
- `items`：策略知识条目，包含标题、分类、内容、标签、状态、active revision、source、author、confidence、schema_version 和 governance decision。
- `revisions`：偏好或知识条目的不可变版本记录。
- `audit`：草稿保存、激活、回滚等审计事件。

## `POST /api/users/{user_id}/preferences/drafts`

保存用户偏好 / `agent.md` 草稿，不立即启用。

请求字段：

- `risk_preference`
- `communication_style`
- `default_strategy_level`
- `preferred_assets`
- `blocked_assets`
- `review_frequency_days`
- `agent_md`
- `source`
- `author`
- `confidence`
- `summary`

## `POST /api/users/{user_id}/preferences/activate`

显式激活一个偏好 revision。

请求字段：

- `revision_id`

## `POST /api/users/{user_id}/knowledge/drafts`

保存一条策略知识草稿，不立即启用。

请求字段：

- `item_id`：可选；为空时创建新条目。
- `title`
- `category`
- `content`
- `tags`
- `source`
- `author`
- `confidence`
- `summary`

## `POST /api/users/{user_id}/knowledge/{item_id}/activate`

显式激活某条知识的一个 revision。

请求字段：

- `revision_id`

## `POST /api/users/{user_id}/knowledge/{item_id}/rollback`

回滚到某条知识的历史 revision，并记录审计事件。

请求字段：

- `revision_id`

## `POST /api/journals`

保存用户选择的一个方案，并生成复盘任务。

请求字段：

- `matrix`：来自 `/api/analysis/fund` 的 `decision_matrix`。
- `selected_option_id`：用户选择的方案 ID。
- `user_notes`：用户备注。

响应字段：

- `journal`
- `review`

系统会把 matrix 保存为不可变的证据快照。设置 `DATABASE_URL` 时，journal 和 review task 会在一个 PostgreSQL 事务中创建，并可跨 API 重启读取；未设置时明确使用非持久化本地 fallback。

## `GET /api/journals/{journal_id}`

按 ID 返回不可变决策日志快照；不存在时返回 `404`。

## `GET /api/reviews/{review_id}`

按 ID 返回生成的复盘任务；不存在时返回 `404`。

## 当前边界

- 当前 preference / knowledge store 使用内存存储，服务重启后会回到 demo seed；PostgreSQL 持久化和权限审批是后续项。
- 当前 account overview 在 `DATABASE_URL` 存在时使用 PostgreSQL 持久化；未设置时使用内存 demo store。
- 当前 session、consent grant 和授权审计在 `DATABASE_URL` 存在时使用 PostgreSQL；未设置时使用非持久化内存 store。
- 当前 data provider 是 mock provider，不能作为生产行情。
- 当前 API 只有本地 demo subject 的 Bearer session bootstrap，不包含生产 OAuth/OIDC；不做资金托管、自动交易或券商下单。
- Redis、附件解析/OCR/PDF/CSV parser、journal/review 与账户的持久化关联和真实 provider 是后续实现项。
