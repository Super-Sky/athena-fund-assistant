// This file verifies authorization decisions, revocation, and secret minimization.
// 本文件验证授权决定、撤销与敏感信息最小化。
package authorization

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestCoreReadScopes(t *testing.T) {
	want := []Scope{
		"fund.account.summary.read",
		"fund.holding.snapshot.read",
		"fund.decision_journal.read",
		"fund.provider.sync.read",
		"fund.broker.sync.read",
	}
	got := []Scope{
		ScopeAccountSummaryRead,
		ScopeHoldingSnapshotRead,
		ScopeDecisionJournalRead,
		ScopeProviderSyncRead,
		ScopeBrokerSyncRead,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("core scopes = %#v, want %#v", got, want)
	}
}

func TestIssueSessionPersistsOnlySHA256Hash(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC)
	store := NewMemoryStore()
	service := NewService(store)
	service.now = func() time.Time { return now }

	bearer, err := service.IssueSession(ctx, "user-1", time.Hour)
	if err != nil {
		t.Fatalf("IssueSession() error = %v", err)
	}
	if strings.Contains(bearer.Token, "user-1") {
		t.Fatalf("raw token contains subject: %q", bearer.Token)
	}
	parts := strings.SplitN(bearer.Token, "_", 2)
	if len(parts) != 2 {
		t.Fatalf("raw token format = %q", bearer.Token)
	}
	randomBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode raw token: %v", err)
	}
	if len(randomBytes) != rawTokenBytes {
		t.Fatalf("raw token entropy bytes = %d, want %d", len(randomBytes), rawTokenBytes)
	}

	store.mu.RLock()
	record, ok := store.sessionsByRef[bearer.Session.Ref]
	store.mu.RUnlock()
	if !ok {
		t.Fatal("session was not persisted")
	}
	wantHash := sha256.Sum256([]byte(bearer.Token))
	if record.TokenHash != wantHash {
		t.Fatalf("persisted hash = %x, want %x", record.TokenHash, wantHash)
	}
	if strings.Contains(record.Ref, bearer.Token) || strings.Contains(record.Subject, bearer.Token) {
		t.Fatal("persisted session metadata contains the raw token")
	}

	authenticated, err := service.Authenticate(ctx, bearer.Token, "user-1")
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if authenticated.Ref != bearer.Session.Ref {
		t.Fatalf("authenticated ref = %q, want %q", authenticated.Ref, bearer.Session.Ref)
	}
}

func TestAuthenticateRejectsInvalidSessionStates(t *testing.T) {
	t.Run("subject mismatch", func(t *testing.T) {
		service, _, bearer, _ := authorizationFixture(t, "user-1", "user-1", "fund-api", []Scope{ScopeAccountSummaryRead})
		_, err := service.Authenticate(context.Background(), bearer.Token, "other-user")
		assertDenialError(t, err, DenialSubjectMismatch)
	})

	t.Run("expired", func(t *testing.T) {
		service, now, bearer, _ := authorizationFixture(t, "user-1", "user-1", "fund-api", []Scope{ScopeAccountSummaryRead})
		*now = now.Add(2 * time.Hour)
		_, err := service.Authenticate(context.Background(), bearer.Token, "user-1")
		assertDenialError(t, err, DenialSessionInvalid)
	})

	t.Run("revoked", func(t *testing.T) {
		service, _, bearer, _ := authorizationFixture(t, "user-1", "user-1", "fund-api", []Scope{ScopeAccountSummaryRead})
		if err := service.RevokeSession(context.Background(), bearer.Session.Ref); err != nil {
			t.Fatalf("RevokeSession() error = %v", err)
		}
		_, err := service.Authenticate(context.Background(), bearer.Token, "user-1")
		assertDenialError(t, err, DenialSessionInvalid)
	})

	t.Run("unknown token", func(t *testing.T) {
		service, _, _, _ := authorizationFixture(t, "user-1", "user-1", "fund-api", []Scope{ScopeAccountSummaryRead})
		_, err := service.Authenticate(context.Background(), "fs_not-a-real-token", "user-1")
		assertDenialError(t, err, DenialSessionInvalid)
	})
}

