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
	maxParsed int64
}

// NewMessageIDFilter creates a filter with max parsed message id
func NewMessageIDFilter(maxParsed int64) *MessageIDFilter {
	return &MessageIDFilter{maxParsed: maxParsed}
}

// FilterNew returns only message ids that are newer than max parsed
func (f *MessageIDFilter) FilterNew(messageIDs []int64) []int64 {
	if len(messageIDs) == 0 {
		return []int64{}
	}

	var newIDs []int64
	for _, id := range messageIDs {
		if id > f.maxParsed {
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
	maxID, err := r.GetMaxMessageID(ctx, targetID)
	if err != nil {
		return nil, err
	}
	return NewMessageIDFilter(maxID), nil
}
