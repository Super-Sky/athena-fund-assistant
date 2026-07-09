package journal

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"

	"github.com/Super-Sky/athena-fund-assistant/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// PostgresStore persists decision journals and review tasks in PostgreSQL.
// PostgresStore 在 PostgreSQL 中持久化决策日志和复盘任务。
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore creates, verifies, and migrates a PostgreSQL journal store.
// NewPostgresStore 创建、验证并迁移 PostgreSQL 决策日志存储。
func NewPostgresStore(ctx context.Context, databaseURL string) (*PostgresStore, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create journal postgres pool: %w", err)
	}
	store := &PostgresStore{pool: pool}
	if err := store.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping journal postgres: %w", err)
	}
	if err := store.migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return store, nil
}

// Create atomically stores a selected option and its first review task.
// Create 以事务方式保存用户选择的方案及其首次复盘任务。
func (s *PostgresStore) Create(ctx context.Context, matrix domain.DecisionMatrix, selectedOptionID, notes string) (domain.JournalEntry, domain.ReviewTask, error) {
	entry, review, err := buildJournalRecords(matrix, selectedOptionID, notes)
	if err != nil {
		return domain.JournalEntry{}, domain.ReviewTask{}, err
	}
	entrySnapshot, err := json.Marshal(entry)
	if err != nil {
		return domain.JournalEntry{}, domain.ReviewTask{}, fmt.Errorf("marshal journal entry: %w", err)
	}
	reviewSnapshot, err := json.Marshal(review)
	if err != nil {
		return domain.JournalEntry{}, domain.ReviewTask{}, fmt.Errorf("marshal review task: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.JournalEntry{}, domain.ReviewTask{}, fmt.Errorf("begin journal transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	if _, err := tx.Exec(
		ctx,
		`INSERT INTO journal_entries (id, created_at, matrix_id, selected_option_id, snapshot)
		 VALUES ($1, $2, $3, $4, $5)`,
		entry.ID,
		entry.CreatedAt,
		entry.MatrixID,
		entry.SelectedOptionID,
		entrySnapshot,
	); err != nil {
		return domain.JournalEntry{}, domain.ReviewTask{}, fmt.Errorf("insert journal entry: %w", err)
	}
	if _, err := tx.Exec(
		ctx,
		`INSERT INTO review_tasks (id, journal_id, due_at, status, snapshot)
		 VALUES ($1, $2, $3, $4, $5)`,
		review.ID,
		review.JournalID,
		review.DueAt,
		review.Status,
		reviewSnapshot,
	); err != nil {
		return domain.JournalEntry{}, domain.ReviewTask{}, fmt.Errorf("insert review task: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.JournalEntry{}, domain.ReviewTask{}, fmt.Errorf("commit journal transaction: %w", err)
	}
	return entry, review, nil
}

// Entry returns a persisted journal entry by ID.
// Entry 按 ID 返回已持久化的决策日志。
func (s *PostgresStore) Entry(ctx context.Context, id string) (domain.JournalEntry, error) {
	var snapshot []byte
	if err := s.pool.QueryRow(ctx, `SELECT snapshot FROM journal_entries WHERE id = $1`, id).Scan(&snapshot); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.JournalEntry{}, ErrEntryNotFound
		}
		return domain.JournalEntry{}, fmt.Errorf("read journal entry: %w", err)
	}
	var entry domain.JournalEntry
	if err := json.Unmarshal(snapshot, &entry); err != nil {
		return domain.JournalEntry{}, fmt.Errorf("decode journal entry snapshot: %w", err)
	}
	return entry, nil
}

// Review returns a persisted review task by ID.
// Review 按 ID 返回已持久化的复盘任务。
func (s *PostgresStore) Review(ctx context.Context, id string) (domain.ReviewTask, error) {
	var snapshot []byte
	if err := s.pool.QueryRow(ctx, `SELECT snapshot FROM review_tasks WHERE id = $1`, id).Scan(&snapshot); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ReviewTask{}, ErrReviewNotFound
		}
		return domain.ReviewTask{}, fmt.Errorf("read review task: %w", err)
	}
	var review domain.ReviewTask
	if err := json.Unmarshal(snapshot, &review); err != nil {
		return domain.ReviewTask{}, fmt.Errorf("decode review task snapshot: %w", err)
	}
	return review, nil
}

// Ping verifies the PostgreSQL connection pool.
// Ping 验证 PostgreSQL 连接池。
func (s *PostgresStore) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

// Close releases PostgreSQL connection pool resources.
// Close 释放 PostgreSQL 连接池资源。
func (s *PostgresStore) Close(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.pool.Close()
	return nil
}

func (s *PostgresStore) migrate(ctx context.Context) error {
	names, err := fs.Glob(migrationFiles, "migrations/*.sql")
	if err != nil {
		return fmt.Errorf("list journal migrations: %w", err)
	}
	for _, name := range names {
		query, err := migrationFiles.ReadFile(name)
		if err != nil {
			return fmt.Errorf("read journal migration %s: %w", name, err)
		}
		if _, err := s.pool.Exec(ctx, string(query)); err != nil {
			return fmt.Errorf("apply journal migration %s: %w", name, err)
		}
	}
	return nil
}

var _ Store = (*PostgresStore)(nil)
