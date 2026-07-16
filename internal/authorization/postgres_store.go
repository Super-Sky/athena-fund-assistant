// This file implements PostgreSQL authorization persistence with embedded migrations.
// 本文件使用嵌入式 migration 实现 PostgreSQL 授权持久化。
package authorization

import (
	"context"
	"crypto/sha256"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// PostgresStore persists authorization sessions, grants, and audit events in PostgreSQL.
// PostgresStore 在 PostgreSQL 中持久化授权会话、授权与审计事件。
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore creates, verifies, and migrates a PostgreSQL authorization store.
// NewPostgresStore 创建、验证并迁移 PostgreSQL 授权存储。
func NewPostgresStore(ctx context.Context, databaseURL string) (*PostgresStore, error) {
	if databaseURL == "" {
		return nil, errors.New("authorization database url is required")
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create authorization postgres pool: %w", err)
	}
	store := &PostgresStore{pool: pool}
	if err := store.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping authorization postgres: %w", err)
	}
	if err := store.migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return store, nil
}

// SaveSession persists a session record containing only a SHA-256 token hash.
// SaveSession 持久化仅包含 SHA-256 token 哈希的会话记录。
func (s *PostgresStore) SaveSession(ctx context.Context, record SessionRecord) error {
	_, err := s.pool.Exec(ctx, `
INSERT INTO authorization_sessions (ref, subject, token_hash, created_at, expires_at, revoked_at)
VALUES ($1, $2, $3, $4, $5, $6)`,
		record.Ref,
		record.Subject,
		record.TokenHash[:],
		record.CreatedAt,
		record.ExpiresAt,
		record.RevokedAt,
	)
	if err != nil {
		return fmt.Errorf("insert authorization session: %w", err)
	}
	return nil
}

// SessionByTokenHash returns a session by its SHA-256 token hash.
// SessionByTokenHash 按 SHA-256 token 哈希返回会话。
func (s *PostgresStore) SessionByTokenHash(ctx context.Context, hash [sha256.Size]byte) (SessionRecord, error) {
	row := s.pool.QueryRow(ctx, `
SELECT ref, subject, token_hash, created_at, expires_at, revoked_at
FROM authorization_sessions
WHERE token_hash = $1`, hash[:])
	record, err := scanSessionRecord(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SessionRecord{}, ErrSessionNotFound
		}
		return SessionRecord{}, fmt.Errorf("select authorization session: %w", err)
	}
	return record, nil
}

