// This file verifies the HTTP and remote-tool read-only authorization boundary.
// 本文件验证 HTTP 与远程工具的只读授权边界。
package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Super-Sky/athena-fund-assistant/internal/account"
	"github.com/Super-Sky/athena-fund-assistant/internal/authorization"
	"github.com/Super-Sky/athena-fund-assistant/internal/data"
	"github.com/Super-Sky/athena-fund-assistant/internal/decision"
	"github.com/Super-Sky/athena-fund-assistant/internal/journal"
)

func TestAuthorizationSessionAndConsentLifecycle(t *testing.T) {
	store := authorization.NewMemoryStore()
	handler := newAuthorizedTestServer(store)
	session := issueTestSession(t, handler)

	current := performAuthorizedJSON(t, handler, http.MethodGet, "/api/auth/session", nil, session.Token)
	if current.Code != http.StatusOK || strings.Contains(current.Body.String(), session.Token) {
		t.Fatalf("current session status=%d body=%s", current.Code, current.Body.String())
	}

	crossUser := performAuthorizedJSON(t, handler, http.MethodGet, "/api/accounts/another-user/overview", nil, session.Token)
	assertAuthorizationCode(t, crossUser, http.StatusForbidden, authorization.DenialSubjectMismatch)

	grantRR := performAuthorizedJSON(t, handler, http.MethodPost, "/api/consents", createGrantRequest{
		Scopes: []authorization.Scope{
			authorization.ScopeAccountSummaryRead,
			authorization.ScopeHoldingSnapshotRead,
		},
	}, session.Token)
	if grantRR.Code != http.StatusCreated {
		t.Fatalf("create consent status=%d body=%s", grantRR.Code, grantRR.Body.String())
	}
	if strings.Contains(grantRR.Body.String(), session.Token) {
		t.Fatal("consent response leaked the raw session token")
	}
	var grant authorization.ConsentGrant
	decodeResponse(t, grantRR, &grant)
	if grant.Subject != "demo-user" || grant.Audience != athenaAudience || grant.Revision != 1 {
		t.Fatalf("unexpected grant = %#v", grant)
	}

	listRR := performAuthorizedJSON(t, handler, http.MethodGet, "/api/consents", nil, session.Token)
	if listRR.Code != http.StatusOK || strings.Contains(listRR.Body.String(), session.Token) {
		t.Fatalf("list consent status=%d body=%s", listRR.Code, listRR.Body.String())
	}
	var list consentListResponse
	decodeResponse(t, listRR, &list)
	if len(list.Items) != 1 || list.Items[0].Ref != grant.Ref {
		t.Fatalf("unexpected grant list = %#v", list.Items)
	}

	revokeRR := performAuthorizedJSON(t, handler, http.MethodPost, "/api/consents/"+grant.Ref+"/revoke", map[string]any{}, session.Token)
	if revokeRR.Code != http.StatusOK {
		t.Fatalf("revoke consent status=%d body=%s", revokeRR.Code, revokeRR.Body.String())
	}
	var revoked authorization.ConsentGrant
	decodeResponse(t, revokeRR, &revoked)
	if revoked.RevokedAt == nil || revoked.Revision != 2 {
		t.Fatalf("unexpected revoked grant = %#v", revoked)
	}

	revokeSession := performAuthorizedJSON(t, handler, http.MethodDelete, "/api/auth/sessions/current", nil, session.Token)
	if revokeSession.Code != http.StatusNoContent {
		t.Fatalf("revoke session status=%d body=%s", revokeSession.Code, revokeSession.Body.String())
	}
	assertAuthorizationCode(t,
		performAuthorizedJSON(t, handler, http.MethodGet, "/api/auth/session", nil, session.Token),
		http.StatusUnauthorized,
		authorization.DenialSessionInvalid,
	)
}

