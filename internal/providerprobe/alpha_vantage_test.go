package providerprobe

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProbeAlphaVantageValidatesResponseShapes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("function") {
		case "ETF_PROFILE":
			_, _ = w.Write([]byte(`{"net_assets":"1","net_expense_ratio":"0.001","holdings":[]}`))
		case "GLOBAL_QUOTE":
			_, _ = w.Write([]byte(`{"Global Quote":{"01. symbol":"QQQ"}}`))
		case "TIME_SERIES_DAILY":
			_, _ = w.Write([]byte(`{"Meta Data":{},"Time Series (Daily)":{"2026-07-08":{}}}`))
		case "FX_DAILY":
			_, _ = w.Write([]byte(`{"Meta Data":{},"Time Series FX (Daily)":{"2026-07-08":{}}}`))
		default:
			http.Error(w, "bad function", http.StatusBadRequest)
		}
	}))
	defer server.Close()

	report := ProbeAlphaVantage(context.Background(), AlphaVantageConfig{BaseURL: server.URL, APIKey: "test"})
	if !report.Passed {
		t.Fatalf("expected report to pass: %+v", report)
	}
	if len(report.Checks) != 4 {
		t.Fatalf("expected 4 checks, got %d", len(report.Checks))
	}
}

func TestProbeAlphaVantageFlagsProviderMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"Information":"demo key requires free API key"}`))
	}))
	defer server.Close()

	report := ProbeAlphaVantage(context.Background(), AlphaVantageConfig{BaseURL: server.URL, APIKey: "demo"})
	if report.Passed {
		t.Fatal("expected provider information response to fail validation")
	}
}