// RevokeSession marks a persisted session as revoked.
// RevokeSession 将持久化会话标记为已撤销。
func (s *PostgresStore) RevokeSession(ctx context.Context, ref string, revokedAt time.Time) error {
	tag, err := s.pool.Exec(ctx, `
UPDATE authorization_sessions
SET revoked_at = COALESCE(revoked_at, $2)
WHERE ref = $1`, ref, revokedAt)
	if err != nil {
		return fmt.Errorf("update authorization session revocation: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrSessionNotFound
	}
	return nil
}

// SaveGrant persists a consent grant.
// SaveGrant 持久化一条同意授权。
func (s *PostgresStore) SaveGrant(ctx context.Context, grant ConsentGrant) error {
	_, err := s.pool.Exec(ctx, `
INSERT INTO authorization_consent_grants
  (ref, subject, audience, scopes, revision, created_at, updated_at, expires_at, revoked_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		grant.Ref,
		grant.Subject,
		grant.Audience,
		scopeStrings(grant.Scopes),
		grant.Revision,
		grant.CreatedAt,
		grant.UpdatedAt,
		grant.ExpiresAt,
		grant.RevokedAt,
	)
	if err != nil {
		return fmt.Errorf("insert consent grant: %w", err)
	}
	return nil
}

// Grant returns a consent grant by reference.
// Grant 按引用返回同意授权。
func (s *PostgresStore) Grant(ctx context.Context, ref string) (ConsentGrant, error) {
	row := s.pool.QueryRow(ctx, `
SELECT ref, subject, audience, scopes, revision, created_at, updated_at, expires_at, revoked_at
FROM authorization_consent_grants
WHERE ref = $1`, ref)
	grant, err := scanGrant(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ConsentGrant{}, ErrGrantNotFound
		}
		return ConsentGrant{}, fmt.Errorf("select consent grant: %w", err)
	}
	return grant, nil
}

// GrantsBySubject returns grants newest first with reference as a stable tie-breaker.
// GrantsBySubject 按最新优先返回主体授权，并以引用作为稳定的同时间排序条件。
func (s *PostgresStore) GrantsBySubject(ctx context.Context, subject string) ([]ConsentGrant, error) {
	rows, err := s.pool.Query(ctx, `
SELECT ref, subject, audience, scopes, revision, created_at, updated_at, expires_at, revoked_at
FROM authorization_consent_grants
WHERE subject = $1
ORDER BY created_at DESC, ref ASC`, subject)
	if err != nil {
		return nil, fmt.Errorf("select consent grants by subject: %w", err)
	}
	defer rows.Close()
	grants := make([]ConsentGrant, 0)
	for rows.Next() {
		grant, err := scanGrant(rows)
		if err != nil {
			return nil, fmt.Errorf("scan consent grant: %w", err)
		}
		grants = append(grants, grant)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate consent grants: %w", err)
	}
	return grants, nil
}

// RevokeGrant marks a grant revoked and advances its revision on the first call.
// RevokeGrant 将授权标记为已撤销，并在首次调用时推进版本。
func (s *PostgresStore) RevokeGrant(ctx context.Context, ref string, revokedAt time.Time) (ConsentGrant, error) {
	row := s.pool.QueryRow(ctx, `
UPDATE authorization_consent_grants
SET revoked_at = COALESCE(revoked_at, $2),
    updated_at = CASE WHEN revoked_at IS NULL THEN $2 ELSE updated_at END,
    revision = CASE WHEN revoked_at IS NULL THEN revision + 1 ELSE revision END
WHERE ref = $1
RETURNING ref, subject, audience, scopes, revision, created_at, updated_at, expires_at, revoked_at`, ref, revokedAt)
	grant, err := scanGrant(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ConsentGrant{}, ErrGrantNotFound
		}
		return ConsentGrant{}, fmt.Errorf("update consent grant revocation: %w", err)
	}
	return grant, nil
}

// AppendAuditEvent stores one validated, minimal audit event.
// AppendAuditEvent 保存一条经过校验的最小审计事件。
func (s *PostgresStore) AppendAuditEvent(ctx context.Context, event AuditEvent) error {
	if err := validateAuditEvent(event); err != nil {
		return err
	}
	_, err := s.pool.Exec(ctx, `
INSERT INTO authorization_audit_events (session_ref, grant_ref, scope, decision, revision)
VALUES ($1, $2, $3, $4, $5)`, event.SessionRef, event.GrantRef, event.Scope, event.Decision, event.Revision)
	if err != nil {
		return fmt.Errorf("insert authorization audit event: %w", err)
	}
	return nil
}

// AuditEvents returns stored minimal audit events in insertion order.
// AuditEvents 按写入顺序返回已保存的最小审计事件。
func (s *PostgresStore) AuditEvents(ctx context.Context) ([]AuditEvent, error) {
	rows, err := s.pool.Query(ctx, `
SELECT session_ref, grant_ref, scope, decision, revision
FROM authorization_audit_events
ORDER BY sequence`)
	if err != nil {
		return nil, fmt.Errorf("select authorization audit events: %w", err)
	}
	defer rows.Close()
	var events []AuditEvent
	for rows.Next() {
		var event AuditEvent
		if err := rows.Scan(&event.SessionRef, &event.GrantRef, &event.Scope, &event.Decision, &event.Revision); err != nil {
			return nil, fmt.Errorf("scan authorization audit event: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate authorization audit events: %w", err)
	}
	return events, nil
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
		return fmt.Errorf("list authorization migrations: %w", err)
	}
	for _, name := range names {
		query, err := migrationFiles.ReadFile(name)
		if err != nil {
			return fmt.Errorf("read authorization migration %s: %w", name, err)
		}
		if _, err := s.pool.Exec(ctx, string(query)); err != nil {
			return fmt.Errorf("apply authorization migration %s: %w", name, err)
		}
	}
	return nil
}

type scanner interface {
	Scan(...any) error
}

func scanSessionRecord(row scanner) (SessionRecord, error) {
	var record SessionRecord
	var tokenHash []byte
	if err := row.Scan(
		&record.Ref,
		&record.Subject,
		&tokenHash,
		&record.CreatedAt,
		&record.ExpiresAt,
		&record.RevokedAt,
	); err != nil {
		return SessionRecord{}, err
	}
	if len(tokenHash) != sha256.Size {
		return SessionRecord{}, fmt.Errorf("invalid persisted token hash size %d", len(tokenHash))
	}
	copy(record.TokenHash[:], tokenHash)
	return record, nil
}

func scanGrant(row scanner) (ConsentGrant, error) {
	var grant ConsentGrant
	var scopes []string
	if err := row.Scan(
		&grant.Ref,
		&grant.Subject,
		&grant.Audience,
		&scopes,
		&grant.Revision,
		&grant.CreatedAt,
		&grant.UpdatedAt,
		&grant.ExpiresAt,
		&grant.RevokedAt,
	); err != nil {
		return ConsentGrant{}, err
	}
	grant.Scopes = make([]Scope, len(scopes))
	for i, scope := range scopes {
		grant.Scopes[i] = Scope(scope)
	}
	return grant, nil
}

func scopeStrings(scopes []Scope) []string {
	values := make([]string, len(scopes))
	for i, scope := range scopes {
		values[i] = string(scope)
	}
	return values
}

var _ Store = (*PostgresStore)(nil)