func TestAuthorizeConsentStates(t *testing.T) {
	tests := []struct {
		name            string
		sessionUser     string
		grantUser       string
		grantAudience   string
		grantScopes     []Scope
		requestScope    Scope
		requestAudience string
		prepare         func(*testing.T, *Service, *time.Time, BearerSession, ConsentGrant)
		wantAllowed     bool
		wantCode        DenialCode
	}{
		{
			name:            "active",
			sessionUser:     "user-1",
			grantUser:       "user-1",
			grantAudience:   "fund-api",
			grantScopes:     []Scope{ScopeAccountSummaryRead},
			requestScope:    ScopeAccountSummaryRead,
			requestAudience: "fund-api",
			wantAllowed:     true,
		},
		{
			name:            "expired",
			sessionUser:     "user-1",
			grantUser:       "user-1",
			grantAudience:   "fund-api",
			grantScopes:     []Scope{ScopeAccountSummaryRead},
			requestScope:    ScopeAccountSummaryRead,
			requestAudience: "fund-api",
			prepare: func(_ *testing.T, _ *Service, now *time.Time, _ BearerSession, _ ConsentGrant) {
				*now = now.Add(31 * time.Minute)
			},
			wantCode: DenialGrantExpired,
		},
		{
			name:            "revoked",
			sessionUser:     "user-1",
			grantUser:       "user-1",
			grantAudience:   "fund-api",
			grantScopes:     []Scope{ScopeAccountSummaryRead},
			requestScope:    ScopeAccountSummaryRead,
			requestAudience: "fund-api",
			prepare: func(t *testing.T, service *Service, _ *time.Time, _ BearerSession, grant ConsentGrant) {
				revoked, err := service.RevokeGrant(context.Background(), grant.Ref)
				if err != nil {
					t.Fatalf("RevokeGrant() error = %v", err)
				}
				if revoked.Revision != 2 {
					t.Fatalf("revoked revision = %d, want 2", revoked.Revision)
				}
			},
			wantCode: DenialGrantRevoked,
		},
		{
			name:            "missing scope",
			sessionUser:     "user-1",
			grantUser:       "user-1",
			grantAudience:   "fund-api",
			grantScopes:     []Scope{ScopeAccountSummaryRead},
			requestScope:    ScopeDecisionJournalRead,
			requestAudience: "fund-api",
			wantCode:        DenialMissingScope,
		},
		{
			name:            "subject mismatch",
			sessionUser:     "user-1",
			grantUser:       "user-2",
			grantAudience:   "fund-api",
			grantScopes:     []Scope{ScopeAccountSummaryRead},
			requestScope:    ScopeAccountSummaryRead,
			requestAudience: "fund-api",
			wantCode:        DenialSubjectMismatch,
		},
		{
			name:            "audience mismatch",
			sessionUser:     "user-1",
			grantUser:       "user-1",
			grantAudience:   "remote-tool",
			grantScopes:     []Scope{ScopeAccountSummaryRead},
			requestScope:    ScopeAccountSummaryRead,
			requestAudience: "fund-api",
			wantCode:        DenialAudienceMismatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, now, bearer, grant := authorizationFixture(t, tt.sessionUser, tt.grantUser, tt.grantAudience, tt.grantScopes)
			if tt.prepare != nil {
				tt.prepare(t, service, now, bearer, grant)
			}
			decision, err := service.Authorize(context.Background(), AuthorizationRequest{
				Token:    bearer.Token,
				GrantRef: grant.Ref,
				Subject:  tt.sessionUser,
				Audience: tt.requestAudience,
				Scope:    tt.requestScope,
			})
			if err != nil {
				t.Fatalf("Authorize() error = %v", err)
			}
			if decision.Allowed != tt.wantAllowed || decision.Code != tt.wantCode {
				t.Fatalf("Authorize() = %#v, want allowed=%v code=%q", decision, tt.wantAllowed, tt.wantCode)
			}
		})
	}
}

func TestAuthorizeMissingGrantAndInvalidSession(t *testing.T) {
	service, _, bearer, _ := authorizationFixture(t, "user-1", "user-1", "fund-api", []Scope{ScopeAccountSummaryRead})

	missing, err := service.Authorize(context.Background(), AuthorizationRequest{
		Token:    bearer.Token,
		GrantRef: "grant_missing",
		Subject:  "user-1",
		Audience: "fund-api",
		Scope:    ScopeAccountSummaryRead,
	})
	if err != nil {
		t.Fatalf("missing grant Authorize() error = %v", err)
	}
	if missing.Allowed || missing.Code != DenialMissingGrant {
		t.Fatalf("missing grant decision = %#v", missing)
	}

	invalid, err := service.Authorize(context.Background(), AuthorizationRequest{
		Token:    "fs_invalid",
		GrantRef: "grant_missing",
		Subject:  "user-1",
		Audience: "fund-api",
		Scope:    ScopeAccountSummaryRead,
	})
	if err != nil {
		t.Fatalf("invalid session Authorize() error = %v", err)
	}
	if invalid.Allowed || invalid.Code != DenialSessionInvalid {
		t.Fatalf("invalid session decision = %#v", invalid)
	}
}

