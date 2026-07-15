// This file verifies optional PostgreSQL authorization persistence and hash-only storage.
// 本文件验证可选 PostgreSQL 授权持久化与仅哈希存储。
package authorization

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestPostgresStorePersistenceAndTokenHashing(t *testing.T) {
	databaseURL := os.Getenv("ATHENA_FUND_PG_TEST_DSN")
	if databaseURL == "" {
		t.Skip("ATHENA_FUND_PG_TEST_DSN is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	now := time.Now().UTC().Truncate(time.Microsecond)

	store, err := NewPostgresStore(ctx, databaseURL)
	if err != nil {
		t.Fatalf("NewPostgresStore() error = %v", err)
	}
	service := NewService(store)
	service.now = func() time.Time { return now }
	subject := fmt.Sprintf("postgres-user-%d", time.Now().UnixNano())
	bearer, err := service.IssueSession(ctx, subject, time.Hour)
	if err != nil {
		_ = store.Close(context.Background())
		t.Fatalf("IssueSession() error = %v", err)
	}
	grant, err := service.CreateGrant(ctx, CreateGrantRequest{
		Subject:   subject,
		Audience:  "postgres-test",
		Scopes:    []Scope{ScopeAccountSummaryRead},
		ExpiresAt: now.Add(time.Hour),
	})
	if err != nil {
		_ = store.Close(context.Background())
		t.Fatalf("CreateGrant() error = %v", err)
	}
	secondGrant, err := service.CreateGrant(ctx, CreateGrantRequest{
		Subject:   subject,
		Audience:  "postgres-test",
		Scopes:    []Scope{ScopeHoldingSnapshotRead},
		ExpiresAt: now.Add(time.Hour),
	})
	if err != nil {
		_ = store.Close(context.Background())
		t.Fatalf("second CreateGrant() error = %v", err)
	}
	decision, err := service.Authorize(ctx, AuthorizationRequest{
		Token:    bearer.Token,
		GrantRef: grant.Ref,
		Subject:  subject,
		Audience: "postgres-test",
		Scope:    ScopeAccountSummaryRead,
	})
	if err != nil || !decision.Allowed {
		_ = store.Close(context.Background())
		t.Fatalf("Authorize() = %#v, %v", decision, err)
	}

	var persistedHash string
	if err := store.pool.QueryRow(ctx, `SELECT encode(token_hash, 'hex') FROM authorization_sessions WHERE ref = $1`, bearer.Session.Ref).Scan(&persistedHash); err != nil {
		_ = store.Close(context.Background())
		t.Fatalf("read persisted token hash: %v", err)
	}
	wantHash := sha256.Sum256([]byte(bearer.Token))
	if persistedHash != hex.EncodeToString(wantHash[:]) {
		_ = store.Close(context.Background())
		t.Fatalf("persisted hash = %q, want %q", persistedHash, hex.EncodeToString(wantHash[:]))
	}
	if strings.Contains(persistedHash, bearer.Token) {
		_ = store.Close(context.Background())
		t.Fatal("persisted token hash contains raw token")
	}
	if err := store.Close(ctx); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	reopened, err := NewPostgresStore(ctx, databaseURL)
	if err != nil {
		t.Fatalf("reopen PostgresStore: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = reopened.pool.Exec(cleanupCtx, `DELETE FROM authorization_audit_events WHERE session_ref = $1 OR grant_ref IN ($2, $3)`, bearer.Session.Ref, grant.Ref, secondGrant.Ref)
		_, _ = reopened.pool.Exec(cleanupCtx, `DELETE FROM authorization_consent_grants WHERE ref = $1`, grant.Ref)
		_, _ = reopened.pool.Exec(cleanupCtx, `DELETE FROM authorization_consent_grants WHERE ref = $1`, secondGrant.Ref)
		_, _ = reopened.pool.Exec(cleanupCtx, `DELETE FROM authorization_sessions WHERE ref = $1`, bearer.Session.Ref)
		_ = reopened.Close(cleanupCtx)
	})
	reopenedService := NewService(reopened)
	reopenedService.now = func() time.Time { return now }
	if _, err := reopenedService.Authenticate(ctx, bearer.Token, subject); err != nil {
		t.Fatalf("Authenticate() after reopen error = %v", err)
	}
	persistedGrant, err := reopenedService.Grant(ctx, grant.Ref)
	if err != nil {
		t.Fatalf("Grant() after reopen error = %v", err)
	}
	if persistedGrant.Revision != 1 || len(persistedGrant.Scopes) != 1 || persistedGrant.Scopes[0] != ScopeAccountSummaryRead {
		t.Fatalf("persisted grant = %#v", persistedGrant)
	}
	listed, err := reopenedService.ListGrants(ctx, subject)
	if err != nil {
		t.Fatalf("ListGrants() after reopen error = %v", err)
	}
	wantRefs := []string{grant.Ref, secondGrant.Ref}
	sort.Strings(wantRefs)
	if len(listed) != len(wantRefs) {
		t.Fatalf("listed grant count = %d, want %d", len(listed), len(wantRefs))
	}
	for i, ref := range wantRefs {
		if listed[i].Ref != ref {
			t.Fatalf("listed grant %d = %q, want %q", i, listed[i].Ref, ref)
		}
	}
}

func TestAuthorizationMigrationHasNoSecretColumns(t *testing.T) {
	migration, err := migrationFiles.ReadFile("migrations/001_authorization.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	lower := strings.ToLower(string(migration))
	for _, forbidden := range []string{"raw_token", "bearer_token", "api_key", "account_payload"} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf("migration contains forbidden secret column %q", forbidden)
		}
	}
	if !strings.Contains(lower, "token_hash bytea") {
		t.Fatal("migration does not persist a bytea token hash")
	}
}
