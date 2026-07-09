package conversation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Super-Sky/athena-fund-assistant/internal/domain"
)

const (
	defaultUserID          = "demo-user"
	defaultSkillID         = "fund_research"
	maxAttachmentSizeBytes = 10 << 20
	retentionDays          = 7
)

// Store defines the Agent conversation workspace persistence boundary.
// Store 定义 Agent 对话工作台的持久化边界。
type Store interface {
	Skills(context.Context) []domain.ConversationSkill
	Create(context.Context, CreateInput) (domain.ConversationDetail, error)
	Detail(context.Context, string) (domain.ConversationDetail, error)
	AddMessage(context.Context, string, MessageInput) (domain.ConversationDetail, error)
	SaveAttachment(context.Context, string, AttachmentInput) (domain.ConversationAttachment, error)
	RecordTrace(context.Context, string, TraceInput) (domain.ConversationDetail, error)
}

// CreateInput describes a new conversation request.
// CreateInput 描述新建对话请求。
type CreateInput struct {
	UserID  string
	SkillID string
	Title   string
}

// MessageInput describes one user message and optional attachment references.
// MessageInput 描述一条用户消息及可选附件引用。
type MessageInput struct {
	Role          string
	Content       string
	SkillID       string
	AttachmentIDs []string
}

// AttachmentInput carries one uploaded file stream and safe metadata.
// AttachmentInput 携带一份上传文件流及安全元数据。
type AttachmentInput struct {
	UserID      string
	FileName    string
	ContentType string
	SizeBytes   int64
	Reader      io.Reader
}

// TraceInput carries one safe timeline event from application integrations.
// TraceInput 携带一条来自应用集成的安全时间线事件。
type TraceInput struct {
	Kind     string
	Status   string
	Summary  string
	Metadata map[string]string
}

// MemoryStore stores conversation workspace state for local MVP runs.
// MemoryStore 为本地 MVP 保存对话工作台状态。
type MemoryStore struct {
	mu          sync.Mutex
	uploadDir   string
	skills      []domain.ConversationSkill
	sessions    map[string]domain.ConversationSession
	messages    map[string][]domain.ConversationMessage
	attachments map[string][]domain.ConversationAttachment
	trace       map[string][]domain.ConversationTraceEvent
}

// NewMemoryStore creates a local conversation store and upload directory.
// NewMemoryStore 创建本地对话存储和上传目录。
func NewMemoryStore(uploadDir string) (*MemoryStore, error) {
	if strings.TrimSpace(uploadDir) == "" {
		uploadDir = filepath.Join(os.TempDir(), "athena-fund-assistant-uploads")
	}
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		return nil, err
	}
	store := &MemoryStore{
		uploadDir:   uploadDir,
		skills:      defaultSkills(),
		sessions:    map[string]domain.ConversationSession{},
		messages:    map[string][]domain.ConversationMessage{},
		attachments: map[string][]domain.ConversationAttachment{},
		trace:       map[string][]domain.ConversationTraceEvent{},
	}
	return store, nil
}

// Skills returns the selectable local Agent skills.
// Skills 返回本地 Agent 可选择 skill 列表。
func (s *MemoryStore) Skills(context.Context) []domain.ConversationSkill {
	return append([]domain.ConversationSkill(nil), s.skills...)
}

// Create opens a new conversation session.
// Create 创建一条新的对话 session。
func (s *MemoryStore) Create(_ context.Context, input CreateInput) (domain.ConversationDetail, error) {
	now := time.Now().UTC()
	if input.UserID == "" {
		input.UserID = defaultUserID
	}
	if input.SkillID == "" {
		input.SkillID = defaultSkillID
	}
	if !s.skillExists(input.SkillID) {
		return domain.ConversationDetail{}, fmt.Errorf("skill %q is not enabled", input.SkillID)
	}
	if input.Title == "" {
		input.Title = "Fund research workspace"
	}
	session := domain.ConversationSession{
		ID:        fmt.Sprintf("conv_%d", now.UnixNano()),
		UserID:    input.UserID,
		SkillID:   input.SkillID,
		Title:     input.Title,
		Status:    "open",
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.ID] = session
	s.trace[session.ID] = append(s.trace[session.ID], traceEvent(session.ID, "conversation_created", "ok", "Conversation workspace opened.", map[string]string{
		"skill_id": input.SkillID,
	}))
	return s.detailLocked(session.ID)
}

// Detail returns one conversation read model.
// Detail 返回一条对话读模型。
func (s *MemoryStore) Detail(_ context.Context, conversationID string) (domain.ConversationDetail, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.detailLocked(conversationID)
}

