// This file implements bearer sessions and read-only consent evaluation.
// 本文件实现 Bearer 会话与只读同意授权评估。
package authorization

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

const (
	rawTokenBytes  = 32
	referenceBytes = 16
)

// Service issues sessions and evaluates revisioned consent grants.
// Service 签发会话并评估带版本的同意授权。
type Service struct {
	store   Store
	now     func() time.Time
	entropy io.Reader
}

// NewService creates an authorization service backed by the supplied truth store.
// NewService 使用给定的真相存储创建授权服务。
func NewService(store Store) *Service {
	return &Service{store: store, now: time.Now, entropy: rand.Reader}
}

// IssueSession creates a high-entropy bearer token and persists only its SHA-256 hash.
// IssueSession 创建高熵 Bearer token，并仅持久化其 SHA-256 哈希。
func (s *Service) IssueSession(ctx context.Context, subject string, ttl time.Duration) (BearerSession, error) {
	if err := s.ready(); err != nil {
		return BearerSession{}, err
	}
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return BearerSession{}, errors.New("session subject is required")
	}
	if ttl <= 0 {
		return BearerSession{}, errors.New("session ttl must be positive")
	}
	ref, err := s.randomValue("session", referenceBytes)
	if err != nil {
		return BearerSession{}, err
	}
	raw, err := s.randomValue("fs", rawTokenBytes)
	if err != nil {
		return BearerSession{}, err
	}
	now := s.now().UTC()
	session := Session{
		Ref:       ref,
		Subject:   subject,
		CreatedAt: now,
		ExpiresAt: now.Add(ttl),
	}
	record := SessionRecord{Session: session, TokenHash: sha256.Sum256([]byte(raw))}
	if err := s.store.SaveSession(ctx, record); err != nil {
		return BearerSession{}, fmt.Errorf("save authorization session: %w", err)
	}
	return BearerSession{Token: raw, Session: session}, nil
}

// Authenticate validates a raw bearer token, expiry, revocation, and optional expected subject.
// Authenticate 校验原始 Bearer token、过期、撤销与可选预期主体。
func (s *Service) Authenticate(ctx context.Context, rawToken, expectedSubject string) (Session, error) {
	if err := s.ready(); err != nil {
		return Session{}, err
	}
	if rawToken == "" {
		return Session{}, DenialSessionInvalid.error()
	}
	record, err := s.store.SessionByTokenHash(ctx, sha256.Sum256([]byte(rawToken)))
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return Session{}, DenialSessionInvalid.error()
		}
		return Session{}, fmt.Errorf("read authorization session: %w", err)
	}
	now := s.now().UTC()
	if record.RevokedAt != nil || !now.Before(record.ExpiresAt) {
		return record.Session, DenialSessionInvalid.error()
	}
	if expectedSubject != "" && record.Subject != expectedSubject {
		return record.Session, DenialSubjectMismatch.error()
	}
	return record.Session, nil
}

// RevokeSession invalidates a bearer session immediately in the truth store.
// RevokeSession 在真相存储中立即使 Bearer 会话失效。
func (s *Service) RevokeSession(ctx context.Context, sessionRef string) error {
	if err := s.ready(); err != nil {
		return err
	}
	if sessionRef == "" {
		return errors.New("session ref is required")
	}
	if err := s.store.RevokeSession(ctx, sessionRef, s.now().UTC()); err != nil {
		return fmt.Errorf("revoke authorization session: %w", err)
	}
	return nil
}

// CreateGrant creates revision one of a read-only consent grant.
// CreateGrant 创建只读同意授权的第一个版本。
func (s *Service) CreateGrant(ctx context.Context, request CreateGrantRequest) (ConsentGrant, error) {
	if err := s.ready(); err != nil {
		return ConsentGrant{}, err
	}
	request.Subject = strings.TrimSpace(request.Subject)
	request.Audience = strings.TrimSpace(request.Audience)
	if request.Subject == "" {
		return ConsentGrant{}, errors.New("grant subject is required")
	}
	if request.Audience == "" {
		return ConsentGrant{}, errors.New("grant audience is required")
	}
	scopes, err := normalizeScopes(request.Scopes)
	if err != nil {
		return ConsentGrant{}, err
	}
	now := s.now().UTC()
	if !now.Before(request.ExpiresAt) {
		return ConsentGrant{}, errors.New("grant expiry must be in the future")
	}
	ref, err := s.randomValue("grant", referenceBytes)
	if err != nil {
		return ConsentGrant{}, err
	}
	grant := ConsentGrant{
		Ref:       ref,
		Subject:   request.Subject,
		Audience:  request.Audience,
		Scopes:    scopes,
		Revision:  1,
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: request.ExpiresAt.UTC(),
	}
	if err := s.store.SaveGrant(ctx, grant); err != nil {
		return ConsentGrant{}, fmt.Errorf("save consent grant: %w", err)
	}
	return grant, nil
}