func TestProtectedRoutesFailClosedWithoutAuthorization(t *testing.T) {
	handler := New(Dependencies{Accounts: account.NewMemoryStore()}).Routes()
	accountRR := httptest.NewRecorder()
	handler.ServeHTTP(accountRR, httptest.NewRequest(http.MethodGet, "/api/accounts/demo-user/overview", nil))
	assertAuthorizationCode(t, accountRR, http.StatusServiceUnavailable, authorization.DenialSessionInvalid)

	remoteRR := performJSON(t, handler, http.MethodPost, "/internal/tools/execute", remoteToolExecutionRequest{
		ContractVersion: remoteToolContractVersion,
		ToolCallID:      "call_fail_closed",
		ToolName:        "account_overview",
		Arguments:       json.RawMessage(`{"user_id":"demo-user","consent_grant_ref":"grant_test"}`),
	})
	assertRemoteErrorCode(t, remoteRR, http.StatusServiceUnavailable, "service_auth_unconfigured")
}

func TestBoundedTTLRejectsOverflowAndOutOfRangeValues(t *testing.T) {
	for _, seconds := range []int64{-1, 1<<63 - 1, int64(maximumSessionTTL/time.Second) + 1} {
		if _, ok := boundedTTL(seconds, defaultSessionTTL, maximumSessionTTL); ok {
			t.Fatalf("boundedTTL(%d) unexpectedly accepted", seconds)
		}
	}
	if ttl, ok := boundedTTL(60, defaultSessionTTL, maximumSessionTTL); !ok || ttl != time.Minute {
		t.Fatalf("boundedTTL(60) = %s, %t", ttl, ok)
	}
}

func TestRemoteAccountToolRequiresServiceIdentityAndConsentScopes(t *testing.T) {
	store := authorization.NewMemoryStore()
	handler := newAuthorizedTestServer(store)
	session := issueTestSession(t, handler)

	summaryOnly := createTestGrant(t, handler, session.Token, []authorization.Scope{authorization.ScopeAccountSummaryRead})
	request := remoteToolExecutionRequest{
		ContractVersion: remoteToolContractVersion,
		RequestID:       "req_authorized_account",
		ToolCallID:      "call_authorized_account",
		RegistrationID:  "fund_account_overview_v1",
		AppID:           "athena-fund-assistant",
		ToolName:        "account_overview",
		Arguments: mustJSON(t, accountOverviewToolArgs{
			UserID:          "demo-user",
			ConsentGrantRef: summaryOnly.Ref,
		}),
		Attempt: 1,
	}

	missingService := performAuthorizedJSON(t, handler, http.MethodPost, "/internal/tools/execute", request, "")
	assertRemoteErrorCode(t, missingService, http.StatusUnauthorized, "service_auth_denied")
	wrongService := performAuthorizedJSON(t, handler, http.MethodPost, "/internal/tools/execute", request, "wrong-service-token")
	assertRemoteErrorCode(t, wrongService, http.StatusUnauthorized, "service_auth_denied")

	missingScope := performAuthorizedJSON(t, handler, http.MethodPost, "/internal/tools/execute", request, "test-service-token")
	assertRemoteAuthorizationDenial(t, missingScope, authorization.DenialMissingScope)

	fullGrant := createTestGrant(t, handler, session.Token, []authorization.Scope{
		authorization.ScopeAccountSummaryRead,
		authorization.ScopeHoldingSnapshotRead,
	})
	request.Arguments = mustJSON(t, accountOverviewToolArgs{
		UserID:          "demo-user",
		ConsentGrantRef: fullGrant.Ref,
	})
	allowed := performAuthorizedJSON(t, handler, http.MethodPost, "/internal/tools/execute", request, "test-service-token")
	if allowed.Code != http.StatusOK {
		t.Fatalf("authorized tool status=%d body=%s", allowed.Code, allowed.Body.String())
	}
	if strings.Contains(allowed.Body.String(), session.Token) || strings.Contains(allowed.Body.String(), "test-service-token") {
		t.Fatal("remote tool response leaked a raw credential")
	}

	revokeRR := performAuthorizedJSON(t, handler, http.MethodPost, "/api/consents/"+fullGrant.Ref+"/revoke", map[string]any{}, session.Token)
	if revokeRR.Code != http.StatusOK {
		t.Fatalf("revoke consent status=%d body=%s", revokeRR.Code, revokeRR.Body.String())
	}
	revoked := performAuthorizedJSON(t, handler, http.MethodPost, "/internal/tools/execute", request, "test-service-token")
	assertRemoteAuthorizationDenial(t, revoked, authorization.DenialGrantRevoked)

	events, err := store.AuditEvents(context.Background())
	if err != nil {
		t.Fatalf("read audit events: %v", err)
	}
	encoded, err := json.Marshal(events)
	if err != nil {
		t.Fatalf("marshal audit events: %v", err)
	}
	if len(events) < 4 || bytes.Contains(encoded, []byte(session.Token)) || bytes.Contains(encoded, []byte("test-service-token")) {
		t.Fatalf("unsafe or incomplete audit events = %s", encoded)
	}
}