// AddMessage appends a message and local trace events.
// AddMessage 追加消息和本地 trace 事件。
func (s *MemoryStore) AddMessage(_ context.Context, conversationID string, input MessageInput) (domain.ConversationDetail, error) {
	now := time.Now().UTC()
	if input.Role == "" {
		input.Role = "user"
	}
	if input.SkillID == "" {
		input.SkillID = defaultSkillID
	}
	message := domain.ConversationMessage{
		ID:             fmt.Sprintf("msg_%d", now.UnixNano()),
		ConversationID: conversationID,
		Role:           input.Role,
		Content:        input.Content,
		SkillID:        input.SkillID,
		AttachmentIDs:  append([]string(nil), input.AttachmentIDs...),
		CreatedAt:      now,
	}
	if err := message.Validate(); err != nil {
		return domain.ConversationDetail{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[conversationID]
	if !ok {
		return domain.ConversationDetail{}, errors.New("conversation not found")
	}
	if !s.skillExists(input.SkillID) {
		return domain.ConversationDetail{}, fmt.Errorf("skill %q is not enabled", input.SkillID)
	}
	for _, attachmentID := range input.AttachmentIDs {
		if !attachmentExists(s.attachments[conversationID], attachmentID) {
			return domain.ConversationDetail{}, fmt.Errorf("attachment %q not found", attachmentID)
		}
	}
	session.SkillID = input.SkillID
	session.UpdatedAt = now
	s.sessions[conversationID] = session
	s.messages[conversationID] = append(s.messages[conversationID], message)
	s.trace[conversationID] = append(s.trace[conversationID],
		traceEvent(conversationID, "message_received", "ok", "User message saved for Agent workspace.", map[string]string{
			"role":     input.Role,
			"skill_id": input.SkillID,
		}),
		traceEvent(conversationID, "athena_agent_run", "pending", "Athena agent run contract is ready for integration; local MVP does not call Athena yet.", map[string]string{
			"tool_contract": "remote_business_tools",
		}),
	)
	if len(input.AttachmentIDs) > 0 {
		s.trace[conversationID] = append(s.trace[conversationID], traceEvent(conversationID, "attachment_context", "pending", "Attachments are metadata-only until parser/OCR tools process them.", map[string]string{
			"attachment_count": fmt.Sprintf("%d", len(input.AttachmentIDs)),
		}))
	}
	return s.detailLocked(conversationID)
}

// SaveAttachment stores one uploaded file and returns metadata.
// SaveAttachment 保存一份上传文件并返回元数据。
func (s *MemoryStore) SaveAttachment(_ context.Context, conversationID string, input AttachmentInput) (domain.ConversationAttachment, error) {
	if input.UserID == "" {
		input.UserID = defaultUserID
	}
	if input.Reader == nil {
		return domain.ConversationAttachment{}, errors.New("attachment reader is required")
	}
	if input.SizeBytes <= 0 {
		return domain.ConversationAttachment{}, errors.New("attachment size is required")
	}
	if input.SizeBytes > maxAttachmentSizeBytes {
		return domain.ConversationAttachment{}, fmt.Errorf("attachment exceeds %d bytes", maxAttachmentSizeBytes)
	}
	if input.ContentType == "" {
		input.ContentType = mime.TypeByExtension(filepath.Ext(input.FileName))
	}
	if input.ContentType == "" {
		input.ContentType = "application/octet-stream"
	}
	now := time.Now().UTC()
	id := fmt.Sprintf("att_%d", now.UnixNano())
	safeName := safeFileName(input.FileName)
	localPath := filepath.Join(s.uploadDir, id+"_"+safeName)
	file, err := os.Create(localPath)
	if err != nil {
		return domain.ConversationAttachment{}, err
	}
	hasher := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(file, hasher), io.LimitReader(input.Reader, maxAttachmentSizeBytes+1))
	closeErr := file.Close()
	if copyErr != nil {
		_ = os.Remove(localPath)
		return domain.ConversationAttachment{}, copyErr
	}
	if closeErr != nil {
		_ = os.Remove(localPath)
		return domain.ConversationAttachment{}, closeErr
	}
	if written > maxAttachmentSizeBytes {
		_ = os.Remove(localPath)
		return domain.ConversationAttachment{}, fmt.Errorf("attachment exceeds %d bytes", maxAttachmentSizeBytes)
	}
	attachment := domain.ConversationAttachment{
		ID:             id,
		ConversationID: conversationID,
		UserID:         input.UserID,
		FileName:       safeName,
		ContentType:    input.ContentType,
		SizeBytes:      written,
		LocalPath:      localPath,
		Status:         "pending_parse",
		Unsupported:    !supportedContentType(input.ContentType),
		SHA256:         hex.EncodeToString(hasher.Sum(nil)),
		CreatedAt:      now,
		RetentionUntil: now.AddDate(0, 0, retentionDays),
	}
	if err := attachment.Validate(); err != nil {
		_ = os.Remove(localPath)
		return domain.ConversationAttachment{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.sessions[conversationID]; !ok {
		_ = os.Remove(localPath)
		return domain.ConversationAttachment{}, errors.New("conversation not found")
	}
	s.attachments[conversationID] = append(s.attachments[conversationID], attachment)
	status := "pending"
	if attachment.Unsupported {
		status = "unsupported"
	}
	s.trace[conversationID] = append(s.trace[conversationID], traceEvent(conversationID, "attachment_uploaded", status, "Attachment saved as metadata and not parsed as fact.", map[string]string{
		"file_name":    attachment.FileName,
		"content_type": attachment.ContentType,
	}))
	return attachment, nil
}

// RecordTrace appends one safe trace event to a conversation.
// RecordTrace 向对话追加一条安全 trace 事件。
func (s *MemoryStore) RecordTrace(_ context.Context, conversationID string, input TraceInput) (domain.ConversationDetail, error) {
	if strings.TrimSpace(input.Kind) == "" {
		return domain.ConversationDetail{}, errors.New("trace kind is required")
	}
	if strings.TrimSpace(input.Status) == "" {
		return domain.ConversationDetail{}, errors.New("trace status is required")
	}
	if strings.TrimSpace(input.Summary) == "" {
		return domain.ConversationDetail{}, errors.New("trace summary is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.sessions[conversationID]; !ok {
		return domain.ConversationDetail{}, errors.New("conversation not found")
	}
	s.trace[conversationID] = append(s.trace[conversationID], traceEvent(conversationID, input.Kind, input.Status, input.Summary, input.Metadata))
	return s.detailLocked(conversationID)
}

func (s *MemoryStore) detailLocked(conversationID string) (domain.ConversationDetail, error) {
	session, ok := s.sessions[conversationID]
	if !ok {
		return domain.ConversationDetail{}, errors.New("conversation not found")
	}
	detail := domain.ConversationDetail{
		Session:     session,
		Messages:    append([]domain.ConversationMessage(nil), s.messages[conversationID]...),
		Attachments: append([]domain.ConversationAttachment(nil), s.attachments[conversationID]...),
		Trace:       append([]domain.ConversationTraceEvent(nil), s.trace[conversationID]...),
	}
	if err := detail.Validate(); err != nil {
		return domain.ConversationDetail{}, err
	}
	return detail, nil
}

func (s *MemoryStore) skillExists(skillID string) bool {
	for _, skill := range s.skills {
		if skill.ID == skillID && skill.Enabled {
			return true
		}
	}
	return false
}

func defaultSkills() []domain.ConversationSkill {
	return []domain.ConversationSkill{
		{
			ID:          "fund_research",
			Name:        "Fund Research",
			Description: "Analyze fund holdings, market data, and decision options.",
			ToolNames:   []string{"account_overview", "fund_market_snapshot", "decision_journal"},
			Enabled:     true,
		},
		{
			ID:          "portfolio_review",
			Name:        "Portfolio Review",
			Description: "Review account-level holdings, returns, and follow-up tasks.",
			ToolNames:   []string{"account_overview", "review_task"},
			Enabled:     true,
		},
		{
			ID:          "document_intake",
			Name:        "Document Intake",
			Description: "Prepare uploaded screenshots, CSV, PDF, or strategy notes for parser tools.",
			ToolNames:   []string{"attachment_parser", "knowledge_draft"},
			Enabled:     true,
		},
	}
}

func traceEvent(conversationID, kind, status, summary string, metadata map[string]string) domain.ConversationTraceEvent {
	now := time.Now().UTC()
	return domain.ConversationTraceEvent{
		ID:             fmt.Sprintf("trace_%d", now.UnixNano()),
		ConversationID: conversationID,
		Kind:           kind,
		Status:         status,
		Summary:        summary,
		Metadata:       metadata,
		CreatedAt:      now,
	}
}

func attachmentExists(items []domain.ConversationAttachment, id string) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
}

func safeFileName(value string) string {
	value = filepath.Base(strings.TrimSpace(value))
	if value == "" || value == "." {
		return "upload.bin"
	}
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_")
	return replacer.Replace(value)
}

func supportedContentType(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return strings.HasPrefix(value, "image/") ||
		value == "application/pdf" ||
		value == "text/csv" ||
		value == "text/plain" ||
		value == "application/vnd.ms-excel"
}
