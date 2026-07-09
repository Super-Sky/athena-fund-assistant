package account

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Super-Sky/athena-fund-assistant/internal/domain"
)

// Store defines the account dashboard persistence boundary.
// Store 定义账户看板的持久化边界。
type Store interface {
	Overview(context.Context, string) (domain.AccountOverview, error)
	ReplaceHoldings(context.Context, string, []domain.AccountHoldingSnapshot) (domain.AccountOverview, error)
}

// MemoryStore keeps local-first account snapshots before PostgreSQL is wired.
// MemoryStore 在接入 PostgreSQL 前保存本地优先的账户快照。
type MemoryStore struct {
	mu         sync.Mutex
	accounts   map[string]domain.UserAccount
	holdings   map[string][]domain.AccountHoldingSnapshot
	operations map[string][]domain.AccountOperationRecord
	trends     map[string][]domain.AccountPerformancePoint
}

// NewMemoryStore creates an account store with one demo user for local MVP runs.
// NewMemoryStore 创建带本地 MVP 演示用户的账户存储。
func NewMemoryStore() *MemoryStore {
	store := &MemoryStore{
		accounts:   map[string]domain.UserAccount{},
		holdings:   map[string][]domain.AccountHoldingSnapshot{},
		operations: map[string][]domain.AccountOperationRecord{},
		trends:     map[string][]domain.AccountPerformancePoint{},
	}
	store.seedDemo()
	return store
}

// Overview returns one account dashboard read model by user ID.
// Overview 按用户 ID 返回一份账户看板读模型。
func (s *MemoryStore) Overview(_ context.Context, userID string) (domain.AccountOverview, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.overviewLocked(userID)
}

