// alpha_vantage_provider.go normalizes user-key-backed US market observations.
// alpha_vantage_provider.go 标准化基于用户 Key 的美股市场观测数据。
package data

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Super-Sky/athena-fund-assistant/internal/domain"
)

const alphaVantageSchemaVersion = "alpha_vantage_market_data.v1"

// AlphaVantageConfig configures a user-key-backed US market data adapter.
// AlphaVantageConfig 配置基于用户 Key 的美股市场数据适配器。
type AlphaVantageConfig struct {
	BaseURL string
	APIKey  string
	Client  *http.Client
	Now     func() time.Time
}

// AlphaVantageProvider normalizes Alpha Vantage US ETF, equity, index-proxy, and FX responses.
// AlphaVantageProvider 标准化 Alpha Vantage 的美股 ETF、个股、指数代理和汇率响应。
type AlphaVantageProvider struct {
	baseURL string
	apiKey  string
	http    *liveHTTPClient
	now     func() time.Time
}

// NewAlphaVantageProvider creates a provider that remains inactive until its key validates.
// NewAlphaVantageProvider 创建一个在 Key 验证通过前不会被默认启用的 provider。
func NewAlphaVantageProvider(cfg AlphaVantageConfig) (*AlphaVantageProvider, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, fmt.Errorf("ALPHA_VANTAGE_API_KEY is required")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://www.alphavantage.co/query"
	}
	parsed, err := url.Parse(cfg.BaseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("alpha vantage base url is invalid")
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &AlphaVantageProvider{
		baseURL: strings.TrimRight(parsed.String(), "?"),
		apiKey:  strings.TrimSpace(cfg.APIKey),
		http:    newLiveHTTPClient(cfg.Client),
		now:     cfg.Now,
	}, nil
}

// ProviderName returns the stable trace provider identifier.
// ProviderName 返回稳定的 trace provider 标识。
func (*AlphaVantageProvider) ProviderName() string { return "alpha_vantage_provider" }

// ValidateCredentials verifies the minimum response shape needed for a safe user-key admission.
// ValidateCredentials 验证安全启用用户 Key 所需的最小响应结构。
func (p *AlphaVantageProvider) ValidateCredentials(ctx context.Context) error {
	_, _, err := p.globalQuote(ctx, "QQQ")
	return err
}

// GetFundSnapshot returns US ETF data with ETF profile, quote, and daily-series provenance.
// GetFundSnapshot 返回带 ETF profile、报价和日序列来源信息的美股 ETF 数据。
func (p *AlphaVantageProvider) GetFundSnapshot(ctx context.Context, code string) (domain.FundSnapshot, error) {
	symbol := normalizeLiveSymbol(code)
	profile, profileHash, err := p.request(ctx, "ETF_PROFILE", map[string]string{"symbol": symbol})
	if err != nil {
		return domain.FundSnapshot{}, err
	}
	quote, quoteHash, err := p.globalQuote(ctx, symbol)
	if err != nil {
		return domain.FundSnapshot{}, err
	}
	series, seriesHash, err := p.dailySeries(ctx, symbol)
	if err != nil {
		return domain.FundSnapshot{}, err
	}
	metrics := calculateDailyMetrics(series)
	metadata := p.metadata("alpha_vantage:ETF_PROFILE,GLOBAL_QUOTE,TIME_SERIES_DAILY", quote.marketTime, "America/New_York", joinHashes(profileHash, quoteHash, seriesHash), 0.88)
	return domain.FundSnapshot{
		Instrument:      domain.FundInstrument{Code: symbol, Name: stringValue(profile["name"], symbol), Market: "US", Currency: "USD", Type: domain.InstrumentETF},
		Price:           quote.price,
		DailyChangePct:  quote.changePct,
		OneYearReturn:   metrics.oneYearReturn,
		MaxDrawdownPct:  metrics.maxDrawdown,
		VolatilityPct:   metrics.volatility,
		ExpenseRatioPct: parsePercent(profile["net_expense_ratio"]),
		AssetSize:       stringValue(profile["net_assets"], ""),
		TopHoldings:     holdingNames(profile["holdings"]),
		Metadata:        metadata,
	}, nil
}

