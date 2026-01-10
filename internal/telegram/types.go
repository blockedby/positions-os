package telegram

import (
	"time"
)

// Message represents a parsed telegram message
type Message struct {
	ID        int       `json:"id"`
	ChannelID int64     `json:"channel_id"`
	Text      string    `json:"text"`
	Date      time.Time `json:"date"`
	TopicID   *int      `json:"topic_id"`
	Views     int       `json:"views"`
	Forwards  int       `json:"forwards"`
}

// Topic represents a forum topic
type Topic struct {
	ID         int    `json:"id"`
	Title      string `json:"title"`
	TopMessage int    `json:"top_message"`
	Closed     bool   `json:"closed"`
	Pinned     bool   `json:"pinned"`
}

// Channel represents a telegram channel info
type Channel struct {
	ID         int64  `json:"id"`
	AccessHash int64  `json:"access_hash"`
	Username   string `json:"username"`
	Title      string `json:"title"`
	IsForum    bool   `json:"is_forum"`
}

// ParsedRange represents a range of scraped message ids
type ParsedRange struct {
	MinMsgID int64 `json:"min_msg_id"`
	MaxMsgID int64 `json:"max_msg_id"`
}

// ScrapeStats tracks statistics during scraping
type ScrapeStats struct {
	TotalFetched int `json:"total_fetched"`
	NewMessages  int `json:"new_messages"`
	SkippedOld   int `json:"skipped_old"`
	SkippedEmpty int `json:"skipped_empty"`
	Errors       int `json:"errors"`
}
