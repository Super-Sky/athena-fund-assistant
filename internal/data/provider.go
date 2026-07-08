package data

import (
	"context"
	"time"

	"github.com/Super-Sky/athena-fund-assistant/internal/domain"
)

// Provider defines the stable boundary for market and fund data adapters.
// Provider 定义市场和基金数据适配器的稳定边界。
type Provider interface {
	GetFundSnapshot(ctx context.Context, code string) (domain.FundSnapshot, error)
	GetEquitySnapshot(ctx context.Context, symbol string) (domain.EquitySnapshot, error)
	GetIndexSnapshot(ctx context.Context, code string) (domain.IndexSnapshot, error)
	GetFXRate(ctx context.Context, baseCurrency, quoteCurrency string) (domain.FXRate, error)
	GetMarketCalendar(ctx context.Context, market string, date time.Time) (domain.MarketCalendar, error)
}