// GetEquitySnapshot returns a normalized US equity quote and daily-history view.
// GetEquitySnapshot 返回标准化的美股个股报价和日历史数据视图。
func (p *AlphaVantageProvider) GetEquitySnapshot(ctx context.Context, symbol string) (domain.EquitySnapshot, error) {
	symbol = normalizeLiveSymbol(symbol)
	quote, quoteHash, err := p.globalQuote(ctx, symbol)
	if err != nil {
		return domain.EquitySnapshot{}, err
	}
	series, seriesHash, err := p.dailySeries(ctx, symbol)
	if err != nil {
		return domain.EquitySnapshot{}, err
	}
	metrics := calculateDailyMetrics(series)
	return domain.EquitySnapshot{
		Instrument:     domain.FundInstrument{Code: symbol, Name: symbol, Market: "US", Currency: "USD", Type: domain.InstrumentEquity},
		Price:          quote.price,
		DailyChangePct: quote.changePct,
		OneYearReturn:  metrics.oneYearReturn,
		MaxDrawdownPct: metrics.maxDrawdown,
		VolatilityPct:  metrics.volatility,
		Metadata:       p.metadata("alpha_vantage:GLOBAL_QUOTE,TIME_SERIES_DAILY", quote.marketTime, "America/New_York", joinHashes(quoteHash, seriesHash), 0.86),
	}, nil
}

// GetIndexSnapshot returns an explicitly labelled ETF proxy for a requested US benchmark.
// GetIndexSnapshot 为请求的美股基准返回带明确标记的 ETF 代理数据。
func (p *AlphaVantageProvider) GetIndexSnapshot(ctx context.Context, code string) (domain.IndexSnapshot, error) {
	indexCode, name, proxy, ok := alphaIndexProxy(code)
	if !ok {
		return domain.IndexSnapshot{}, fmt.Errorf("%w: alpha vantage index %q", ErrUnsupportedCapability, code)
	}
	quote, quoteHash, err := p.globalQuote(ctx, proxy)
	if err != nil {
		return domain.IndexSnapshot{}, err
	}
	series, seriesHash, err := p.dailySeries(ctx, proxy)
	if err != nil {
		return domain.IndexSnapshot{}, err
	}
	metrics := calculateDailyMetrics(series)
	return domain.IndexSnapshot{
		Code:           indexCode,
		Name:           name + " (ETF proxy: " + proxy + ")",
		Market:         "US",
		Currency:       "USD",
		Level:          quote.price,
		DailyChangePct: quote.changePct,
		OneYearReturn:  metrics.oneYearReturn,
		MaxDrawdownPct: metrics.maxDrawdown,
		Metadata:       p.metadata("alpha_vantage:ETF_proxy:"+proxy, quote.marketTime, "America/New_York", joinHashes(quoteHash, seriesHash), 0.72),
	}, nil
}

// GetFXRate returns the latest daily FX observation for a currency pair.
// GetFXRate 返回货币对最新的日频汇率观测值。
func (p *AlphaVantageProvider) GetFXRate(ctx context.Context, baseCurrency, quoteCurrency string) (domain.FXRate, error) {
	base := strings.ToUpper(strings.TrimSpace(baseCurrency))
	quote := strings.ToUpper(strings.TrimSpace(quoteCurrency))
	payload, rawHash, err := p.request(ctx, "FX_DAILY", map[string]string{"from_symbol": base, "to_symbol": quote, "outputsize": "compact"})
	if err != nil {
		return domain.FXRate{}, err
	}
	series, err := dailyFXSeries(payload)
	if err != nil {
		return domain.FXRate{}, fmt.Errorf("alpha vantage FX_DAILY: %w", err)
	}
	if len(series) == 0 {
		return domain.FXRate{}, fmt.Errorf("alpha vantage FX_DAILY returned no rows")
	}
	latest := series[0]
	return domain.FXRate{
		BaseCurrency:  base,
		QuoteCurrency: quote,
		Rate:          latest.close,
		Metadata:      p.metadata("alpha_vantage:FX_DAILY", latest.at, "UTC", rawHash, 0.84),
	}, nil
}

// GetMarketCalendar returns an explicit gap because Alpha Vantage is not the calendar authority.
// GetMarketCalendar 明确返回能力缺口，因为 Alpha Vantage 不是交易日历权威源。
func (*AlphaVantageProvider) GetMarketCalendar(_ context.Context, market string, _ time.Time) (domain.MarketCalendar, error) {
	return domain.MarketCalendar{}, fmt.Errorf("%w: Alpha Vantage does not admit an exchange calendar for %s", ErrUnsupportedCapability, market)
}

type alphaQuote struct {
	price      float64
	changePct  float64
	marketTime time.Time
}

