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
- 尝试运行 `ATHENA_REPO=../Athena-remote-tools ./scripts/smoke_dual_docker.sh`：
  - 已完成基础镜像拉取、Athena Dockerfile 解析、依赖下载和进入 Athena `go build` 阶段。
  - 本机 Docker 首次构建在 Athena `go build` 阶段长时间无输出，已人工中断并清理 `athena-fund-dual-smoke` compose 资源。
  - 后续复查发现新建 `docker run --rm alpine:3.20 sh -lc 'echo ok'` 和 `docker run --rm golang:1.23-alpine ...` 都停留在 `Created`，未进入 `Running`；这说明当前 Docker Desktop 新容器启动路径不健康，不是 fund assistant 业务代码的确定性失败证据。
  - 已移除停留在 `Created` 的测试容器，并终止挂住的 Docker CLI 进程。
  - 后续需要在 Docker Desktop 恢复、Docker cache 就绪或 CI 资源更稳定时重新运行完整 smoke，取得最终 pass 证据。
