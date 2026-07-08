package data

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/Super-Sky/athena-fund-assistant/internal/domain"
)

// MockProvider supplies deterministic sample data while real providers are validated.
// MockProvider 在真实 provider 验证前提供确定性的样例数据。
type MockProvider struct {
	now func() time.Time
}

// NewMockProvider creates a mock provider with wall-clock timestamps.
// NewMockProvider 创建使用当前时间戳的 mock provider。
func NewMockProvider() *MockProvider {
	return &MockProvider{now: time.Now}
}

// GetFundSnapshot returns normalized mock fund or ETF data with explicit metadata.
// GetFundSnapshot 返回带显式 metadata 的标准化 mock 基金或 ETF 数据。
func (p *MockProvider) GetFundSnapshot(_ context.Context, code string) (domain.FundSnapshot, error) {
	now := p.now().UTC()
	normalized := strings.ToUpper(strings.TrimSpace(code))
	switch normalized {
	case "000001", "CN-FUND-000001":
		return p.snapshot(now, domain.FundInstrument{
			Code:     "000001",
			Name:     "Sample China Balanced Fund",
			Market:   "CN",
			Currency: "CNY",
			Type:     domain.InstrumentFund,
		}, 1.238, 0, -0.42, 6.8, 18.5, 13.2, 1.2, "Mock Manager", "12.4B CNY", []string{"CSI 300 exposure", "China government bond sleeve", "Cash buffer"}), nil
	case "510300", "CN-ETF-510300":
		return p.snapshot(now, domain.FundInstrument{
			Code:     "510300",
			Name:     "Sample CSI 300 ETF",
			Market:   "CN",
			Currency: "CNY",
			Type:     domain.InstrumentETF,
		}, 4.12, 4.12, 0.84, 3.4, 21.7, 18.4, 0.5, "Index ETF", "35.8B CNY", []string{"CSI 300 index basket"}), nil
	case "QQQ", "US-ETF-QQQ":
		return p.snapshot(now, domain.FundInstrument{
			Code:     "QQQ",
			Name:     "Sample Nasdaq 100 ETF",
			Market:   "US",
			Currency: "USD",
			Type:     domain.InstrumentETF,
		}, 0, 512.35, 1.08, 22.4, 24.9, 19.8, 0.2, "Index ETF", "290B USD", []string{"Large-cap technology", "Semiconductors", "Consumer platforms"}), nil
	default:
		return domain.FundSnapshot{}, fmt.Errorf("mock provider has no instrument %q", code)
	}
}

// GetEquitySnapshot returns normalized mock US equity data.
// GetEquitySnapshot 返回标准化 mock 美股个股数据。
func (p *MockProvider) GetEquitySnapshot(_ context.Context, symbol string) (domain.EquitySnapshot, error) {
	now := p.now().UTC()
	normalized := strings.ToUpper(strings.TrimSpace(symbol))
	if normalized != "AAPL" {
		return domain.EquitySnapshot{}, fmt.Errorf("mock provider has no equity %q", symbol)
	}
	metadata := p.metadata(now, domain.FundInstrument{Code: "AAPL", Market: "US"})
	return domain.EquitySnapshot{
		Instrument: domain.FundInstrument{
			Code:     "AAPL",
			Name:     "Sample Apple Equity",
			Market:   "US",
			Currency: "USD",
			Type:     domain.InstrumentEquity,
		},
		Price:          226.5,
		DailyChangePct: 0.7,
		OneYearReturn:  16.2,
		MaxDrawdownPct: 22.1,
		VolatilityPct:  21.3,
		Metadata:       metadata,
	}, nil
}

// GetIndexSnapshot returns normalized mock benchmark index data.
// GetIndexSnapshot 返回标准化 mock 基准指数数据。
func (p *MockProvider) GetIndexSnapshot(_ context.Context, code string) (domain.IndexSnapshot, error) {
	now := p.now().UTC()
	normalized := strings.ToUpper(strings.TrimSpace(code))
	switch normalized {
	case "SPX", "S&P500", "SP500":
		return p.indexSnapshot(now, "SPX", "Sample S&P 500 Index", "US", "USD", 6820.1, 0.4, 14.3, 12.2), nil
	case "NDX", "NASDAQ100":
		return p.indexSnapshot(now, "NDX", "Sample Nasdaq 100 Index", "US", "USD", 25210.4, 0.9, 21.7, 20.3), nil
	case "CSI300", "000300":
		return p.indexSnapshot(now, "000300", "Sample CSI 300 Index", "CN", "CNY", 4120.8, 0.6, 4.2, 18.6), nil
	default:
		return domain.IndexSnapshot{}, fmt.Errorf("mock provider has no index %q", code)
	}
}