// RevokeGrant invalidates a grant and advances its revision exactly once.
// RevokeGrant 使授权失效，并且仅在首次撤销时推进版本。
func (s *Service) RevokeGrant(ctx context.Context, grantRef string) (ConsentGrant, error) {
	if err := s.ready(); err != nil {
		return ConsentGrant{}, err
	}
	if grantRef == "" {
		return ConsentGrant{}, errors.New("grant ref is required")
	}
	grant, err := s.store.RevokeGrant(ctx, grantRef, s.now().UTC())
	if err != nil {
		return ConsentGrant{}, fmt.Errorf("revoke consent grant: %w", err)
	}
	return grant, nil
}

// Grant returns one persisted consent grant by reference.
// Grant 按引用返回一条持久化同意授权。
func (s *Service) Grant(ctx context.Context, grantRef string) (ConsentGrant, error) {
	if err := s.ready(); err != nil {
		return ConsentGrant{}, err
	}
	grant, err := s.store.Grant(ctx, grantRef)
	if err != nil {
		return ConsentGrant{}, fmt.Errorf("read consent grant: %w", err)
	}
	return grant, nil
}

// GrantsBySubject returns grants newest first with reference as a stable tie-breaker.
// GrantsBySubject 按最新优先返回主体授权，并以引用作为稳定的同时间排序条件。
func (s *Service) GrantsBySubject(ctx context.Context, subject string) ([]ConsentGrant, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return nil, errors.New("grant subject is required")
	}
	grants, err := s.store.GrantsBySubject(ctx, subject)
	if err != nil {
		return nil, fmt.Errorf("list consent grants: %w", err)
	}
	return grants, nil
}

// ListGrants returns one subject's grants in stable newest-first order.
// ListGrants 以稳定的最新优先顺序返回一个主体的授权列表。
func (s *Service) ListGrants(ctx context.Context, subject string) ([]ConsentGrant, error) {
	return s.GrantsBySubject(ctx, subject)
}

// Authorize authenticates a session and checks subject, audience, grant state, and scope.
// Authorize 认证会话并检查主体、受众、授权状态与 scope。
func (s *Service) Authorize(ctx context.Context, request AuthorizationRequest) (AuthorizationDecision, error) {
	if err := s.ready(); err != nil {
		return AuthorizationDecision{}, err
	}
	if err := validateScope(request.Scope); err != nil {
		return AuthorizationDecision{}, err
	}
	if strings.TrimSpace(request.Audience) == "" {
		return AuthorizationDecision{}, errors.New("authorization audience is required")
	}

	session, err := s.Authenticate(ctx, request.Token, request.Subject)
	if err != nil {
		if code, ok := DenialCodeFromError(err); ok {
			return s.recordDecision(ctx, deniedDecision(code, session.Ref, request.GrantRef, request.Scope, 0))
		}
		return AuthorizationDecision{}, err
	}
	if request.GrantRef == "" {
		return s.recordDecision(ctx, deniedDecision(DenialMissingGrant, session.Ref, "", request.Scope, 0))
	}
	return s.authorizeGrant(ctx, session.Ref, GrantAuthorizationRequest{
		GrantRef: request.GrantRef,
		Subject:  session.Subject,
		Audience: request.Audience,
		Scope:    request.Scope,
	})
}

// AuthorizeGrant checks a grant for an authenticated service without accepting a user bearer token.
// AuthorizeGrant 为已认证服务检查授权，且不接受用户 Bearer token。
func (s *Service) AuthorizeGrant(ctx context.Context, request GrantAuthorizationRequest) (AuthorizationDecision, error) {
	if err := s.ready(); err != nil {
		return AuthorizationDecision{}, err
	}
	request.Subject = strings.TrimSpace(request.Subject)
	request.Audience = strings.TrimSpace(request.Audience)
	if request.Subject == "" {
		return AuthorizationDecision{}, errors.New("authorization subject is required")
	}
	if request.Audience == "" {
		return AuthorizationDecision{}, errors.New("authorization audience is required")
	}
	if err := validateScope(request.Scope); err != nil {
		return AuthorizationDecision{}, err
	}
	return s.authorizeGrant(ctx, "", request)
}

