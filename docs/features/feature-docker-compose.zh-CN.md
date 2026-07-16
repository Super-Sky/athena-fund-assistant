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
  - 验证错误服务 token 拒绝、正确 token + consent 通过、撤销后拒绝、fund conversation trace 回写、artifacts no-leak 和 CSV 决策 trace。

## 边界

- 双服务 overlay 面向本地 MVP 演示和 contract 验证，不是云生产部署。
- fake 模型只用于 smoke，不代表真实模型质量。
- CSV provider 只用于本地兜底，不冒充授权实时行情。
- 不包含支付、订阅、券商账号集成或自动交易能力。
- `ATHENA_FUND_REMOTE_TOOL_TOKEN` 必须通过本地 `.env` / production secret 注入，不能提交真实值；catalog 和 Athena registration 只保存 `env://ATHENA_FUND_REMOTE_TOOL_TOKEN` 引用。
- 双服务 overlay 在 token 为空时让 Athena / fund API healthcheck fail closed，但不阻断 `docker compose config/down/ps/logs` 等生命周期命令。
- Athena 在双服务 smoke 中启用 debug observability；脚本导出容器日志和 control-plane JSON，与主机 artifacts 一起执行 credential no-leak 扫描。

## 验证

- `docker compose -f docker-compose.yml -f docker-compose.dual.yml config`
- `bash -n scripts/smoke_dual_docker.sh`
- `git diff --check`
- `ATHENA_REPO=../Athena ./scripts/smoke_dual_docker.sh` 通过：
  - Compose 启动 Athena、fund API/Web、PostgreSQL、Redis 和 fake model。
  - 错误 token 返回 `service_auth_denied`，正确 token + 有效 grant 完成 `account_overview`。
  - fund conversation 写回完成态 Athena trace；grant 撤销后返回 `authorization_denied`。
  - smoke artifacts、容器日志、Athena remote trace 和 control-plane JSON 未出现服务 token 或用户 session token value。
  - CSV provider 继续返回 `temporary_data=true` 和 conservative/balanced/aggressive 三档策略。
