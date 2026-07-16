// This file defines the read-only authorization and consent contract.
// 本文件定义只读授权与同意契约。
package authorization

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"
)

// Scope identifies one read-only capability protected by a consent grant.
// Scope 标识一项由同意授权保护的只读能力。
type Scope string

const (
	// ScopeAccountSummaryRead permits reading an account summary.
	// ScopeAccountSummaryRead 允许读取账户摘要。
	ScopeAccountSummaryRead Scope = "fund.account.summary.read"
	// ScopeHoldingSnapshotRead permits reading holding snapshots.
	// ScopeHoldingSnapshotRead 允许读取持仓快照。
	ScopeHoldingSnapshotRead Scope = "fund.holding.snapshot.read"
	// ScopeDecisionJournalRead permits reading decision journals.
	// ScopeDecisionJournalRead 允许读取决策日志。
	ScopeDecisionJournalRead Scope = "fund.decision_journal.read"
	// ScopeProviderSyncRead permits read-only provider synchronization.
	// ScopeProviderSyncRead 允许只读数据提供方同步。
	ScopeProviderSyncRead Scope = "fund.provider.sync.read"
	// ScopeBrokerSyncRead permits read-only broker synchronization.
	// ScopeBrokerSyncRead 允许只读券商同步。
	ScopeBrokerSyncRead Scope = "fund.broker.sync.read"
)

// DenialCode is a stable machine-readable authorization refusal code.
// DenialCode 是稳定且机器可读的授权拒绝码。
type DenialCode string

const (
	// DenialMissingScope indicates that the grant lacks the requested scope.
	// DenialMissingScope 表示授权缺少请求的 scope。
	DenialMissingScope DenialCode = "missing_scope"
	// DenialGrantExpired indicates that the consent grant has expired.
	// DenialGrantExpired 表示同意授权已过期。
	DenialGrantExpired DenialCode = "grant_expired"
	// DenialGrantRevoked indicates that the consent grant has been revoked.
	// DenialGrantRevoked 表示同意授权已撤销。
	DenialGrantRevoked DenialCode = "grant_revoked"
	// DenialSubjectMismatch indicates that identities do not match.
	// DenialSubjectMismatch 表示身份主体不匹配。
	DenialSubjectMismatch DenialCode = "subject_mismatch"
	// DenialMissingGrant indicates that the requested grant does not exist.
	// DenialMissingGrant 表示请求的授权不存在。
	DenialMissingGrant DenialCode = "missing_grant"
	// DenialAudienceMismatch indicates that the grant targets another service.
	// DenialAudienceMismatch 表示授权面向其他服务。
	DenialAudienceMismatch DenialCode = "audience_mismatch"
	// DenialSessionInvalid indicates that the bearer session is unusable.
	// DenialSessionInvalid 表示 Bearer 会话不可用。
	DenialSessionInvalid DenialCode = "session_invalid"
)

// AuditDecision is the minimal allow-or-deny value stored in an audit event.
// AuditDecision 是审计事件中保存的最小允许或拒绝值。
type AuditDecision string

const (
	// AuditDecisionAllow records a successful authorization decision.
	// AuditDecisionAllow 记录一次授权成功决定。
	AuditDecisionAllow AuditDecision = "allow"
	// AuditDecisionDeny records a refused authorization decision.
	// AuditDecisionDeny 记录一次授权拒绝决定。
	AuditDecisionDeny AuditDecision = "deny"
)

var (
	// ErrSessionNotFound indicates that no persisted session matches a reference or hash.
	// ErrSessionNotFound 表示没有持久化会话匹配给定引用或哈希。
	ErrSessionNotFound = errors.New("authorization session not found")
	// ErrGrantNotFound indicates that no persisted consent grant matches a reference.
	// ErrGrantNotFound 表示没有持久化同意授权匹配给定引用。
	ErrGrantNotFound = errors.New("consent grant not found")
)

