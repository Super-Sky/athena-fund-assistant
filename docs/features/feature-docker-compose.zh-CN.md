# Docker Compose MVP Runtime

## 范围

本功能提供基金助手 MVP 的本地 Docker 运行方式，并增加可选双服务 overlay，让 Athena 和 athena-fund-assistant 可以在同一个 Docker Compose 项目中联调演示。

## 已实现

- `Dockerfile.api`
  - 构建 Go API 容器。
  - 将 `examples/` 复制到运行镜像，供 CSV provider 在容器内读取样例数据。
- `Dockerfile.web`
  - 构建 React + TypeScript + Vite 前端，并用 Nginx 提供静态资源和 API 代理。
- `docker-compose.yml`
  - 启动 fund assistant Web、API、PostgreSQL 和 Redis。
  - 通过 `DATABASE_URL`、`REDIS_URL`、`ATHENA_BASE_URL` 配置运行时依赖。
- `docker-compose.dual.yml`
  - 增加 Athena API、fake OpenAI-compatible 模型和双服务网络配置。
  - 将 fund assistant API 指向容器内 Athena：`http://athena-api:8080`。
  - 默认启用 CSV provider，避免双服务演示依赖第三方市场数据 key。
- `scripts/fake_openai_tool_model.js`
  - 提供 Docker smoke 使用的 OpenAI-compatible tool-call 模型替身。
- `scripts/smoke_dual_docker.sh`
  - 构建并启动双服务 Docker 拓扑。
  - 注册 fake 模型和 fund remote tools。
  - 验证 Athena Agent Run、remote tool callback、fund conversation trace 回写和 CSV 决策 trace。

## 边界

- 双服务 overlay 面向本地 MVP 演示和 contract 验证，不是云生产部署。
- fake 模型只用于 smoke，不代表真实模型质量。
- CSV provider 只用于本地兜底，不冒充授权实时行情。
- 不包含支付、订阅、券商账号集成或自动交易能力。

## 验证

- `docker compose -f docker-compose.yml -f docker-compose.dual.yml config`
- `bash -n scripts/smoke_dual_docker.sh`
- `git diff --check`
- 本机基础 Compose 运行时验证：
  - `docker compose up -d --build` 已成功完成。
  - Web、API、PostgreSQL 和 Redis 均进入 healthy 状态。
  - `GET http://127.0.0.1:8081/readyz` 返回 `{"status":"ready"}`，基金分析接口返回三档 matrix，且治理结论为 `passed`。
- `ATHENA_REPO=../Athena ./scripts/smoke_dual_docker.sh` 已通过；也可指向待验证的 Athena 功能 worktree：
  - Athena 成功完成 Agent Run，并调用已注册的 `account_overview` remote business tool。
  - fund 对话记录了 completed Athena trace，包含一次 tool call 和已存在的输出。
  - fund 分析使用 `csv_provider`，正确标记用户提供本地数据的边界与临时数据状态，并返回稳健、均衡、激进三档方案。
  - Athena 首次镜像构建在 BuildKit 中可能连续数分钟没有输出；后续运行会复用缓存并正常完成。