func (p *AlphaVantageProvider) globalQuote(ctx context.Context, symbol string) (alphaQuote, string, error) {
	payload, rawHash, err := p.request(ctx, "GLOBAL_QUOTE", map[string]string{"symbol": symbol})
	if err != nil {
		return alphaQuote{}, "", err
	}
	quote, ok := objectValue(payload["Global Quote"])
	if !ok {
		return alphaQuote{}, "", fmt.Errorf("alpha vantage GLOBAL_QUOTE is missing Global Quote")
	}
	price, err := floatValue(quote["05. price"])
	if err != nil {
		return alphaQuote{}, "", fmt.Errorf("alpha vantage GLOBAL_QUOTE price: %w", err)
	}
	if price <= 0 {
		return alphaQuote{}, "", fmt.Errorf("alpha vantage GLOBAL_QUOTE price must be positive")
	}
	marketTime, err := parseMarketDate(stringValue(quote["07. latest trading day"], ""), "America/New_York")
	if err != nil {
		marketTime = p.now().UTC()
	}
	return alphaQuote{price: price, changePct: parsePercent(quote["10. change percent"]), marketTime: marketTime}, rawHash, nil
}

func (p *AlphaVantageProvider) dailySeries(ctx context.Context, symbol string) ([]dailyPoint, string, error) {
	payload, rawHash, err := p.request(ctx, "TIME_SERIES_DAILY", map[string]string{"symbol": symbol, "outputsize": "full"})
	if err != nil {
		return nil, "", err
	}
	series, err := dailySeries(payload)
	return series, rawHash, err
}

func (p *AlphaVantageProvider) request(ctx context.Context, function string, values map[string]string) (map[string]any, string, error) {
	query := url.Values{}
	query.Set("function", function)
	query.Set("apikey", p.apiKey)
	for key, value := range values {
		query.Set(key, value)
	}
	payload, rawHash, err := p.http.getJSON(ctx, p.baseURL+"?"+query.Encode())
	if err != nil {
		return nil, "", err
	}
	if message, ok := providerError(payload); ok {
		return nil, "", fmt.Errorf("alpha vantage %s: %s", function, message)
	}
	return payload, rawHash, nil
}

func (p *AlphaVantageProvider) metadata(source string, marketTime time.Time, timezone, rawHash string, confidence float64) domain.SourceMetadata {
	return domain.SourceMetadata{Source: source, Provider: p.ProviderName(), FetchedAt: p.now().UTC(), MarketTime: marketTime, Timezone: timezone, Delay: "provider_delay_not_guaranteed", LicenseTerms: "alpha_vantage_user_key_terms_required", Confidence: confidence, SchemaVersion: alphaVantageSchemaVersion, RawPayloadHash: rawHash}
}

type dailyPoint struct {
	at    time.Time
	close float64
}

type dailyMetrics struct{ oneYearReturn, maxDrawdown, volatility float64 }

func dailySeries(payload map[string]any) ([]dailyPoint, error) {
	value, ok := objectValue(payload["Time Series (Daily)"])
	if !ok {
		return nil, fmt.Errorf("Time Series (Daily) is missing")
	}
	points := make([]dailyPoint, 0, len(value))
	for date, raw := range value {
		row, ok := objectValue(raw)
		if !ok {
			continue
		}
		close, err := floatValue(row["4. close"])
		if err != nil || close <= 0 {
			continue
		}
		at, err := parseMarketDate(date, "America/New_York")
		if err != nil {
			continue
		}
		points = append(points, dailyPoint{at: at, close: close})
	}
	if len(points) == 0 {
		return nil, fmt.Errorf("Time Series (Daily) contains no valid close")
	}
	sort.Slice(points, func(i, j int) bool { return points[i].at.After(points[j].at) })
	return points, nil
}

func dailyFXSeries(payload map[string]any) ([]dailyPoint, error) {
	value, ok := objectValue(payload["Time Series FX (Daily)"])
	if !ok {
		return nil, fmt.Errorf("Time Series FX (Daily) is missing")
	}
	points := make([]dailyPoint, 0, len(value))
	for date, raw := range value {
		row, ok := objectValue(raw)
		if !ok {
			continue
		}
		close, err := floatValue(row["4. close"])
		if err != nil || close <= 0 {
			continue
		}
		at, err := time.Parse(time.DateOnly, date)
		if err != nil {
			continue
		}
		points = append(points, dailyPoint{at: at.UTC(), close: close})
	}
	if len(points) == 0 {
		return nil, fmt.Errorf("Time Series FX (Daily) contains no valid close")
	}
	sort.Slice(points, func(i, j int) bool { return points[i].at.After(points[j].at) })
	return points, nil
}

