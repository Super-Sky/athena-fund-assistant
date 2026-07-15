// This file implements the thread-safe in-memory authorization truth store.
// 本文件实现线程安全的内存授权真相存储。
package authorization

import (
	"context"
	"crypto/sha256"
	"sort"
	"sync"
	"time"
)

// MemoryStore keeps authorization state in process for local runs and tests.
// MemoryStore 为本地运行与测试在进程内保存授权状态。
type MemoryStore struct {
	mu               sync.RWMutex
	sessionsByRef    map[string]SessionRecord
	sessionRefByHash map[[sha256.Size]byte]string
	grants           map[string]ConsentGrant
	auditEvents      []AuditEvent
}

// NewMemoryStore creates an empty thread-safe authorization store.
// NewMemoryStore 创建空的线程安全授权存储。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		sessionsByRef:    make(map[string]SessionRecord),
		sessionRefByHash: make(map[[sha256.Size]byte]string),
		grants:           make(map[string]ConsentGrant),
	}
}

// SaveSession persists a session record containing only a token hash.
// SaveSession 持久化仅包含 token 哈希的会话记录。
func (s *MemoryStore) SaveSession(ctx context.Context, record SessionRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessionsByRef[record.Ref] = cloneSessionRecord(record)
	s.sessionRefByHash[record.TokenHash] = record.Ref
	return nil
}

// SessionByTokenHash returns a session by its SHA-256 token hash.
// SessionByTokenHash 按 SHA-256 token 哈希返回会话。
func (s *MemoryStore) SessionByTokenHash(ctx context.Context, hash [sha256.Size]byte) (SessionRecord, error) {
	if err := ctx.Err(); err != nil {
		return SessionRecord{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	ref, ok := s.sessionRefByHash[hash]
	if !ok {
		return SessionRecord{}, ErrSessionNotFound
	}
	return cloneSessionRecord(s.sessionsByRef[ref]), nil
}

// RevokeSession marks a persisted session as revoked.
// RevokeSession 将持久化会话标记为已撤销。
func (s *MemoryStore) RevokeSession(ctx context.Context, ref string, revokedAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.sessionsByRef[ref]
	if !ok {
		return ErrSessionNotFound
	}
	if record.RevokedAt == nil {
		at := revokedAt.UTC()
		record.RevokedAt = &at
		s.sessionsByRef[ref] = record
	}
	return nil
}

// SaveGrant persists a consent grant.
// SaveGrant 持久化一条同意授权。
func (s *MemoryStore) SaveGrant(ctx context.Context, grant ConsentGrant) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.grants[grant.Ref] = cloneGrant(grant)
	return nil
}

// Grant returns a consent grant by reference.
// Grant 按引用返回同意授权。
func (s *MemoryStore) Grant(ctx context.Context, ref string) (ConsentGrant, error) {
	if err := ctx.Err(); err != nil {
		return ConsentGrant{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	grant, ok := s.grants[ref]
	if !ok {
		return ConsentGrant{}, ErrGrantNotFound
	}
	return cloneGrant(grant), nil
}

// GrantsBySubject returns grants newest first with reference as a stable tie-breaker.
// GrantsBySubject 按最新优先返回主体授权，并以引用作为稳定的同时间排序条件。
func (s *MemoryStore) GrantsBySubject(ctx context.Context, subject string) ([]ConsentGrant, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	grants := make([]ConsentGrant, 0)
	for _, grant := range s.grants {
		if grant.Subject == subject {
			grants = append(grants, cloneGrant(grant))
		}
	}
	sort.Slice(grants, func(i, j int) bool {
		if grants[i].CreatedAt.Equal(grants[j].CreatedAt) {
			return grants[i].Ref < grants[j].Ref
		}
		return grants[i].CreatedAt.After(grants[j].CreatedAt)
	})
	return grants, nil
}

// RevokeGrant marks a grant revoked and advances its revision on the first call.
// RevokeGrant 将授权标记为已撤销，并在首次调用时推进版本。
func (s *MemoryStore) RevokeGrant(ctx context.Context, ref string, revokedAt time.Time) (ConsentGrant, error) {
	if err := ctx.Err(); err != nil {
		return ConsentGrant{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	grant, ok := s.grants[ref]
	if !ok {
		return ConsentGrant{}, ErrGrantNotFound
	}
	if grant.RevokedAt == nil {
		at := revokedAt.UTC()
		grant.RevokedAt = &at
		grant.UpdatedAt = at
		grant.Revision++
		s.grants[ref] = grant
	}
	return cloneGrant(grant), nil
}

// AppendAuditEvent stores one validated, minimal audit event.
// AppendAuditEvent 保存一条经过校验的最小审计事件。
func (s *MemoryStore) AppendAuditEvent(ctx context.Context, event AuditEvent) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateAuditEvent(event); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.auditEvents = append(s.auditEvents, event)
	return nil
}

// AuditEvents returns a snapshot of stored minimal audit events.
// AuditEvents 返回已保存最小审计事件的快照。
func (s *MemoryStore) AuditEvents(ctx context.Context) ([]AuditEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]AuditEvent(nil), s.auditEvents...), nil
}

func cloneSessionRecord(record SessionRecord) SessionRecord {
	if record.RevokedAt != nil {
		at := *record.RevokedAt
		record.RevokedAt = &at
	}
	return record
}

func cloneGrant(grant ConsentGrant) ConsentGrant {
	grant.Scopes = append([]Scope(nil), grant.Scopes...)
	if grant.RevokedAt != nil {
		at := *grant.RevokedAt
		grant.RevokedAt = &at
	}
	return grant
}

var _ Store = (*MemoryStore)(nil)
