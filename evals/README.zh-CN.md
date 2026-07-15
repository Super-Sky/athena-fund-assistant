# 确定性金融评测包

## 范围

本包是 #31A 的固定金融安全 fixture 发布门禁。它仅使用 Promptfoo `0.121.19`、Node.js `>=22.22.0`、本地 file provider 和 JavaScript 断言，不依赖生产账户、API key、LLM provider、网络请求或 LLM-as-judge。

仓库通过 `.node-version` 固定 Node.js `22.22.0`；可使用支持该文件的版本管理器，或任何满足 `>=22.22.0` 的标准 Node.js 发行版运行。

fixture 矩阵覆盖安全基线、缺失与陈旧数据、provider 与 tool 失败、来源 metadata 缺失、保证收益措辞、单一路径结论、风险与失效条件缺失、无依据百分比以及未授权账户读取。每条用例还包含可关联的 Athena run 安全摘要，但不携带基金业务 payload 或凭据。

`go.mod` 仅作为工具目录边界，防止仓库根目录的 Go 命令扫描 Promptfoo 依赖中附带的 Go adapter；本包不承载基金业务 Go 代码。

## 命令

```bash
npm ci
npm test
npm run eval:deterministic
```

`npm run eval:deterministic` 会先删除陈旧结果，关闭 Promptfoo cache、数据库写入、结果分享、telemetry 与 remote generation，并同时写出：

- `artifacts/results.json`：完整 Promptfoo 诊断和断言原因。
- `artifacts/results.junit.xml`：依靠必需的 `.junit.xml` 后缀识别出的 JUnit 报告。

## 门禁

确定性阈值为 100%：每条 fixture 及其全部生效组件断言都必须通过。任一断言失败或 provider/runtime 错误都会使命令非零退出。应阻断场景必须 fail closed；标记场景必须保留机器可读披露。

输出是否有帮助、语气、适当性、provider 许可，以及未来任何模型辅助 rubric 仍需人工复核。此类复核可以补充本门禁，但不能替代或削弱确定性合规规则。
