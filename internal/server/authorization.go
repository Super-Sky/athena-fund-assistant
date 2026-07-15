// This file maps bearer sessions and read-only consent into HTTP boundaries.
// 本文件将 Bearer 会话与只读同意授权映射到 HTTP 边界。
package server

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Super-Sky/athena-fund-assistant/internal/authorization"
)

const (
	defaultSessionTTL = 24 * time.Hour
	maximumSessionTTL = 7 * 24 * time.Hour
	defaultGrantTTL   = 30 * 24 * time.Hour
	maximumGrantTTL   = 90 * 24 * time.Hour
	athenaAudience    = "athena-runtime"
)

// AuthorizationService is the HTTP-facing subset of the authorization core.
// AuthorizationService 是 HTTP 层使用的授权核心能力子集。
type AuthorizationService interface {
	IssueSession(context.Context, string, time.Duration) (authorization.BearerSession, error)
	Authenticate(context.Context, string, string) (authorization.Session, error)
	RevokeSession(context.Context, string) error
	CreateGrant(context.Context, authorization.CreateGrantRequest) (authorization.ConsentGrant, error)
	ListGrants(context.Context, string) ([]authorization.ConsentGrant, error)
	Grant(context.Context, string) (authorization.ConsentGrant, error)
	RevokeGrant(context.Context, string) (authorization.ConsentGrant, error)
	AuthorizeGrant(context.Context, authorization.GrantAuthorizationRequest) (authorization.AuthorizationDecision, error)
}

type createSessionRequest struct {
	UserID     string `json:"user_id"`
	TTLSeconds int64  `json:"ttl_seconds,omitempty"`
}

type createGrantRequest struct {
	Audience   string                `json:"audience"`
	Scopes     []authorization.Scope `json:"scopes"`
	TTLSeconds int64                 `json:"ttl_seconds,omitempty"`
}

type consentListResponse struct {
	Items []authorization.ConsentGrant `json:"items"`
}

func (s *Server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	if s.deps.Authorization == nil {
		writeAuthorizationError(w, http.StatusServiceUnavailable, authorization.DenialSessionInvalid, "authorization service is not configured")
		return
	}
	var req createSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAuthorizationError(w, http.StatusBadRequest, authorization.DenialSessionInvalid, "invalid session request")
		return
	}
	subject := strings.TrimSpace(req.UserID)
	localSubject := strings.TrimSpace(s.deps.LocalAuthSubject)
	if localSubject == "" || subject != localSubject {
		writeAuthorizationError(w, http.StatusForbidden, authorization.DenialSubjectMismatch, "local session issuance is disabled for this subject")
		return
	}
	ttl, ok := boundedTTL(req.TTLSeconds, defaultSessionTTL, maximumSessionTTL)
	if !ok {
		writeAuthorizationError(w, http.StatusBadRequest, authorization.DenialSessionInvalid, "session ttl is outside the allowed range")
		return
	}
	session, err := s.deps.Authorization.IssueSession(r.Context(), subject, ttl)
	if err != nil {
		writeAuthorizationError(w, http.StatusBadRequest, authorization.DenialSessionInvalid, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, session)
}

func (s *Server) handleCurrentSession(w http.ResponseWriter, r *http.Request) {
	if s.deps.Authorization == nil {
		writeAuthorizationError(w, http.StatusServiceUnavailable, authorization.DenialSessionInvalid, "authorization service is not configured")
		return
	}
	session, ok := s.requireSession(w, r, "")
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, session)
}

