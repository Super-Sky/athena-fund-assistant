package preference

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Super-Sky/athena-fund-assistant/internal/domain"
)

const schemaVersion = "preference_knowledge.v1"

// Store defines durable preference and strategy knowledge operations.
// Store 定义长期偏好和策略知识操作。
type Store interface {
	Workspace(ctx context.Context, userID string) (domain.KnowledgeWorkspace, error)
	SavePreferenceDraft(ctx context.Context, userID string, input PreferenceInput) (domain.KnowledgeWorkspace, error)
	ActivatePreference(ctx context.Context, userID string, revisionID string) (domain.KnowledgeWorkspace, error)
	SaveKnowledgeDraft(ctx context.Context, userID string, input KnowledgeInput) (domain.KnowledgeWorkspace, error)
	ActivateKnowledge(ctx context.Context, userID string, itemID string, revisionID string) (domain.KnowledgeWorkspace, error)
	RollbackKnowledge(ctx context.Context, userID string, itemID string, revisionID string) (domain.KnowledgeWorkspace, error)
}

// PreferenceInput carries a preference draft submitted by UI, tool, or MCP.
// PreferenceInput 承载 UI、tool 或 MCP 提交的偏好草稿。
type PreferenceInput struct {
	RiskPreference       domain.RiskPreference `json:"risk_preference"`
	CommunicationStyle   string                `json:"communication_style"`
	DefaultStrategyLevel string                `json:"default_strategy_level"`
	PreferredAssets      []string              `json:"preferred_assets"`
	BlockedAssets        []string              `json:"blocked_assets"`
	ReviewFrequencyDays  int                   `json:"review_frequency_days"`
	AgentMarkdown        string                `json:"agent_md"`
	Source               string                `json:"source"`
	Author               string                `json:"author"`
	Confidence           float64               `json:"confidence"`
	Summary              string                `json:"summary"`
}

// KnowledgeInput carries one strategy knowledge draft.
// KnowledgeInput 承载一条策略知识草稿。
type KnowledgeInput struct {
	ItemID     string   `json:"item_id"`
	Title      string   `json:"title"`
	Category   string   `json:"category"`
	Content    string   `json:"content"`
	Tags       []string `json:"tags"`
	Source     string   `json:"source"`
	Author     string   `json:"author"`
	Confidence float64  `json:"confidence"`
	Summary    string   `json:"summary"`
}

type memoryUserState struct {
	preference domain.PreferenceProfile
	items      map[string]domain.KnowledgeItem
	revisions  []domain.KnowledgeRevision
	audit      []domain.KnowledgeAuditEvent
}

// MemoryStore keeps preference and knowledge state for local MVP demos.
// MemoryStore 为本地 MVP 演示保存偏好和知识状态。
type MemoryStore struct {
	mu     sync.Mutex
	now    func() time.Time
	states map[string]*memoryUserState
}

// NewMemoryStore creates a seeded in-memory preference store.
// NewMemoryStore 创建带种子数据的内存偏好 store。
func NewMemoryStore() *MemoryStore {
	store := &MemoryStore{
		now:    time.Now,
		states: map[string]*memoryUserState{},
	}
	store.seed("demo-user")
	return store
}

// Workspace returns active preference, knowledge items, revisions, and audit trail.
// Workspace 返回已启用偏好、知识条目、版本和审计轨迹。
func (s *MemoryStore) Workspace(_ context.Context, userID string) (domain.KnowledgeWorkspace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.ensure(userID)
	return snapshot(state), nil
}

