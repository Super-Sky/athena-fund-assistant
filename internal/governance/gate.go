// Package governance applies financial-output guardrails before delivery.
// Package governance 在金融输出交付前执行治理护栏。
package governance

import (
	"strings"
	"time"

	"github.com/Super-Sky/athena-fund-assistant/internal/domain"
)

// Status describes whether a governance check passed, needs disclosure, or blocks delivery.
// Status 描述治理检查是通过、需要披露，还是阻断交付。
type Status string

const (
	StatusPassed  Status = "passed"
	StatusFlagged Status = "flagged"
	StatusBlocked Status = "blocked"
)

// Check records one user-visible governance decision without exposing raw provider payloads.
// Check 记录一条面向用户的治理结论，不暴露原始 provider payload。
type Check struct {
	Rule    string `json:"rule"`
	Status  Status `json:"status"`
	Message string `json:"message"`
}

// Result is the deterministic audit record for one decision matrix.
// Result 是一份决策矩阵的确定性审计记录。
type Result struct {
	Decision Status  `json:"decision"`
	Checks   []Check `json:"checks"`
}

// Allowed reports whether the result can be delivered as financial decision support.
// Allowed 判断结果是否可作为金融决策支持交付。
func (r Result) Allowed() bool {
	return r.Decision != StatusBlocked
}

// Gate evaluates product-specific financial output rules.
// Gate 评估产品级金融输出规则。
type Gate struct{}

// NewGate creates the default deterministic financial governance gate.
// NewGate 创建默认的确定性金融治理门。
func NewGate() *Gate {
	return &Gate{}
}

// Evaluate checks multi-option shape, required disclosures, and disallowed language.
// Evaluate 检查多方案结构、必要披露和禁止措辞。
func (g *Gate) Evaluate(matrix domain.DecisionMatrix) Result {
	checks := []Check{
		checkTrace(matrix.Trace),
		checkOptionDisclosures(matrix.Options),
		checkStrategyBasis(matrix.Options),
		checkForbiddenLanguage(matrix.Options),
	}
	decision := StatusPassed
	for _, check := range checks {
		if check.Status == StatusBlocked {
			decision = StatusBlocked
			break
		}
		if check.Status == StatusFlagged {
			decision = StatusFlagged
		}
	}
	return Result{Decision: decision, Checks: checks}
}

func checkTrace(trace domain.TraceSummary) Check {
	missing := make([]string, 0, 5)
	if strings.TrimSpace(trace.DataSource) == "" {
		missing = append(missing, "source")
	}
	if strings.TrimSpace(trace.DataProvider) == "" {
		missing = append(missing, "provider")
	}
	if strings.TrimSpace(trace.DataFetchedAt) == "" {
		missing = append(missing, "fetched_at")
	}
	if strings.TrimSpace(trace.MarketTime) == "" {
		missing = append(missing, "market_time")
	}
	if strings.TrimSpace(trace.Timezone) == "" {
		missing = append(missing, "timezone")
	}
	if len(missing) > 0 {
		return Check{Rule: "source_and_freshness_metadata", Status: StatusFlagged, Message: "missing required data metadata: " + strings.Join(missing, ", ")}
	}
	if _, err := time.Parse(time.RFC3339, trace.DataFetchedAt); err != nil {
		return Check{Rule: "source_and_freshness_metadata", Status: StatusFlagged, Message: "fetched_at is not RFC3339"}
	}
	if _, err := time.Parse(time.RFC3339, trace.MarketTime); err != nil {
		return Check{Rule: "source_and_freshness_metadata", Status: StatusFlagged, Message: "market_time is not RFC3339"}
	}
	return Check{Rule: "source_and_freshness_metadata", Status: StatusPassed, Message: "source and freshness metadata are present"}
}

func checkOptionDisclosures(options []domain.DecisionOption) Check {
	if len(options) < 2 {
		return Check{Rule: "multi_option_risk_and_invalidation", Status: StatusBlocked, Message: "financial output requires at least two decision options"}
	}
	for _, option := range options {
		if len(option.Risks) == 0 || strings.TrimSpace(option.Invalidation) == "" || option.ReviewAfterDays <= 0 {
			return Check{Rule: "risk_invalidation_and_review", Status: StatusFlagged, Message: "option " + option.ID + " is missing risk, invalidation, or review timing"}
		}
	}
	return Check{Rule: "risk_invalidation_and_review", Status: StatusPassed, Message: "multiple options include risk, invalidation, and review timing"}
}

func checkStrategyBasis(options []domain.DecisionOption) Check {
	for _, option := range options {
		if option.AllocationChangePct != 0 && len(option.StrategyBasis) == 0 {
			return Check{Rule: "allocation_percentage_basis", Status: StatusBlocked, Message: "option " + option.ID + " has an allocation change without a derivation basis"}
		}
	}
	return Check{Rule: "allocation_percentage_basis", Status: StatusPassed, Message: "allocation changes have a traceable basis"}
}

func checkForbiddenLanguage(options []domain.DecisionOption) Check {
	for _, option := range options {
		text := strings.ToLower(strings.Join([]string{
			option.Action,
			strings.Join(option.Conditions, " "),
			strings.Join(option.Evidence, " "),
			strings.Join(option.Risks, " "),
			option.Invalidation,
			option.PortfolioImpact,
		}, " "))
		if containsAny(text, "guaranteed return", "guarantee profit", "稳赚", "保本", "必涨", "一定赚钱") {
			return Check{Rule: "no_guaranteed_returns", Status: StatusBlocked, Message: "option " + option.ID + " contains guaranteed-return language"}
		}
		if containsAny(text, "automatic trading", "place order", "execute order", "自动交易", "自动下单", "执行交易") {
			return Check{Rule: "no_automatic_trading", Status: StatusBlocked, Message: "option " + option.ID + " contains trading-execution language"}
		}
		if containsAny(text, "must buy", "must sell", "all in", "必须买", "必须卖", "必须卖出", "满仓", "清仓") {
			return Check{Rule: "no_single_absolute_conclusion", Status: StatusBlocked, Message: "option " + option.ID + " contains an absolute trading conclusion"}
		}
	}
	return Check{Rule: "disallowed_financial_language", Status: StatusPassed, Message: "no guaranteed-return, automatic-trading, or absolute-command language detected"}
}

func containsAny(text string, terms ...string) bool {
	for _, term := range terms {
		if strings.Contains(text, term) {
			return true
		}
	}
	return false
}
