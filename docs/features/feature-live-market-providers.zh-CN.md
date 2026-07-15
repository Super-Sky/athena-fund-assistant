# 凭据型实时市场 Provider

## 范围

本功能在基金助手应用层增加了需要用户凭据的市场数据 adapter。它不改变 Athena core、不保存券商凭据，也不启用交易。

## Provider

- `alpha_vantage_provider`
  - 要求用户自有 `ALPHA_VANTAGE_API_KEY`。
  - 覆盖美股 ETF 快照、美股个股快照、USD/CNY 汇率，以及带明确标记的标普 500、纳斯达克 100、道琼斯 ETF 代理。
  - 使用完整日线历史，并以最近约 252 个交易日计算一年收益、回撤和波动率。
  - 不宣称具备交易所日历覆盖；日历调用会返回明确的不支持能力错误。
- `tushare_provider`
  - 要求用户自有 `TUSHARE_TOKEN`。
  - 覆盖中国公募基金净值、沪深 300 指数和上交所交易日历记录。
  - 不宣称覆盖美股、汇率或美股日历。

## 准入与 Trace

- 两个 provider 都通过 `ATHENA_FUND_PROVIDER` 显式选择；默认仍然是 `mock`。
- API 在监听前执行 `data.ValidateProvider`。缺失凭据、授权失败、额度返回、字段畸形或 metadata 不合格都会阻止启动。
- 每条标准化观测都记录来源、provider、抓取时间、市场时间、时区、延迟、条款、置信度、schema 版本和原始载荷哈希。
- 不支持的能力会显式报错，绝不使用猜测值或 mock 观测悄悄替代。

## 验证

- `go test ./internal/data -run 'Test(AlphaVantageProvider|TushareProvider)' -count=1`
- `go test ./...`
- 在 `apps/web` 中执行 `yarn build`
- `docker compose -f docker-compose.yml config --quiet`
- `docker compose -f docker-compose.yml -f docker-compose.dual.yml config --quiet`

真实 provider 的最终准入仍等待用户提供 Alpha Vantage Key 与 Tushare Token。测试套件使用本地 HTTP fixture，不能把它当作真实 provider 授权已经通过的证据。

## 维护 Skill

本次未新增专用维护 skill。当前 provider contract 规模较小，`internal/data/README.md` 已能完整索引；当后续跨市场组合、Redis 缓存、provider 健康监测或凭据生命周期形成可复用运维流程时，再创建专用 skill。
