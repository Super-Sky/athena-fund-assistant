# Fund Assistant Web MVP

## 背景

本功能把基金助手从本地 API slice 推进到可交互的 Web MVP。第一版目标是让真实使用者可以在浏览器里录入风险画像和单标的持仓，生成基金体检、三档策略、数据 trace，并保存一条 decision journal。

## 规则

- Web 只属于 `athena-fund-assistant` 业务应用层，不进入 Athena core。
- Web 通过 HTTP 调用 Go API，不 import 后端内部包。
- 当前默认数据仍来自通过启动前 validation 的 mock provider。
- UI 必须显示 `mock_data_temporary`、provider、source、license、confidence、market time 和 fetched time。
- 输出必须保持多方案，不给单一路径绝对结论。
- 本功能不做自动交易、券商下单、资金托管或账户凭据存储。

## 实现

- `apps/web/` 新增 React + TypeScript + Vite 研究台。
- `internal/server` 增加本地开发 CORS，允许 `localhost` / `127.0.0.1` 的端口化 origin。
- `Dockerfile.web` 使用 Yarn lock 构建静态资源，并通过 nginx 代理 `/api` 与 `/healthz` 到 Compose 内的 `api` 服务。
- `docker-compose.yml` 新增 `web` 服务，默认暴露 `5173`。
- `docs/local-runtime.*.md` 与 `docs/api.*.md` 同步记录 Web、本地 CORS 和 Compose 运行方式。

## 交互链路

1. 用户在 Web 输入风险偏好、回撤约束、单标的上限和持仓信息。
2. Web 调用 `POST /api/analysis/fund`。
3. API 使用已验证的 provider 快照生成 diagnosis 和 conservative / balanced / aggressive 三档 matrix。
4. Web 展示策略卡、依据、风险、反证条件、复盘时间和 trace。
5. 用户选择一个方案后，Web 调用 `POST /api/journals`。
6. API 创建 journal 和 review task，Web 展示下一次复盘任务。

## 风险

- 当前 mock provider 不能作为生产行情或真实投资依据。
- PostgreSQL 和 Redis 已进入 Compose 拓扑，但 journal 仍是内存存储。
- Docker daemon 未运行时只能验证 `docker compose config`，不能完成本机镜像构建。
- 真实 provider 仍必须先通过 `cmd/providerprobe` 或等价 validation report，不能直接接入 UI 默认决策流。

## 验证

- `go test ./...`
- `mkdir -p build && go build -o build/athena-fund-api ./cmd/api`
- `cd apps/web && yarn build`
- `docker compose config`
- 内置浏览器 smoke：
  - 打开 `http://127.0.0.1:5173/`
  - 点击 `生成三档策略`
  - 确认三张策略卡出现
  - 选择均衡方案
  - 点击 `保存 journal`
  - 确认 review task 出现且页面无错误横幅

## Skill 结论

暂不新增专属 feature skill。原因是当前 Web MVP 仍是应用层第一版交互台，维护流程可以复用现有 `frontend-design`、`webapp-testing` 和本仓 provider validation 文档；等 UI 工作流、真实数据源和 Athena trace 接入稳定后，再沉淀专属维护 skill。
