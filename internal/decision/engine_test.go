package decision

import (
	"context"
	"testing"

	"github.com/Super-Sky/athena-fund-assistant/internal/data"
	"github.com/Super-Sky/athena-fund-assistant/internal/domain"
)

func TestEngineGeneratesTraceableThreeOptionMatrix(t *testing.T) {
	provider := data.NewMockProvider()
	snapshot, err := provider.GetFundSnapshot(context.Background(), "QQQ")
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}

	profile := domain.InvestorProfile{
		RiskPreference:                   domain.RiskAggressive,
		InvestmentHorizonMonths:          36,
		MaxAcceptableDrawdownPct:         30,
		SingleInstrumentMaxAllocationPct: 25,
		CashPreferencePct:                10,
		DefaultDecisionStyle:             "three_options",
	}
	portfolio := domain.Portfolio{
		Holdings: []domain.PortfolioHolding{{
			InstrumentCode: "QQQ",
			InstrumentName: "Sample Nasdaq 100 ETF",
			Market:         "US",
			Currency:       "USD",
			HoldingAmount:  10000,
			CostBasis:      450,
			AllocationPct:  18,
			UserThesis:     "long-term US technology exposure",
		}},
	}

	diagnosis, matrix, err := NewEngine().Generate(profile, portfolio, snapshot)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(diagnosis.Evidence) == 0 {
		t.Fatal("expected diagnosis evidence")
	}
	if err := matrix.Validate(); err != nil {
		t.Fatalf("matrix validate: %v", err)
	}
	if !matrix.Trace.MockDataTemporary {
		t.Fatal("expected mock data to be explicitly marked")
	}
	for _, option := range matrix.Options {
		if len(option.StrategyBasis) == 0 {
			t.Fatalf("option %s missing strategy basis", option.ID)
		}
	}
}

func TestEngineDoesNotAddAboveSingleInstrumentCap(t *testing.T) {
	provider := data.NewMockProvider()
	snapshot, err := provider.GetFundSnapshot(context.Background(), "QQQ")
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}

	profile := domain.InvestorProfile{
		RiskPreference:                   domain.RiskAggressive,
		InvestmentHorizonMonths:          36,
		MaxAcceptableDrawdownPct:         30,
		SingleInstrumentMaxAllocationPct: 20,
		CashPreferencePct:                5,
		DefaultDecisionStyle:             "three_options",
	}
	portfolio := domain.Portfolio{
		Holdings: []domain.PortfolioHolding{{
			InstrumentCode: "QQQ",
			Market:         "US",
			Currency:       "USD",
			HoldingAmount:  10000,
			CostBasis:      450,
			AllocationPct:  22,
		}},
	}

	_, matrix, err := NewEngine().Generate(profile, portfolio, snapshot)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	for _, option := range matrix.Options {
		if option.ID == "option_aggressive" && option.AllocationChangePct > 0 {
			t.Fatalf("aggressive option should not add above cap, got %.2f", option.AllocationChangePct)
		}
	}
}
