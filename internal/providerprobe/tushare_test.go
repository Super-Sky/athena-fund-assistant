package providerprobe

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProbeTushareRequiresToken(t *testing.T) {
	report := ProbeTushare(context.Background(), TushareConfig{})
	if report.Passed {
		t.Fatal("expected missing token to fail")
	}
}

func TestProbeTushareValidatesEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":0,"msg":"","data":{"fields":["ts_code","name","fund_type","end_date","unit_nav","trade_date","close"],"items":[["000001.OF","name","type","20260708","1.0","20260708","4000"]]}}`))
	}))
	defer server.Close()

	report := ProbeTushare(context.Background(), TushareConfig{BaseURL: server.URL, Token: "test"})
	if !report.Passed {
		t.Fatalf("expected report to pass: %+v", report)
	}
}
