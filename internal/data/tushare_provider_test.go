package data

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTushareProviderNormalizesFundIndexAndCalendar(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			APIName string `json:"api_name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		switch request.APIName {
		case "fund_basic":
			_, _ = w.Write([]byte(`{"code":0,"msg":"","data":{"fields":["ts_code","name","management","fund_type","m_fee","issue_amount"],"items":[["000001.OF","Sample Fund","Sample Manager","Mixed","0.15","100"]]}}`))
		case "fund_nav":
			_, _ = w.Write([]byte(`{"code":0,"msg":"","data":{"fields":["ts_code","end_date","unit_nav","accum_nav","net_asset","total_netasset"],"items":[["000001.OF","20260710","1.234","1.234","100","200"]]}}`))
		case "index_daily":
			_, _ = w.Write([]byte(`{"code":0,"msg":"","data":{"fields":["ts_code","trade_date","close","pct_chg"],"items":[["000300.SH","20260710","4000","1.20"]]}}`))
		case "trade_cal":
			_, _ = w.Write([]byte(`{"code":0,"msg":"","data":{"fields":["exchange","cal_date","is_open","pretrade_date"],"items":[["SSE","20260710","1","20260709"]]}}`))
		default:
			http.Error(w, "unsupported", http.StatusBadRequest)
		}
	}))
	defer server.Close()

	provider, err := NewTushareProvider(TushareConfig{BaseURL: server.URL, Token: "test", Now: func() time.Time { return time.Date(2026, 7, 11, 0, 0, 0, 0, time.UTC) }})
	if err != nil {
		t.Fatalf("NewTushareProvider() error = %v", err)
	}
	ctx := context.Background()
	fund, err := provider.GetFundSnapshot(ctx, "000001")
	if err != nil || fund.NAV != 1.234 || fund.Metadata.Timezone != "Asia/Shanghai" {
		t.Fatalf("GetFundSnapshot() = %#v, %v", fund, err)
	}
	if err := fund.Validate(); err != nil {
		t.Fatalf("fund.Validate() error = %v", err)
	}
	index, err := provider.GetIndexSnapshot(ctx, "000300")
	if err != nil || index.Level != 4000 || index.Code != "000300.SH" {
		t.Fatalf("GetIndexSnapshot() = %#v, %v", index, err)
	}
	calendar, err := provider.GetMarketCalendar(ctx, "CN", time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC))
	if err != nil || !calendar.IsTradingDay || calendar.Timezone != "Asia/Shanghai" {
		t.Fatalf("GetMarketCalendar() = %#v, %v", calendar, err)
	}
	if err := provider.ValidateCredentials(ctx); err != nil {
		t.Fatalf("ValidateCredentials() error = %v", err)
	}
}

func TestTushareProviderRejectsMissingCredentialAndUnsupportedFX(t *testing.T) {
	if _, err := NewTushareProvider(TushareConfig{}); err == nil {
		t.Fatal("expected missing token to be rejected")
	}
	provider, err := NewTushareProvider(TushareConfig{Token: "test"})
	if err != nil {
		t.Fatalf("NewTushareProvider() error = %v", err)
	}
	if _, err := provider.GetFXRate(context.Background(), "USD", "CNY"); err == nil {
		t.Fatal("expected FX capability error")
	}
}