func calculateDailyMetrics(points []dailyPoint) dailyMetrics {
	if len(points) < 2 {
		return dailyMetrics{}
	}
	window := points
	if len(window) > 253 {
		window = window[:253]
	}
	oldest := window[len(window)-1].close
	oneYearReturn := 0.0
	if oldest > 0 {
		oneYearReturn = (window[0].close/oldest - 1) * 100
	}
	peak := window[len(window)-1].close
	maxDrawdown := 0.0
	returns := make([]float64, 0, len(window)-1)
	for index := len(window) - 2; index >= 0; index-- {
		price := window[index].close
		previous := window[index+1].close
		if price > peak {
			peak = price
		}
		if peak > 0 {
			maxDrawdown = math.Max(maxDrawdown, (peak-price)/peak*100)
		}
		if previous > 0 {
			returns = append(returns, price/previous-1)
		}
	}
	if len(returns) < 2 {
		return dailyMetrics{oneYearReturn: oneYearReturn, maxDrawdown: maxDrawdown}
	}
	mean := 0.0
	for _, value := range returns {
		mean += value
	}
	mean /= float64(len(returns))
	variance := 0.0
	for _, value := range returns {
		variance += (value - mean) * (value - mean)
	}
	volatility := math.Sqrt(variance/float64(len(returns)-1)) * math.Sqrt(252) * 100
	return dailyMetrics{oneYearReturn: oneYearReturn, maxDrawdown: maxDrawdown, volatility: volatility}
}

func alphaIndexProxy(code string) (string, string, string, bool) {
	switch strings.ToUpper(strings.TrimSpace(code)) {
	case "SPX", "S&P500", "SP500":
		return "SPX", "S&P 500 Index", "SPY", true
	case "NDX", "NASDAQ100":
		return "NDX", "Nasdaq 100 Index", "QQQ", true
	case "DJI", "DOW", "DOWJONES":
		return "DJI", "Dow Jones Industrial Average", "DIA", true
	default:
		return "", "", "", false
	}
}

func providerError(payload map[string]any) (string, bool) {
	for _, key := range []string{"Error Message", "Information", "Note"} {
		if value, ok := payload[key]; ok {
			return stringValue(value, "provider error"), true
		}
	}
	return "", false
}

func objectValue(value any) (map[string]any, bool) {
	result, ok := value.(map[string]any)
	return result, ok
}

func floatValue(value any) (float64, error) {
	text := strings.ReplaceAll(strings.TrimSpace(stringValue(value, "")), ",", "")
	if text == "" {
		return 0, fmt.Errorf("value is empty")
	}
	return strconv.ParseFloat(text, 64)
}

func stringValue(value any, fallback string) string {
	if value == nil {
		return fallback
	}
	text := strings.TrimSpace(fmt.Sprint(value))
	if text == "" || text == "<nil>" {
		return fallback
	}
	return text
}

func parsePercent(value any) float64 {
	text := strings.TrimSuffix(strings.TrimSpace(stringValue(value, "")), "%")
	parsed, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return 0
	}
	if math.Abs(parsed) <= 1 && !strings.Contains(stringValue(value, ""), "%") {
		return parsed * 100
	}
	return parsed
}

func parseMarketDate(value, timezone string) (time.Time, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return time.Time{}, err
	}
	date, err := time.ParseInLocation(time.DateOnly, value, loc)
	if err != nil {
		return time.Time{}, err
	}
	return date, nil
}

func normalizeLiveSymbol(value string) string {
	return strings.ToUpper(strings.TrimSpace(strings.TrimPrefix(value, "US-ETF-")))
}

// joinHashes produces one trace-safe hash when a normalized record uses multiple raw responses.
// joinHashes 在一个标准化记录使用多个原始响应时生成单个可追溯哈希。
func joinHashes(values ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(values, ":")))
	return hex.EncodeToString(sum[:])
}

func holdingNames(value any) []string {
	rows, ok := value.([]any)
	if !ok {
		return nil
	}
	items := make([]string, 0, len(rows))
	for _, row := range rows {
		entry, ok := objectValue(row)
		if !ok {
			continue
		}
		if name := stringValue(entry["name"], stringValue(entry["symbol"], "")); name != "" {
			items = append(items, name)
		}
	}
	return items
}
