# China Fund, ETF, And US Market Data Source Strategy

## Goal

The MVP needs real fund and market data, but data sources with unclear licensing must not become default production dependencies. Every data integration must go through a provider interface and preserve source, timing, licensing, and confidence metadata.

US market data must be treated as a first-class data domain, not merely as a supplement to US ETFs. Domestic users who hold QDII, NASDAQ, S&P 500, overseas technology, USD bond, or US-themed funds need underlying US equities, ETFs, indices, FX rates, and market calendars for credible analysis.

## Required Metadata

Every normalized data item must preserve:

- `source`
- `provider`
- `fetched_at`
- `market_time`
- `timezone`
- `delay`
- `license_terms`
- `confidence`
- `schema_version`
- `raw_payload_hash`

## China Fund / ETF Candidates

### Tushare Pro

Position: preferred candidate for A-share, index, ETF, and fund-data research and MVP provider work.

Evidence:

- Tushare Pro has public documentation and a token mechanism.
- The official docs describe a points-based permission system where different APIs and frequencies require different point levels.
- The docs also state that some minute-level and special datasets need separate permission, and the minute data is for strategy research and study rather than commercial use.

Risks:

- It is not a fully open, no-threshold free API.
- Commercial use, display, and redistribution boundaries must be checked per endpoint.
- The MVP can support user-supplied Tushare tokens, but should not claim production-safe commercial use by default.

Recommendation:

- Implement `tushare_provider` in the first live-provider batch.
- Require users to provide their own token.
- Default usage should be research, validation, and local personal use.
- Record `license_terms=tushare_user_token_required` in UI and trace.

References:

- https://tushare.pro/
- https://tushare.pro/document/1?doc_id=108
- https://tushare.pro/document/1?doc_id=290

### AKShare / Eastmoney / Tiantian Fund Path

Position: development and validation candidate, not the default production source.

Evidence:

- AKShare public-fund docs cover open-end fund real-time data, money-market funds, historical NAV, and related endpoints.
- AKShare docs identify many target addresses as Eastmoney / Tiantian Fund.

Risks:

- AKShare is an open-source data interface library; it does not automatically grant rights to the underlying target data.
- Scraping, display, and redistribution terms for Eastmoney / Tiantian Fund must be checked separately.
- Default production SaaS use has licensing uncertainty.

Recommendation:

- Only expose as `akshare_experimental_provider` or local-development fallback.
- Mark all data from this path as experimental in UI and trace.
- Do not use it as the default commercial deployment provider.

References:

- https://akshare.akfamily.xyz/data/fund/fund_public.html
- https://quantapi.eastmoney.com/Manual?from=web

### Exchange / Association Public Data

Position: reference material for rules, statistics, and market data mechanics; not the first live business API.

Evidence:

- AMAC publishes public-fund market statistics.
- SZSE fund-company data interface documents describe ETF, LOF, and fund NAV publication mechanics.
- SSE Info historical data docs describe Level-1, K-line, and CSV historical data forms.

Risks:

- These sources are mostly market-participant technical specs, service documents, or statistical reports, not a ready free API for normal SaaS products.
- Service application, authorization, and display rules must be confirmed before product use.

Recommendation:

- Use them for schema design, field validation, and market-statistics references.
- Do not use them as the first MVP live data provider.

References:

- https://www.amac.org.cn/sjtj/
- https://docs.static.szse.cn/www/marketServices/technicalservice/notice/W020190408723859697048.pdf
- https://www.sseinfo.com/services/assortment/document/

## US Equity / ETF / Index Data Candidates

### US Data Scope

The MVP US provider should cover at least:

- US equities: large-cap technology names, sector leaders, ADRs, and fund-heavy holdings.
- US ETFs: S&P 500, NASDAQ, sector, bond, gold, and USD-asset ETFs.
- US indices: S&P 500, NASDAQ 100, Dow Jones, Russell 2000, and similar benchmarks.
- FX rates: at least USD/CNY for unified return attribution between RMB holdings and USD assets.
- Market calendar and timezone: `America/New_York`, US holidays, half trading days, extended-hours status, and delayed feeds.

These data are used only for fund research and portfolio analysis, not for automated trading, order routing, or brokerage instructions.

### Alpha Vantage

Position: first-priority live provider candidate for US equities, ETFs, and indices.

Evidence:

- Official docs cover stocks, ETFs, mutual funds, market indices, FX, and technical indicators.
- Alpha Vantage provides a free API-key path suitable for MVP validation.
- Alpha Vantage also offers an official MCP server entry for AI-agent use cases.

