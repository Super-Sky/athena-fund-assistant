# Agent 对话工作台

## 范围

本功能把基金助手从固定表单工作流推进到日常 Agent 工作台。用户可以选择 skill，上传图片或文件，发送自然语言请求，并查看附件状态和本地 trace timeline。

## 已实现

- `internal/domain/conversation.go`
  - 定义 skill、conversation session、message、attachment metadata 和 trace event。
- `internal/conversation/store.go`
  - 提供 conversation store interface、本地内存实现、上传目录、SHA256、大小限制、保留期和 pending/unsupported 状态。
- `GET /api/conversations/skills`
  - 返回可选 skill 列表。
- `POST /api/conversations`
  - 创建对话 session。
- `GET /api/conversations/{conversation_id}`
  - 读取对话详情、消息、附件和 trace。
- `POST /api/conversations/{conversation_id}/attachments`
  - 上传文件并返回 metadata。当前不解析附件，不把附件当作事实。
- `POST /api/conversations/{conversation_id}/messages`
  - 追加消息并写入本地 trace。Athena agent run 当前标记为 `pending`，等待真实 Athena client 接线。
- `GET /internal/tools/catalog`
  - 输出可注册到 Athena remote tool registry 的 fund assistant 工具清单。
- `POST /internal/tools/execute`
  - 执行 `remote_tool_execution.v1` callback，目前支持 `account_overview` 和 `fund_market_snapshot` 两个只读工具。
- `apps/web`
  - 展示 Agent 对话、skill selector、文件上传、消息列表、附件状态和 trace timeline。

## 上传边界

- 单文件上限：`10 MiB`。
- 默认保留期：`7 天`。
- 上传目录：`ATHENA_FUND_UPLOAD_DIR`，未设置时使用系统临时目录。
- 支持类型第一版包括 image、PDF、CSV、TXT 和 Excel MIME；未知类型会标记 `unsupported=true`。
- 未解析附件只能作为 metadata / context candidate，不得冒充已解析账单、事实或策略知识。

## Athena 边界

- 当前 UI 和 API 已具备发起 Agent run 的本地 contract 形态，但还不实际调用 Athena Agent Run API。
- fund assistant 已暴露 Athena remote tools callback；Athena 可以把只读业务工具注册到 remote registry 后回调本应用。
- trace 中的 `athena_agent_run=pending` 表示下一步会通过 Athena Agent Run API 真实发起 run，并把 remote tool result 回写到 conversation trace。
- 基金业务对象、附件文件和业务 tool 实现仍保留在 fund assistant，不写入 Athena core。
- 当前 remote tools 均为只读，`side_effect_level=none`，不执行自动交易或资金动作。

## 验证

- `go test ./...`
- `yarn build` in `apps/web`
- Browser smoke: 工作台、skill selector、上传入口和 trace timeline 可见。
- Server test: remote tool catalog、`account_overview`、`fund_market_snapshot` 和 unknown-tool error envelope。
