package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// SourceMetadata records provenance for every normalized market data point.
// SourceMetadata 记录每条标准化市场数据的来源和新鲜度信息。
type SourceMetadata struct {
	Source         string    `json:"source"`
	Provider       string    `json:"provider"`
	FetchedAt      time.Time `json:"fetched_at"`
	MarketTime     time.Time `json:"market_time"`
	Timezone       string    `json:"timezone"`
	Delay          string    `json:"delay"`
	LicenseTerms   string    `json:"license_terms"`
	Confidence     float64   `json:"confidence"`
	SchemaVersion  string    `json:"schema_version"`
	RawPayloadHash string    `json:"raw_payload_hash,omitempty"`
}

// Validate checks whether data provenance is strong enough for a decision trace.
// Validate 检查数据来源信息是否足以进入决策 trace。
func (m SourceMetadata) Validate() error {
	var missing []string
	if m.Source == "" {
		missing = append(missing, "source")
	}
	if m.Provider == "" {
		missing = append(missing, "provider")
	}
	if m.FetchedAt.IsZero() {
		missing = append(missing, "fetched_at")
	}
	if m.MarketTime.IsZero() {
		missing = append(missing, "market_time")
	}
	if m.Timezone == "" {
		missing = append(missing, "timezone")
	}
	if m.LicenseTerms == "" {
		missing = append(missing, "license_terms")
	}
	if m.SchemaVersion == "" {
		missing = append(missing, "schema_version")
	}
	if len(missing) > 0 {
		return fmt.Errorf("source metadata missing %s", strings.Join(missing, ", "))
	}
	if m.Confidence <= 0 || m.Confidence > 1 {
		return errors.New("source metadata confidence must be between 0 and 1")
	}
	return nil
}

// RiskPreference describes the user's default risk posture.
// RiskPreference 描述用户默认风险偏好。
type RiskPreference string

const (
	RiskConservative RiskPreference = "conservative"
	RiskBalanced     RiskPreference = "balanced"
	RiskAggressive   RiskPreference = "aggressive"
)

// InvestorProfile captures constraints used to derive position percentages.
// InvestorProfile 记录用于推导仓位百分比的用户约束。
type InvestorProfile struct {
	ID                               string         `json:"id,omitempty"`
	RiskPreference                   RiskPreference `json:"risk_preference"`
	InvestmentHorizonMonths          int            `json:"investment_horizon_months"`
	MaxAcceptableDrawdownPct         float64        `json:"max_acceptable_drawdown_pct"`
	SingleInstrumentMaxAllocationPct float64        `json:"single_instrument_max_allocation_pct"`
	CashPreferencePct                float64        `json:"cash_preference_pct"`
	DefaultDecisionStyle             string         `json:"default_decision_style"`
}

// Validate checks profile constraints before decision generation.
// Validate 在生成决策前检查用户画像约束。
func (p InvestorProfile) Validate() error {
	switch p.RiskPreference {
	case RiskConservative, RiskBalanced, RiskAggressive:
	default:
		return fmt.Errorf("unsupported risk preference %q", p.RiskPreference)
	}
	if p.InvestmentHorizonMonths <= 0 {
		return errors.New("investment horizon must be positive")
	}
	if p.MaxAcceptableDrawdownPct <= 0 || p.MaxAcceptableDrawdownPct > 100 {
		return errors.New("max acceptable drawdown must be within 0-100")
	}
	if p.SingleInstrumentMaxAllocationPct <= 0 || p.SingleInstrumentMaxAllocationPct > 100 {
		return errors.New("single instrument max allocation must be within 0-100")
	}
	if p.CashPreferencePct < 0 || p.CashPreferencePct > 100 {
		return errors.New("cash preference must be within 0-100")
	}
	return nil
}

// PortfolioHolding records a user-owned fund, ETF, equity, or index-like exposure.
// PortfolioHolding 记录用户持有的基金、ETF、股票或指数类敞口。
type PortfolioHolding struct {
	InstrumentCode string  `json:"instrument_code"`
	InstrumentName string  `json:"instrument_name,omitempty"`
	Market         string  `json:"market"`
	Currency       string  `json:"currency"`
	HoldingAmount  float64 `json:"holding_amount"`
	CostBasis      float64 `json:"cost_basis"`
	AllocationPct  float64 `json:"allocation_pct"`
	UserThesis     string  `json:"user_thesis,omitempty"`
}

