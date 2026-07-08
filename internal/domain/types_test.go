package domain

import (
	"testing"
	"time"
)

func TestDecisionMatrixRequiresThreeGovernedOptions(t *testing.T) {
	matrix := DecisionMatrix{
		ID:          "matrix_test",
		GeneratedAt: time.Now(),
		Options: []DecisionOption{
			option("conservative"),
			option("balanced"),
			option("aggressive"),
		},
	}
	if err := matrix.Validate(); err != nil {
		t.Fatalf("expected matrix to validate: %v", err)
	}
}

func TestDecisionMatrixRejectsSinglePathOutput(t *testing.T) {
	matrix := DecisionMatrix{
		ID:          "matrix_test",
		GeneratedAt: time.Now(),
		Options:     []DecisionOption{option("aggressive")},
	}
	if err := matrix.Validate(); err == nil {
		t.Fatal("expected single-path matrix to fail validation")
	}
}

func option(style string) DecisionOption {
	return DecisionOption{
		ID:                  "option_" + style,
		Style:               style,
		Action:              "hold",
		AllocationChangePct: 0,
		Evidence:            []string{"source evidence"},
		Risks:               []string{"risk"},
		Invalidation:        "condition",
		ReviewAfterDays:     7,
		StrategyBasis:       []string{"profile"},
	}
}