// GetFXRate returns a normalized mock FX rate with explicit temporary metadata.
// GetFXRate 返回带显式临时标记的标准化 mock 汇率。
func (p *MockProvider) GetFXRate(_ context.Context, baseCurrency, quoteCurrency string) (domain.FXRate, error) {
	now := p.now().UTC()
	base := strings.ToUpper(strings.TrimSpace(baseCurrency))
	quote := strings.ToUpper(strings.TrimSpace(quoteCurrency))
	if base == quote {
		return domain.FXRate{BaseCurrency: base, QuoteCurrency: quote, Rate: 1, Metadata: p.metadata(now, domain.FundInstrument{Code: base + quote, Market: "FX"})}, nil
	}
	if base == "USD" && quote == "CNY" {
		return domain.FXRate{BaseCurrency: base, QuoteCurrency: quote, Rate: 7.18, Metadata: p.metadata(now, domain.FundInstrument{Code: "USDCNY", Market: "FX"})}, nil
	}
	if base == "CNY" && quote == "USD" {
		return domain.FXRate{BaseCurrency: base, QuoteCurrency: quote, Rate: round(1 / 7.18), Metadata: p.metadata(now, domain.FundInstrument{Code: "CNYUSD", Market: "FX"})}, nil
	}
	return domain.FXRate{}, fmt.Errorf("mock provider has no fx rate %s/%s", baseCurrency, quoteCurrency)
}

// GetMarketCalendar returns deterministic trading-day metadata for CN and US markets.
// GetMarketCalendar 返回中美市场的确定性交易日信息。
func (p *MockProvider) GetMarketCalendar(_ context.Context, market string, date time.Time) (domain.MarketCalendar, error) {
	normalized := strings.ToUpper(strings.TrimSpace(market))
	if normalized != "CN" && normalized != "US" {
		return domain.MarketCalendar{}, fmt.Errorf("mock provider has no market calendar %q", market)
	}
	weekday := date.Weekday()
	isTradingDay := weekday != time.Saturday && weekday != time.Sunday
	session := "regular"
	if !isTradingDay {
		session = "closed"
	}
	return domain.MarketCalendar{
		Market:       normalized,
		Date:         date.Format("2006-01-02"),
		IsTradingDay: isTradingDay,
		IsHalfDay:    false,
		Session:      session,
		Timezone:     timezoneForMarket(normalized),
		Delay:        "mock_calendar",
		Metadata:     p.metadata(p.now().UTC(), domain.FundInstrument{Code: normalized + "_CALENDAR", Market: normalized}),
	}, nil
}

func (p *MockProvider) snapshot(now time.Time, instrument domain.FundInstrument, nav, price, daily, oneYear, drawdown, volatility, expense float64, manager, size string, top []string) domain.FundSnapshot {
	return domain.FundSnapshot{
		Instrument:      instrument,
		NAV:             nav,
		Price:           price,
		DailyChangePct:  daily,
		OneYearReturn:   oneYear,
		MaxDrawdownPct:  drawdown,
		VolatilityPct:   volatility,
		ExpenseRatioPct: expense,
		Manager:         manager,
		AssetSize:       size,
		TopHoldings:     top,
		Metadata:        p.metadata(now, instrument),
	}
}

func (p *MockProvider) indexSnapshot(now time.Time, code, name, market, currency string, level, daily, oneYear, drawdown float64) domain.IndexSnapshot {
	return domain.IndexSnapshot{
		Code:           code,
		Name:           name,
		Market:         market,
		Currency:       currency,
		Level:          level,
		DailyChangePct: daily,
		OneYearReturn:  oneYear,
		MaxDrawdownPct: drawdown,
		Metadata:       p.metadata(now, domain.FundInstrument{Code: code, Market: market}),
	}
}

func (p *MockProvider) metadata(now time.Time, instrument domain.FundInstrument) domain.SourceMetadata {
	marketTime := now.Add(-15 * time.Minute)
	if instrument.Market == "US" {
		loc, err := time.LoadLocation("America/New_York")
		if err == nil {
			marketTime = now.In(loc).Add(-15 * time.Minute)
		}
	}
	raw := fmt.Sprintf("%s|%s|%s", instrument.Code, instrument.Market, now.Format(time.RFC3339))
	sum := sha256.Sum256([]byte(raw))
	return domain.SourceMetadata{
		Source:         "mock_seed_dataset",
		Provider:       "mock_provider",
		FetchedAt:      now,
		MarketTime:     marketTime,
		Timezone:       timezoneForMarket(instrument.Market),
		Delay:          "mock_15m_delay",
		LicenseTerms:   "mock_data_temporary_not_for_production",
		Confidence:     0.6,
		SchemaVersion:  "market_data.v1",
		RawPayloadHash: hex.EncodeToString(sum[:]),
	}
}

func timezoneForMarket(market string) string {
	if market == "US" {
		return "America/New_York"
	}
	if market == "FX" {
		return "UTC"
	}
	return "Asia/Shanghai"
}

func round(value float64) float64 {
	return math.Round(value*1000000) / 1000000
}
