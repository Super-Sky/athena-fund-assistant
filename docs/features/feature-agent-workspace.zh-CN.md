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
  - 追加消息并写入本地 trace，然后通过 Athena client 发起 Agent Run。未配置 `ATHENA_BASE_URL` 时使用 mock client，本地可演示；配置后调用外部 Athena `/api/agent/runs`。
- `GET /internal/tools/catalog`
  - 输出可注册到 Athena remote tool registry 的 fund assistant 工具清单。
- `POST /internal/tools/execute`
  - 执行 `remote_tool_execution.v1` callback。目前支持两个只读工具；`account_overview` 额外校验 Athena 服务身份和用户 consent scopes。
- `apps/web`
  - 以 Agent 对话作为默认首页，展示账户核心上下文、skill selector、文件上传、消息列表、附件状态和 trace timeline。
  - 账户、持仓、收益、策略分析、偏好、知识库和数据授权拆分为独立导航页面，仅保留核心数据与核心配置。
  - 使用分组 Lucide 导航和对话 composer，保留明确的附件操作，并以带无障碍名称的图标按钮发送消息；附件/trace 辅助栏不抢占对话主任务。

## 上传边界

- 单文件上限：`10 MiB`。
- 默认保留期：`7 天`。
- 上传目录：`ATHENA_FUND_UPLOAD_DIR`，未设置时使用系统临时目录。
- 支持类型第一版包括 image、PDF、CSV、TXT 和 Excel MIME；未知类型会标记 `unsupported=true`。
- 未解析附件只能作为 metadata / context candidate，不得冒充已解析账单、事实或策略知识。

## Athena 边界

- 当前 UI 和 API 已具备发起 Agent run 的 contract 形态，并通过 app-side Athena client 写回 run trace。
- fund assistant 已暴露 Athena remote tools callback；账户回调由 Athena 注入独立服务身份，并携带模型上下文中的安全 `consent_grant_ref`。
- 未配置 `ATHENA_BASE_URL` 时 trace 会显示 mock run；配置后 trace 会记录真实 Athena run_id、status 和 trace_available。
- 基金业务对象、附件文件和业务 tool 实现仍保留在 fund assistant，不写入 Athena core。
- 当前 remote tools 均为只读，`side_effect_level=none`，不执行自动交易或资金动作。
- Athena 的 remote-tool secret reference / header 注入由 `Super-Sky/Athena#24` 承接；本机与 Docker 双服务 smoke 均已验证错误 token 拒绝、正确 token 通过和 token no-leak。

## 验证

- `go test ./...`
- `yarn build` in `apps/web`
- 1440px 与 390px Browser smoke：对话默认打开，skill selector、上传入口、composer 和 trace timeline 可见；桌面与移动导航无横向页面溢出，浏览器 console 无错误。
- Server test: remote tool catalog、服务身份、授权成功、缺 scope、撤销后拒绝、`fund_market_snapshot` 和 unknown-tool error envelope。
- Server test: conversation message starts Athena mock run and writes `athena_agent_run=ok` trace。
- Dual-service smoke: `ATHENA_REPO=../Athena ./scripts/smoke_dual_service.sh` 与 `./scripts/smoke_dual_docker.sh` 均通过。
