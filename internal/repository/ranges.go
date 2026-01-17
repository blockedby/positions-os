package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ParsedRange represents a range of scraped message ids for a target
type ParsedRange struct {
	ID       uuid.UUID
	TargetID uuid.UUID
	MinMsgID int64
	MaxMsgID int64
}

// Contains checks if a message id is within the parsed range
func (r *ParsedRange) Contains(msgID int64) bool {
	return msgID >= r.MinMsgID && msgID <= r.MaxMsgID
}

// Extend expands the range to include new min/max values
func (r *ParsedRange) Extend(newMin, newMax int64) {
	// handle first initialization (0,0 is empty range)
	if r.MinMsgID == 0 && r.MaxMsgID == 0 {
		r.MinMsgID = newMin
		r.MaxMsgID = newMax
		return
	}

	if newMin < r.MinMsgID {
		r.MinMsgID = newMin
	}
	if newMax > r.MaxMsgID {
		r.MaxMsgID = newMax
	}
}

// MessageIDFilter filters messages based on already parsed ranges
type MessageIDFilter struct {
	minParsed int64
	maxParsed int64
}

// NewMessageIDFilter creates a filter with min and max parsed message ids
func NewMessageIDFilter(minParsed, maxParsed int64) *MessageIDFilter {
	return &MessageIDFilter{minParsed: minParsed, maxParsed: maxParsed}
}

// FilterNew returns message ids that are outside the parsed range [min, max]
// Messages below min (older) or above max (newer) are considered new
func (f *MessageIDFilter) FilterNew(messageIDs []int64) []int64 {
	if len(messageIDs) == 0 {
		return []int64{}
	}

	// If no range exists (both 0), all messages are new
	if f.minParsed == 0 && f.maxParsed == 0 {
		return messageIDs
	}

	var newIDs []int64
	for _, id := range messageIDs {
		// Message is new if it's outside the [min, max] range
		if id < f.minParsed || id > f.maxParsed {
			newIDs = append(newIDs, id)
		}
	}

	if newIDs == nil {
		return []int64{}
	}
	return newIDs
}

// SmartMessageFilter filters messages based on both range AND existing jobs.
// A message is "new" if:
// 1. It's outside the parsed range [min, max], OR
// 2. It's inside the range but no job exists for it
type SmartMessageFilter struct {
	minParsed    int64
	maxParsed    int64
	existingJobs map[int64]bool
}

// NewSmartMessageFilter creates a filter with range and existing job IDs
func NewSmartMessageFilter(minParsed, maxParsed int64, existingJobIDs []int64) *SmartMessageFilter {
	jobSet := make(map[int64]bool, len(existingJobIDs))
	for _, id := range existingJobIDs {
		jobSet[id] = true
	}
	return &SmartMessageFilter{
		minParsed:    minParsed,
		maxParsed:    maxParsed,
		existingJobs: jobSet,
	}
}

// FilterNew returns message IDs that should be processed.
// Messages are new if outside range OR if inside range but no job exists.
func (f *SmartMessageFilter) FilterNew(messageIDs []int64) []int64 {
	if len(messageIDs) == 0 {
		return []int64{}
	}

	// If no range exists, all messages are new
	if f.minParsed == 0 && f.maxParsed == 0 {
		return messageIDs
	}

	var newIDs []int64
	for _, id := range messageIDs {
		// Outside range = definitely new
		if id < f.minParsed || id > f.maxParsed {
			newIDs = append(newIDs, id)
			continue
		}
		// Inside range but no job exists = also new
		if !f.existingJobs[id] {
			newIDs = append(newIDs, id)
		}
	}

	if newIDs == nil {
		return []int64{}
	}
	return newIDs
}

// RangesRepository handles parsed_ranges table operations
type RangesRepository struct {
	pool *pgxpool.Pool
}

// NewRangesRepository creates a new ranges repository
func NewRangesRepository(pool *pgxpool.Pool) *RangesRepository {
	return &RangesRepository{pool: pool}
}

// GetRange returns the parsed range for a target, or nil if not exists
func (r *RangesRepository) GetRange(ctx context.Context, targetID uuid.UUID) (*ParsedRange, error) {
	var pr ParsedRange
	err := r.pool.QueryRow(ctx, `
		SELECT id, target_id, min_msg_id, max_msg_id
		FROM parsed_ranges
		WHERE target_id = $1
	`, targetID).Scan(&pr.ID, &pr.TargetID, &pr.MinMsgID, &pr.MaxMsgID)

	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil // no range exists yet
		}
		return nil, fmt.Errorf("get parsed range: %w", err)
	}

	return &pr, nil
}

// UpdateRange creates or extends the parsed range for a target
// uses upsert - if range exists, it extends; otherwise creates new
func (r *RangesRepository) UpdateRange(ctx context.Context, targetID uuid.UUID, minID, maxID int64) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO parsed_ranges (target_id, min_msg_id, max_msg_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (target_id)
		DO UPDATE SET
			min_msg_id = LEAST(parsed_ranges.min_msg_id, $2),
			max_msg_id = GREATEST(parsed_ranges.max_msg_id, $3),
			updated_at = NOW()
	`, targetID, minID, maxID)

	if err != nil {
		return fmt.Errorf("update parsed range: %w", err)
	}

	return nil
}

// GetMaxMessageID returns the maximum parsed message id for a target
// returns 0 if no range exists (meaning all messages are new)
func (r *RangesRepository) GetMaxMessageID(ctx context.Context, targetID uuid.UUID) (int64, error) {
	var maxID int64
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(max_msg_id, 0)
		FROM parsed_ranges
		WHERE target_id = $1
	`, targetID).Scan(&maxID)

	if err != nil {
		if err.Error() == "no rows in result set" {
			return 0, nil // no range exists, all messages are new
		}
		return 0, fmt.Errorf("get max message id: %w", err)
	}

	return maxID, nil
}

// NewFilter creates a message filter for a target based on its parsed range
func (r *RangesRepository) NewFilter(ctx context.Context, targetID uuid.UUID) (*MessageIDFilter, error) {
	pr, err := r.GetRange(ctx, targetID)
	if err != nil {
		return nil, err
	}
	if pr == nil {
		return NewMessageIDFilter(0, 0), nil
	}
	return NewMessageIDFilter(pr.MinMsgID, pr.MaxMsgID), nil
}

// NewSmartFilter creates a smart message filter that checks both range AND job existence
func (r *RangesRepository) NewSmartFilter(ctx context.Context, targetID uuid.UUID, existingJobIDs []int64) (*SmartMessageFilter, error) {
	pr, err := r.GetRange(ctx, targetID)
	if err != nil {
		return nil, err
	}
	if pr == nil {
		return NewSmartMessageFilter(0, 0, existingJobIDs), nil
	}
	return NewSmartMessageFilter(pr.MinMsgID, pr.MaxMsgID, existingJobIDs), nil
}
