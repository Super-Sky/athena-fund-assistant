package journal

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/Super-Sky/athena-fund-assistant/internal/domain"
)

var (
	// ErrEntryNotFound indicates that a journal entry does not exist.
	// ErrEntryNotFound 表示决策日志不存在。
	ErrEntryNotFound = errors.New("journal entry not found")
	// ErrReviewNotFound indicates that a review task does not exist.
	// ErrReviewNotFound 表示复盘任务不存在。
	ErrReviewNotFound = errors.New("review task not found")
)

// Store persists decision journal entries and their follow-up review tasks.
// Store 持久化决策日志及其后续复盘任务。
type Store interface {
	Create(context.Context, domain.DecisionMatrix, string, string) (domain.JournalEntry, domain.ReviewTask, error)
	Entry(context.Context, string) (domain.JournalEntry, error)
	Review(context.Context, string) (domain.ReviewTask, error)
	Ping(context.Context) error
	Close(context.Context) error
}

func buildJournalRecords(matrix domain.DecisionMatrix, selectedOptionID, notes string) (domain.JournalEntry, domain.ReviewTask, error) {
	if err := matrix.Validate(); err != nil {
		return domain.JournalEntry{}, domain.ReviewTask{}, err
	}
	selected, ok := selectedOption(matrix, selectedOptionID)
	if !ok {
		return domain.JournalEntry{}, domain.ReviewTask{}, fmt.Errorf("selected option %q not found", selectedOptionID)
	}
	entryID, err := newRecordID("journal")
	if err != nil {
		return domain.JournalEntry{}, domain.ReviewTask{}, err
	}
	reviewID, err := newRecordID("review")
	if err != nil {
		return domain.JournalEntry{}, domain.ReviewTask{}, err
	}
	now := time.Now().UTC()
	entry := domain.JournalEntry{
		ID:               entryID,
		CreatedAt:        now,
		MatrixID:         matrix.ID,
		SelectedOptionID: selectedOptionID,
		UserNotes:        notes,
		EvidenceSnapshot: matrix,
	}
	review := domain.ReviewTask{
		ID:          reviewID,
		JournalID:   entry.ID,
		DueAt:       now.AddDate(0, 0, selected.ReviewAfterDays),
		Question:    fmt.Sprintf("Review whether %s remains valid for %s", selected.Style, matrix.Instrument.Code),
		TriggerHint: selected.Invalidation,
		Status:      "open",
	}
	return entry, review, nil
}

func newRecordID(prefix string) (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", fmt.Errorf("generate %s id: %w", prefix, err)
	}
	return prefix + "_" + hex.EncodeToString(value[:]), nil
}

func selectedOption(matrix domain.DecisionMatrix, id string) (domain.DecisionOption, bool) {
	for _, option := range matrix.Options {
		if option.ID == id {
			return option, true
		}
	}
	return domain.DecisionOption{}, false
}
