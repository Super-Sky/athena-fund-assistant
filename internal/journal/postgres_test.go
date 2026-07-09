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
	if err := store.Ping(ctx); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}

	matrix := testDecisionMatrix()
	entry, review, err := store.Create(ctx, matrix, "option-balanced", "persist the complete decision")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := store.Close(ctx); err != nil {
		t.Fatalf("Close() before reopen error = %v", err)
	}

	reopened, err := NewPostgresStore(ctx, databaseURL)
	if err != nil {
		t.Fatalf("reopen NewPostgresStore() error = %v", err)
	}
	t.Cleanup(func() {
		_, _ = reopened.pool.Exec(context.Background(), `DELETE FROM journal_entries WHERE id = $1`, entry.ID)
		_ = reopened.Close(context.Background())
	})

	gotEntry, err := reopened.Entry(ctx, entry.ID)
	if err != nil {
		t.Fatalf("Entry() error = %v", err)
	}
	if !reflect.DeepEqual(gotEntry, entry) {
		t.Fatalf("Entry() snapshot = %#v, want %#v", gotEntry, entry)
	}
	gotReview, err := reopened.Review(ctx, review.ID)
	if err != nil {
		t.Fatalf("Review() error = %v", err)
	}
	if !reflect.DeepEqual(gotReview, review) {
		t.Fatalf("Review() snapshot = %#v, want %#v", gotReview, review)
	}

	if _, err := reopened.Entry(ctx, "missing"); !errors.Is(err, ErrEntryNotFound) {
		t.Fatalf("Entry() error = %v, want ErrEntryNotFound", err)
	}
	if _, err := reopened.Review(ctx, "missing"); !errors.Is(err, ErrReviewNotFound) {
		t.Fatalf("Review() error = %v, want ErrReviewNotFound", err)
	}
}
