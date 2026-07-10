package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// PreferenceProfile stores durable user-level agent preferences.
// PreferenceProfile 保存用户级长期 Agent 偏好。
type PreferenceProfile struct {
	UserID                  string         `json:"user_id"`
	RiskPreference          RiskPreference `json:"risk_preference"`
	CommunicationStyle      string         `json:"communication_style"`
	DefaultStrategyLevel    string         `json:"default_strategy_level"`
	PreferredAssets         []string       `json:"preferred_assets"`
	BlockedAssets           []string       `json:"blocked_assets"`
	ReviewFrequencyDays     int            `json:"review_frequency_days"`
	AgentMarkdown           string         `json:"agent_md"`
	ActiveRevisionID        string         `json:"active_revision_id"`
	UpdatedAt               time.Time      `json:"updated_at"`
	Source                  string         `json:"source"`
	Author                  string         `json:"author"`
	Confidence              float64        `json:"confidence"`
	SchemaVersion           string         `json:"schema_version"`
	GovernanceDecision      string         `json:"governance_decision"`
	GovernanceDecisionNotes string         `json:"governance_decision_notes"`
}

// Validate checks preference shape before activation.
// Validate 在偏好正式启用前检查结构。
func (p PreferenceProfile) Validate() error {
	if strings.TrimSpace(p.UserID) == "" {
		return errors.New("user_id is required")
	}
	switch p.RiskPreference {
	case RiskConservative, RiskBalanced, RiskAggressive:
	default:
		return fmt.Errorf("unsupported risk preference %q", p.RiskPreference)
	}
	if strings.TrimSpace(p.DefaultStrategyLevel) == "" {
		return errors.New("default_strategy_level is required")
	}
	if p.ReviewFrequencyDays <= 0 {
		return errors.New("review_frequency_days must be positive")
	}
	if p.Confidence <= 0 || p.Confidence > 1 {
		return errors.New("confidence must be between 0 and 1")
	}
	if strings.TrimSpace(p.SchemaVersion) == "" {
		return errors.New("schema_version is required")
	}
	return nil
}

// KnowledgeStatus identifies whether a preference or knowledge revision is active.
// KnowledgeStatus 标识偏好或知识版本是否已启用。
type KnowledgeStatus string

const (
	KnowledgeDraft    KnowledgeStatus = "draft"
	KnowledgeActive   KnowledgeStatus = "active"
	KnowledgeArchived KnowledgeStatus = "archived"
)

// KnowledgeItem stores one reusable fund-strategy rule or context document.
// KnowledgeItem 保存一条可复用基金策略规则或上下文文档。
type KnowledgeItem struct {
	ID                      string          `json:"id"`
	Title                   string          `json:"title"`
	Category                string          `json:"category"`
	Content                 string          `json:"content"`
	Tags                    []string        `json:"tags"`
	Status                  KnowledgeStatus `json:"status"`
	ActiveRevisionID        string          `json:"active_revision_id"`
	Source                  string          `json:"source"`
	Author                  string          `json:"author"`
	Confidence              float64         `json:"confidence"`
	SchemaVersion           string          `json:"schema_version"`
	GovernanceDecision      string          `json:"governance_decision"`
	GovernanceDecisionNotes string          `json:"governance_decision_notes"`
	CreatedAt               time.Time       `json:"created_at"`
	UpdatedAt               time.Time       `json:"updated_at"`
}

// Validate checks knowledge item fields before draft or activation.
// Validate 在草稿或启用知识条目前检查字段。
func (i KnowledgeItem) Validate() error {
	if strings.TrimSpace(i.Title) == "" {
		return errors.New("knowledge title is required")
	}
	if strings.TrimSpace(i.Category) == "" {
		return errors.New("knowledge category is required")
	}
	if strings.TrimSpace(i.Content) == "" {
		return errors.New("knowledge content is required")
	}
	if i.Confidence <= 0 || i.Confidence > 1 {
		return errors.New("confidence must be between 0 and 1")
	}
	if strings.TrimSpace(i.SchemaVersion) == "" {
		return errors.New("schema_version is required")
	}
	return nil
}

// KnowledgeRevision records an immutable preference or strategy knowledge change.
// KnowledgeRevision 记录不可变的偏好或策略知识变更。
type KnowledgeRevision struct {
	ID                 string          `json:"id"`
	TargetType         string          `json:"target_type"`
	TargetID           string          `json:"target_id"`
	Status             KnowledgeStatus `json:"status"`
	Summary            string          `json:"summary"`
	ContentSnapshot    string          `json:"content_snapshot"`
	Source             string          `json:"source"`
	Author             string          `json:"author"`
	Confidence         float64         `json:"confidence"`
	SchemaVersion      string          `json:"schema_version"`
	GovernanceDecision string          `json:"governance_decision"`
	GovernanceTrace    []string        `json:"governance_trace"`
	CreatedAt          time.Time       `json:"created_at"`
	ActivatedAt        *time.Time      `json:"activated_at,omitempty"`
}

// KnowledgeAuditEvent records user, tool, or governance actions.
// KnowledgeAuditEvent 记录用户、工具或治理动作。
type KnowledgeAuditEvent struct {
	ID        string            `json:"id"`
	UserID    string            `json:"user_id"`
	Action    string            `json:"action"`
	TargetID  string            `json:"target_id"`
	Summary   string            `json:"summary"`
	Metadata  map[string]string `json:"metadata"`
	CreatedAt time.Time         `json:"created_at"`
}

// KnowledgeWorkspace is the user-facing preference and strategy knowledge view.
// KnowledgeWorkspace 是面向用户的偏好和策略知识视图。
type KnowledgeWorkspace struct {
	Preference PreferenceProfile     `json:"preference"`
	Items      []KnowledgeItem       `json:"items"`
	Revisions  []KnowledgeRevision   `json:"revisions"`
	Audit      []KnowledgeAuditEvent `json:"audit"`
}
