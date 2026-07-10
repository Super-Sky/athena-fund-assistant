package data

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCSVProviderCoversChinaAndUSProbeSet(t *testing.T) {
	provider, err := NewCSVProvider(filepath.Join("..", "..", "examples", "market-data-sample.csv"))
	if err != nil {
		t.Fatalf("csv provider: %v", err)
	}

	report := ValidateProvider(context.Background(), provider, ValidationOptions{
		FundCodes:     []string{"510300", "QQQ"},
		EquitySymbols: []string{"AAPL"},
		IndexCodes:    []string{"000300", "NDX"},
		FXPairs:       []FXPair{{BaseCurrency: "USD", QuoteCurrency: "CNY"}},
		Calendars: []CalendarProbe{
			{Market: "CN", Date: time.Date(2026, 7, 8, 0, 0, 0, 0, time.UTC)},
			{Market: "US", Date: time.Date(2026, 7, 8, 0, 0, 0, 0, time.UTC)},
		},
	})
	if !report.Passed {
		t.Fatalf("expected csv provider validation to pass: %+v", report)
	}

	snapshot, err := provider.GetFundSnapshot(context.Background(), "QQQ")
	if err != nil {
		t.Fatalf("fund snapshot: %v", err)
	}
	if snapshot.Metadata.Provider != "csv_provider" {
		t.Fatalf("unexpected provider %s", snapshot.Metadata.Provider)
	}
	if snapshot.Metadata.LicenseTerms != "user_supplied_csv_for_local_mvp_not_licensed_live_feed" {
		t.Fatalf("unexpected license terms %s", snapshot.Metadata.LicenseTerms)
	}
	if snapshot.Metadata.RawPayloadHash == "" {
		t.Fatal("expected raw payload hash")
	}
}

func TestCSVProviderRejectsMissingProvenance(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.csv")
	csv := "kind,code,name,market,currency,type,price,fetched_at,market_time,timezone,license_terms,confidence\n" +
		"ETF,QQQ,Bad ETF,US,USD,etf,512.35,2026-07-08T20:20:00Z,2026-07-08T16:00:00-04:00,America/New_York,,0.70\n"
	if err := os.WriteFile(path, []byte(csv), 0o600); err != nil {
		t.Fatalf("write csv: %v", err)
	}

	if _, err := NewCSVProvider(path); err == nil {
		t.Fatal("expected missing license terms to fail")
	}
}