Risks:

- Free quota is limited.
- Real-time feeds, high-frequency data, and commercial display boundaries must be checked against terms.

Recommendation:

- Implement `alpha_vantage_provider` first.
- Default to US equity / ETF daily prices, basic quotes, indices, USD/CNY FX rates, and technical indicators.
- Cache responses in Redis to avoid quota pressure.

References:

- https://www.alphavantage.co/documentation/
- https://www.alphavantage.co/

### Financial Modeling Prep

Position: candidate provider for US equity / ETF / mutual-fund information and supplemental quotes.

Evidence:

- Official docs include an ETF & Mutual Fund Information API and market-data endpoints such as stock quotes / historical prices.
- Pricing docs describe free-plan bandwidth limits.
- Pricing docs state that displaying or redistributing FMP data requires a Data Display and Licensing Agreement.

Risks:

- Display and redistribution requirements need careful review.
- Better suited as an optional user-key provider than the default free production source.

Recommendation:

- Implement `fmp_provider` as a second-priority provider.
- Require explicit user acceptance of FMP terms in configuration.
- Record `license_terms=fmp_terms_required` in UI and trace.

References:

- https://site.financialmodelingprep.com/developer/docs
- https://site.financialmodelingprep.com/developer/docs/pricing
- https://site.financialmodelingprep.com/developer/docs/stable/information

### Tiingo

Position: candidate provider for US EOD, ETF, mutual-fund, and news data.

Evidence:

- Official docs describe Tiingo market-data APIs.
- Tiingo product pages cover EOD stocks, ETFs, and mutual funds.

Risks:

- Free account limits and commercial display terms must be checked.

Recommendation:

- Keep as an optional provider, not the first default.
- Use mainly for EOD historical prices and news extension.

References:

- https://www.tiingo.com/documentation/
- https://www.tiingo.com/products/end-of-day-stock-price-data

### Nasdaq Data Link

Position: supplemental dataset provider.

Evidence:

- Official docs state Nasdaq Data Link offers free and premium data through Streaming API, REST API, and Tables API.

Risks:

- Each dataset must be checked for free availability, equity / ETF / index coverage, and display rights.

Recommendation:

- Use for supplemental macro, index, and thematic datasets.
- Do not make it the first default quote provider.

References:

- https://docs.data.nasdaq.com/docs/getting-started
- https://www.nasdaq.com/solutions/data/nasdaq-data-link/api

### Stooq

Position: historical-data fallback candidate.

Evidence:

- Stooq provides free historical data downloads.

Risks:

- Automated usage, commercial display, and redistribution terms are unclear.
- It is not suitable as a production default provider.

Recommendation:

- Use only as local-research fallback or CSV import source.

Reference:

- https://stooq.com/db/h/

### Yahoo / yfinance

Position: not recommended as an official provider.

Evidence:

- `yfinance` documentation states it is not affiliated, endorsed, or vetted by Yahoo and uses publicly available APIs for research and educational purposes.
- Yahoo Developer does not provide a current official Yahoo Finance market-data API entry.

Recommendation:

- Do not use as production provider.
- If a user uses it locally, mark it as unofficial / research-only.

References:

- https://ranaroussi.github.io/yfinance/
- https://developer.yahoo.com/api/

## MVP Provider Priority

### First Batch

- `csv_provider`
- `mock_provider`
- `alpha_vantage_provider`
- `tushare_provider`

### Second Batch

- `fmp_provider`
- `tiingo_provider`
- `nasdaq_data_link_provider`

### Experimental Only

- `akshare_experimental_provider`
- `stooq_csv_provider`
- `yfinance_unofficial_provider`

## Provider Interface Requirements

Providers must implement:

- `ListInstruments`
- `GetFundSnapshot`
- `GetMarketSnapshot`
- `GetHistoricalPrices`
- `GetEquitySnapshot`
- `GetIndexSnapshot`
- `GetFxRate`
- `GetMarketCalendar`
- `GetProviderStatus`

Provider responses must include:

- normalized data
- raw source metadata
- freshness metadata
- provider confidence
- license marker
- cache key

## Cache Strategy

- Redis stores provider responses, rate-limit state, and async refresh state.
- PostgreSQL stores normalized snapshots, journal evidence snapshots, and user selections.
- Unauthorized raw payloads must not be stored as long-lived redistributable assets.
