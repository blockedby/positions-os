package collector

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// validation errors
var (
	ErrChannelRequired = errors.New("either target_id or channel is required")
	ErrChannelNotFound = errors.New("channel not found")
	ErrNotAChannel     = errors.New("specified username is not a channel")
	ErrInvalidDate     = errors.New("until date must be in YYYY-MM-DD format")
	ErrFutureDate      = errors.New("until date cannot be in the future")
	ErrInvalidLimit    = errors.New("limit must be non-negative")
	ErrTopicsForForum  = errors.New("topic_ids can only be used with TG_FORUM targets")
	ErrTopicNotFound   = errors.New("one or more topic_ids not found in the forum")
)

// ScrapeRequest represents a request to scrape a telegram channel
type ScrapeRequest struct {
	// TargetID - id from scraping_targets table.
	// if specified, channel is ignored.
	TargetID *uuid.UUID `json:"target_id,omitempty"`

	// Channel - username (with or without @).
	// used if targetid is not specified.
	Channel string `json:"channel,omitempty"`

	// Limit - maximum messages to scrape.
	// 0 means no limit.
	Limit int `json:"limit,omitempty"`

	// Until - date to scrape until (YYYY-MM-DD).
	// messages older than this are ignored.
	Until string `json:"until,omitempty"`

	// TopicIDs - list of forum topic ids to scrape.
	// only for TG_FORUM targets. empty means all topics.
	TopicIDs []int `json:"topic_ids,omitempty"`
}

// Validate performs basic validation of the request
// does not check if channel exists (that requires network call)
func (r *ScrapeRequest) Validate() error {
	// check that we have a source
	if r.TargetID == nil && r.Channel == "" {
		return ErrChannelRequired
	}

	// normalize channel name
	r.Channel = strings.TrimPrefix(r.Channel, "@")

	// validate limit
	if r.Limit < 0 {
		return ErrInvalidLimit
	}

	// validate until date
	if r.Until != "" {
		until, err := time.Parse("2006-01-02", r.Until)
		if err != nil {
			return ErrInvalidDate
		}
		if until.After(time.Now()) {
			return ErrFutureDate
		}
	}

	return nil
}

// UntilTime returns the Until date as *time.Time
// returns nil if Until is empty or invalid
func (r *ScrapeRequest) UntilTime() *time.Time {
	if r.Until == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", r.Until)
	if err != nil {
		return nil
	}
	return &t
}

// ScrapeResponse represents response to scrape request
type ScrapeResponse struct {
	ScrapeID  uuid.UUID  `json:"scrape_id"`
	Status    string     `json:"status"` // "running" | "completed" | "failed"
	Target    TargetInfo `json:"target"`
	StartedAt time.Time  `json:"started_at"`
}

// TargetInfo contains brief info about scraping target
type TargetInfo struct {
	ID      uuid.UUID `json:"id"`
	Name    string    `json:"name"`
	Channel string    `json:"channel"`
}