// SavePreferenceDraft stores a preference revision without activating it.
// SavePreferenceDraft 保存偏好版本草稿但不启用。
func (s *MemoryStore) SavePreferenceDraft(_ context.Context, userID string, input PreferenceInput) (domain.KnowledgeWorkspace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.ensure(userID)
	now := s.now().UTC()
	profile := domain.PreferenceProfile{
		UserID:               userID,
		RiskPreference:       input.RiskPreference,
		CommunicationStyle:   strings.TrimSpace(input.CommunicationStyle),
		DefaultStrategyLevel: firstNonEmpty(input.DefaultStrategyLevel, "balanced"),
		PreferredAssets:      compact(input.PreferredAssets),
		BlockedAssets:        compact(input.BlockedAssets),
		ReviewFrequencyDays:  input.ReviewFrequencyDays,
		AgentMarkdown:        strings.TrimSpace(input.AgentMarkdown),
		UpdatedAt:            now,
		Source:               firstNonEmpty(input.Source, "manual_ui"),
		Author:               firstNonEmpty(input.Author, "demo-user"),
		Confidence:           fallbackConfidence(input.Confidence),
		SchemaVersion:        schemaVersion,
		GovernanceDecision:   "draft_saved_pending_activation",
	}
	if err := profile.Validate(); err != nil {
		return domain.KnowledgeWorkspace{}, err
	}
	contentSnapshot, err := json.Marshal(profile)
	if err != nil {
		return domain.KnowledgeWorkspace{}, err
	}
	revision := domain.KnowledgeRevision{
		ID:                 nextID("pref", now, len(state.revisions)+1),
		TargetType:         "preference",
		TargetID:           userID,
		Status:             domain.KnowledgeDraft,
		Summary:            firstNonEmpty(input.Summary, "Preference draft saved"),
		ContentSnapshot:    string(contentSnapshot),
		Source:             profile.Source,
		Author:             profile.Author,
		Confidence:         profile.Confidence,
		SchemaVersion:      profile.SchemaVersion,
		GovernanceDecision: profile.GovernanceDecision,
		GovernanceTrace:    []string{"validated_schema", "saved_as_draft", "activation_requires_explicit_call"},
		CreatedAt:          now,
	}
	state.revisions = append(state.revisions, revision)
	state.audit = append(state.audit, audit(now, userID, "preference_draft_saved", userID, revision.Summary, map[string]string{"revision_id": revision.ID}))
	return snapshot(state), nil
}

// ActivatePreference promotes a preference draft to the active agent profile.
// ActivatePreference 将偏好草稿提升为启用中的 Agent profile。
func (s *MemoryStore) ActivatePreference(_ context.Context, userID string, revisionID string) (domain.KnowledgeWorkspace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.ensure(userID)
	revisionIndex := findRevision(state.revisions, "preference", userID, revisionID)
	if revisionIndex < 0 {
		return domain.KnowledgeWorkspace{}, errors.New("preference revision not found")
	}
	var profile domain.PreferenceProfile
	if err := json.Unmarshal([]byte(state.revisions[revisionIndex].ContentSnapshot), &profile); err != nil {
		return domain.KnowledgeWorkspace{}, fmt.Errorf("decode preference revision: %w", err)
	}
	now := s.now().UTC()
	profile.ActiveRevisionID = revisionID
	profile.UpdatedAt = now
	profile.GovernanceDecision = "activated_after_schema_validation"
	profile.GovernanceDecisionNotes = "Manual activation gate passed in local MVP."
	state.preference = profile
	state.revisions[revisionIndex].Status = domain.KnowledgeActive
	state.revisions[revisionIndex].GovernanceDecision = "activated_after_schema_validation"
	state.revisions[revisionIndex].ActivatedAt = &now
	state.audit = append(state.audit, audit(now, userID, "preference_activated", userID, "Preference revision activated", map[string]string{"revision_id": revisionID}))
	return snapshot(state), nil
}

