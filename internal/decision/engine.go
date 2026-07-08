package decision

import (
	"fmt"
	"math"
	"time"

	"github.com/Super-Sky/athena-fund-assistant/internal/domain"
)

// Engine generates deterministic decision matrices from profile and portfolio rules.
// Engine 基于用户画像和组合规则生成确定性决策矩阵。
type Engine struct {
	now func() time.Time
}

// NewEngine creates a decision engine using wall-clock generation time.
// NewEngine 创建使用当前时间作为生成时间的决策引擎。
func NewEngine() *Engine {
	return &Engine{now: time.Now}
}

// Generate creates conservative, balanced, and aggressive options with traceable percentages.
// Generate 生成带可追溯百分比的稳健、均衡和激进三档方案。
func (e *Engine) Generate(profile domain.InvestorProfile, portfolio domain.Portfolio, snapshot domain.FundSnapshot) (domain.Diagnosis, domain.DecisionMatrix, error) {
	if err := profile.Validate(); err != nil {
		return domain.Diagnosis{}, domain.DecisionMatrix{}, err
	}
	if err := portfolio.Validate(); err != nil {
		return domain.Diagnosis{}, domain.DecisionMatrix{}, err
	}
	if err := snapshot.Validate(); err != nil {
		return domain.Diagnosis{}, domain.DecisionMatrix{}, err
	}

	holding, ok := portfolio.HoldingByCode(snapshot.Instrument.Code)
	if !ok {
		return domain.Diagnosis{}, domain.DecisionMatrix{}, fmt.Errorf("portfolio does not include instrument %s", snapshot.Instrument.Code)
	}

	diagnosis := diagnose(profile, holding, snapshot)
	options := buildOptions(profile, holding, snapshot, diagnosis)
	matrix := domain.DecisionMatrix{
		ID:          fmt.Sprintf("matrix_%s_%d", snapshot.Instrument.Code, e.now().UnixNano()),
		Instrument:  snapshot.Instrument,
		GeneratedAt: e.now().UTC(),
		Options:     options,
		GovernanceTags: []string{
			"no_auto_trading",
			"multi_option_output",
			"percentage_from_profile_and_portfolio_rules",
			"mock_data_marked",
		},
		Trace: domain.TraceSummary{
			DataProvider:      snapshot.Metadata.Provider,
			DataSource:        snapshot.Metadata.Source,
			DataFetchedAt:     snapshot.Metadata.FetchedAt.Format(time.RFC3339),
			MarketTime:        snapshot.Metadata.MarketTime.Format(time.RFC3339),
			Timezone:          snapshot.Metadata.Timezone,
			LicenseTerms:      snapshot.Metadata.LicenseTerms,
			Confidence:        snapshot.Metadata.Confidence,
			RuleEvaluations:   ruleEvaluations(profile, holding, snapshot),
			GovernanceChecks:  []string{"blocked_single_absolute_conclusion", "included_risks", "included_invalidation", "included_review_timing"},
			MockDataTemporary: snapshot.Metadata.Provider == "mock_provider",
		},
	}
	return diagnosis, matrix, matrix.Validate()
}

func diagnose(profile domain.InvestorProfile, holding domain.PortfolioHolding, snapshot domain.FundSnapshot) domain.Diagnosis {
	var risks []string
	var warnings []string
	var evidence []string

	if snapshot.MaxDrawdownPct > profile.MaxAcceptableDrawdownPct {
		risks = append(risks, fmt.Sprintf("max drawdown %.1f%% exceeds profile limit %.1f%%", snapshot.MaxDrawdownPct, profile.MaxAcceptableDrawdownPct))
	}
	if holding.AllocationPct > profile.SingleInstrumentMaxAllocationPct {
		risks = append(risks, fmt.Sprintf("allocation %.1f%% exceeds single-instrument limit %.1f%%", holding.AllocationPct, profile.SingleInstrumentMaxAllocationPct))
	}
	if snapshot.VolatilityPct > 18 {
		risks = append(risks, fmt.Sprintf("volatility %.1f%% is elevated", snapshot.VolatilityPct))
	}
	if snapshot.Metadata.Provider == "mock_provider" {
		warnings = append(warnings, "mock data is temporary and must not be treated as production market data")
	}

	evidence = append(evidence,
		fmt.Sprintf("one-year return %.1f%%", snapshot.OneYearReturn),
		fmt.Sprintf("max drawdown %.1f%%", snapshot.MaxDrawdownPct),
		fmt.Sprintf("current allocation %.1f%%", holding.AllocationPct),
		fmt.Sprintf("data source %s via %s", snapshot.Metadata.Source, snapshot.Metadata.Provider),
	)

	summary := "holding is within the user's declared constraints"
	if len(risks) > 0 {
		summary = "holding has risk or concentration pressure that needs review"
	}

	return domain.Diagnosis{
		InstrumentCode: snapshot.Instrument.Code,
		Summary:        summary,
		RiskFactors:    risks,
		DataWarnings:   warnings,
		Evidence:       evidence,
	}
}

