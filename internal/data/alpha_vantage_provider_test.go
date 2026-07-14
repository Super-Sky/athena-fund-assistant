package data

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAlphaVantageProviderNormalizesETFEquityIndexProxyAndFX(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("function") {
		case "ETF_PROFILE":
			_, _ = w.Write([]byte(`{"name":"Invesco QQQ Trust","net_assets":"300000000000","net_expense_ratio":"0.20","holdings":[{"name":"NVIDIA"}]}`))
		case "GLOBAL_QUOTE":
			_, _ = w.Write([]byte(`{"Global Quote":{"05. price":"500.00","07. latest trading day":"2026-07-10","10. change percent":"1.25%"}}`))
		case "TIME_SERIES_DAILY":
			if r.URL.Query().Get("outputsize") != "full" {
				http.Error(w, "daily history must request full output", http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte(`{"Time Series (Daily)":{"2026-07-10":{"4. close":"500.00"},"2026-07-09":{"4. close":"490.00"},"2026-07-08":{"4. close":"480.00"}}}`))
		case "FX_DAILY":
			_, _ = w.Write([]byte(`{"Time Series FX (Daily)":{"2026-07-10":{"4. close":"7.1800"}}}`))
		default:
			http.Error(w, "unsupported", http.StatusBadRequest)
		}
	}))
	defer server.Close()

	provider, err := NewAlphaVantageProvider(AlphaVantageConfig{BaseURL: server.URL, APIKey: "test", Now: func() time.Time { return time.Date(2026, 7, 11, 0, 0, 0, 0, time.UTC) }})
	if err != nil {
		t.Fatalf("NewAlphaVantageProvider() error = %v", err)
	}
	ctx := context.Background()
	fund, err := provider.GetFundSnapshot(ctx, "QQQ")
	if err != nil {
		t.Fatalf("GetFundSnapshot() error = %v", err)
	}
	if fund.Instrument.Name != "Invesco QQQ Trust" || fund.Price != 500 || fund.Metadata.Provider != "alpha_vantage_provider" || fund.Metadata.Timezone != "America/New_York" || fund.Metadata.RawPayloadHash == "" {
		t.Fatalf("fund = %#v", fund)
	}
	if err := fund.Validate(); err != nil {
		t.Fatalf("fund.Validate() error = %v", err)
	}
	equity, err := provider.GetEquitySnapshot(ctx, "AAPL")
	if err != nil || equity.Price != 500 {
		t.Fatalf("GetEquitySnapshot() = %#v, %v", equity, err)
	}
	index, err := provider.GetIndexSnapshot(ctx, "SPX")
	if err != nil || index.Code != "SPX" || index.Metadata.Confidence >= 0.8 {
		t.Fatalf("GetIndexSnapshot() = %#v, %v", index, err)
	}
	fx, err := provider.GetFXRate(ctx, "USD", "CNY")
	if err != nil || fx.Rate != 7.18 || fx.Metadata.Timezone != "UTC" {
		t.Fatalf("GetFXRate() = %#v, %v", fx, err)
	}
	if err := provider.ValidateCredentials(ctx); err != nil {
		t.Fatalf("ValidateCredentials() error = %v", err)
	}
}

func TestAlphaVantageProviderRejectsMissingCredentialAndCalendarAssumption(t *testing.T) {
	if _, err := NewAlphaVantageProvider(AlphaVantageConfig{}); err == nil {
		t.Fatal("expected missing API key to be rejected")
	}
	provider, err := NewAlphaVantageProvider(AlphaVantageConfig{APIKey: "test"})
	if err != nil {
		t.Fatalf("NewAlphaVantageProvider() error = %v", err)
	}
	if _, err := provider.GetMarketCalendar(context.Background(), "US", time.Now()); err == nil {
		t.Fatal("expected calendar capability error")
	}
}
