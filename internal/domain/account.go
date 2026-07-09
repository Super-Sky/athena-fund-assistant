package domain

import (
	"errors"
	"fmt"
	"time"
)

// UserAccount identifies one local fund-assistant user without brokerage custody.
// UserAccount 标识一个本地基金助手用户，不承载券商资金托管。
type UserAccount struct {
	UserID       string    `json:"user_id"`
	DisplayName  string    `json:"display_name"`
	BaseCurrency string    `json:"base_currency"`
	AuthMode     string    `json:"auth_mode"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Validate checks the minimum account identity required by dashboard APIs.
// Validate 检查账户看板 API 所需的最小账户身份。
func (a UserAccount) Validate() error {
	if a.UserID == "" {
		return errors.New("user_id is required")
	}
	if a.BaseCurrency == "" {
		return errors.New("base_currency is required")
	}
	if a.AuthMode == "" {
		return errors.New("auth_mode is required")
	}
	return nil
}

// AccountHoldingSnapshot captures one manual or synced holding in account context.
// AccountHoldingSnapshot 记录账户上下文中的一条手动或同步持仓。
type AccountHoldingSnapshot struct {
	ID                string         `json:"id"`
	UserID            string         `json:"user_id"`
	InstrumentCode    string         `json:"instrument_code"`
	InstrumentName    string         `json:"instrument_name"`
	Market            string         `json:"market"`
	Currency          string         `json:"currency"`
	Units             float64        `json:"units"`
	CostBasis         float64        `json:"cost_basis"`
	CurrentPrice      float64        `json:"current_price"`
	FXToBase          float64        `json:"fx_to_base"`
	MarketValue       float64        `json:"market_value"`
	CostValue         float64        `json:"cost_value"`
	BaseMarketValue   float64        `json:"base_market_value"`
	BaseCostValue     float64        `json:"base_cost_value"`
	UnrealizedPnL     float64        `json:"unrealized_pnl"`
	UnrealizedPnLPct  float64        `json:"unrealized_pnl_pct"`
	AllocationPct     float64        `json:"allocation_pct"`
	UserThesis        string         `json:"user_thesis,omitempty"`
	DataAuthorization string         `json:"data_authorization"`
	Metadata          SourceMetadata `json:"metadata"`
}

// Validate checks holding math and provenance before it enters account totals.
// Validate 在持仓进入账户汇总前检查计算字段和来源信息。
func (h AccountHoldingSnapshot) Validate() error {
	if h.ID == "" || h.UserID == "" {
		return errors.New("holding id and user_id are required")
	}
	if h.InstrumentCode == "" || h.Market == "" || h.Currency == "" {
		return errors.New("holding instrument_code, market, and currency are required")
	}
	if h.Units < 0 || h.CostBasis < 0 || h.CurrentPrice < 0 {
		return fmt.Errorf("holding %s units, cost_basis, and current_price must be non-negative", h.InstrumentCode)
	}
	if h.FXToBase <= 0 {
		return fmt.Errorf("holding %s fx_to_base must be positive", h.InstrumentCode)
	}
	if h.AllocationPct < 0 || h.AllocationPct > 100 {
		return fmt.Errorf("holding %s allocation_pct must be within 0-100", h.InstrumentCode)
	}
	if h.DataAuthorization == "" {
		return fmt.Errorf("holding %s data_authorization is required", h.InstrumentCode)
	}
	return h.Metadata.Validate()
}

// AccountOperationRecord preserves a user-visible operation without placing trades.
// AccountOperationRecord 保存用户可见操作记录，但不执行交易。
type AccountOperationRecord struct {
	ID             string         `json:"id"`
	UserID         string         `json:"user_id"`
	OccurredAt     time.Time      `json:"occurred_at"`
	InstrumentCode string         `json:"instrument_code"`
	Type           string         `json:"type"`
	Units          float64        `json:"units"`
	Price          float64        `json:"price"`
	Amount         float64        `json:"amount"`
	BaseAmount     float64        `json:"base_amount"`
	RealizedPnL    float64        `json:"realized_pnl"`
	Currency       string         `json:"currency"`
	Notes          string         `json:"notes,omitempty"`
	Metadata       SourceMetadata `json:"metadata"`
}

// Validate checks operation provenance and prevents broker-action semantics.
// Validate 检查操作来源，并避免表达成券商下单语义。
func (o AccountOperationRecord) Validate() error {
	if o.ID == "" || o.UserID == "" {
		return errors.New("operation id and user_id are required")
	}
	if o.Type == "" {
		return errors.New("operation type is required")
	}
	if o.OccurredAt.IsZero() {
		return errors.New("operation occurred_at is required")
	}
	if o.Currency == "" {
		return errors.New("operation currency is required")
	}
	return o.Metadata.Validate()
}

// AccountPerformancePoint is one chartable account-level performance point.
// AccountPerformancePoint 是一条可用于图表展示的账户级收益点。
type AccountPerformancePoint struct {
	Date             string         `json:"date"`
	BaseCurrency     string         `json:"base_currency"`
	TotalMarketValue float64        `json:"total_market_value"`
	TotalCostValue   float64        `json:"total_cost_value"`
	TotalPnL         float64        `json:"total_pnl"`
	TotalPnLPct      float64        `json:"total_pnl_pct"`
	OperationPnL     float64        `json:"operation_pnl"`
	Metadata         SourceMetadata `json:"metadata"`
}

// Validate checks performance point provenance and currency boundaries.
// Validate 检查收益点来源和币种边界。
func (p AccountPerformancePoint) Validate() error {
	if p.Date == "" {
		return errors.New("performance date is required")
	}
	if p.BaseCurrency == "" {
		return errors.New("performance base_currency is required")
	}
	return p.Metadata.Validate()
}

// AccountTraceSummary records dashboard computation provenance and temporary limits.
// AccountTraceSummary 记录账户看板计算来源和临时限制。
type AccountTraceSummary struct {
	Provider              string   `json:"provider"`
	Source                string   `json:"source"`
	FetchedAt             string   `json:"fetched_at"`
	MarketTime            string   `json:"market_time"`
	Timezone              string   `json:"timezone"`
	LicenseTerms          string   `json:"license_terms"`
	Confidence            float64  `json:"confidence"`
	SchemaVersion         string   `json:"schema_version"`
	MockDataTemporary     bool     `json:"mock_data_temporary"`
	ReadOnlySyncAvailable bool     `json:"read_only_sync_available"`
	Warnings              []string `json:"warnings"`
}

// AccountOverview is the first dashboard read model for the account homepage.
// AccountOverview 是账户首页使用的第一版看板读模型。
type AccountOverview struct {
	Account            UserAccount               `json:"account"`
	Holdings           []AccountHoldingSnapshot  `json:"holdings"`
	TotalMarketValue   float64                   `json:"total_market_value"`
	TotalCostValue     float64                   `json:"total_cost_value"`
	TotalPnL           float64                   `json:"total_pnl"`
	TotalPnLPct        float64                   `json:"total_pnl_pct"`
	RecentOperationPnL float64                   `json:"recent_operation_pnl"`
	BaseCurrency       string                    `json:"base_currency"`
	PerformanceTrend   []AccountPerformancePoint `json:"performance_trend"`
	RecentOperations   []AccountOperationRecord  `json:"recent_operations"`
	Trace              AccountTraceSummary       `json:"trace"`
}

// Validate checks the dashboard read model before returning it to UI or Athena.
// Validate 在看板读模型返回 UI 或 Athena 前检查结构。
func (o AccountOverview) Validate() error {
	if err := o.Account.Validate(); err != nil {
		return err
	}
	if len(o.Holdings) == 0 {
		return errors.New("account overview must include holdings")
	}
	for _, holding := range o.Holdings {
		if err := holding.Validate(); err != nil {
			return err
		}
	}
	for _, point := range o.PerformanceTrend {
		if err := point.Validate(); err != nil {
			return err
		}
	}
	for _, operation := range o.RecentOperations {
		if err := operation.Validate(); err != nil {
			return err
		}
	}
	if o.BaseCurrency == "" {
		return errors.New("overview base_currency is required")
	}
	return nil
}
