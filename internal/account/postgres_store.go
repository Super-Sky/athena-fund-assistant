package account

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Super-Sky/athena-fund-assistant/internal/domain"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// PostgresStore persists account dashboard data in PostgreSQL.
// PostgresStore 将账户看板数据持久化到 PostgreSQL。
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore opens PostgreSQL, ensures schema, and seeds local demo data when empty.
// NewPostgresStore 打开 PostgreSQL、确保 schema 存在，并在为空时写入本地演示数据。
func NewPostgresStore(ctx context.Context, dsn string) (*PostgresStore, error) {
	if dsn == "" {
		return nil, errors.New("database dsn is required")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	store := &PostgresStore{db: db}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := store.ensureSchema(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := store.seedDemoIfEmpty(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

// Close releases the database handle.
// Close 释放数据库连接句柄。
func (s *PostgresStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// Overview returns one persisted account dashboard read model by user ID.
// Overview 按用户 ID 返回一份持久化账户看板读模型。
func (s *PostgresStore) Overview(ctx context.Context, userID string) (domain.AccountOverview, error) {
	if userID == "" {
		userID = "demo-user"
	}
	account, err := s.account(ctx, userID)
	if err != nil {
		return domain.AccountOverview{}, err
	}
	holdings, err := s.holdings(ctx, userID)
	if err != nil {
		return domain.AccountOverview{}, err
	}
	operations, err := s.operations(ctx, userID)
	if err != nil {
		return domain.AccountOverview{}, err
	}
	trend, err := s.trend(ctx, userID)
	if err != nil {
		return domain.AccountOverview{}, err
	}
	overview := buildOverview(account, holdings, operations, trend)
	if err := overview.Validate(); err != nil {
		return domain.AccountOverview{}, err
	}
	return overview, nil
}

// ReplaceHoldings stores manually entered holdings and recalculates persisted trend points.
// ReplaceHoldings 保存手动录入持仓，并重新计算持久化趋势点。
func (s *PostgresStore) ReplaceHoldings(ctx context.Context, userID string, holdings []domain.AccountHoldingSnapshot) (domain.AccountOverview, error) {
	if userID == "" {
		return domain.AccountOverview{}, errors.New("user_id is required")
	}
	normalized, err := normalizeHoldings(userID, holdings)
	if err != nil {
		return domain.AccountOverview{}, err
	}
	account, err := s.account(ctx, userID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return domain.AccountOverview{}, err
		}
		now := time.Now().UTC()
		account = domain.UserAccount{
			UserID:       userID,
			DisplayName:  "Local Investor",
			BaseCurrency: "CNY",
			AuthMode:     "local_demo",
			CreatedAt:    now,
			UpdatedAt:    now,
		}
	}
	trend := buildTrend(account.BaseCurrency, normalized, nil)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.AccountOverview{}, err
	}
	if err := upsertAccount(ctx, tx, account); err != nil {
		_ = tx.Rollback()
		return domain.AccountOverview{}, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM account_holdings WHERE user_id = $1`, userID); err != nil {
		_ = tx.Rollback()
		return domain.AccountOverview{}, err
	}
	for _, holding := range normalized {
		if err := insertHolding(ctx, tx, holding); err != nil {
			_ = tx.Rollback()
			return domain.AccountOverview{}, err
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM account_performance_points WHERE user_id = $1`, userID); err != nil {
		_ = tx.Rollback()
		return domain.AccountOverview{}, err
	}
	for _, point := range trend {
		if err := insertTrendPoint(ctx, tx, userID, point); err != nil {
			_ = tx.Rollback()
			return domain.AccountOverview{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return domain.AccountOverview{}, err
	}
	return s.Overview(ctx, userID)
}

func (s *PostgresStore) ensureSchema(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS fund_accounts (
  user_id TEXT PRIMARY KEY,
  display_name TEXT NOT NULL,
  base_currency TEXT NOT NULL,
  auth_mode TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS account_holdings (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES fund_accounts(user_id) ON DELETE CASCADE,
  instrument_code TEXT NOT NULL,
  instrument_name TEXT NOT NULL,
  market TEXT NOT NULL,
  currency TEXT NOT NULL,
  units DOUBLE PRECISION NOT NULL,
  cost_basis DOUBLE PRECISION NOT NULL,
  current_price DOUBLE PRECISION NOT NULL,
  fx_to_base DOUBLE PRECISION NOT NULL,
  market_value DOUBLE PRECISION NOT NULL,
  cost_value DOUBLE PRECISION NOT NULL,
  base_market_value DOUBLE PRECISION NOT NULL,
  base_cost_value DOUBLE PRECISION NOT NULL,
  unrealized_pnl DOUBLE PRECISION NOT NULL,
  unrealized_pnl_pct DOUBLE PRECISION NOT NULL,
  allocation_pct DOUBLE PRECISION NOT NULL,
  user_thesis TEXT NOT NULL DEFAULT '',
  data_authorization TEXT NOT NULL,
  metadata JSONB NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS account_operations (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES fund_accounts(user_id) ON DELETE CASCADE,
  occurred_at TIMESTAMPTZ NOT NULL,
  instrument_code TEXT NOT NULL,
  type TEXT NOT NULL,
  units DOUBLE PRECISION NOT NULL,
  price DOUBLE PRECISION NOT NULL,
  amount DOUBLE PRECISION NOT NULL,
  base_amount DOUBLE PRECISION NOT NULL,
  realized_pnl DOUBLE PRECISION NOT NULL,
  currency TEXT NOT NULL,
  notes TEXT NOT NULL DEFAULT '',
  metadata JSONB NOT NULL
);

CREATE TABLE IF NOT EXISTS account_performance_points (
  user_id TEXT NOT NULL REFERENCES fund_accounts(user_id) ON DELETE CASCADE,
  date TEXT NOT NULL,
  base_currency TEXT NOT NULL,
  total_market_value DOUBLE PRECISION NOT NULL,
  total_cost_value DOUBLE PRECISION NOT NULL,
  total_pnl DOUBLE PRECISION NOT NULL,
  total_pnl_pct DOUBLE PRECISION NOT NULL,
  operation_pnl DOUBLE PRECISION NOT NULL,
  metadata JSONB NOT NULL,
  PRIMARY KEY (user_id, date)
);
`)
	return err
}

func (s *PostgresStore) seedDemoIfEmpty(ctx context.Context) error {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM fund_accounts`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	account, holdings, operations, trend := demoAccountData(time.Now().UTC())
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := upsertAccount(ctx, tx, account); err != nil {
		_ = tx.Rollback()
		return err
	}
	for _, holding := range holdings {
		if err := insertHolding(ctx, tx, holding); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	for _, operation := range operations {
		if err := insertOperation(ctx, tx, operation); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	for _, point := range trend {
		if err := insertTrendPoint(ctx, tx, account.UserID, point); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (s *PostgresStore) account(ctx context.Context, userID string) (domain.UserAccount, error) {
	var account domain.UserAccount
	err := s.db.QueryRowContext(ctx, `
SELECT user_id, display_name, base_currency, auth_mode, created_at, updated_at
FROM fund_accounts
WHERE user_id = $1`, userID).Scan(
		&account.UserID,
		&account.DisplayName,
		&account.BaseCurrency,
		&account.AuthMode,
		&account.CreatedAt,
		&account.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.UserAccount{}, fmt.Errorf("account %q not found: %w", userID, sql.ErrNoRows)
		}
		return domain.UserAccount{}, err
	}
	return account, nil
}

func (s *PostgresStore) holdings(ctx context.Context, userID string) ([]domain.AccountHoldingSnapshot, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, user_id, instrument_code, instrument_name, market, currency, units, cost_basis, current_price, fx_to_base,
       market_value, cost_value, base_market_value, base_cost_value, unrealized_pnl, unrealized_pnl_pct,
       allocation_pct, user_thesis, data_authorization, metadata
FROM account_holdings
WHERE user_id = $1
ORDER BY base_market_value DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.AccountHoldingSnapshot
	for rows.Next() {
		var item domain.AccountHoldingSnapshot
		var metadata []byte
		if err := rows.Scan(
			&item.ID,
			&item.UserID,
			&item.InstrumentCode,
			&item.InstrumentName,
			&item.Market,
			&item.Currency,
			&item.Units,
			&item.CostBasis,
			&item.CurrentPrice,
			&item.FXToBase,
			&item.MarketValue,
			&item.CostValue,
			&item.BaseMarketValue,
			&item.BaseCostValue,
			&item.UnrealizedPnL,
			&item.UnrealizedPnLPct,
			&item.AllocationPct,
			&item.UserThesis,
			&item.DataAuthorization,
			&metadata,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(metadata, &item.Metadata); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) operations(ctx context.Context, userID string) ([]domain.AccountOperationRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, user_id, occurred_at, instrument_code, type, units, price, amount, base_amount, realized_pnl, currency, notes, metadata
FROM account_operations
WHERE user_id = $1
ORDER BY occurred_at DESC
LIMIT 20`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.AccountOperationRecord
	for rows.Next() {
		var item domain.AccountOperationRecord
		var metadata []byte
		if err := rows.Scan(
			&item.ID,
			&item.UserID,
			&item.OccurredAt,
			&item.InstrumentCode,
			&item.Type,
			&item.Units,
			&item.Price,
			&item.Amount,
			&item.BaseAmount,
			&item.RealizedPnL,
			&item.Currency,
			&item.Notes,
			&metadata,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(metadata, &item.Metadata); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) trend(ctx context.Context, userID string) ([]domain.AccountPerformancePoint, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT date, base_currency, total_market_value, total_cost_value, total_pnl, total_pnl_pct, operation_pnl, metadata
FROM account_performance_points
WHERE user_id = $1
ORDER BY date ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.AccountPerformancePoint
	for rows.Next() {
		var item domain.AccountPerformancePoint
		var metadata []byte
		if err := rows.Scan(
			&item.Date,
			&item.BaseCurrency,
			&item.TotalMarketValue,
			&item.TotalCostValue,
			&item.TotalPnL,
			&item.TotalPnLPct,
			&item.OperationPnL,
			&metadata,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(metadata, &item.Metadata); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func upsertAccount(ctx context.Context, tx *sql.Tx, account domain.UserAccount) error {
	_, err := tx.ExecContext(ctx, `
INSERT INTO fund_accounts (user_id, display_name, base_currency, auth_mode, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (user_id) DO UPDATE SET
  display_name = EXCLUDED.display_name,
  base_currency = EXCLUDED.base_currency,
  auth_mode = EXCLUDED.auth_mode,
  updated_at = EXCLUDED.updated_at`,
		account.UserID,
		account.DisplayName,
		account.BaseCurrency,
		account.AuthMode,
		account.CreatedAt,
		account.UpdatedAt,
	)
	return err
}

func insertHolding(ctx context.Context, tx *sql.Tx, holding domain.AccountHoldingSnapshot) error {
	metadata, err := json.Marshal(holding.Metadata)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO account_holdings (
  id, user_id, instrument_code, instrument_name, market, currency, units, cost_basis, current_price, fx_to_base,
  market_value, cost_value, base_market_value, base_cost_value, unrealized_pnl, unrealized_pnl_pct,
  allocation_pct, user_thesis, data_authorization, metadata
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
  $11, $12, $13, $14, $15, $16,
  $17, $18, $19, $20
)`,
		holding.ID,
		holding.UserID,
		holding.InstrumentCode,
		holding.InstrumentName,
		holding.Market,
		holding.Currency,
		holding.Units,
		holding.CostBasis,
		holding.CurrentPrice,
		holding.FXToBase,
		holding.MarketValue,
		holding.CostValue,
		holding.BaseMarketValue,
		holding.BaseCostValue,
		holding.UnrealizedPnL,
		holding.UnrealizedPnLPct,
		holding.AllocationPct,
		holding.UserThesis,
		holding.DataAuthorization,
		metadata,
	)
	return err
}

func insertOperation(ctx context.Context, tx *sql.Tx, operation domain.AccountOperationRecord) error {
	metadata, err := json.Marshal(operation.Metadata)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO account_operations (
  id, user_id, occurred_at, instrument_code, type, units, price, amount, base_amount, realized_pnl, currency, notes, metadata
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
)`,
		operation.ID,
		operation.UserID,
		operation.OccurredAt,
		operation.InstrumentCode,
		operation.Type,
		operation.Units,
		operation.Price,
		operation.Amount,
		operation.BaseAmount,
		operation.RealizedPnL,
		operation.Currency,
		operation.Notes,
		metadata,
	)
	return err
}

func insertTrendPoint(ctx context.Context, tx *sql.Tx, userID string, point domain.AccountPerformancePoint) error {
	metadata, err := json.Marshal(point.Metadata)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO account_performance_points (
  user_id, date, base_currency, total_market_value, total_cost_value, total_pnl, total_pnl_pct, operation_pnl, metadata
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9
)`,
		userID,
		point.Date,
		point.BaseCurrency,
		point.TotalMarketValue,
		point.TotalCostValue,
		point.TotalPnL,
		point.TotalPnLPct,
		point.OperationPnL,
		metadata,
	)
	return err
}
