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

	ctx := context.Background()
	store := NewMemoryStore()
	matrix := testDecisionMatrix()

	entry, review, err := store.Create(ctx, matrix, "option-aggressive", "accept short-term volatility")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if entry.MatrixID != matrix.ID {
		t.Fatalf("Create() MatrixID = %q, want %q", entry.MatrixID, matrix.ID)
	}
	if entry.SelectedOptionID != "option-aggressive" {
		t.Fatalf("Create() SelectedOptionID = %q", entry.SelectedOptionID)
	}
	if !reflect.DeepEqual(entry.EvidenceSnapshot, matrix) {
		t.Fatalf("Create() evidence snapshot differs from input matrix")
	}
	wantDueAt := entry.CreatedAt.AddDate(0, 0, 3)
	if !review.DueAt.Equal(wantDueAt) {
		t.Fatalf("Create() review DueAt = %v, want %v", review.DueAt, wantDueAt)
	}

	gotEntry, err := store.Entry(ctx, entry.ID)
	if err != nil {
		t.Fatalf("Entry() error = %v", err)
	}
	if !reflect.DeepEqual(gotEntry, entry) {
		t.Fatalf("Entry() = %#v, want %#v", gotEntry, entry)
	}
	gotReview, err := store.Review(ctx, review.ID)
	if err != nil {
		t.Fatalf("Review() error = %v", err)
	}
	if !reflect.DeepEqual(gotReview, review) {
		t.Fatalf("Review() = %#v, want %#v", gotReview, review)
	}
	if err := store.Ping(ctx); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}
	if err := store.Close(ctx); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestMemoryStoreNotFoundSentinels(t *testing.T) {
	t.Parallel()

	store := NewMemoryStore()
	if _, err := store.Entry(context.Background(), "missing"); !errors.Is(err, ErrEntryNotFound) {
		t.Fatalf("Entry() error = %v, want ErrEntryNotFound", err)
	}
	if _, err := store.Review(context.Background(), "missing"); !errors.Is(err, ErrReviewNotFound) {
		t.Fatalf("Review() error = %v, want ErrReviewNotFound", err)
	}
}

func TestMemoryStoreHonorsCanceledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	store := NewMemoryStore()

	if _, _, err := store.Create(ctx, testDecisionMatrix(), "option-balanced", ""); !errors.Is(err, context.Canceled) {
		t.Fatalf("Create() error = %v, want context.Canceled", err)
	}
	if _, err := store.Entry(ctx, "missing"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Entry() error = %v, want context.Canceled", err)
	}
	if _, err := store.Review(ctx, "missing"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Review() error = %v, want context.Canceled", err)
	}
	if err := store.Ping(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Ping() error = %v, want context.Canceled", err)
	}
	if err := store.Close(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Close() error = %v, want context.Canceled", err)
	}
}

func TestMemoryStoreRejectsUnknownOption(t *testing.T) {
	t.Parallel()

	store := NewMemoryStore()
	if _, _, err := store.Create(context.Background(), testDecisionMatrix(), "missing", ""); err == nil {
		t.Fatal("Create() error = nil, want unknown option error")
	}
}

func testDecisionMatrix() domain.DecisionMatrix {
	generatedAt := time.Date(2026, time.July, 9, 9, 30, 0, 123456789, time.UTC)
	return domain.DecisionMatrix{
		ID: "matrix-test-001",
		Instrument: domain.FundInstrument{
			Code:     "510300",
			Name:     "CSI 300 ETF",
			Market:   "CN",
			Currency: "CNY",
			Type:     domain.InstrumentETF,
		},
		GeneratedAt: generatedAt,
		Options: []domain.DecisionOption{
			testDecisionOption("option-conservative", "conservative", 30),
			testDecisionOption("option-balanced", "balanced", 14),
			testDecisionOption("option-aggressive", "aggressive", 3),
		},
		GovernanceTags: []string{"not-investment-advice", "user-decision-required"},
		Trace: domain.TraceSummary{
			DataProvider:      "verified-provider",
			DataSource:        "provider-endpoint",
			DataFetchedAt:     generatedAt.Add(-time.Minute).Format(time.RFC3339Nano),
			MarketTime:        generatedAt.Add(-2 * time.Minute).Format(time.RFC3339Nano),
			Timezone:          "Asia/Shanghai",
			LicenseTerms:      "test-only",
			Confidence:        0.92,
			RuleEvaluations:   []string{"allocation-limit:pass"},
			GovernanceChecks:  []string{"three-options:pass"},
			AthenaRunID:       "run-test-001",
			MockDataTemporary: false,
		},
	}
}

func testDecisionOption(id, style string, reviewAfterDays int) domain.DecisionOption {
	return domain.DecisionOption{
		ID:                  id,
		Style:               style,
		Action:              "rebalance",
		AllocationChangePct: 5,
		Conditions:          []string{"market remains liquid"},
		Evidence:            []string{"valuation percentile is below its five-year median"},
		Risks:               []string{"benchmark drawdown may continue"},
		Invalidation:        "tracking error exceeds policy limit",
		ReviewAfterDays:     reviewAfterDays,
		PortfolioImpact:     "keeps allocation within profile limit",
		StrategyBasis:       []string{"risk budget", "valuation"},
	}
}
