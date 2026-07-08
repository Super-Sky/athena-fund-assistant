package journal

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Super-Sky/athena-fund-assistant/internal/domain"
)

// MemoryStore keeps MVP journals until PostgreSQL persistence is added.
// MemoryStore 在接入 PostgreSQL 前保存 MVP 决策日志。
type MemoryStore struct {
	mu      sync.Mutex
	entries map[string]domain.JournalEntry
	reviews map[string]domain.ReviewTask
}

// NewMemoryStore creates an in-memory decision journal store.
// NewMemoryStore 创建内存决策日志存储。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		entries: map[string]domain.JournalEntry{},
		reviews: map[string]domain.ReviewTask{},
	}
}

// Create stores a selected option and schedules its first review task.
// Create 保存用户选择的方案并创建首次复盘任务。
func (s *MemoryStore) Create(matrix domain.DecisionMatrix, selectedOptionID, notes string) (domain.JournalEntry, domain.ReviewTask, error) {
	if err := matrix.Validate(); err != nil {
		return domain.JournalEntry{}, domain.ReviewTask{}, err
	}
	selected, ok := selectedOption(matrix, selectedOptionID)
	if !ok {
		return domain.JournalEntry{}, domain.ReviewTask{}, fmt.Errorf("selected option %q not found", selectedOptionID)
	}
	now := time.Now().UTC()
	entry := domain.JournalEntry{
		ID:               fmt.Sprintf("journal_%d", now.UnixNano()),
		CreatedAt:        now,
		MatrixID:         matrix.ID,
		SelectedOptionID: selectedOptionID,
		UserNotes:        notes,
		EvidenceSnapshot: matrix,
	}
	review := domain.ReviewTask{
		ID:          fmt.Sprintf("review_%d", now.UnixNano()),
		JournalID:   entry.ID,
		DueAt:       now.AddDate(0, 0, selected.ReviewAfterDays),
		Question:    fmt.Sprintf("Review whether %s remains valid for %s", selected.Style, matrix.Instrument.Code),
		TriggerHint: selected.Invalidation,
		Status:      "open",
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[entry.ID] = entry
	s.reviews[review.ID] = review
	return entry, review, nil
}

// Entry returns a stored journal entry by id.
// Entry 按 ID 返回已保存的决策日志。
func (s *MemoryStore) Entry(id string) (domain.JournalEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.entries[id]
	if !ok {
		return domain.JournalEntry{}, errors.New("journal entry not found")
	}
	return entry, nil
}

func selectedOption(matrix domain.DecisionMatrix, id string) (domain.DecisionOption, bool) {
	for _, option := range matrix.Options {
		if option.ID == id {
			return option, true
		}
	}
	return domain.DecisionOption{}, false
}
