# 只读账户授权与同意审计

## 目标

本功能为 fund assistant 建立用户会话、只读 consent grant、scope、有效期、撤销和审计闭环。它允许用户明确授权 Athena 读取账户摘要和持仓快照，但不增加下单、资金转移、券商写入或自动交易能力。

## 当前实现

- 本地会话通过 `POST /api/auth/sessions` 签发；当前只允许 `ATHENA_FUND_LOCAL_AUTH_SUBJECT` 指定的主体，默认是 `demo-user`。
- 原始 Bearer token 只在签发响应中返回一次；内存和 PostgreSQL store 只保存 SHA-256 hash。
- consent grant 包含 `subject`、`audience`、只读 `scopes`、`revision`、`expires_at` 和 `revoked_at`。
- 当前 scope 包括账户摘要、持仓快照、决策日志，以及未来 provider / broker 只读同步预留项。
- Web 工作台自动建立本地会话，但账户读取授权必须由用户显式创建或撤销。
- 对话发起 Athena Agent Run 时只传 `user_id` 和非秘密 `consent_grant_ref`，不会传用户 Bearer token。
- `account_overview` remote tool 同时检查 Athena 服务身份、用户主体、audience、grant 状态、账户摘要 scope 和持仓快照 scope。
- grant 缺失、scope 缺失、过期、撤销、主体或 audience 不匹配时返回稳定拒绝码。
- 审计事件只保存 session/grant 引用、scope、allow/deny 和 grant revision，不保存 token、券商密钥或账户 payload。

## 持久化

未设置 `DATABASE_URL` 时使用线程安全的内存 store。设置后，API 会自动创建并使用：

- `authorization_sessions`
- `authorization_consent_grants`
- `authorization_audit_events`

该 migration 不包含 token、password、API key 或 brokerage credential 明文字段。

## 跨服务边界

fund assistant 使用 `ATHENA_FUND_REMOTE_TOOL_TOKEN` 校验 Athena remote callback 的 Bearer 服务身份。该 token 只来自 HTTP header，不进入工具参数、模型上下文或 trace。

remote tool catalog 只发布 `env://ATHENA_FUND_REMOTE_TOOL_TOKEN` 引用。Athena 在出站 HTTP 边界解析并注入 token，fund assistant 在 consent 校验前验证该服务身份；注册、trace 和 smoke artifacts 都不保存 token value。平台实现由 [Super-Sky/Athena#24](https://github.com/Super-Sky/Athena/issues/24) 承接。

## 验证

- `go test ./internal/authorization ./internal/server`
- `go test -race -count=1 ./internal/authorization`
- `go vet ./internal/authorization`
- `yarn build`（`apps/web`）
- `ATHENA_REPO=../Athena ./scripts/smoke_dual_service.sh`
- `ATHENA_REPO=../Athena ./scripts/smoke_dual_docker.sh`
- Browser smoke：本地 session 自动建立、grant 创建/撤销状态切换、对话发送和移动视口布局通过，console 无 warning/error。

服务端测试覆盖会话签发/撤销、跨用户拒绝、grant 创建/列表/撤销、服务身份拒绝、缺 scope 拒绝、授权成功、撤销后拒绝和原始凭据不落审计。双服务 smoke 额外覆盖错误服务 token 拒绝、正确 token + 有效 grant 通过、撤销后拒绝、对话 trace 回写和 artifacts no-leak。

## 非目标

- 真实 OAuth / OIDC 身份提供方
- 券商凭据托管
- 券商写接口、订单、自动交易或资金操作
- 绕过应用授权检查的前端直连
