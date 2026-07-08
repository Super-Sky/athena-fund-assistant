package data

import (
	"context"
	"testing"
	"time"
)

func TestValidateProviderPassesMockProbeSet(t *testing.T) {
	report := ValidateProvider(context.Background(), NewMockProvider(), ValidationOptions{
		FundCodes:     []string{"QQQ"},
		EquitySymbols: []string{"AAPL"},
		IndexCodes:    []string{"NDX"},
		FXPairs:       []FXPair{{BaseCurrency: "USD", QuoteCurrency: "CNY"}},
		Calendars:     []CalendarProbe{{Market: "US", Date: time.Date(2026, 7, 8, 0, 0, 0, 0, time.UTC)}},
	})
	if !report.Passed {
		t.Fatalf("expected mock provider validation to pass: %+v", report)
	}
	if len(report.Checks) != 5 {
		t.Fatalf("expected 5 checks, got %d", len(report.Checks))
	}
}

func TestValidateProviderRejectsEmptyProbeSet(t *testing.T) {
	report := ValidateProvider(context.Background(), NewMockProvider(), ValidationOptions{})
	if report.Passed {
		t.Fatal("expected empty validation options to fail")
	}
}