func buildOptions(profile domain.InvestorProfile, holding domain.PortfolioHolding, snapshot domain.FundSnapshot, diagnosis domain.Diagnosis) []domain.DecisionOption {
	baseTrim := clamp(math.Min(10, math.Max(5, holding.AllocationPct-profile.SingleInstrumentMaxAllocationPct)), 0, 10)
	if holding.AllocationPct <= profile.SingleInstrumentMaxAllocationPct {
		baseTrim = 5
	}
	drawdownPressure := snapshot.MaxDrawdownPct > profile.MaxAcceptableDrawdownPct
	momentumPositive := snapshot.OneYearReturn > 8 && snapshot.DailyChangePct >= 0

	conservativeChange := -baseTrim
	if drawdownPressure {
		conservativeChange = -math.Min(15, baseTrim+5)
	}

	balancedChange := 0.0
	if holding.AllocationPct > profile.SingleInstrumentMaxAllocationPct {
		balancedChange = -5
	}

	remainingCap := profile.SingleInstrumentMaxAllocationPct - holding.AllocationPct
	aggressiveChange := clamp(remainingCap, 0, 5)
	if profile.RiskPreference == domain.RiskAggressive && momentumPositive && !drawdownPressure {
		aggressiveChange = clamp(remainingCap, 0, 10)
	}
	if drawdownPressure {
		aggressiveChange = 0
	}

	commonEvidence := append([]string{}, diagnosis.Evidence...)
	return []domain.DecisionOption{
		{
			ID:                  "option_conservative",
			Style:               "conservative",
			Action:              "reduce exposure and move the difference to cash or lower-volatility funds",
			AllocationChangePct: conservativeChange,
			Conditions:          []string{"use when drawdown tolerance or concentration limit matters more than upside capture"},
			Evidence:            commonEvidence,
			Risks:               append([]string{"may miss a short-term rebound"}, diagnosis.RiskFactors...),
			Invalidation:        "if drawdown falls back inside profile limit and volatility normalizes for two review periods",
			ReviewAfterDays:     14,
			PortfolioImpact:     fmt.Sprintf("target allocation moves from %.1f%% to %.1f%%", holding.AllocationPct, clamp(holding.AllocationPct+conservativeChange, 0, 100)),
			StrategyBasis:       []string{"profile.max_acceptable_drawdown_pct", "profile.single_instrument_max_allocation_pct", "portfolio.current_allocation_pct"},
		},
		{
			ID:                  "option_balanced",
			Style:               "balanced",
			Action:              "hold or make a small rebalance while waiting for fresher confirmation",
			AllocationChangePct: balancedChange,
			Conditions:          []string{"use when evidence is mixed or data source confidence is not production-grade"},
			Evidence:            commonEvidence,
			Risks:               append([]string{"risk may persist if market weakness broadens"}, diagnosis.RiskFactors...),
			Invalidation:        "if the instrument breaches the user's max drawdown or allocation limit again",
			ReviewAfterDays:     7,
			PortfolioImpact:     fmt.Sprintf("target allocation moves from %.1f%% to %.1f%%", holding.AllocationPct, clamp(holding.AllocationPct+balancedChange, 0, 100)),
			StrategyBasis:       []string{"mock_data_confidence", "portfolio.current_allocation_pct", "decision_template.balanced_review"},
		},
		{
			ID:                  "option_aggressive",
			Style:               "aggressive",
			Action:              "add only within the single-instrument cap and keep a strict review trigger",
			AllocationChangePct: aggressiveChange,
			Conditions:          []string{"use only when the user accepts volatility and the position remains inside allocation limits"},
			Evidence:            commonEvidence,
			Risks:               append([]string{"larger drawdown if US or China risk assets reverse"}, diagnosis.RiskFactors...),
			Invalidation:        "if price momentum reverses, max drawdown expands, or allocation exceeds the configured cap",
			ReviewAfterDays:     5,
			PortfolioImpact:     fmt.Sprintf("target allocation moves from %.1f%% to %.1f%%", holding.AllocationPct, clamp(holding.AllocationPct+aggressiveChange, 0, 100)),
			StrategyBasis:       []string{"profile.risk_preference", "profile.single_instrument_max_allocation_pct", "snapshot.one_year_return_pct"},
		},
	}
}

func ruleEvaluations(profile domain.InvestorProfile, holding domain.PortfolioHolding, snapshot domain.FundSnapshot) []string {
	return []string{
		fmt.Sprintf("risk_preference=%s", profile.RiskPreference),
		fmt.Sprintf("drawdown %.1f <= limit %.1f => %t", snapshot.MaxDrawdownPct, profile.MaxAcceptableDrawdownPct, snapshot.MaxDrawdownPct <= profile.MaxAcceptableDrawdownPct),
		fmt.Sprintf("allocation %.1f <= single limit %.1f => %t", holding.AllocationPct, profile.SingleInstrumentMaxAllocationPct, holding.AllocationPct <= profile.SingleInstrumentMaxAllocationPct),
		fmt.Sprintf("data_provider=%s license=%s", snapshot.Metadata.Provider, snapshot.Metadata.LicenseTerms),
	}
}

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