// ReplaceHoldings stores manually entered holdings and recalculates the account read model.
// ReplaceHoldings 保存手动录入持仓，并重新计算账户读模型。
func (s *MemoryStore) ReplaceHoldings(_ context.Context, userID string, holdings []domain.AccountHoldingSnapshot) (domain.AccountOverview, error) {
	if userID == "" {
		return domain.AccountOverview{}, errors.New("user_id is required")
	}
	if len(holdings) == 0 {
		return domain.AccountOverview{}, errors.New("holdings are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	account, ok := s.accounts[userID]
	if !ok {
		now := time.Now().UTC()
		account = domain.UserAccount{
			UserID:       userID,
			DisplayName:  "Local Investor",
			BaseCurrency: "CNY",
			AuthMode:     "local_demo",
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		s.accounts[userID] = account
	}
	normalized, err := normalizeHoldings(userID, holdings)
	if err != nil {
		return domain.AccountOverview{}, err
	}
	s.holdings[userID] = normalized
	s.trends[userID] = buildTrend(account.BaseCurrency, normalized, nil)
	return s.overviewLocked(userID)
}

func (s *MemoryStore) overviewLocked(userID string) (domain.AccountOverview, error) {
	if userID == "" {
		userID = "demo-user"
	}
	account, ok := s.accounts[userID]
	if !ok {
		return domain.AccountOverview{}, fmt.Errorf("account %q not found", userID)
	}
	holdings := append([]domain.AccountHoldingSnapshot(nil), s.holdings[userID]...)
	operations := append([]domain.AccountOperationRecord(nil), s.operations[userID]...)
	trend := append([]domain.AccountPerformancePoint(nil), s.trends[userID]...)
	overview := buildOverview(account, holdings, operations, trend)
	if err := overview.Validate(); err != nil {
		return domain.AccountOverview{}, err
	}
	return overview, nil
}

func (s *MemoryStore) seedDemo() {
	account, holdings, operations, trend := demoAccountData(time.Now().UTC())
	s.accounts[account.UserID] = account
	s.holdings[account.UserID] = holdings
	s.operations[account.UserID] = operations
	s.trends[account.UserID] = trend
}

func demoAccountData(now time.Time) (domain.UserAccount, []domain.AccountHoldingSnapshot, []domain.AccountOperationRecord, []domain.AccountPerformancePoint) {
	metadata := demoMetadata(now)
	account := domain.UserAccount{
		UserID:       "demo-user",
		DisplayName:  "Demo Investor",
		BaseCurrency: "CNY",
		AuthMode:     "local_demo",
		CreatedAt:    now.AddDate(0, -1, 0),
		UpdatedAt:    now,
	}
	holdings := []domain.AccountHoldingSnapshot{
		{
			ID:                "holding_demo_510300",
			UserID:            account.UserID,
			InstrumentCode:    "510300",
			InstrumentName:    "Sample CSI 300 ETF",
			Market:            "CN",
			Currency:          "CNY",
			Units:             12000,
			CostBasis:         4.2,
			CurrentPrice:      4.38,
			FXToBase:          1,
			UserThesis:        "Broad China equity beta with controlled concentration.",
			DataAuthorization: "manual_entry",
			Metadata:          metadata,
		},
		{
			ID:                "holding_demo_qqq",
			UserID:            account.UserID,
			InstrumentCode:    "QQQ",
			InstrumentName:    "Sample Nasdaq 100 ETF",
			Market:            "US",
			Currency:          "USD",
			Units:             20,
			CostBasis:         420,
			CurrentPrice:      455,
			FXToBase:          7.18,
			UserThesis:        "US growth exposure with explicit FX and US market-time handling.",
			DataAuthorization: "manual_entry",
			Metadata:          metadata,
		},
	}
	normalized, _ := normalizeHoldings(account.UserID, holdings)
	operations := []domain.AccountOperationRecord{
		{
			ID:             "operation_demo_reduce_510300",
			UserID:         account.UserID,
			OccurredAt:     now.AddDate(0, 0, -5),
			InstrumentCode: "510300",
			Type:           "manual_journal_reduce_plan",
			Units:          600,
			Price:          4.35,
			Amount:         2610,
			BaseAmount:     2610,
			RealizedPnL:    90,
			Currency:       "CNY",
			Notes:          "Recorded as a decision journal operation, not an executed broker order.",
			Metadata:       metadata,
		},
	}
	return account, normalized, operations, buildTrend(account.BaseCurrency, normalized, operations)
}

func normalizeHoldings(userID string, holdings []domain.AccountHoldingSnapshot) ([]domain.AccountHoldingSnapshot, error) {
	normalized := make([]domain.AccountHoldingSnapshot, len(holdings))
	total := 0.0
	for i, holding := range holdings {
		holding.UserID = userID
		if holding.ID == "" {
			holding.ID = fmt.Sprintf("holding_%s_%d", userID, i+1)
		}
		if holding.FXToBase == 0 {
			holding.FXToBase = 1
		}
		if holding.DataAuthorization == "" {
			holding.DataAuthorization = "manual_entry"
		}
		holding.MarketValue = roundMoney(holding.Units * holding.CurrentPrice)
		holding.CostValue = roundMoney(holding.Units * holding.CostBasis)
		holding.BaseMarketValue = roundMoney(holding.MarketValue * holding.FXToBase)
		holding.BaseCostValue = roundMoney(holding.CostValue * holding.FXToBase)
		holding.UnrealizedPnL = roundMoney(holding.BaseMarketValue - holding.BaseCostValue)
		if holding.BaseCostValue > 0 {
			holding.UnrealizedPnLPct = roundPct(holding.UnrealizedPnL / holding.BaseCostValue * 100)
		}
		total += holding.BaseMarketValue
		normalized[i] = holding
	}
	for i := range normalized {
		if total > 0 {
			normalized[i].AllocationPct = roundPct(normalized[i].BaseMarketValue / total * 100)
		}
		if err := normalized[i].Validate(); err != nil {
			return nil, err
		}
	}
	return normalized, nil
}

func buildOverview(account domain.UserAccount, holdings []domain.AccountHoldingSnapshot, operations []domain.AccountOperationRecord, trend []domain.AccountPerformancePoint) domain.AccountOverview {
	totalMarket := 0.0
	totalCost := 0.0
	recentOperationPnL := 0.0
	var metadata domain.SourceMetadata
	if len(holdings) > 0 {
		metadata = holdings[0].Metadata
	}
	for _, holding := range holdings {
		totalMarket += holding.BaseMarketValue
		totalCost += holding.BaseCostValue
	}
	for _, operation := range operations {
		recentOperationPnL += operation.RealizedPnL
	}
	totalPnL := roundMoney(totalMarket - totalCost)
	totalPnLPct := 0.0
	if totalCost > 0 {
		totalPnLPct = roundPct(totalPnL / totalCost * 100)
	}
	return domain.AccountOverview{
		Account:            account,
		Holdings:           holdings,
		TotalMarketValue:   roundMoney(totalMarket),
		TotalCostValue:     roundMoney(totalCost),
		TotalPnL:           totalPnL,
		TotalPnLPct:        totalPnLPct,
		RecentOperationPnL: roundMoney(recentOperationPnL),
		BaseCurrency:       account.BaseCurrency,
		PerformanceTrend:   trend,
		RecentOperations:   operations,
		Trace: domain.AccountTraceSummary{
			Provider:              metadata.Provider,
			Source:                metadata.Source,
			FetchedAt:             metadata.FetchedAt.Format(time.RFC3339),
			MarketTime:            metadata.MarketTime.Format(time.RFC3339),
			Timezone:              metadata.Timezone,
			LicenseTerms:          metadata.LicenseTerms,
			Confidence:            metadata.Confidence,
			SchemaVersion:         metadata.SchemaVersion,
			MockDataTemporary:     true,
			ReadOnlySyncAvailable: false,
			Warnings: []string{
				"Local MVP account data is manually entered and mock-backed.",
				"Brokerage synchronization is reserved as a read-only future direction.",
			},
		},
	}
}

func buildTrend(baseCurrency string, holdings []domain.AccountHoldingSnapshot, operations []domain.AccountOperationRecord) []domain.AccountPerformancePoint {
	now := time.Now().UTC()
	totalMarket := 0.0
	totalCost := 0.0
	operationPnL := 0.0
	var metadata domain.SourceMetadata
	if len(holdings) > 0 {
		metadata = holdings[0].Metadata
	}
	for _, holding := range holdings {
		totalMarket += holding.BaseMarketValue
		totalCost += holding.BaseCostValue
	}
	for _, operation := range operations {
		operationPnL += operation.RealizedPnL
	}
	points := make([]domain.AccountPerformancePoint, 0, 4)
	for i, scale := range []float64{0.965, 0.982, 0.994, 1} {
		market := roundMoney(totalMarket * scale)
		pnl := roundMoney(market - totalCost)
		pct := 0.0
		if totalCost > 0 {
			pct = roundPct(pnl / totalCost * 100)
		}
		points = append(points, domain.AccountPerformancePoint{
			Date:             now.AddDate(0, 0, -21+i*7).Format("2006-01-02"),
			BaseCurrency:     baseCurrency,
			TotalMarketValue: market,
			TotalCostValue:   roundMoney(totalCost),
			TotalPnL:         pnl,
			TotalPnLPct:      pct,
			OperationPnL:     roundMoney(operationPnL),
			Metadata:         metadata,
		})
	}
	return points
}

func demoMetadata(now time.Time) domain.SourceMetadata {
	return domain.SourceMetadata{
		Source:        "local_demo_account_snapshot",
		Provider:      "mock_account_provider",
		FetchedAt:     now,
		MarketTime:    now.Add(-15 * time.Minute),
		Timezone:      "Asia/Shanghai",
		Delay:         "15m",
		LicenseTerms:  "mock data for local MVP only",
		Confidence:    0.5,
		SchemaVersion: "account_snapshot.v1",
	}
}

func roundMoney(value float64) float64 {
	return float64(int(value*100+0.5)) / 100
}

func roundPct(value float64) float64 {
	return float64(int(value*100+0.5)) / 100
}