// Portfolio stores the user-entered first-version holdings.
// Portfolio 保存第一版由用户手动录入的持仓。
type Portfolio struct {
	ID       string             `json:"id,omitempty"`
	UserID   string             `json:"user_id,omitempty"`
	Holdings []PortfolioHolding `json:"holdings"`
}

// Validate checks holdings and allocation constraints.
// Validate 检查持仓和占比约束。
func (p Portfolio) Validate() error {
	if len(p.Holdings) == 0 {
		return errors.New("portfolio must include at least one holding")
	}
	total := 0.0
	for i, holding := range p.Holdings {
		if holding.InstrumentCode == "" {
			return fmt.Errorf("holding %d instrument_code is required", i)
		}
		if holding.Market == "" {
			return fmt.Errorf("holding %s market is required", holding.InstrumentCode)
		}
		if holding.Currency == "" {
			return fmt.Errorf("holding %s currency is required", holding.InstrumentCode)
		}
		if holding.HoldingAmount < 0 || holding.CostBasis < 0 {
			return fmt.Errorf("holding %s amount and cost basis must be non-negative", holding.InstrumentCode)
		}
		if holding.AllocationPct < 0 || holding.AllocationPct > 100 {
			return fmt.Errorf("holding %s allocation must be within 0-100", holding.InstrumentCode)
		}
		total += holding.AllocationPct
	}
	if total > 100.01 {
		return fmt.Errorf("portfolio allocation exceeds 100%%: %.2f", total)
	}
	return nil
}

// HoldingByCode finds the portfolio holding that matches an instrument code.
// HoldingByCode 查找指定代码对应的持仓。
func (p Portfolio) HoldingByCode(code string) (PortfolioHolding, bool) {
	for _, holding := range p.Holdings {
		if strings.EqualFold(holding.InstrumentCode, code) {
			return holding, true
		}
	}
	return PortfolioHolding{}, false
}

// InstrumentType identifies the asset class behind a fund-analysis target.
// InstrumentType 标识基金分析对象背后的资产类型。
type InstrumentType string

const (
	InstrumentFund   InstrumentType = "fund"
	InstrumentETF    InstrumentType = "etf"
	InstrumentIndex  InstrumentType = "index"
	InstrumentEquity InstrumentType = "equity"
)

// FundInstrument is the normalized instrument identity used by providers.
// FundInstrument 是 provider 使用的标准化标的身份信息。
type FundInstrument struct {
	Code     string         `json:"code"`
	Name     string         `json:"name"`
	Market   string         `json:"market"`
	Currency string         `json:"currency"`
	Type     InstrumentType `json:"type"`
}

// FundSnapshot is a normalized point-in-time data view for diagnosis.
// FundSnapshot 是用于体检的标准化时间点数据视图。
type FundSnapshot struct {
	Instrument      FundInstrument `json:"instrument"`
	NAV             float64        `json:"nav,omitempty"`
	Price           float64        `json:"price,omitempty"`
	DailyChangePct  float64        `json:"daily_change_pct"`
	OneYearReturn   float64        `json:"one_year_return_pct"`
	MaxDrawdownPct  float64        `json:"max_drawdown_pct"`
	VolatilityPct   float64        `json:"volatility_pct"`
	ExpenseRatioPct float64        `json:"expense_ratio_pct,omitempty"`
	Manager         string         `json:"manager,omitempty"`
	AssetSize       string         `json:"asset_size,omitempty"`
	TopHoldings     []string       `json:"top_holdings,omitempty"`
	Metadata        SourceMetadata `json:"metadata"`
}

// EquitySnapshot is the normalized view for US equity data used by fund analysis.
// EquitySnapshot 是基金分析使用的美股个股标准化数据视图。
type EquitySnapshot struct {
	Instrument     FundInstrument `json:"instrument"`
	Price          float64        `json:"price"`
	DailyChangePct float64        `json:"daily_change_pct"`
	OneYearReturn  float64        `json:"one_year_return_pct"`
	MaxDrawdownPct float64        `json:"max_drawdown_pct"`
	VolatilityPct  float64        `json:"volatility_pct"`
	Metadata       SourceMetadata `json:"metadata"`
}