func newAuthorizedTestServer(store *authorization.MemoryStore) http.Handler {
	return New(Dependencies{
		Provider:         data.NewMockProvider(),
		DecisionMaker:    decision.NewEngine(),
		Journals:         journal.NewMemoryStore(),
		Accounts:         account.NewMemoryStore(),
		Authorization:    authorization.NewService(store),
		LocalAuthSubject: "demo-user",
		RemoteToolToken:  "test-service-token",
	}).Routes()
}

func issueTestSession(t *testing.T, handler http.Handler) authorization.BearerSession {
	t.Helper()
	rr := performJSON(t, handler, http.MethodPost, "/api/auth/sessions", createSessionRequest{UserID: "demo-user"})
	if rr.Code != http.StatusCreated {
		t.Fatalf("issue session status=%d body=%s", rr.Code, rr.Body.String())
	}
	var session authorization.BearerSession
	decodeResponse(t, rr, &session)
	if session.Token == "" || session.Session.Subject != "demo-user" {
		t.Fatalf("unexpected bearer session = %#v", session)
	}
	return session
}

func createTestGrant(t *testing.T, handler http.Handler, token string, scopes []authorization.Scope) authorization.ConsentGrant {
	t.Helper()
	rr := performAuthorizedJSON(t, handler, http.MethodPost, "/api/consents", createGrantRequest{Scopes: scopes}, token)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create test grant status=%d body=%s", rr.Code, rr.Body.String())
	}
	var grant authorization.ConsentGrant
	decodeResponse(t, rr, &grant)
	return grant
}

func performAuthorizedJSON(t *testing.T, handler http.Handler, method, path string, body any, token string) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

func decodeResponse(t *testing.T, rr *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.Unmarshal(rr.Body.Bytes(), target); err != nil {
		t.Fatalf("decode response: %v body=%s", err, rr.Body.String())
	}
}

func mustJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	return payload
}

func assertAuthorizationCode(t *testing.T, rr *httptest.ResponseRecorder, status int, code authorization.DenialCode) {
	t.Helper()
	if rr.Code != status {
		t.Fatalf("authorization status=%d want=%d body=%s", rr.Code, status, rr.Body.String())
	}
	var payload map[string]string
	decodeResponse(t, rr, &payload)
	if payload["code"] != string(code) {
		t.Fatalf("authorization code=%q want=%q body=%s", payload["code"], code, rr.Body.String())
	}
}

func assertRemoteErrorCode(t *testing.T, rr *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if rr.Code != status {
		t.Fatalf("remote tool status=%d want=%d body=%s", rr.Code, status, rr.Body.String())
	}
	var payload remoteToolExecutionResponse
	decodeResponse(t, rr, &payload)
	if payload.Error == nil || payload.Error.Code != code {
		t.Fatalf("remote tool error=%#v want=%q", payload.Error, code)
	}
}

func assertRemoteAuthorizationDenial(t *testing.T, rr *httptest.ResponseRecorder, code authorization.DenialCode) {
	t.Helper()
	assertRemoteErrorCode(t, rr, http.StatusForbidden, "authorization_denied")
	var payload remoteToolExecutionResponse
	decodeResponse(t, rr, &payload)
	if payload.Metadata["authorization_code"] != string(code) {
		t.Fatalf("authorization denial=%v want=%q body=%s", payload.Metadata["authorization_code"], code, rr.Body.String())
	}
}
