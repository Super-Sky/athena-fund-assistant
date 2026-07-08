package data

import (
	"context"
	"fmt"
	"time"
)

// ValidationOptions declares the exact provider probes to run before coding against a data source.
// ValidationOptions 声明在编码接入数据源前必须执行的 provider 探针。
type ValidationOptions struct {
	FundCodes     []string
	EquitySymbols []string
	IndexCodes    []string
	FXPairs       []FXPair
	Calendars     []CalendarProbe
}

// FXPair identifies one currency pair that must be validated.
// FXPair 标识一个必须验证的汇率对。
type FXPair struct {
	BaseCurrency  string
	QuoteCurrency string
}

// CalendarProbe identifies one market/date calendar probe.
// CalendarProbe 标识一个市场和日期的交易日历探针。
type CalendarProbe struct {
	Market string
	Date   time.Time
}

// ValidationReport records whether a provider is ready to feed analysis code.
// ValidationReport 记录 provider 是否可以接入分析代码。
type ValidationReport struct {
	GeneratedAt time.Time         `json:"generated_at"`
	Passed      bool              `json:"passed"`
	Checks      []ValidationCheck `json:"checks"`
}

// ValidationCheck records one provider probe result.
// ValidationCheck 记录单个 provider 探针结果。
type ValidationCheck struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Message string `json:"message"`
}

// ValidateProvider runs structural and metadata probes before a provider is trusted by workflows.
// ValidateProvider 在 workflow 信任 provider 前执行结构和 metadata 探针。
func ValidateProvider(ctx context.Context, provider Provider, options ValidationOptions) ValidationReport {
	report := ValidationReport{
		GeneratedAt: time.Now().UTC(),
		Passed:      true,
	}

	for _, code := range options.FundCodes {
		name := fmt.Sprintf("fund_snapshot:%s", code)
		snapshot, err := provider.GetFundSnapshot(ctx, code)
		report.add(name, err == nil && snapshot.Validate() == nil, firstErr(err, snapshot.Validate()))
	}
	for _, symbol := range options.EquitySymbols {
		name := fmt.Sprintf("equity_snapshot:%s", symbol)
		snapshot, err := provider.GetEquitySnapshot(ctx, symbol)
		report.add(name, err == nil && snapshot.Validate() == nil, firstErr(err, snapshot.Validate()))
	}
	for _, code := range options.IndexCodes {
		name := fmt.Sprintf("index_snapshot:%s", code)
		snapshot, err := provider.GetIndexSnapshot(ctx, code)
		report.add(name, err == nil && snapshot.Validate() == nil, firstErr(err, snapshot.Validate()))
	}
	for _, pair := range options.FXPairs {
		name := fmt.Sprintf("fx_rate:%s/%s", pair.BaseCurrency, pair.QuoteCurrency)
		rate, err := provider.GetFXRate(ctx, pair.BaseCurrency, pair.QuoteCurrency)
		report.add(name, err == nil && rate.Validate() == nil, firstErr(err, rate.Validate()))
	}
	for _, calendar := range options.Calendars {
		name := fmt.Sprintf("market_calendar:%s:%s", calendar.Market, calendar.Date.Format("2006-01-02"))
		value, err := provider.GetMarketCalendar(ctx, calendar.Market, calendar.Date)
		report.add(name, err == nil && value.Validate() == nil, firstErr(err, value.Validate()))
	}

	if len(report.Checks) == 0 {
		report.add("probe_set", false, "validation options must include at least one probe")
	}
	return report
}

func (r *ValidationReport) add(name string, passed bool, message string) {
	if message == "" && passed {
		message = "ok"
	}
	if message == "" && !passed {
		message = "validation failed"
	}
	r.Checks = append(r.Checks, ValidationCheck{Name: name, Passed: passed, Message: message})
	if !passed {
		r.Passed = false
	}
}

func firstErr(primary error, secondary error) string {
	if primary != nil {
		return primary.Error()
	}
	if secondary != nil {
		return secondary.Error()
	}
	return ""
}
