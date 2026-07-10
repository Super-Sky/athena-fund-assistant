# 用户偏好与策略知识库

## 范围

本功能为基金助手 MVP 增加用户级 `agent.md`、长期偏好和基金策略知识库。它让业务应用可以管理偏好资产与策略模板，并把这些资产作为 Athena 上下文或 remote tool 数据提供出去，而不是写入 Athena core。

## 已实现

- `internal/domain/knowledge.go`
  - 定义 `PreferenceProfile`、`KnowledgeItem`、`KnowledgeRevision`、`KnowledgeAuditEvent` 和 `KnowledgeWorkspace`。
- `internal/preference/store.go`
  - 提供内存版 preference / knowledge store。
  - 默认为 `demo-user` 种子化 agent.md、偏好、仓位规则和审计事件。
  - 支持保存偏好草稿、激活偏好、保存知识草稿、激活知识和知识回滚。
- API:
  - `GET /api/users/{user_id}/knowledge`
  - `POST /api/users/{user_id}/preferences/drafts`
  - `POST /api/users/{user_id}/preferences/activate`
  - `POST /api/users/{user_id}/knowledge/drafts`
  - `POST /api/users/{user_id}/knowledge/{item_id}/activate`
  - `POST /api/users/{user_id}/knowledge/{item_id}/rollback`
- `apps/web`
  - 新增“用户偏好 · agent.md”和“策略知识库”面板。
  - 支持查看当前偏好、保存知识草稿、激活最新知识草稿和查看审计事件。

## 治理边界

- 草稿可以由 UI、未来 function call 或 MCP 自动保存。
- 正式启用必须显式调用 activation API；当前 MVP 用 schema validation + manual activation 作为治理门。
- 每个条目和版本保留 source、author、confidence、schema_version、governance_decision 和 audit。
- 当前 store 是内存实现，适合本地 MVP；后续需要 PostgreSQL 持久化和更细的权限/审批流。
- Athena 不持有基金业务对象；偏好和策略知识由 fund assistant 管理。

## 验证

- `go test ./internal/preference ./internal/server ./internal/domain`
- `go test ./...`
- `yarn build` in `apps/web`
- API smoke:
  - `GET /api/users/demo-user/knowledge` 返回种子偏好、知识条目、revision 和 audit。
  - `POST /api/users/demo-user/knowledge/drafts` 返回 draft revision。
  - `POST /api/users/demo-user/knowledge/{item_id}/activate` 将 draft 激活为 active。
