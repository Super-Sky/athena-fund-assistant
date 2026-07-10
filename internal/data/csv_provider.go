package data

import (
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Super-Sky/athena-fund-assistant/internal/domain"
)

const csvSchemaVersion = "market_data_csv.v1"

// CSVProvider loads user-supplied normalized market data from CSV files.
// CSVProvider 从用户提供的标准化 CSV 文件加载市场数据。
type CSVProvider struct {
	funds     map[string]domain.FundSnapshot
	equities  map[string]domain.EquitySnapshot
	indexes   map[string]domain.IndexSnapshot
	fxRates   map[string]domain.FXRate
	calendars map[string]domain.MarketCalendar
}

// NewCSVProvider creates a provider from one CSV file or every CSV file in a directory.
// NewCSVProvider 从单个 CSV 文件或目录下的所有 CSV 文件创建 provider。
func NewCSVProvider(path string) (*CSVProvider, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("csv provider path is required")
	}
	provider := &CSVProvider{
		funds:     map[string]domain.FundSnapshot{},
		equities:  map[string]domain.EquitySnapshot{},
		indexes:   map[string]domain.IndexSnapshot{},
		fxRates:   map[string]domain.FXRate{},
		calendars: map[string]domain.MarketCalendar{},
	}
	paths, err := csvPaths(path)
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("csv provider found no csv files under %q", path)
	}
	for _, filePath := range paths {
		if err := provider.loadFile(filePath); err != nil {
			return nil, err
		}
	}
	if provider.empty() {
		return nil, fmt.Errorf("csv provider loaded no supported rows from %q", path)
	}
	return provider, nil
}

// GetFundSnapshot returns a normalized fund or ETF row from user-supplied CSV data.
// GetFundSnapshot 从用户提供的 CSV 数据返回标准化基金或 ETF 行。
func (p *CSVProvider) GetFundSnapshot(_ context.Context, code string) (domain.FundSnapshot, error) {
	if snapshot, ok := p.funds[normalizeKey(code)]; ok {
		return snapshot, nil
	}
	return domain.FundSnapshot{}, fmt.Errorf("csv provider has no fund %q", code)
}

// GetEquitySnapshot returns a normalized equity row from user-supplied CSV data.
// GetEquitySnapshot 从用户提供的 CSV 数据返回标准化股票行。
func (p *CSVProvider) GetEquitySnapshot(_ context.Context, symbol string) (domain.EquitySnapshot, error) {
	if snapshot, ok := p.equities[normalizeKey(symbol)]; ok {
		return snapshot, nil
	}
	return domain.EquitySnapshot{}, fmt.Errorf("csv provider has no equity %q", symbol)
}

// GetIndexSnapshot returns a normalized index row from user-supplied CSV data.
// GetIndexSnapshot 从用户提供的 CSV 数据返回标准化指数行。
func (p *CSVProvider) GetIndexSnapshot(_ context.Context, code string) (domain.IndexSnapshot, error) {
	if snapshot, ok := p.indexes[normalizeKey(code)]; ok {
		return snapshot, nil
	}
	return domain.IndexSnapshot{}, fmt.Errorf("csv provider has no index %q", code)
}

// GetFXRate returns a normalized FX row from user-supplied CSV data.
// GetFXRate 从用户提供的 CSV 数据返回标准化汇率行。
func (p *CSVProvider) GetFXRate(_ context.Context, baseCurrency, quoteCurrency string) (domain.FXRate, error) {
	if rate, ok := p.fxRates[fxKey(baseCurrency, quoteCurrency)]; ok {
		return rate, nil
	}
	return domain.FXRate{}, fmt.Errorf("csv provider has no fx rate %s/%s", baseCurrency, quoteCurrency)
}

// GetMarketCalendar returns a normalized market-calendar row from user-supplied CSV data.
// GetMarketCalendar 从用户提供的 CSV 数据返回标准化交易日历行。
func (p *CSVProvider) GetMarketCalendar(_ context.Context, market string, date time.Time) (domain.MarketCalendar, error) {
	key := calendarKey(market, date.Format("2006-01-02"))
	if calendar, ok := p.calendars[key]; ok {
		return calendar, nil
	}
	return domain.MarketCalendar{}, fmt.Errorf("csv provider has no market calendar %s:%s", market, date.Format("2006-01-02"))
}

func (p *CSVProvider) empty() bool {
	return len(p.funds) == 0 && len(p.equities) == 0 && len(p.indexes) == 0 && len(p.fxRates) == 0 && len(p.calendars) == 0
}

