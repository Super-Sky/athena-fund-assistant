// This file provides the explicit non-durable journal fallback for local workflows.
// 本文件为本地工作流提供明确的非持久化决策日志 fallback。
package journal

import (
	"context"
	"sync"

	"github.com/Super-Sky/athena-fund-assistant/internal/domain"
)

// MemoryStore keeps decision journals in memory for local and test workflows.
// MemoryStore 为本地和测试流程在内存中保存决策日志。
type MemoryStore struct {
	mu      sync.RWMutex
	entries map[string]domain.JournalEntry
	reviews map[string]domain.ReviewTask
}

// NewMemoryStore creates an in-memory decision journal store.
// NewMemoryStore 创建内存决策日志存储。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{entries: map[string]domain.JournalEntry{}, reviews: map[string]domain.ReviewTask{}}
}

// Create stores a selected option and schedules its first review task.
// Create 保存用户选择的方案并创建首次复盘任务。
func (s *MemoryStore) Create(ctx context.Context, matrix domain.DecisionMatrix, selectedOptionID, notes string) (domain.JournalEntry, domain.ReviewTask, error) {
	if err := ctx.Err(); err != nil {
		return domain.JournalEntry{}, domain.ReviewTask{}, err
	}
	entry, review, err := buildJournalRecords(matrix, selectedOptionID, notes)
	if err != nil {
		return domain.JournalEntry{}, domain.ReviewTask{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[entry.ID] = entry
	s.reviews[review.ID] = review
	return entry, review, nil
}

// Entry returns a stored journal entry by ID.
// Entry 按 ID 返回已保存的决策日志。
func (s *MemoryStore) Entry(ctx context.Context, id string) (domain.JournalEntry, error) {
	if err := ctx.Err(); err != nil {
		return domain.JournalEntry{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.entries[id]
	if !ok {
		return domain.JournalEntry{}, ErrEntryNotFound
	}
	return entry, nil
}

// Review returns a stored review task by ID.
// Review 按 ID 返回已保存的复盘任务。
func (s *MemoryStore) Review(ctx context.Context, id string) (domain.ReviewTask, error) {
	if err := ctx.Err(); err != nil {
		return domain.ReviewTask{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	review, ok := s.reviews[id]
	if !ok {
		return domain.ReviewTask{}, ErrReviewNotFound
	}
	return review, nil
}

// Ping verifies that the memory store can accept work.
// Ping 验证内存存储可以接收工作。
func (s *MemoryStore) Ping(ctx context.Context) error { return ctx.Err() }

// Close releases memory store resources.
// Close 释放内存存储资源。
func (s *MemoryStore) Close(ctx context.Context) error { return ctx.Err() }

var _ Store = (*MemoryStore)(nil)