// SaveKnowledgeDraft stores a strategy knowledge draft without activating it.
// SaveKnowledgeDraft 保存策略知识草稿但不启用。
func (s *MemoryStore) SaveKnowledgeDraft(_ context.Context, userID string, input KnowledgeInput) (domain.KnowledgeWorkspace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.ensure(userID)
	now := s.now().UTC()
	itemID := strings.TrimSpace(input.ItemID)
	if itemID == "" {
		itemID = nextID("knowledge", now, len(state.items)+1)
	}
	item := domain.KnowledgeItem{
		ID:                 itemID,
		Title:              strings.TrimSpace(input.Title),
		Category:           strings.TrimSpace(input.Category),
		Content:            strings.TrimSpace(input.Content),
		Tags:               compact(input.Tags),
		Status:             domain.KnowledgeDraft,
		Source:             firstNonEmpty(input.Source, "manual_ui"),
		Author:             firstNonEmpty(input.Author, userID),
		Confidence:         fallbackConfidence(input.Confidence),
		SchemaVersion:      schemaVersion,
		GovernanceDecision: "draft_saved_pending_activation",
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if existing, ok := state.items[itemID]; ok {
		item.CreatedAt = existing.CreatedAt
		item.ActiveRevisionID = existing.ActiveRevisionID
	}
	if err := item.Validate(); err != nil {
		return domain.KnowledgeWorkspace{}, err
	}
	revision := domain.KnowledgeRevision{
		ID:                 nextID("rev", now, len(state.revisions)+1),
		TargetType:         "knowledge_item",
		TargetID:           itemID,
		Status:             domain.KnowledgeDraft,
		Summary:            firstNonEmpty(input.Summary, "Knowledge draft saved"),
		ContentSnapshot:    item.Content,
		Source:             item.Source,
		Author:             item.Author,
		Confidence:         item.Confidence,
		SchemaVersion:      item.SchemaVersion,
		GovernanceDecision: item.GovernanceDecision,
		GovernanceTrace:    []string{"validated_schema", "saved_as_draft", "activation_requires_explicit_call"},
		CreatedAt:          now,
	}
	state.items[itemID] = item
	state.revisions = append(state.revisions, revision)
	state.audit = append(state.audit, audit(now, userID, "knowledge_draft_saved", itemID, revision.Summary, map[string]string{"revision_id": revision.ID}))
	return snapshot(state), nil
}

// ActivateKnowledge promotes a strategy knowledge draft to active status.
// ActivateKnowledge 将策略知识草稿提升为启用状态。
func (s *MemoryStore) ActivateKnowledge(_ context.Context, userID string, itemID string, revisionID string) (domain.KnowledgeWorkspace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.ensure(userID)
	item, ok := state.items[itemID]
	if !ok {
		return domain.KnowledgeWorkspace{}, errors.New("knowledge item not found")
	}
	revisionIndex := findRevision(state.revisions, "knowledge_item", itemID, revisionID)
	if revisionIndex < 0 {
		return domain.KnowledgeWorkspace{}, errors.New("knowledge revision not found")
	}
	now := s.now().UTC()
	item.Status = domain.KnowledgeActive
	item.ActiveRevisionID = revisionID
	item.UpdatedAt = now
	item.GovernanceDecision = "activated_after_schema_validation"
	item.GovernanceDecisionNotes = "Manual activation gate passed in local MVP."
	item.Content = state.revisions[revisionIndex].ContentSnapshot
	state.items[itemID] = item
	state.revisions[revisionIndex].Status = domain.KnowledgeActive
	state.revisions[revisionIndex].GovernanceDecision = "activated_after_schema_validation"
	state.revisions[revisionIndex].ActivatedAt = &now
	state.audit = append(state.audit, audit(now, userID, "knowledge_activated", itemID, "Knowledge revision activated", map[string]string{"revision_id": revisionID}))
	return snapshot(state), nil
}

// RollbackKnowledge activates a previous knowledge revision and records governance trace.
// RollbackKnowledge 启用历史知识版本并记录治理 trace。
func (s *MemoryStore) RollbackKnowledge(_ context.Context, userID string, itemID string, revisionID string) (domain.KnowledgeWorkspace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.ensure(userID)
	item, ok := state.items[itemID]
	if !ok {
		return domain.KnowledgeWorkspace{}, errors.New("knowledge item not found")
	}
	revisionIndex := findRevision(state.revisions, "knowledge_item", itemID, revisionID)
	if revisionIndex < 0 {
		return domain.KnowledgeWorkspace{}, errors.New("knowledge revision not found")
	}
	now := s.now().UTC()
	item.Status = domain.KnowledgeActive
	item.ActiveRevisionID = revisionID
	item.Content = state.revisions[revisionIndex].ContentSnapshot
	item.UpdatedAt = now
	item.GovernanceDecision = "rollback_activated"
	item.GovernanceDecisionNotes = "Rollback selected an existing audited revision."
	state.items[itemID] = item
	state.audit = append(state.audit, audit(now, userID, "knowledge_rollback", itemID, "Knowledge rollback activated", map[string]string{"revision_id": revisionID}))
	return snapshot(state), nil
}

func (s *MemoryStore) ensure(userID string) *memoryUserState {
	userID = firstNonEmpty(userID, "demo-user")
	if _, ok := s.states[userID]; !ok {
		s.seed(userID)
	}
	return s.states[userID]
}

func (s *MemoryStore) seed(userID string) {
	now := s.now().UTC()
	preferenceRevisionID := "pref_seed_v1"
	itemID := "knowledge_position_rules_v1"
	itemRevisionID := "rev_position_rules_v1"
	s.states[userID] = &memoryUserState{
		preference: domain.PreferenceProfile{
			UserID:                  userID,
			RiskPreference:          domain.RiskBalanced,
			CommunicationStyle:      "concise_with_evidence",
			DefaultStrategyLevel:    "balanced",
			PreferredAssets:         []string{"broad_index_etf", "public_fund"},
			BlockedAssets:           []string{"leveraged_etf", "single_name_concentration"},
			ReviewFrequencyDays:     7,
			AgentMarkdown:           "# Agent Profile\n\n- Keep outputs in conservative / balanced / aggressive options.\n- Explain percentage changes from profile limits, portfolio allocation, strategy templates, or data evidence.\n- Never present automatic trading as available.\n",
			ActiveRevisionID:        preferenceRevisionID,
			UpdatedAt:               now,
			Source:                  "seed",
			Author:                  "system",
			Confidence:              0.75,
			SchemaVersion:           schemaVersion,
			GovernanceDecision:      "seed_activated",
			GovernanceDecisionNotes: "Seed profile for local MVP demo.",
		},
		items: map[string]domain.KnowledgeItem{
			itemID: {
				ID:                      itemID,
				Title:                   "Position sizing guardrails",
				Category:                "allocation_rule",
				Content:                 "Conservative trims exposure when current allocation exceeds the single-instrument cap or drawdown exceeds profile tolerance. Balanced holds or trims 5% when evidence is mixed. Aggressive can add only inside the configured cap and must include a review trigger.",
				Tags:                    []string{"allocation", "risk", "review"},
				Status:                  domain.KnowledgeActive,
				ActiveRevisionID:        itemRevisionID,
				Source:                  "seed",
				Author:                  "system",
				Confidence:              0.75,
				SchemaVersion:           schemaVersion,
				GovernanceDecision:      "seed_activated",
				GovernanceDecisionNotes: "Seed strategy template for local MVP demo.",
				CreatedAt:               now,
				UpdatedAt:               now,
			},
		},
		revisions: []domain.KnowledgeRevision{
			{
				ID:                 preferenceRevisionID,
				TargetType:         "preference",
				TargetID:           userID,
				Status:             domain.KnowledgeActive,
				Summary:            "Seed preference profile activated",
				ContentSnapshot:    "seed agent.md",
				Source:             "seed",
				Author:             "system",
				Confidence:         0.75,
				SchemaVersion:      schemaVersion,
				GovernanceDecision: "seed_activated",
				GovernanceTrace:    []string{"seeded", "schema_validated", "activated"},
				CreatedAt:          now,
				ActivatedAt:        &now,
			},
			{
				ID:                 itemRevisionID,
				TargetType:         "knowledge_item",
				TargetID:           itemID,
				Status:             domain.KnowledgeActive,
				Summary:            "Seed allocation strategy template activated",
				ContentSnapshot:    "Conservative / balanced / aggressive allocation guardrails.",
				Source:             "seed",
				Author:             "system",
				Confidence:         0.75,
				SchemaVersion:      schemaVersion,
				GovernanceDecision: "seed_activated",
				GovernanceTrace:    []string{"seeded", "schema_validated", "activated"},
				CreatedAt:          now,
				ActivatedAt:        &now,
			},
		},
		audit: []domain.KnowledgeAuditEvent{
			audit(now, userID, "workspace_seeded", userID, "Seed preference and strategy knowledge activated", map[string]string{"schema_version": schemaVersion}),
		},
	}
}

func snapshot(state *memoryUserState) domain.KnowledgeWorkspace {
	items := make([]domain.KnowledgeItem, 0, len(state.items))
	for _, item := range state.items {
		items = append(items, item)
	}
	revisions := append([]domain.KnowledgeRevision{}, state.revisions...)
	auditTrail := append([]domain.KnowledgeAuditEvent{}, state.audit...)
	return domain.KnowledgeWorkspace{
		Preference: state.preference,
		Items:      items,
		Revisions:  revisions,
		Audit:      auditTrail,
	}
}

func audit(now time.Time, userID string, action string, targetID string, summary string, metadata map[string]string) domain.KnowledgeAuditEvent {
	return domain.KnowledgeAuditEvent{
		ID:        nextID("audit", now, 0),
		UserID:    userID,
		Action:    action,
		TargetID:  targetID,
		Summary:   summary,
		Metadata:  metadata,
		CreatedAt: now,
	}
}

func findRevision(revisions []domain.KnowledgeRevision, targetType string, targetID string, revisionID string) int {
	for i, revision := range revisions {
		if revision.TargetType == targetType && revision.TargetID == targetID && revision.ID == revisionID {
			return i
		}
	}
	return -1
}

func nextID(prefix string, now time.Time, seq int) string {
	if seq <= 0 {
		return fmt.Sprintf("%s_%d", prefix, now.UnixNano())
	}
	return fmt.Sprintf("%s_%d_%d", prefix, now.UnixNano(), seq)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func fallbackConfidence(value float64) float64 {
	if value <= 0 {
		return 0.7
	}
	return value
}

func compact(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
