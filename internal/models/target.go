// Package models defines shared data types for the application.
package models

import (
	"time"

	"github.com/google/uuid"
)

// ScrapingTargetType represents the type of scraping source.
type ScrapingTargetType string

// ScrapingTargetType constants define the supported scraping source types.
const (
	TargetTypeTGChannel ScrapingTargetType = "TG_CHANNEL"
	TargetTypeTGGroup   ScrapingTargetType = "TG_GROUP"
	TargetTypeTGForum   ScrapingTargetType = "TG_FORUM"
	TargetTypeHHSearch  ScrapingTargetType = "HH_SEARCH"
	TargetTypeLinkedIn  ScrapingTargetType = "LINKEDIN_SEARCH"
)

// ScrapingTarget represents a source for job parsing.
type ScrapingTarget struct {
	ID   uuid.UUID          `json:"id" db:"id"`
	Name string             `json:"name" db:"name"`
	Type ScrapingTargetType `json:"type" db:"type"`
	URL  string             `json:"url" db:"url"`

	// telegram specific
	TGAccessHash *int64 `json:"tg_access_hash,omitempty" db:"tg_access_hash"`
	TGChannelID  *int64 `json:"tg_channel_id,omitempty" db:"tg_channel_id"`

	// parsing config
	Metadata map[string]any `json:"metadata" db:"metadata"`

	// state
	LastScrapedAt *time.Time `json:"last_scraped_at,omitempty" db:"last_scraped_at"`
	LastMessageID *int64     `json:"last_message_id,omitempty" db:"last_message_id"`
	IsActive      bool       `json:"is_active" db:"is_active"`

	// timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// TargetMetadata represents parsing configuration in metadata field.
type TargetMetadata struct {
	Keywords      []string `json:"keywords,omitempty"`
	Limit         int      `json:"limit,omitempty"`
	IncludeTopics bool     `json:"include_topics,omitempty"`
	Until         string   `json:"until,omitempty"` // date string YYYY-MM-DD
}