// Validate checks equity data shape and provenance before it can support analysis.
// Validate 在美股个股数据支持分析前检查数据结构和来源信息。
func (s EquitySnapshot) Validate() error {
	if s.Instrument.Code == "" {
		return errors.New("equity instrument code is required")
	}
	if s.Instrument.Market == "" {
		return errors.New("equity instrument market is required")
	}
	if s.Price <= 0 {
		return errors.New("equity snapshot price must be positive")
	}
	return s.Metadata.Validate()
}

// IndexSnapshot is the normalized view for benchmark index data.
// IndexSnapshot 是基准指数数据的标准化视图。
type IndexSnapshot struct {
	Code           string         `json:"code"`
	Name           string         `json:"name"`
	Market         string         `json:"market"`
	Currency       string         `json:"currency"`
	Level          float64        `json:"level"`
	DailyChangePct float64        `json:"daily_change_pct"`
	OneYearReturn  float64        `json:"one_year_return_pct"`
	MaxDrawdownPct float64        `json:"max_drawdown_pct"`
	Metadata       SourceMetadata `json:"metadata"`
}

// Validate checks index data shape and provenance before it can support analysis.
// Validate 在指数数据支持分析前检查数据结构和来源信息。
func (s IndexSnapshot) Validate() error {
	if s.Code == "" {
		return errors.New("index code is required")
	}
	if s.Market == "" {
		return errors.New("index market is required")
	}
	if s.Level <= 0 {
		return errors.New("index level must be positive")
	}
	return s.Metadata.Validate()
}

// FXRate records a normalized currency conversion rate with provenance.
// FXRate 记录带来源信息的标准化汇率。
type FXRate struct {
	BaseCurrency  string         `json:"base_currency"`
	QuoteCurrency string         `json:"quote_currency"`
	Rate          float64        `json:"rate"`
	Metadata      SourceMetadata `json:"metadata"`
}

// Validate checks FX data shape and provenance before it can support attribution.
// Validate 在汇率数据支持归因前检查数据结构和来源信息。
func (r FXRate) Validate() error {
	if r.BaseCurrency == "" || r.QuoteCurrency == "" {
		return errors.New("fx currencies are required")
	}
	if r.Rate <= 0 {
		return errors.New("fx rate must be positive")
	}
	return r.Metadata.Validate()
}

// MarketCalendar records trading-day state for a market and date.
// MarketCalendar 记录某个市场和日期的交易状态。
type MarketCalendar struct {
	Market       string         `json:"market"`
	Date         string         `json:"date"`
	IsTradingDay bool           `json:"is_trading_day"`
	IsHalfDay    bool           `json:"is_half_day"`
	Session      string         `json:"session"`
	Timezone     string         `json:"timezone"`
	Delay        string         `json:"delay"`
	Metadata     SourceMetadata `json:"metadata"`
}

// Validate checks market-calendar data shape and provenance before it can align timelines.
// Validate 在交易日历用于时间线对齐前检查数据结构和来源信息。
func (c MarketCalendar) Validate() error {
	if c.Market == "" {
		return errors.New("calendar market is required")
	}
	if c.Date == "" {
		return errors.New("calendar date is required")
	}
	if c.Timezone == "" {
		return errors.New("calendar timezone is required")
	}
	return c.Metadata.Validate()
}

// Validate checks snapshot values and required provenance.
// Validate 检查快照数值和必需来源信息。
func (s FundSnapshot) Validate() error {
	if s.Instrument.Code == "" {
		return errors.New("instrument code is required")
	}
	if s.Instrument.Name == "" {
		return errors.New("instrument name is required")
	}
	if s.Price <= 0 && s.NAV <= 0 {
		return errors.New("snapshot must include positive price or nav")
	}
	return s.Metadata.Validate()
}

// Diagnosis summarizes risk and freshness signals before option generation.
// Diagnosis 在生成方案前汇总风险和数据新鲜度信号。
type Diagnosis struct {
	InstrumentCode string   `json:"instrument_code"`
	Summary        string   `json:"summary"`
	RiskFactors    []string `json:"risk_factors"`
	DataWarnings   []string `json:"data_warnings"`
	Evidence       []string `json:"evidence"`
}

