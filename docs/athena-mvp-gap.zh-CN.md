# Athena MVP 缺口清单

## 背景

Athena 当前已经具备一组 runtime foundation 能力：RuntimeContract、TaskTypeRegistration、HookBinding、System Truth lifecycle、projection boundary、runtime trace / usage / checkpoint readout、tool governance、Validation MCP 和 Control Plane 验证面。

fund assistant MVP 需要的是“业务应用可调用”的稳定 runtime API。当前 Athena 更偏验证和控制面，仍需补齐业务应用接入面。

## 已具备能力

- Runtime persistence 基础对象：run、step、lifecycle、trace、usage、projection、checkpoint safe metadata。
- RuntimeContract foundation 与 registered task type validator。
- Tool governance policy / decision API。
- Validation MCP deterministic tool invocation。
- Context assets 注入与 direct respond rich delivery read model。
- System Validation 页面可展示 runtime persistence readout。
- OpenAPI / Swagger 暴露已有控制面接口。

## MVP 必补能力

### 1. Goal-driven Agent Run API

需要新增业务应用可调用的 agent run API，而不是只依赖 chat/respond 或 control-plane validation run。

建议接口：

- `POST /api/agent/runs`
- `GET /api/agent/runs/:runID`
- `POST /api/agent/runs/:runID/resume`
- `POST /api/agent/runs/:runID/cancel`
- `GET /api/agent/runs/:runID/trace`

最小请求语义：

- `goal`
- `success_criteria`
- `constraints`
- `budget`
- `context_assets`
- `tools`
- `tool_choice`
- `memory_scope`
- `governance_policy_refs`

### 2. OpenAI-Compatible Tools / Tool Calls

需要对外兼容 OpenAI 风格：

- `tools`
- `tool_choice`
- assistant message `tool_calls`
- tool result messages
- streaming tool-call delta

Athena 内部仍保持 canonical tool contract，避免被单一供应商格式锁死。

### 3. Business Tool Registry / Execution Surface

fund assistant 需要注册自己的业务 tools：

- fund snapshot tool
- market snapshot tool
- portfolio context tool
- decision journal tool
- review task tool

Athena 需要提供：

- tool schema registry
- tool execution request/response contract
- tool timeout / retry / error contract
- tool trace
- governance pre-check
- business app callback 或 remote tool endpoint 支持

### 4. Memory / Context API

fund assistant 需要把用户画像、持仓摘要、决策日志、复盘结论作为可治理上下文注入，而不是写入 Athena core 业务表。

建议接口：

- `POST /api/memory/query`
- `POST /api/memory/write`
- `POST /api/context-assets/resolve`
- `POST /api/context-assets/assemble`

最小能力：

- query scoped memory
- write decision summary
- context compression
- trace memory read/write
- preserve app ownership

### 5. Trace Timeline API

已有 runtime trace readout，但 fund assistant 需要业务可读 timeline。

需要统一展示：

- agent goal
- plan / loop step
- model calls
- tool calls
- data provider calls
- memory reads/writes
- context assembly
- governance decisions
- generated decision matrix
- final report delivery

### 6. Built-in Basic Tools

MVP 需要至少有：

- HTTP fetch tool
- web/search tool 或可插拔 search provider
- calculator tool
- time / market calendar helper
- file/CSV import helper
- JSON/schema validation helper

金融业务 tool 仍由 fund assistant 注册，不进入 Athena core。

### 7. Docker Integration Contract

Athena 需要提供 Docker Compose 友好的配置：

- PostgreSQL
- Redis
- Athena API
- Athena web control plane
- healthcheck
- env example

fund assistant 通过环境变量配置 Athena base URL、auth token、PostgreSQL、Redis 和 provider keys。

## 不放入 Athena core 的内容

- 基金代码 / ETF 代码业务表。
- 用户持仓业务表。
- 基金净值、回撤、收益率、基金经理等业务字段。
- 稳健 / 均衡 / 激进方案业务模板。
- 金融数据 provider 的业务账号和授权配置。
- 投资决策日志业务表。

这些内容属于 `athena-fund-assistant`。

## 建议 Athena Issue 切分

1. Agent Run API foundation。
2. OpenAI-compatible tool call contract。
3. Remote/business tool registry。
4. Memory/context external API。
5. Agent trace timeline read API。
6. Built-in basic tools pack。
7. Docker Compose runtime profile with PostgreSQL and Redis。