func (p *CSVProvider) loadFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("read csv header %s: %w", path, err)
	}
	columns := normalizeHeader(header)
	line := 1
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		line++
		if err != nil {
			return fmt.Errorf("read csv row %s:%d: %w", path, line, err)
		}
		row := rowValues(columns, record)
		if emptyRow(row) {
			continue
		}
		if err := p.addRow(row, record); err != nil {
			return fmt.Errorf("load csv row %s:%d: %w", path, line, err)
		}
	}
	return nil
}

func (p *CSVProvider) addRow(row map[string]string, record []string) error {
	kind := normalizeKey(row["kind"])
	switch kind {
	case "FUND", "ETF", "LOF":
		snapshot, err := parseCSVFund(row, record)
		if err != nil {
			return err
		}
		if err := snapshot.Validate(); err != nil {
			return err
		}
		p.funds[normalizeKey(snapshot.Instrument.Code)] = snapshot
	case "EQUITY", "STOCK":
		snapshot, err := parseCSVEquity(row, record)
		if err != nil {
			return err
		}
		if err := snapshot.Validate(); err != nil {
			return err
		}
		p.equities[normalizeKey(snapshot.Instrument.Code)] = snapshot
	case "INDEX":
		snapshot, err := parseCSVIndex(row, record)
		if err != nil {
			return err
		}
		if err := snapshot.Validate(); err != nil {
			return err
		}
		p.indexes[normalizeKey(snapshot.Code)] = snapshot
	case "FX":
		rate, err := parseCSVFX(row, record)
		if err != nil {
			return err
		}
		if err := rate.Validate(); err != nil {
			return err
		}
		p.fxRates[fxKey(rate.BaseCurrency, rate.QuoteCurrency)] = rate
	case "CALENDAR":
		calendar, err := parseCSVCalendar(row, record)
		if err != nil {
			return err
		}
		if err := calendar.Validate(); err != nil {
			return err
		}
		p.calendars[calendarKey(calendar.Market, calendar.Date)] = calendar
	default:
		return fmt.Errorf("unsupported csv row kind %q", row["kind"])
	}
	return nil
}

func parseCSVFund(row map[string]string, record []string) (domain.FundSnapshot, error) {
	instrumentType := domain.InstrumentType(strings.ToLower(firstNonEmpty(row["type"], row["kind"])))
	if instrumentType == "lof" {
		instrumentType = domain.InstrumentETF
	}
	snapshot := domain.FundSnapshot{
		Instrument: domain.FundInstrument{
			Code:     firstNonEmpty(row["code"], row["symbol"]),
			Name:     row["name"],
			Market:   strings.ToUpper(row["market"]),
			Currency: strings.ToUpper(row["currency"]),
			Type:     instrumentType,
		},
		NAV:             parseOptionalFloat(row["nav"]),
		Price:           parseOptionalFloat(row["price"]),
		DailyChangePct:  parseOptionalFloat(row["daily_change_pct"]),
		OneYearReturn:   parseOptionalFloat(row["one_year_return_pct"]),
		MaxDrawdownPct:  parseOptionalFloat(row["max_drawdown_pct"]),
		VolatilityPct:   parseOptionalFloat(row["volatility_pct"]),
		ExpenseRatioPct: parseOptionalFloat(row["expense_ratio_pct"]),
		Manager:         row["manager"],
		AssetSize:       row["asset_size"],
		TopHoldings:     splitList(row["top_holdings"]),
	}
	metadata, err := parseCSVMetadata(row, record)
	if err != nil {
		return domain.FundSnapshot{}, err
	}
	snapshot.Metadata = metadata
	return snapshot, nil
}

func parseCSVEquity(row map[string]string, record []string) (domain.EquitySnapshot, error) {
	snapshot := domain.EquitySnapshot{
		Instrument: domain.FundInstrument{
			Code:     firstNonEmpty(row["symbol"], row["code"]),
			Name:     row["name"],
			Market:   strings.ToUpper(row["market"]),
			Currency: strings.ToUpper(row["currency"]),
			Type:     domain.InstrumentEquity,
		},
		Price:          parseOptionalFloat(row["price"]),
		DailyChangePct: parseOptionalFloat(row["daily_change_pct"]),
		OneYearReturn:  parseOptionalFloat(row["one_year_return_pct"]),
		MaxDrawdownPct: parseOptionalFloat(row["max_drawdown_pct"]),
		VolatilityPct:  parseOptionalFloat(row["volatility_pct"]),
	}
	metadata, err := parseCSVMetadata(row, record)
	if err != nil {
		return domain.EquitySnapshot{}, err
	}
	snapshot.Metadata = metadata
	return snapshot, nil
}

