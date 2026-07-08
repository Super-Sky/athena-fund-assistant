package providerprobe

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// AlphaVantageConfig defines validation probes for Alpha Vantage.
// AlphaVantageConfig 定义 Alpha Vantage 的验证探针。
type AlphaVantageConfig struct {
	BaseURL        string
	APIKey         string
	ETFSymbol      string
	QuoteSymbol    string
	FXFromCurrency string
	FXToCurrency   string
	Timeout        time.Duration
}

// ProbeAlphaVantage validates Alpha Vantage response shapes without wiring a business provider.
// ProbeAlphaVantage 只验证 Alpha Vantage 响应结构，不接入业务 provider。
func ProbeAlphaVantage(ctx context.Context, cfg AlphaVantageConfig) Report {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://www.alphavantage.co/query"
	}
	if cfg.APIKey == "" {
		cfg.APIKey = "demo"
	}
	if cfg.ETFSymbol == "" {
		cfg.ETFSymbol = "QQQ"
	}
	if cfg.QuoteSymbol == "" {
		cfg.QuoteSymbol = cfg.ETFSymbol
	}
	if cfg.FXFromCurrency == "" {
		cfg.FXFromCurrency = "USD"
	}
	if cfg.FXToCurrency == "" {
		cfg.FXToCurrency = "CNY"
	}

	report := NewReport("alpha_vantage", "alpha_vantage_terms_required")
	client := NewJSONClient(cfg.Timeout)

	probes := []struct {
		name     string
		endpoint string
		required []string
	}{
		{
			name:     "etf_profile",
			endpoint: alphaURL(cfg.BaseURL, map[string]string{"function": "ETF_PROFILE", "symbol": cfg.ETFSymbol, "apikey": cfg.APIKey}),
			required: []string{"net_assets", "net_expense_ratio", "holdings"},
		},
		{
			name:     "global_quote",
			endpoint: alphaURL(cfg.BaseURL, map[string]string{"function": "GLOBAL_QUOTE", "symbol": cfg.QuoteSymbol, "apikey": cfg.APIKey}),
			required: []string{"Global Quote"},
		},
		{
			name:     "time_series_daily",
			endpoint: alphaURL(cfg.BaseURL, map[string]string{"function": "TIME_SERIES_DAILY", "symbol": cfg.QuoteSymbol, "apikey": cfg.APIKey}),
			required: []string{"Meta Data", "Time Series (Daily)"},
		},
		{
			name:     "fx_daily",
			endpoint: alphaURL(cfg.BaseURL, map[string]string{"function": "FX_DAILY", "from_symbol": cfg.FXFromCurrency, "to_symbol": cfg.FXToCurrency, "apikey": cfg.APIKey}),
			required: []string{"Meta Data", "Time Series FX (Daily)"},
		},
	}

	for _, probe := range probes {
		payload, err := client.Get(ctx, probe.endpoint)
		report.Add(validatePayload(probe.name, probe.endpoint, probe.required, payload, err))
	}
	return report
}

func alphaURL(base string, params map[string]string) string {
	values := url.Values{}
	for key, value := range params {
		values.Set(key, value)
	}
	separator := "?"
	if strings.Contains(base, "?") {
		separator = "&"
	}
	return base + separator + values.Encode()
}

func validatePayload(name, endpoint string, required []string, payload map[string]any, err error) Check {
	check := Check{
		Name:           name,
		Endpoint:       scrubAPIKey(endpoint),
		RequiredFields: required,
	}
	if err != nil {
		check.Message = err.Error()
		return check
	}
	check.ObservedFields = observedFields(payload)
	if providerMessage, ok := providerMessage(payload); ok {
		check.Message = providerMessage
		return check
	}
	missing := missingFields(payload, required)
	if len(missing) > 0 {
		check.Message = fmt.Sprintf("missing fields: %s", strings.Join(missing, ", "))
		return check
	}
	check.Passed = true
	return check
}

func providerMessage(payload map[string]any) (string, bool) {
	for _, field := range []string{"Error Message", "Information", "Note"} {
		if value, ok := payload[field]; ok {
			return fmt.Sprintf("%s: %v", field, value), true
		}
	}
	return "", false
}

func scrubAPIKey(endpoint string) string {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return endpoint
	}
	values := parsed.Query()
	if values.Has("apikey") {
		values.Set("apikey", "redacted")
		parsed.RawQuery = values.Encode()
	}
	if values.Has("token") {
		values.Set("token", "redacted")
		parsed.RawQuery = values.Encode()
	}
	return parsed.String()
}
