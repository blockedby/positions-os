package telegram

import (
	"time"
)

// Message represents a parsed telegram message
type Message struct {
	ID        int       // message id (unique within channel)
	ChannelID int64     // channel id
	Text      string    // message text content
	Date      time.Time // message creation timestamp
	TopicID   *int      // forum topic id (nil for non-forum channels)
	Views     int       // view count
	Forwards  int       // forward count
}

// Topic represents a forum topic
type Topic struct {
	ID         int    // topic id (same as message_thread_id)
	Title      string // topic title
	TopMessage int    // id of last message in topic
	Closed     bool   // whether topic is closed
	Pinned     bool   // whether topic is pinned
}

// Channel represents a telegram channel info
type Channel struct {
	ID         int64  // channel id
	AccessHash int64  // access hash for api calls
	Username   string // channel username (without @)
	Title      string // channel title
	IsForum    bool   // whether it's a forum-type supergroup
}

// ParsedRange represents a range of scraped message ids
type ParsedRange struct {
	MinMsgID int64 // lowest message id scraped
	MaxMsgID int64 // highest message id scraped
}

// ScrapeStats tracks statistics during scraping
type ScrapeStats struct {
	TotalFetched int // total messages fetched from telegram
	NewMessages  int // new messages saved to database
	SkippedOld   int // messages skipped due to deduplication
	SkippedEmpty int // messages skipped due to empty content
	Errors       int // error count
}
