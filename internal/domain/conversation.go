package domain

import (
	"errors"
	"fmt"
	"time"
)

// ConversationSkill describes one selectable Agent workspace skill.
// ConversationSkill 描述 Agent 工作台中可选择的一个 skill。
type ConversationSkill struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	ToolNames   []string `json:"tool_names"`
	Enabled     bool     `json:"enabled"`
}

// ConversationSession stores the durable conversation header for one user.
// ConversationSession 保存单个用户的一条持久对话头信息。
type ConversationSession struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	SkillID   string    `json:"skill_id"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Validate checks the minimal conversation fields required for workspace display.
// Validate 检查工作台展示所需的最小对话字段。
func (s ConversationSession) Validate() error {
	if s.ID == "" || s.UserID == "" {
		return errors.New("conversation id and user_id are required")
	}
	if s.SkillID == "" {
		return errors.New("conversation skill_id is required")
	}
	if s.Status == "" {
		return errors.New("conversation status is required")
	}
	return nil
}

// ConversationAttachment stores local upload metadata without parsed facts.
// ConversationAttachment 保存本地上传元数据，但不把附件冒充为已解析事实。
type ConversationAttachment struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	UserID         string    `json:"user_id"`
	FileName       string    `json:"file_name"`
	ContentType    string    `json:"content_type"`
	SizeBytes      int64     `json:"size_bytes"`
	LocalPath      string    `json:"local_path,omitempty"`
	Status         string    `json:"status"`
	Unsupported    bool      `json:"unsupported"`
	SHA256         string    `json:"sha256"`
	CreatedAt      time.Time `json:"created_at"`
	RetentionUntil time.Time `json:"retention_until"`
}

// Validate checks attachment metadata before it can be referenced by a message.
// Validate 在消息引用附件前检查附件元数据。
func (a ConversationAttachment) Validate() error {
	if a.ID == "" || a.ConversationID == "" || a.UserID == "" {
		return errors.New("attachment id, conversation_id, and user_id are required")
	}
	if a.FileName == "" {
		return errors.New("attachment file_name is required")
	}
	if a.SizeBytes <= 0 {
		return fmt.Errorf("attachment %s size must be positive", a.ID)
	}
	if a.Status == "" {
		return fmt.Errorf("attachment %s status is required", a.ID)
	}
	if a.SHA256 == "" {
		return fmt.Errorf("attachment %s sha256 is required", a.ID)
	}
	return nil
}

// ConversationMessage stores a user, assistant, tool, or system message.
// ConversationMessage 保存用户、助手、工具或系统消息。
type ConversationMessage struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	Role           string    `json:"role"`
	Content        string    `json:"content"`
	SkillID        string    `json:"skill_id"`
	AttachmentIDs  []string  `json:"attachment_ids,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// Validate checks the minimal message fields required by the workspace.
// Validate 检查工作台所需的最小消息字段。
func (m ConversationMessage) Validate() error {
	if m.ID == "" || m.ConversationID == "" {
		return errors.New("message id and conversation_id are required")
	}
	if m.Role == "" {
		return errors.New("message role is required")
	}
	if m.Content == "" {
		return errors.New("message content is required")
	}
	return nil
}

// ConversationTraceEvent is a safe timeline event for local or Athena actions.
// ConversationTraceEvent 是本地动作或 Athena 动作的安全时间线事件。
type ConversationTraceEvent struct {
	ID             string            `json:"id"`
	ConversationID string            `json:"conversation_id"`
	Kind           string            `json:"kind"`
	Status         string            `json:"status"`
	Summary        string            `json:"summary"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
}

// ConversationDetail is the read model returned to the workspace.
// ConversationDetail 是返回给工作台的对话读模型。
type ConversationDetail struct {
	Session     ConversationSession      `json:"session"`
	Messages    []ConversationMessage    `json:"messages"`
	Attachments []ConversationAttachment `json:"attachments"`
	Trace       []ConversationTraceEvent `json:"trace"`
}

// Validate checks a conversation detail before API response.
// Validate 在 API 返回前检查对话详情。
func (d ConversationDetail) Validate() error {
	if err := d.Session.Validate(); err != nil {
		return err
	}
	for _, attachment := range d.Attachments {
		if err := attachment.Validate(); err != nil {
			return err
		}
	}
	for _, message := range d.Messages {
		if err := message.Validate(); err != nil {
			return err
		}
	}
	return nil
}
