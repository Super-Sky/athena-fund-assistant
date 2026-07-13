package governance

import (
	"testing"
	"time"

	"github.com/Super-Sky/athena-fund-assistant/internal/domain"
)

func TestGateAllowsTraceableMultiOptionOutput(t *testing.T) {
	result := NewGate().Evaluate(validMatrix())
	if !result.Allowed() || result.Decision != StatusPassed {
		t.Fatalf("result = %#v, want passed", result)
	}
}

func TestGateBlocksDisallowedFinancialLanguage(t *testing.T) {
	cases := []string{"guaranteed return", "automatically place order", "must buy now"}
	for _, action := range cases {
		t.Run(action, func(t *testing.T) {
			matrix := validMatrix()
			matrix.Options[0].Action = action
			result := NewGate().Evaluate(matrix)
			if result.Decision != StatusBlocked || result.Allowed() {
				t.Fatalf("result = %#v, want blocked", result)
			}
		})
	}
}

func TestGateFlagsMissingSourceOrFreshness(t *testing.T) {
	matrix := validMatrix()
	matrix.Trace.DataFetchedAt = ""
	result := NewGate().Evaluate(matrix)
	if result.Decision != StatusFlagged || !result.Allowed() {
		t.Fatalf("result = %#v, want flagged but allowed", result)
	}
}

func TestGateFlagsMissingRiskAndInvalidation(t *testing.T) {
	matrix := validMatrix()
	matrix.Options[0].Risks = nil
	matrix.Options[1].Invalidation = ""
	result := NewGate().Evaluate(matrix)
	if result.Decision != StatusFlagged || !result.Allowed() {
		t.Fatalf("result = %#v, want flagged but allowed", result)
	}
}

func TestGateBlocksAllocationChangeWithoutBasis(t *testing.T) {
	matrix := validMatrix()
	matrix.Options[2].StrategyBasis = nil
	result := NewGate().Evaluate(matrix)
	if result.Decision != StatusBlocked || result.Allowed() {
		t.Fatalf("result = %#v, want blocked", result)
	}
}

func validMatrix() domain.DecisionMatrix {
	now := time.Now().UTC().Format(time.RFC3339)
	return domain.DecisionMatrix{
		Options: []domain.DecisionOption{
			{ID: "conservative", Action: "reduce exposure", AllocationChangePct: -5, Risks: []string{"may miss rebound"}, Invalidation: "risk normalizes", ReviewAfterDays: 14, StrategyBasis: []string{"profile.limit"}},
			{ID: "balanced", Action: "hold", AllocationChangePct: 0, Risks: []string{"weakness can continue"}, Invalidation: "limit breached", ReviewAfterDays: 7, StrategyBasis: []string{"template"}},
			{ID: "aggressive", Action: "add within cap", AllocationChangePct: 5, Risks: []string{"drawdown can expand"}, Invalidation: "momentum reverses", ReviewAfterDays: 5, StrategyBasis: []string{"profile.cap"}},
		},
		Trace: domain.TraceSummary{DataSource: "test", DataProvider: "test", DataFetchedAt: now, MarketTime: now, Timezone: "UTC"},
	}
}
