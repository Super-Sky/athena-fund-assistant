package account

import (
	"context"
	"testing"
	"time"

	"github.com/Super-Sky/athena-fund-assistant/internal/domain"
)

func TestMemoryStoreOverviewPreservesAccountPerformanceTrace(t *testing.T) {
	store := NewMemoryStore()
	overview, err := store.Overview(context.Background(), "demo-user")
	if err != nil {
		t.Fatalf("overview error = %v", err)
	}
	if err := overview.Validate(); err != nil {
		t.Fatalf("overview invalid = %v", err)
	}
	if overview.BaseCurrency != "CNY" {
		t.Fatalf("base currency = %q, want CNY", overview.BaseCurrency)
	}
	if overview.TotalMarketValue <= overview.TotalCostValue {
		t.Fatalf("total market value = %.2f, cost = %.2f, want gains in demo data", overview.TotalMarketValue, overview.TotalCostValue)
	}
	if len(overview.PerformanceTrend) < 2 {
		t.Fatalf("trend length = %d, want at least 2", len(overview.PerformanceTrend))
	}
	if !overview.Trace.MockDataTemporary || overview.Trace.ReadOnlySyncAvailable {
		t.Fatalf("trace = %#v, want mock data and no readonly sync", overview.Trace)
	}
}

func TestMemoryStoreReplaceHoldingsRecalculatesAllocations(t *testing.T) {
	store := NewMemoryStore()
	now := time.Now().UTC()
	overview, err := store.ReplaceHoldings(context.Background(), "manual-user", []domain.AccountHoldingSnapshot{
		{
			InstrumentCode:    "510300",
			InstrumentName:    "Manual CSI 300 ETF",
			Market:            "CN",
			Currency:          "CNY",
			Units:             100,
			CostBasis:         4,
			CurrentPrice:      5,
			FXToBase:          1,
			DataAuthorization: "manual_entry",
			Metadata: domain.SourceMetadata{
				Source:        "unit_test",
				Provider:      "manual",
				FetchedAt:     now,
				MarketTime:    now,
				Timezone:      "Asia/Shanghai",
				Delay:         "0m",
				LicenseTerms:  "test",
				Confidence:    0.9,
				SchemaVersion: "account_snapshot.v1",
			},
		},
	})
	if err != nil {
		t.Fatalf("replace holdings error = %v", err)
	}
	if overview.TotalMarketValue != 500 {
		t.Fatalf("total market value = %.2f, want 500", overview.TotalMarketValue)
	}
	if overview.Holdings[0].AllocationPct != 100 {
		t.Fatalf("allocation = %.2f, want 100", overview.Holdings[0].AllocationPct)
	}
}