func (s *Server) handleRevokeCurrentSession(w http.ResponseWriter, r *http.Request) {
	if s.deps.Authorization == nil {
		writeAuthorizationError(w, http.StatusServiceUnavailable, authorization.DenialSessionInvalid, "authorization service is not configured")
		return
	}
	session, ok := s.requireSession(w, r, "")
	if !ok {
		return
	}
	if err := s.deps.Authorization.RevokeSession(r.Context(), session.Ref); err != nil {
		writeAuthorizationError(w, http.StatusInternalServerError, authorization.DenialSessionInvalid, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListConsents(w http.ResponseWriter, r *http.Request) {
	if s.deps.Authorization == nil {
		writeAuthorizationError(w, http.StatusServiceUnavailable, authorization.DenialMissingGrant, "authorization service is not configured")
		return
	}
	session, ok := s.requireSession(w, r, "")
	if !ok {
		return
	}
	grants, err := s.deps.Authorization.ListGrants(r.Context(), session.Subject)
	if err != nil {
		writeAuthorizationError(w, http.StatusInternalServerError, authorization.DenialMissingGrant, err.Error())
		return
	}
	if grants == nil {
		grants = []authorization.ConsentGrant{}
	}
	writeJSON(w, http.StatusOK, consentListResponse{Items: grants})
}

func (s *Server) handleCreateConsent(w http.ResponseWriter, r *http.Request) {
	if s.deps.Authorization == nil {
		writeAuthorizationError(w, http.StatusServiceUnavailable, authorization.DenialMissingGrant, "authorization service is not configured")
		return
	}
	session, ok := s.requireSession(w, r, "")
	if !ok {
		return
	}
	var req createGrantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAuthorizationError(w, http.StatusBadRequest, authorization.DenialMissingGrant, "invalid consent request")
		return
	}
	audience := strings.TrimSpace(req.Audience)
	if audience == "" {
		audience = athenaAudience
	}
	if audience != athenaAudience {
		writeAuthorizationError(w, http.StatusBadRequest, authorization.DenialAudienceMismatch, "unsupported consent audience")
		return
	}
	ttl, ok := boundedTTL(req.TTLSeconds, defaultGrantTTL, maximumGrantTTL)
	if !ok {
		writeAuthorizationError(w, http.StatusBadRequest, authorization.DenialGrantExpired, "consent ttl is outside the allowed range")
		return
	}
	grant, err := s.deps.Authorization.CreateGrant(r.Context(), authorization.CreateGrantRequest{
		Subject:   session.Subject,
		Audience:  audience,
		Scopes:    req.Scopes,
		ExpiresAt: time.Now().UTC().Add(ttl),
	})
	if err != nil {
		writeAuthorizationError(w, http.StatusBadRequest, authorization.DenialMissingGrant, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, grant)
}

func (s *Server) handleRevokeConsent(w http.ResponseWriter, r *http.Request) {
	if s.deps.Authorization == nil {
		writeAuthorizationError(w, http.StatusServiceUnavailable, authorization.DenialMissingGrant, "authorization service is not configured")
		return
	}
	session, ok := s.requireSession(w, r, "")
	if !ok {
		return
	}
	grantRef := strings.TrimSpace(r.PathValue("grant_ref"))
	grant, err := s.deps.Authorization.Grant(r.Context(), grantRef)
	if err != nil {
		writeAuthorizationError(w, http.StatusNotFound, authorization.DenialMissingGrant, "consent grant not found")
		return
	}
	if grant.Subject != session.Subject {
		writeAuthorizationError(w, http.StatusForbidden, authorization.DenialSubjectMismatch, "consent grant belongs to another subject")
		return
	}
	grant, err = s.deps.Authorization.RevokeGrant(r.Context(), grantRef)
	if err != nil {
		writeAuthorizationError(w, http.StatusInternalServerError, authorization.DenialMissingGrant, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, grant)
}

func (s *Server) requireSession(w http.ResponseWriter, r *http.Request, expectedSubject string) (authorization.Session, bool) {
	if s.deps.Authorization == nil {
		writeAuthorizationError(w, http.StatusServiceUnavailable, authorization.DenialSessionInvalid, "authorization service is not configured")
		return authorization.Session{}, false
	}
	token := bearerToken(r.Header.Get("Authorization"))
	session, err := s.deps.Authorization.Authenticate(r.Context(), token, strings.TrimSpace(expectedSubject))
	if err != nil {
		code := authorization.DenialSessionInvalid
		if value, found := authorization.DenialCodeFromError(err); found {
			code = value
		}
		status := http.StatusUnauthorized
		if code == authorization.DenialSubjectMismatch {
			status = http.StatusForbidden
		}
		writeAuthorizationError(w, status, code, "authorization denied")
		return authorization.Session{}, false
	}
	return session, true
}

func (s *Server) requireRemoteService(w http.ResponseWriter, r *http.Request, req remoteToolExecutionRequest) bool {
	if s.deps.Authorization == nil {
		writeRemoteToolError(w, http.StatusServiceUnavailable, req, "service_auth_unconfigured", "remote tool authorization is not configured", false)
		return false
	}
	expected := strings.TrimSpace(s.deps.RemoteToolToken)
	provided := bearerToken(r.Header.Get("Authorization"))
	if expected == "" {
		writeRemoteToolError(w, http.StatusServiceUnavailable, req, "service_auth_unconfigured", "remote tool service authentication is not configured", false)
		return false
	}
	expectedHash := sha256.Sum256([]byte(expected))
	providedHash := sha256.Sum256([]byte(provided))
	if subtle.ConstantTimeCompare(providedHash[:], expectedHash[:]) != 1 {
		writeRemoteToolError(w, http.StatusUnauthorized, req, "service_auth_denied", "remote tool service authentication failed", false)
		return false
	}
	return true
}

func bearerToken(value string) string {
	parts := strings.Fields(strings.TrimSpace(value))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}

func boundedTTL(seconds int64, fallback, maximum time.Duration) (time.Duration, bool) {
	if seconds == 0 {
		return fallback, true
	}
	if seconds < 0 || seconds > int64(maximum/time.Second) {
		return 0, false
	}
	ttl := time.Duration(seconds) * time.Second
	return ttl, true
}

func writeAuthorizationError(w http.ResponseWriter, status int, code authorization.DenialCode, message string) {
	writeJSON(w, status, map[string]string{
		"code":  string(code),
		"error": message,
	})
}

func authorizationDenied(err error) (authorization.DenialCode, bool) {
	if err == nil {
		return "", false
	}
	code, ok := authorization.DenialCodeFromError(err)
	return code, ok || errors.Is(err, authorization.ErrGrantNotFound)
}
