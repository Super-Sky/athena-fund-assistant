// tushare_provider.go normalizes user-token-backed China fund and market observations.
// tushare_provider.go 标准化基于用户 Token 的中国基金和市场观测数据。
package data

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Super-Sky/athena-fund-assistant/internal/domain"
)

const tushareSchemaVersion = "tushare_market_data.v1"

// TushareConfig configures a user-token-backed China fund and index data adapter.
// TushareConfig 配置基于用户 Token 的中国基金和指数数据适配器。
type TushareConfig struct {
	BaseURL string
	Token   string
	Client  *http.Client
	Now     func() time.Time
}

// TushareProvider normalizes authorized Tushare fund NAV, ETF, index, and calendar responses.
// TushareProvider 标准化已授权 Tushare 基金净值、ETF、指数和日历响应。
type TushareProvider struct {
	baseURL string
	token   string
	http    *liveHTTPClient
	now     func() time.Time
}

// NewTushareProvider creates a provider that requires a user-supplied token.
// NewTushareProvider 创建一个必须提供用户 Token 的 provider。
func NewTushareProvider(cfg TushareConfig) (*TushareProvider, error) {
	if strings.TrimSpace(cfg.Token) == "" {
		return nil, fmt.Errorf("TUSHARE_TOKEN is required")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://api.tushare.pro"
	}
	parsed, err := url.Parse(cfg.BaseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("tushare base url is invalid")
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &TushareProvider{baseURL: parsed.String(), token: strings.TrimSpace(cfg.Token), http: newLiveHTTPClient(cfg.Client), now: cfg.Now}, nil
}

// ProviderName returns the stable trace provider identifier.
// ProviderName 返回稳定的 trace provider 标识。
func (*TushareProvider) ProviderName() string { return "tushare_provider" }

// ValidateCredentials verifies that the token can read a basic fund response.
// ValidateCredentials 验证该 Token 可以读取基础基金响应。
func (p *TushareProvider) ValidateCredentials(ctx context.Context) error {
	_, _, err := p.query(ctx, "fund_basic", map[string]any{"ts_code": "000001.OF"}, "ts_code,name,fund_type")
	return err
}

// GetFundSnapshot returns China public-fund or ETF metadata and its latest NAV.
// GetFundSnapshot 返回中国公募基金或 ETF 的元数据和最新净值。
func (p *TushareProvider) GetFundSnapshot(ctx context.Context, code string) (domain.FundSnapshot, error) {
	tsCode := tushareFundCode(code)
	basic, basicHash, err := p.query(ctx, "fund_basic", map[string]any{"ts_code": tsCode}, "ts_code,name,management,fund_type,m_fee,issue_amount")
	if err != nil {
		return domain.FundSnapshot{}, err
	}
	basicRow, ok := basic.first()
	if !ok {
		return domain.FundSnapshot{}, fmt.Errorf("tushare fund_basic returned no row for %s", tsCode)
	}
	nav, navHash, err := p.query(ctx, "fund_nav", map[string]any{"ts_code": tsCode}, "ts_code,end_date,unit_nav,accum_nav,net_asset,total_netasset")
	if err != nil {
		return domain.FundSnapshot{}, err
	}
	navRow, ok := nav.first()
	if !ok {
		return domain.FundSnapshot{}, fmt.Errorf("tushare fund_nav returned no row for %s", tsCode)
	}
	unitNAV, err := rowFloat(navRow, "unit_nav")
	if err != nil {
		return domain.FundSnapshot{}, fmt.Errorf("tushare fund_nav unit_nav: %w", err)
	}
	if unitNAV <= 0 {
		return domain.FundSnapshot{}, fmt.Errorf("tushare fund_nav unit_nav must be positive")
	}
	marketTime, err := tushareDate(rowString(navRow, "end_date"))
	if err != nil {
		marketTime = p.now().In(shanghaiLocation())
	}
	instrumentType := domain.InstrumentFund
	if strings.Contains(strings.ToUpper(rowString(basicRow, "fund_type")), "ETF") {
		instrumentType = domain.InstrumentETF
	}
	return domain.FundSnapshot{
		Instrument:      domain.FundInstrument{Code: tsCode, Name: rowString(basicRow, "name"), Market: "CN", Currency: "CNY", Type: instrumentType},
		NAV:             unitNAV,
		ExpenseRatioPct: parseTusharePercent(basicRow["m_fee"]),
		Manager:         rowString(basicRow, "management"),
		AssetSize:       rowString(navRow, "total_netasset"),
		Metadata:        p.metadata("tushare:fund_basic,fund_nav", marketTime, joinHashes(basicHash, navHash), 0.9),
	}, nil
}

// GetEquitySnapshot is intentionally unavailable because the China MVP adapter is fund-focused.
// GetEquitySnapshot 有意不支持，因为这个中国 MVP 适配器聚焦基金数据。
func (*TushareProvider) GetEquitySnapshot(_ context.Context, symbol string) (domain.EquitySnapshot, error) {
	return domain.EquitySnapshot{}, fmt.Errorf("%w: Tushare fund adapter does not expose equity %s", ErrUnsupportedCapability, symbol)
}

// GetIndexSnapshot returns the latest China benchmark index record.
// GetIndexSnapshot 返回最新的中国基准指数记录。
func (p *TushareProvider) GetIndexSnapshot(ctx context.Context, code string) (domain.IndexSnapshot, error) {
	tsCode := tushareIndexCode(code)
	result, rawHash, err := p.query(ctx, "index_daily", map[string]any{"ts_code": tsCode}, "ts_code,trade_date,close,pct_chg")
	if err != nil {
		return domain.IndexSnapshot{}, err
	}
	row, ok := result.first()
	if !ok {
		return domain.IndexSnapshot{}, fmt.Errorf("tushare index_daily returned no row for %s", tsCode)
	}
	level, err := rowFloat(row, "close")
	if err != nil {
		return domain.IndexSnapshot{}, fmt.Errorf("tushare index_daily close: %w", err)
	}
	if level <= 0 {
		return domain.IndexSnapshot{}, fmt.Errorf("tushare index_daily close must be positive")
	}
	marketTime, err := tushareDate(rowString(row, "trade_date"))
	if err != nil {
		marketTime = p.now().In(shanghaiLocation())
	}
	return domain.IndexSnapshot{
		Code:           tsCode,
		Name:           tushareIndexName(tsCode),
		Market:         "CN",
		Currency:       "CNY",
		Level:          level,
		DailyChangePct: parseTusharePercent(row["pct_chg"]),
		Metadata:       p.metadata("tushare:index_daily", marketTime, rawHash, 0.9),
	}, nil
}

// GetFXRate is intentionally unavailable because FX requires a separately admitted source.
// GetFXRate 有意不支持，因为汇率需要单独完成准入的数据源。
func (*TushareProvider) GetFXRate(_ context.Context, baseCurrency, quoteCurrency string) (domain.FXRate, error) {
	return domain.FXRate{}, fmt.Errorf("%w: Tushare fund adapter does not expose FX %s/%s", ErrUnsupportedCapability, baseCurrency, quoteCurrency)
}

// GetMarketCalendar returns an exchange-calendar record and keeps the China timezone explicit.
// GetMarketCalendar 返回交易所日历记录并明确保留中国时区。
func (p *TushareProvider) GetMarketCalendar(ctx context.Context, market string, date time.Time) (domain.MarketCalendar, error) {
	if strings.ToUpper(strings.TrimSpace(market)) != "CN" {
		return domain.MarketCalendar{}, fmt.Errorf("%w: Tushare calendar only supports CN", ErrUnsupportedCapability)
	}
	day := date.In(shanghaiLocation()).Format("20060102")
	result, rawHash, err := p.query(ctx, "trade_cal", map[string]any{"exchange": "SSE", "start_date": day, "end_date": day}, "exchange,cal_date,is_open,pretrade_date")
	if err != nil {
		return domain.MarketCalendar{}, err
	}
	row, ok := result.first()
	if !ok {
		return domain.MarketCalendar{}, fmt.Errorf("tushare trade_cal returned no row for %s", day)
	}
	isOpen := rowString(row, "is_open") == "1"
	marketTime, err := tushareDate(rowString(row, "cal_date"))
	if err != nil {
		marketTime = date.In(shanghaiLocation())
	}
	session := "closed"
	if isOpen {
		session = "regular"
	}
	return domain.MarketCalendar{Market: "CN", Date: marketTime.Format(time.DateOnly), IsTradingDay: isOpen, IsHalfDay: false, Session: session, Timezone: "Asia/Shanghai", Delay: "provider_calendar", Metadata: p.metadata("tushare:trade_cal", marketTime, rawHash, 0.92)}, nil
}

func (p *TushareProvider) query(ctx context.Context, apiName string, params map[string]any, fields string) (tushareResult, string, error) {
	payload := map[string]any{"api_name": apiName, "token": p.token, "params": params, "fields": fields}
	response, rawHash, err := p.http.postJSON(ctx, p.baseURL, payload)
	if err != nil {
		return tushareResult{}, "", err
	}
	code, err := intValue(response["code"])
	if err != nil {
		return tushareResult{}, "", fmt.Errorf("tushare %s code: %w", apiName, err)
	}
	if code != 0 {
		return tushareResult{}, "", fmt.Errorf("tushare %s code %d: %s", apiName, code, stringValue(response["msg"], "unknown error"))
	}
	data, ok := objectValue(response["data"])
	if !ok {
		return tushareResult{}, "", fmt.Errorf("tushare %s response data is missing", apiName)
	}
	fieldsRaw, ok := data["fields"].([]any)
	if !ok {
		return tushareResult{}, "", fmt.Errorf("tushare %s fields are missing", apiName)
	}
	fieldsSlice := make([]string, 0, len(fieldsRaw))
	for _, value := range fieldsRaw {
		fieldsSlice = append(fieldsSlice, stringValue(value, ""))
	}
	itemsRaw, ok := data["items"].([]any)
	if !ok {
		return tushareResult{fields: fieldsSlice}, rawHash, nil
	}
	rows := make([]map[string]any, 0, len(itemsRaw))
	for _, raw := range itemsRaw {
		values, ok := raw.([]any)
		if !ok {
			continue
		}
		row := map[string]any{}
		for index, field := range fieldsSlice {
			if index < len(values) {
				row[field] = values[index]
			}
		}
		rows = append(rows, row)
	}
	return tushareResult{fields: fieldsSlice, rows: rows}, rawHash, nil
}

func (p *TushareProvider) metadata(source string, marketTime time.Time, rawHash string, confidence float64) domain.SourceMetadata {
	return domain.SourceMetadata{Source: source, Provider: p.ProviderName(), FetchedAt: p.now().UTC(), MarketTime: marketTime, Timezone: "Asia/Shanghai", Delay: "provider_delay_not_guaranteed", LicenseTerms: "tushare_user_token_required", Confidence: confidence, SchemaVersion: tushareSchemaVersion, RawPayloadHash: rawHash}
}

type tushareResult struct {
	fields []string
	rows   []map[string]any
}

func (r tushareResult) first() (map[string]any, bool) {
	if len(r.rows) == 0 {
		return nil, false
	}
	return r.rows[0], true
}

func rowString(row map[string]any, key string) string          { return stringValue(row[key], "") }
func rowFloat(row map[string]any, key string) (float64, error) { return floatValue(row[key]) }
func intValue(value any) (int, error) {
	parsed, err := strconv.Atoi(stringValue(value, ""))
	return parsed, err
}
func parseTusharePercent(value any) float64 { return parsePercent(value) }
func shanghaiLocation() *time.Location {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("CST", 8*60*60)
	}
	return location
}

func tushareDate(value string) (time.Time, error) {
	return time.ParseInLocation("20060102", strings.TrimSpace(value), shanghaiLocation())
}
func tushareFundCode(code string) string {
	code = strings.ToUpper(strings.TrimSpace(strings.TrimPrefix(code, "CN-FUND-")))
	if strings.Contains(code, ".") {
		return code
	}
	return code + ".OF"
}
func tushareIndexCode(code string) string {
	switch strings.ToUpper(strings.TrimSpace(code)) {
	case "000300", "CSI300":
		return "000300.SH"
	default:
		return strings.ToUpper(strings.TrimSpace(code))
	}
}
func tushareIndexName(code string) string {
	if code == "000300.SH" {
		return "CSI 300 Index"
	}
	return code
}
