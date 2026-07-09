package account

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Super-Sky/athena-fund-assistant/internal/domain"
)

func TestPostgresStoreOverviewAndReplaceHoldings(t *testing.T) {
	dsn := os.Getenv("ATHENA_FUND_PG_TEST_DSN")
	if dsn == "" {
		t.Skip("ATHENA_FUND_PG_TEST_DSN is not set")
	}
	ctx := context.Background()
	store, err := NewPostgresStore(ctx, dsn)
	if err != nil {
		t.Fatalf("new postgres store error = %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close postgres store error = %v", err)
		}
	}()

	overview, err := store.Overview(ctx, "demo-user")
	if err != nil {
		t.Fatalf("overview error = %v", err)
	}
	if overview.TotalMarketValue <= 0 || len(overview.Holdings) == 0 {
		t.Fatalf("unexpected overview = %#v", overview)
	}

	now := time.Now().UTC()
	replaced, err := store.ReplaceHoldings(ctx, "itest-user", []domain.AccountHoldingSnapshot{{
		InstrumentCode:    "510300",
		InstrumentName:    "Integration CSI 300 ETF",
		Market:            "CN",
		Currency:          "CNY",
		Units:             100,
		CostBasis:         4,
		CurrentPrice:      4.5,
		FXToBase:          1,
		DataAuthorization: "manual_entry",
		Metadata: domain.SourceMetadata{
			Source:        "integration_test",
			Provider:      "manual",
			FetchedAt:     now,
			MarketTime:    now,
			Timezone:      "Asia/Shanghai",
			Delay:         "0m",
			LicenseTerms:  "test",
			Confidence:    0.9,
			SchemaVersion: "account_snapshot.v1",
		},
	}})
	if err != nil {
		t.Fatalf("replace holdings error = %v", err)
	}
	if replaced.TotalMarketValue != 450 {
		t.Fatalf("total market value = %.2f, want 450", replaced.TotalMarketValue)
	}
}