// DecisionOption is one traceable action path, not an absolute command.
// DecisionOption 是一条可追溯行动路径，而不是绝对指令。
type DecisionOption struct {
	ID                  string   `json:"id"`
	Style               string   `json:"style"`
	Action              string   `json:"action"`
	AllocationChangePct float64  `json:"allocation_change_pct"`
	Conditions          []string `json:"conditions"`
	Evidence            []string `json:"evidence"`
	Risks               []string `json:"risks"`
	Invalidation        string   `json:"invalidation"`
	ReviewAfterDays     int      `json:"review_after_days"`
	PortfolioImpact     string   `json:"portfolio_impact"`
	StrategyBasis       []string `json:"strategy_basis"`
}

// DecisionMatrix contains conservative, balanced, and aggressive choices.
// DecisionMatrix 包含稳健、均衡和激进三档选择。
type DecisionMatrix struct {
	ID             string           `json:"id"`
	Instrument     FundInstrument   `json:"instrument"`
	GeneratedAt    time.Time        `json:"generated_at"`
	Options        []DecisionOption `json:"options"`
	GovernanceTags []string         `json:"governance_tags"`
	Trace          TraceSummary     `json:"trace"`
}

// Validate checks the matrix governance shape required for financial output.
// Validate 检查金融输出需要的方案治理形态。
func (m DecisionMatrix) Validate() error {
	if len(m.Options) < 2 {
		return errors.New("decision matrix must include at least two options")
	}
	styles := map[string]bool{}
	for _, option := range m.Options {
		if option.ID == "" || option.Style == "" || option.Action == "" {
			return errors.New("decision option must include id, style, and action")
		}
		if len(option.Evidence) == 0 {
			return fmt.Errorf("decision option %s missing evidence", option.ID)
		}
		if len(option.Risks) == 0 {
			return fmt.Errorf("decision option %s missing risks", option.ID)
		}
		if option.Invalidation == "" {
			return fmt.Errorf("decision option %s missing invalidation", option.ID)
		}
		if option.ReviewAfterDays <= 0 {
			return fmt.Errorf("decision option %s missing review timing", option.ID)
		}
		if len(option.StrategyBasis) == 0 {
			return fmt.Errorf("decision option %s missing strategy basis", option.ID)
		}
		styles[option.Style] = true
	}
	if !(styles["conservative"] && styles["balanced"] && styles["aggressive"]) {
		return errors.New("decision matrix must include conservative, balanced, and aggressive options")
	}
	return nil
}

// TraceSummary keeps the local MVP trace fields before Athena integration.
// TraceSummary 在 Athena 接入前保存本地 MVP trace 字段。
type TraceSummary struct {
	DataProvider      string   `json:"data_provider"`
	DataSource        string   `json:"data_source"`
	DataFetchedAt     string   `json:"data_fetched_at"`
	MarketTime        string   `json:"market_time"`
	Timezone          string   `json:"timezone"`
	LicenseTerms      string   `json:"license_terms"`
	Confidence        float64  `json:"confidence"`
	DataBoundary      string   `json:"data_boundary"`
	TemporaryData     bool     `json:"temporary_data"`
	RuleEvaluations   []string `json:"rule_evaluations"`
	GovernanceChecks  []string `json:"governance_checks"`
	AthenaRunID       string   `json:"athena_run_id,omitempty"`
	MockDataTemporary bool     `json:"mock_data_temporary"`
}

// JournalEntry preserves the chosen option and its evidence snapshot.
// JournalEntry 保存用户选择的方案及其证据快照。
type JournalEntry struct {
	ID               string         `json:"id"`
	CreatedAt        time.Time      `json:"created_at"`
	MatrixID         string         `json:"matrix_id"`
	SelectedOptionID string         `json:"selected_option_id"`
	UserNotes        string         `json:"user_notes,omitempty"`
	EvidenceSnapshot DecisionMatrix `json:"evidence_snapshot"`
}

// ReviewTask schedules the next thesis review from a journal entry.
// ReviewTask 基于决策日志安排下一次投资假设复盘。
type ReviewTask struct {
	ID          string    `json:"id"`
	JournalID   string    `json:"journal_id"`
	DueAt       time.Time `json:"due_at"`
	Question    string    `json:"question"`
	TriggerHint string    `json:"trigger_hint"`
	Status      string    `json:"status"`
}