func TestRevocationTakesEffectOnNextAuthorization(t *testing.T) {
	ctx := context.Background()
	service, _, bearer, grant := authorizationFixture(t, "user-1", "user-1", "fund-api", []Scope{ScopeAccountSummaryRead})
	request := AuthorizationRequest{
		Token:    bearer.Token,
		GrantRef: grant.Ref,
		Subject:  "user-1",
		Audience: "fund-api",
		Scope:    ScopeAccountSummaryRead,
	}

	active, err := service.Authorize(ctx, request)
	if err != nil || !active.Allowed {
		t.Fatalf("active Authorize() = %#v, %v", active, err)
	}
	revoked, err := service.RevokeGrant(ctx, grant.Ref)
	if err != nil {
		t.Fatalf("RevokeGrant() error = %v", err)
	}
	revokedAgain, err := service.RevokeGrant(ctx, grant.Ref)
	if err != nil {
		t.Fatalf("second RevokeGrant() error = %v", err)
	}
	if revoked.Revision != 2 || revokedAgain.Revision != revoked.Revision {
		t.Fatalf("revoke revisions = %d, %d; want stable 2", revoked.Revision, revokedAgain.Revision)
	}
	denied, err := service.Authorize(ctx, request)
	if err != nil || denied.Allowed || denied.Code != DenialGrantRevoked || denied.Revision != 2 {
		t.Fatalf("post-revoke Authorize() = %#v, %v", denied, err)
	}

	if err := service.RevokeSession(ctx, bearer.Session.Ref); err != nil {
		t.Fatalf("RevokeSession() error = %v", err)
	}
	_, err = service.Authenticate(ctx, bearer.Token, "user-1")
	assertDenialError(t, err, DenialSessionInvalid)
}

func TestAuditEventsAreMinimalAndRedacted(t *testing.T) {
	ctx := context.Background()
	service, _, bearer, grant := authorizationFixture(t, "user-1", "user-1", "fund-api", []Scope{ScopeAccountSummaryRead})
	decision, err := service.Authorize(ctx, AuthorizationRequest{
		Token:    bearer.Token,
		GrantRef: grant.Ref,
		Subject:  "user-1",
		Audience: "fund-api",
		Scope:    ScopeAccountSummaryRead,
	})
	if err != nil || !decision.Allowed {
		t.Fatalf("Authorize() = %#v, %v", decision, err)
	}
	events, err := service.store.AuditEvents(ctx)
	if err != nil {
		t.Fatalf("AuditEvents() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("audit event count = %d, want 1", len(events))
	}
	want := AuditEvent{
		SessionRef: bearer.Session.Ref,
		GrantRef:   grant.Ref,
		Scope:      ScopeAccountSummaryRead,
		Decision:   AuditDecisionAllow,
		Revision:   1,
	}
	if !reflect.DeepEqual(events[0], want) {
		t.Fatalf("audit event = %#v, want %#v", events[0], want)
	}

	eventType := reflect.TypeOf(AuditEvent{})
	wantFields := []string{"SessionRef", "GrantRef", "Scope", "Decision", "Revision"}
	if eventType.NumField() != len(wantFields) {
		t.Fatalf("AuditEvent field count = %d, want %d", eventType.NumField(), len(wantFields))
	}
	for i, name := range wantFields {
		if eventType.Field(i).Name != name {
			t.Fatalf("AuditEvent field %d = %q, want %q", i, eventType.Field(i).Name, name)
		}
	}
	payload, err := json.Marshal(events[0])
	if err != nil {
		t.Fatalf("marshal audit event: %v", err)
	}
	for _, secret := range []string{bearer.Token, "api-key-secret", "account-payload-secret", "user-1", "fund-api"} {
		if strings.Contains(string(payload), secret) {
			t.Fatalf("audit payload contains sensitive value %q: %s", secret, payload)
		}
	}
}

func TestMemoryStoreConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	service, _, bearer, grant := authorizationFixture(t, "user-1", "user-1", "fund-api", []Scope{ScopeAccountSummaryRead})
	request := AuthorizationRequest{
		Token:    bearer.Token,
		GrantRef: grant.Ref,
		Subject:  "user-1",
		Audience: "fund-api",
		Scope:    ScopeAccountSummaryRead,
	}
	var wg sync.WaitGroup
	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := service.Authorize(ctx, request); err != nil {
				t.Errorf("concurrent Authorize() error = %v", err)
			}
			if _, err := service.store.AuditEvents(ctx); err != nil {
				t.Errorf("concurrent AuditEvents() error = %v", err)
			}
		}()
	}
	wg.Wait()
}