func parseCSVIndex(row map[string]string, record []string) (domain.IndexSnapshot, error) {
	snapshot := domain.IndexSnapshot{
		Code:           firstNonEmpty(row["code"], row["symbol"]),
		Name:           row["name"],
		Market:         strings.ToUpper(row["market"]),
		Currency:       strings.ToUpper(row["currency"]),
		Level:          parseOptionalFloat(firstNonEmpty(row["level"], row["price"])),
		DailyChangePct: parseOptionalFloat(row["daily_change_pct"]),
		OneYearReturn:  parseOptionalFloat(row["one_year_return_pct"]),
		MaxDrawdownPct: parseOptionalFloat(row["max_drawdown_pct"]),
	}
	metadata, err := parseCSVMetadata(row, record)
	if err != nil {
		return domain.IndexSnapshot{}, err
	}
	snapshot.Metadata = metadata
	return snapshot, nil
}

func parseCSVFX(row map[string]string, record []string) (domain.FXRate, error) {
	rate := domain.FXRate{
		BaseCurrency:  strings.ToUpper(row["base_currency"]),
		QuoteCurrency: strings.ToUpper(row["quote_currency"]),
		Rate:          parseOptionalFloat(row["rate"]),
	}
	metadata, err := parseCSVMetadata(row, record)
	if err != nil {
		return domain.FXRate{}, err
	}
	rate.Metadata = metadata
	return rate, nil
}

func parseCSVCalendar(row map[string]string, record []string) (domain.MarketCalendar, error) {
	calendar := domain.MarketCalendar{
		Market:       strings.ToUpper(row["market"]),
		Date:         row["date"],
		IsTradingDay: parseOptionalBool(row["is_trading_day"]),
		IsHalfDay:    parseOptionalBool(row["is_half_day"]),
		Session:      row["session"],
		Timezone:     row["timezone"],
		Delay:        row["delay"],
	}
	metadata, err := parseCSVMetadata(row, record)
	if err != nil {
		return domain.MarketCalendar{}, err
	}
	calendar.Metadata = metadata
	return calendar, nil
}

func parseCSVMetadata(row map[string]string, record []string) (domain.SourceMetadata, error) {
	fetchedAt, err := parseRequiredTime(row["fetched_at"], "fetched_at")
	if err != nil {
		return domain.SourceMetadata{}, err
	}
	marketTime, err := parseRequiredTime(row["market_time"], "market_time")
	if err != nil {
		return domain.SourceMetadata{}, err
	}
	confidence, err := strconv.ParseFloat(strings.TrimSpace(row["confidence"]), 64)
	if err != nil {
		return domain.SourceMetadata{}, fmt.Errorf("confidence must be numeric: %w", err)
	}
	rawHash := row["raw_payload_hash"]
	if rawHash == "" {
		rawHash = hashRecord(record)
	}
	schemaVersion := row["schema_version"]
	if schemaVersion == "" {
		schemaVersion = csvSchemaVersion
	}
	return domain.SourceMetadata{
		Source:         row["source"],
		Provider:       firstNonEmpty(row["provider"], "csv_provider"),
		FetchedAt:      fetchedAt,
		MarketTime:     marketTime,
		Timezone:       row["timezone"],
		Delay:          row["delay"],
		LicenseTerms:   row["license_terms"],
		Confidence:     confidence,
		SchemaVersion:  schemaVersion,
		RawPayloadHash: rawHash,
	}, nil
}

func csvPaths(path string) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return []string{path}, nil
	}
	var paths []string
	err = filepath.WalkDir(path, func(filePath string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(filePath), ".csv") {
			paths = append(paths, filePath)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	return paths, nil
}

func normalizeHeader(header []string) []string {
	columns := make([]string, 0, len(header))
	for _, column := range header {
		columns = append(columns, strings.ToLower(strings.TrimSpace(column)))
	}
	return columns
}

func rowValues(columns []string, record []string) map[string]string {
	row := map[string]string{}
	for i, column := range columns {
		if i >= len(record) {
			break
		}
		row[column] = strings.TrimSpace(record[i])
	}
	return row
}

func emptyRow(row map[string]string) bool {
	for _, value := range row {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}

func normalizeKey(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func fxKey(baseCurrency, quoteCurrency string) string {
	return normalizeKey(baseCurrency) + "/" + normalizeKey(quoteCurrency)
}

func calendarKey(market string, date string) string {
	return normalizeKey(market) + "|" + strings.TrimSpace(date)
}

func parseRequiredTime(value string, field string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, fmt.Errorf("%s is required", field)
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("%s must use RFC3339 or yyyy-mm-dd format", field)
}

func parseOptionalFloat(value string) float64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func parseOptionalBool(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return value == "true" || value == "1" || value == "yes" || value == "y"
}

func splitList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, "|")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		if item := strings.TrimSpace(part); item != "" {
			items = append(items, item)
		}
	}
	return items
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func hashRecord(record []string) string {
	sum := sha256.Sum256([]byte(strings.Join(record, "\x1f")))
	return hex.EncodeToString(sum[:])
}
