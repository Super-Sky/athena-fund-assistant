// This file verifies optional PostgreSQL restart persistence for decision journals.
// 本文件验证决策日志可选 PostgreSQL 重启后的持久化行为。
package journal

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestPostgresStorePersistence(t *testing.T) {
	databaseURL := os.Getenv("ATHENA_FUND_PG_TEST_DSN")
	if databaseURL == "" {
		t.Skip("ATHENA_FUND_PG_TEST_DSN is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	store, err := NewPostgresStore(ctx, databaseURL)
	if err != nil {
		t.Fatalf("NewPostgresStore() error = %v", err)
	}
	entry, review, err := store.Create(ctx, testDecisionMatrix(), "option-balanced", "verify restart persistence")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := store.Close(ctx); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	reopened, err := NewPostgresStore(ctx, databaseURL)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	t.Cleanup(func() {
		_, _ = reopened.pool.Exec(context.Background(), `DELETE FROM journal_entries WHERE id = $1`, entry.ID)
		_ = reopened.Close(context.Background())
	})
	gotEntry, err := reopened.Entry(ctx, entry.ID)
	if err != nil || !reflect.DeepEqual(gotEntry, entry) {
		t.Fatalf("Entry() = %#v, %v; want %#v", gotEntry, err, entry)
	}
	gotReview, err := reopened.Review(ctx, review.ID)
	if err != nil || !reflect.DeepEqual(gotReview, review) {
		t.Fatalf("Review() = %#v, %v; want %#v", gotReview, err, review)
	}
	if _, err := reopened.Entry(ctx, "missing"); !errors.Is(err, ErrEntryNotFound) {
		t.Fatalf("missing Entry() error = %v", err)
	}
}