// Session describes bearer-session metadata without the raw bearer token.
// Session 描述不包含原始 Bearer token 的会话元数据。
type Session struct {
	Ref       string     `json:"ref"`
	Subject   string     `json:"subject"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt time.Time  `json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

// SessionRecord is the persistence form of a session and contains only a SHA-256 token hash.
// SessionRecord 是会话的持久化形式，只包含 SHA-256 token 哈希。
type SessionRecord struct {
	Session
	TokenHash [sha256.Size]byte `json:"-"`
}

// BearerSession contains the one-time raw token returned when a session is issued.
// BearerSession 包含签发会话时一次性返回的原始 token。
type BearerSession struct {
	Token   string  `json:"token"`
	Session Session `json:"session"`
}

// ConsentGrant binds a subject and audience to a revisioned set of read-only scopes.
// ConsentGrant 将主体和受众绑定到一组带版本的只读 scope。
type ConsentGrant struct {
	Ref       string     `json:"ref"`
	Subject   string     `json:"subject"`
	Audience  string     `json:"audience"`
	Scopes    []Scope    `json:"scopes"`
	Revision  int64      `json:"revision"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	ExpiresAt time.Time  `json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

// CreateGrantRequest supplies the identity, audience, scopes, and expiry for a new grant.
// CreateGrantRequest 提供新授权的身份、受众、scope 与过期时间。
type CreateGrantRequest struct {
	Subject   string    `json:"subject"`
	Audience  string    `json:"audience"`
	Scopes    []Scope   `json:"scopes"`
	ExpiresAt time.Time `json:"expires_at"`
}

// AuthorizationRequest describes one bearer-backed consent check.
// AuthorizationRequest 描述一次由 Bearer 会话支持的同意授权检查。
type AuthorizationRequest struct {
	Token    string `json:"-"`
	GrantRef string `json:"grant_ref"`
	Subject  string `json:"subject"`
	Audience string `json:"audience"`
	Scope    Scope  `json:"scope"`
}

// GrantAuthorizationRequest describes a service-to-service consent check without a bearer token.
// GrantAuthorizationRequest 描述不携带 Bearer token 的服务间同意授权检查。
type GrantAuthorizationRequest struct {
	GrantRef string `json:"grant_ref"`
	Subject  string `json:"subject"`
	Audience string `json:"audience"`
	Scope    Scope  `json:"scope"`
}

// AuthorizationDecision reports the result and only audit-safe references.
// AuthorizationDecision 返回鉴权结果及可安全审计的引用。
type AuthorizationDecision struct {
	Allowed    bool       `json:"allowed"`
	Code       DenialCode `json:"code,omitempty"`
	SessionRef string     `json:"session_ref,omitempty"`
	GrantRef   string     `json:"grant_ref,omitempty"`
	Scope      Scope      `json:"scope"`
	Revision   int64      `json:"revision"`
}

// AuditEvent contains only references, scope, decision, and grant revision.
// AuditEvent 仅包含引用、scope、决定与授权版本。
type AuditEvent struct {
	SessionRef string        `json:"session_ref,omitempty"`
	GrantRef   string        `json:"grant_ref,omitempty"`
	Scope      Scope         `json:"scope"`
	Decision   AuditDecision `json:"decision"`
	Revision   int64         `json:"revision"`
}

// DenialError carries a stable refusal code for authentication failures.
// DenialError 为认证失败携带稳定拒绝码。
type DenialError struct {
	Code DenialCode
}

// Error returns the stable refusal code as an error string.
// Error 将稳定拒绝码作为错误字符串返回。
func (e *DenialError) Error() string {
	if e == nil {
		return ""
	}
	return string(e.Code)
}

// DenialCodeFromError extracts a stable refusal code from an error.
// DenialCodeFromError 从错误中提取稳定拒绝码。
func DenialCodeFromError(err error) (DenialCode, bool) {
	var denial *DenialError
	if !errors.As(err, &denial) {
		return "", false
	}
	return denial.Code, true
}

// Store persists session hashes, grants, revocations, and minimal audit events.
// Store 持久化会话哈希、授权、撤销状态与最小审计事件。
type Store interface {
	SaveSession(context.Context, SessionRecord) error
	SessionByTokenHash(context.Context, [sha256.Size]byte) (SessionRecord, error)
	RevokeSession(context.Context, string, time.Time) error
	SaveGrant(context.Context, ConsentGrant) error
	Grant(context.Context, string) (ConsentGrant, error)
	GrantsBySubject(context.Context, string) ([]ConsentGrant, error)
	RevokeGrant(context.Context, string, time.Time) (ConsentGrant, error)
	AppendAuditEvent(context.Context, AuditEvent) error
	AuditEvents(context.Context) ([]AuditEvent, error)
}

func (c DenialCode) error() error { return &DenialError{Code: c} }

func validateAuditEvent(event AuditEvent) error {
	if event.Scope == "" {
		return errors.New("audit scope is required")
	}
	if event.Decision != AuditDecisionAllow && event.Decision != AuditDecisionDeny {
		return fmt.Errorf("invalid audit decision %q", event.Decision)
	}
	if event.Revision < 0 {
		return errors.New("audit revision must not be negative")
	}
	return nil
}
