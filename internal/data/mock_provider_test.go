package data

import (
	"context"
	"testing"
	"time"
)

func TestMockProviderCoversUSMarketSupportData(t *testing.T) {
	provider := NewMockProvider()

	if _, err := provider.GetEquitySnapshot(context.Background(), "AAPL"); err != nil {
		t.Fatalf("equity snapshot: %v", err)
	}
	if _, err := provider.GetIndexSnapshot(context.Background(), "NDX"); err != nil {
		t.Fatalf("index snapshot: %v", err)
	}
	fx, err := provider.GetFXRate(context.Background(), "USD", "CNY")
	if err != nil {
		t.Fatalf("fx rate: %v", err)
	}
	if fx.Rate <= 0 {
		t.Fatalf("expected positive fx rate, got %.4f", fx.Rate)
	}
	calendar, err := provider.GetMarketCalendar(context.Background(), "US", time.Date(2026, 7, 8, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("market calendar: %v", err)
	}
	if calendar.Timezone != "America/New_York" {
		t.Fatalf("unexpected timezone %s", calendar.Timezone)
	}
}