func TestCreateGrantNormalizesReadOnlyScopes(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC)
	service := NewService(NewMemoryStore())
	service.now = func() time.Time { return now }
	grant, err := service.CreateGrant(ctx, CreateGrantRequest{
		Subject:   " user-1 ",
		Audience:  " fund-api ",
		Scopes:    []Scope{ScopeHoldingSnapshotRead, ScopeAccountSummaryRead, ScopeHoldingSnapshotRead},
		ExpiresAt: now.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("CreateGrant() error = %v", err)
	}
	wantScopes := []Scope{ScopeAccountSummaryRead, ScopeHoldingSnapshotRead}
	if grant.Subject != "user-1" || grant.Audience != "fund-api" || !reflect.DeepEqual(grant.Scopes, wantScopes) {
		t.Fatalf("normalized grant = %#v", grant)
	}
	_, err = service.CreateGrant(ctx, CreateGrantRequest{
		Subject:   "user-1",
		Audience:  "fund-api",
		Scopes:    []Scope{"fund.trade.execute"},
		ExpiresAt: now.Add(time.Hour),
	})
	if err == nil || !strings.Contains(err.Error(), "not read-only") {
		t.Fatalf("non-read scope error = %v", err)
	}
}

func TestGrantsBySubjectReturnsStableOrder(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC)
	store := NewMemoryStore()
	service := NewService(store)
	grants := []ConsentGrant{
		{Ref: "grant-b", Subject: "user-1", Scopes: []Scope{ScopeHoldingSnapshotRead}, CreatedAt: now},
		{Ref: "grant-new", Subject: "user-1", Scopes: []Scope{ScopeAccountSummaryRead}, CreatedAt: now.Add(time.Minute)},
		{Ref: "grant-a", Subject: "user-1", Scopes: []Scope{ScopeDecisionJournalRead}, CreatedAt: now},
		{Ref: "grant-other", Subject: "user-2", Scopes: []Scope{ScopeAccountSummaryRead}, CreatedAt: now.Add(time.Hour)},
	}
	for _, grant := range grants {
		if err := store.SaveGrant(ctx, grant); err != nil {
			t.Fatalf("SaveGrant(%q) error = %v", grant.Ref, err)
		}
	}

	got, err := service.ListGrants(ctx, " user-1 ")
	if err != nil {
		t.Fatalf("ListGrants() error = %v", err)
	}
	wantRefs := []string{"grant-new", "grant-a", "grant-b"}
	if len(got) != len(wantRefs) {
		t.Fatalf("grant count = %d, want %d", len(got), len(wantRefs))
	}
	for i, ref := range wantRefs {
		if got[i].Ref != ref {
			t.Fatalf("grant %d ref = %q, want %q", i, got[i].Ref, ref)
		}
	}

	got[0].Scopes[0] = ScopeBrokerSyncRead
	again, err := service.GrantsBySubject(ctx, "user-1")
	if err != nil {
		t.Fatalf("second GrantsBySubject() error = %v", err)
	}
	if again[0].Scopes[0] != ScopeAccountSummaryRead {
		t.Fatal("listed grant scopes alias in-memory truth")
	}
	if _, err := service.GrantsBySubject(ctx, " "); err == nil {
		t.Fatal("empty subject GrantsBySubject() error = nil")
	}
	empty, err := service.GrantsBySubject(ctx, "user-missing")
	if err != nil {
		t.Fatalf("empty GrantsBySubject() error = %v", err)
	}
	if empty == nil || len(empty) != 0 {
		t.Fatalf("empty grants = %#v, want non-nil empty slice", empty)
	}
}