func (s *Service) authorizeGrant(ctx context.Context, sessionRef string, request GrantAuthorizationRequest) (AuthorizationDecision, error) {
	if request.GrantRef == "" {
		return s.recordDecision(ctx, deniedDecision(DenialMissingGrant, sessionRef, "", request.Scope, 0))
	}
	grant, err := s.store.Grant(ctx, request.GrantRef)
	if err != nil {
		if errors.Is(err, ErrGrantNotFound) {
			return s.recordDecision(ctx, deniedDecision(DenialMissingGrant, sessionRef, request.GrantRef, request.Scope, 0))
		}
		return AuthorizationDecision{}, fmt.Errorf("read consent grant: %w", err)
	}

	decision := AuthorizationDecision{
		Allowed:    true,
		SessionRef: sessionRef,
		GrantRef:   grant.Ref,
		Scope:      request.Scope,
		Revision:   grant.Revision,
	}
	now := s.now().UTC()
	switch {
	case grant.Subject != request.Subject:
		decision = deniedDecision(DenialSubjectMismatch, sessionRef, grant.Ref, request.Scope, grant.Revision)
	case grant.Audience != request.Audience:
		decision = deniedDecision(DenialAudienceMismatch, sessionRef, grant.Ref, request.Scope, grant.Revision)
	case grant.RevokedAt != nil:
		decision = deniedDecision(DenialGrantRevoked, sessionRef, grant.Ref, request.Scope, grant.Revision)
	case !now.Before(grant.ExpiresAt):
		decision = deniedDecision(DenialGrantExpired, sessionRef, grant.Ref, request.Scope, grant.Revision)
	case !containsScope(grant.Scopes, request.Scope):
		decision = deniedDecision(DenialMissingScope, sessionRef, grant.Ref, request.Scope, grant.Revision)
	}
	return s.recordDecision(ctx, decision)
}

func (s *Service) recordDecision(ctx context.Context, decision AuthorizationDecision) (AuthorizationDecision, error) {
	auditDecision := AuditDecisionDeny
	if decision.Allowed {
		auditDecision = AuditDecisionAllow
	}
	event := AuditEvent{
		SessionRef: decision.SessionRef,
		GrantRef:   decision.GrantRef,
		Scope:      decision.Scope,
		Decision:   auditDecision,
		Revision:   decision.Revision,
	}
	if err := s.store.AppendAuditEvent(ctx, event); err != nil {
		return decision, fmt.Errorf("append authorization audit event: %w", err)
	}
	return decision, nil
}

func (s *Service) ready() error {
	if s == nil || s.store == nil {
		return errors.New("authorization store is required")
	}
	if s.now == nil || s.entropy == nil {
		return errors.New("authorization service dependencies are required")
	}
	return nil
}

func (s *Service) randomValue(prefix string, size int) (string, error) {
	value := make([]byte, size)
	if _, err := io.ReadFull(s.entropy, value); err != nil {
		return "", fmt.Errorf("generate %s value: %w", prefix, err)
	}
	return prefix + "_" + base64.RawURLEncoding.EncodeToString(value), nil
}

func normalizeScopes(scopes []Scope) ([]Scope, error) {
	if len(scopes) == 0 {
		return nil, errors.New("at least one grant scope is required")
	}
	unique := make(map[Scope]struct{}, len(scopes))
	for _, scope := range scopes {
		if err := validateScope(scope); err != nil {
			return nil, err
		}
		unique[scope] = struct{}{}
	}
	normalized := make([]Scope, 0, len(unique))
	for scope := range unique {
		normalized = append(normalized, scope)
	}
	sort.Slice(normalized, func(i, j int) bool { return normalized[i] < normalized[j] })
	return normalized, nil
}

func validateScope(scope Scope) error {
	value := string(scope)
	if value == "" {
		return errors.New("authorization scope is required")
	}
	if len(value) > 128 || strings.TrimSpace(value) != value || strings.ContainsAny(value, " \t\r\n") {
		return fmt.Errorf("invalid authorization scope %q", scope)
	}
	if !strings.HasSuffix(value, ".read") {
		return fmt.Errorf("authorization scope %q is not read-only", scope)
	}
	return nil
}

func containsScope(scopes []Scope, requested Scope) bool {
	for _, scope := range scopes {
		if scope == requested {
			return true
		}
	}
	return false
}

func deniedDecision(code DenialCode, sessionRef, grantRef string, scope Scope, revision int64) AuthorizationDecision {
	return AuthorizationDecision{
		Code:       code,
		SessionRef: sessionRef,
		GrantRef:   grantRef,
		Scope:      scope,
		Revision:   revision,
	}
}
