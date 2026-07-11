// This file verifies the journal store contract without external services.
// 本文件在不依赖外部服务的情况下验证决策日志 store 契约。
package journal

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/Super-Sky/athena-fund-assistant/internal/domain"
)

func TestMemoryStoreCreateAndRead(t *testing.T) {
	t.Parallel()
	store := NewMemoryStore()
	matrix := testDecisionMatrix()
	entry, review, err := store.Create(context.Background(), matrix, "option-balanced", "preserve the full evidence snapshot")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if !reflect.DeepEqual(entry.EvidenceSnapshot, matrix) {
		t.Fatal("Create() evidence snapshot differs from input matrix")
	}
	if want := entry.CreatedAt.AddDate(0, 0, 14); !review.DueAt.Equal(want) {
		t.Fatalf("Review due time = %v, want %v", review.DueAt, want)
	}
	gotEntry, err := store.Entry(context.Background(), entry.ID)
	if err != nil || !reflect.DeepEqual(gotEntry, entry) {
		t.Fatalf("Entry() = %#v, %v; want %#v", gotEntry, err, entry)
	}
	gotReview, err := store.Review(context.Background(), review.ID)
	if err != nil || !reflect.DeepEqual(gotReview, review) {
		t.Fatalf("Review() = %#v, %v; want %#v", gotReview, err, review)
	}
}

func TestMemoryStoreHonorsContextAndNotFound(t *testing.T) {
	t.Parallel()
	store := NewMemoryStore()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := store.Create(ctx, testDecisionMatrix(), "option-balanced", ""); !errors.Is(err, context.Canceled) {
		t.Fatalf("Create() error = %v, want context.Canceled", err)
	}
	if _, err := store.Entry(context.Background(), "missing"); !errors.Is(err, ErrEntryNotFound) {
		t.Fatalf("Entry() error = %v, want ErrEntryNotFound", err)
	}
	if _, err := store.Review(context.Background(), "missing"); !errors.Is(err, ErrReviewNotFound) {
		t.Fatalf("Review() error = %v, want ErrReviewNotFound", err)
	}
}

func testDecisionMatrix() domain.DecisionMatrix {
	generatedAt := time.Date(2026, time.July, 11, 9, 30, 0, 0, time.UTC)
	return domain.DecisionMatrix{
		ID:          "matrix-journal-test",
		Instrument:  domain.FundInstrument{Code: "510300", Name: "CSI 300 ETF", Market: "CN", Currency: "CNY", Type: domain.InstrumentETF},
		GeneratedAt: generatedAt,
		Options: []domain.DecisionOption{
			{ID: "option-conservative", Style: "conservative", Action: "rebalance", AllocationChangePct: 3, Conditions: []string{"liquidity is normal"}, Evidence: []string{"risk budget"}, Risks: []string{"drawdown can continue"}, Invalidation: "allocation exceeds profile limit", ReviewAfterDays: 30, PortfolioImpact: "within profile limit", StrategyBasis: []string{"risk budget"}},
			{ID: "option-balanced", Style: "balanced", Action: "rebalance", AllocationChangePct: 5, Conditions: []string{"liquidity is normal"}, Evidence: []string{"risk budget"}, Risks: []string{"drawdown can continue"}, Invalidation: "allocation exceeds profile limit", ReviewAfterDays: 14, PortfolioImpact: "within profile limit", StrategyBasis: []string{"risk budget"}},
			{ID: "option-aggressive", Style: "aggressive", Action: "rebalance", AllocationChangePct: 8, Conditions: []string{"liquidity is normal"}, Evidence: []string{"risk budget"}, Risks: []string{"drawdown can continue"}, Invalidation: "allocation exceeds profile limit", ReviewAfterDays: 3, PortfolioImpact: "within profile limit", StrategyBasis: []string{"risk budget"}},
		},
		GovernanceTags: []string{"not-investment-advice", "user-decision-required"},
		Trace:          domain.TraceSummary{DataProvider: "test-provider", DataSource: "test-source", DataFetchedAt: generatedAt.Add(-time.Minute).Format(time.RFC3339), MarketTime: generatedAt.Add(-2 * time.Minute).Format(time.RFC3339), Timezone: "Asia/Shanghai", LicenseTerms: "test-only", Confidence: 0.9, RuleEvaluations: []string{"allocation-limit:pass"}, GovernanceChecks: []string{"three-options:pass"}},
	}
}