func TestAuthorizeGrantWithoutBearerToken(t *testing.T) {
	requestType := reflect.TypeOf(GrantAuthorizationRequest{})
	if _, ok := requestType.FieldByName("Token"); ok {
		t.Fatal("GrantAuthorizationRequest must not accept a bearer token")
	}

	tests := []struct {
		name        string
		prepare     func(*testing.T, *Service, *time.Time, ConsentGrant)
		change      func(*GrantAuthorizationRequest)
		wantAllowed bool
		wantCode    DenialCode
	}{
		{name: "active", wantAllowed: true},
		{
			name: "expired",
			prepare: func(_ *testing.T, _ *Service, now *time.Time, _ ConsentGrant) {
				*now = now.Add(31 * time.Minute)
			},
			wantCode: DenialGrantExpired,
		},
		{
			name: "revoked",
			prepare: func(t *testing.T, service *Service, _ *time.Time, grant ConsentGrant) {
				if _, err := service.RevokeGrant(context.Background(), grant.Ref); err != nil {
					t.Fatalf("RevokeGrant() error = %v", err)
				}
			},
			wantCode: DenialGrantRevoked,
		},
		{
			name: "missing scope",
			change: func(request *GrantAuthorizationRequest) {
				request.Scope = ScopeDecisionJournalRead
			},
			wantCode: DenialMissingScope,
		},
		{
			name: "subject mismatch",
			change: func(request *GrantAuthorizationRequest) {
				request.Subject = "user-2"
			},
			wantCode: DenialSubjectMismatch,
		},
		{
			name: "audience mismatch",
			change: func(request *GrantAuthorizationRequest) {
				request.Audience = "other-service"
			},
			wantCode: DenialAudienceMismatch,
		},
		{
			name: "missing grant",
			change: func(request *GrantAuthorizationRequest) {
				request.GrantRef = "grant_missing"
			},
			wantCode: DenialMissingGrant,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, now, grant := grantAuthorizationFixture(t)
			if tt.prepare != nil {
				tt.prepare(t, service, now, grant)
			}
			request := GrantAuthorizationRequest{
				GrantRef: grant.Ref,
				Subject:  "user-1",
				Audience: "remote-tool",
				Scope:    ScopeAccountSummaryRead,
			}
			if tt.change != nil {
				tt.change(&request)
			}
			serialized, err := json.Marshal(request)
			if err != nil {
				t.Fatalf("marshal GrantAuthorizationRequest: %v", err)
			}
			if strings.Contains(strings.ToLower(string(serialized)), "token") {
				t.Fatalf("grant authorization request exposes token field: %s", serialized)
			}

			decision, err := service.AuthorizeGrant(context.Background(), request)
			if err != nil {
				t.Fatalf("AuthorizeGrant() error = %v", err)
			}
			if decision.Allowed != tt.wantAllowed || decision.Code != tt.wantCode {
				t.Fatalf("AuthorizeGrant() = %#v, want allowed=%v code=%q", decision, tt.wantAllowed, tt.wantCode)
			}
			if decision.SessionRef != "" {
				t.Fatalf("service decision session ref = %q, want empty", decision.SessionRef)
			}
			events, err := service.store.AuditEvents(context.Background())
			if err != nil {
				t.Fatalf("AuditEvents() error = %v", err)
			}
			if len(events) != 1 || events[0].SessionRef != "" {
				t.Fatalf("service audit events = %#v, want one event with empty session ref", events)
			}
		})
	}
}

func authorizationFixture(t *testing.T, sessionSubject, grantSubject, audience string, scopes []Scope) (*Service, *time.Time, BearerSession, ConsentGrant) {
	t.Helper()
	ctx := context.Background()
	now := time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC)
	store := NewMemoryStore()
	service := NewService(store)
	service.now = func() time.Time { return now }
	bearer, err := service.IssueSession(ctx, sessionSubject, time.Hour)
	if err != nil {
		t.Fatalf("IssueSession() error = %v", err)
	}
	grant, err := service.CreateGrant(ctx, CreateGrantRequest{
		Subject:   grantSubject,
		Audience:  audience,
		Scopes:    scopes,
		ExpiresAt: now.Add(30 * time.Minute),
	})
	if err != nil {
		t.Fatalf("CreateGrant() error = %v", err)
	}
	return service, &now, bearer, grant
}

func grantAuthorizationFixture(t *testing.T) (*Service, *time.Time, ConsentGrant) {
	t.Helper()
	ctx := context.Background()
	now := time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC)
	service := NewService(NewMemoryStore())
	service.now = func() time.Time { return now }
	grant, err := service.CreateGrant(ctx, CreateGrantRequest{
		Subject:   "user-1",
		Audience:  "remote-tool",
		Scopes:    []Scope{ScopeAccountSummaryRead},
		ExpiresAt: now.Add(30 * time.Minute),
	})
	if err != nil {
		t.Fatalf("CreateGrant() error = %v", err)
	}
	return service, &now, grant
}

func assertDenialError(t *testing.T, err error, want DenialCode) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want denial %q", want)
	}
	code, ok := DenialCodeFromError(err)
	if !ok || code != want {
		t.Fatalf("denial error = %v, code = %q, want %q", err, code, want)
	}
	if !errors.Is(err, err) {
		t.Fatal("denial error unexpectedly fails errors.Is against itself")
	}
}
