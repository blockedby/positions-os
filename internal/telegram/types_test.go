package telegram

import (
	"testing"
	"time"
)

// test message type initialization
func TestMessage_Fields(t *testing.T) {
	tests := []struct {
		name     string
		msg      Message
		wantID   int
		wantText string
	}{
		{
			name: "basic message with all fields",
			msg: Message{
				ID:        123,
				ChannelID: 456789,
				Text:      "hello world",
				Date:      time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
				TopicID:   nil,
				Views:     100,
				Forwards:  5,
			},
			wantID:   123,
			wantText: "hello world",
		},
		{
			name: "message with topic id",
			msg: Message{
				ID:        456,
				ChannelID: 789,
				Text:      "forum message",
				Date:      time.Now(),
				TopicID:   intPtr(15),
				Views:     50,
				Forwards:  0,
			},
			wantID:   456,
			wantText: "forum message",
		},
		{
			name: "empty message",
			msg: Message{
				ID:        0,
				ChannelID: 0,
				Text:      "",
				Date:      time.Time{},
			},
			wantID:   0,
			wantText: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.msg.ID != tt.wantID {
				t.Errorf("Message.ID = %d, want %d", tt.msg.ID, tt.wantID)
			}
			if tt.msg.Text != tt.wantText {
				t.Errorf("Message.Text = %q, want %q", tt.msg.Text, tt.wantText)
			}
		})
	}
}

// test topic type
func TestTopic_Status(t *testing.T) {
	tests := []struct {
		name       string
		topic      Topic
		wantClosed bool
		wantPinned bool
	}{
		{
			name: "open topic",
			topic: Topic{
				ID:         1,
				Title:      "general",
				TopMessage: 100,
				Closed:     false,
				Pinned:     false,
			},
			wantClosed: false,
			wantPinned: false,
		},
		{
			name: "closed pinned topic",
			topic: Topic{
				ID:         15,
				Title:      "archived",
				TopMessage: 500,
				Closed:     true,
				Pinned:     true,
			},
			wantClosed: true,
			wantPinned: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.topic.Closed != tt.wantClosed {
				t.Errorf("Topic.Closed = %v, want %v", tt.topic.Closed, tt.wantClosed)
			}
			if tt.topic.Pinned != tt.wantPinned {
				t.Errorf("Topic.Pinned = %v, want %v", tt.topic.Pinned, tt.wantPinned)
			}
		})
	}
}

// test channel type
func TestChannel_IsForum(t *testing.T) {
	tests := []struct {
		name      string
		channel   Channel
		wantForum bool
	}{
		{
			name: "regular channel",
			channel: Channel{
				ID:         123456789,
				AccessHash: 987654321,
				Username:   "golang_jobs",
				Title:      "Go Jobs",
				IsForum:    false,
			},
			wantForum: false,
		},
		{
			name: "forum supergroup",
			channel: Channel{
				ID:         111222333,
				AccessHash: 444555666,
				Username:   "go_forum",
				Title:      "Go Forum",
				IsForum:    true,
			},
			wantForum: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.channel.IsForum != tt.wantForum {
				t.Errorf("Channel.IsForum = %v, want %v", tt.channel.IsForum, tt.wantForum)
			}
		})
	}
}

// test parsed range
func TestParsedRange_Valid(t *testing.T) {
	tests := []struct {
		name      string
		r         ParsedRange
		wantValid bool
	}{
		{
			name: "valid range",
			r: ParsedRange{
				MinMsgID: 100,
				MaxMsgID: 200,
			},
			wantValid: true,
		},
		{
			name: "single message range",
			r: ParsedRange{
				MinMsgID: 100,
				MaxMsgID: 100,
			},
			wantValid: true,
		},
		{
			name: "empty range",
			r: ParsedRange{
				MinMsgID: 0,
				MaxMsgID: 0,
			},
			wantValid: true, // 0-0 is valid (no messages parsed yet)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// a valid range has min <= max
			valid := tt.r.MinMsgID <= tt.r.MaxMsgID
			if valid != tt.wantValid {
				t.Errorf("ParsedRange valid = %v, want %v", valid, tt.wantValid)
			}
		})
	}
}

// test scrape stats
func TestScrapeStats_Summary(t *testing.T) {
	stats := ScrapeStats{
		TotalFetched: 100,
		NewMessages:  25,
		SkippedOld:   70,
		SkippedEmpty: 5,
		Errors:       0,
	}

	// verify all fetched messages are accounted for
	accounted := stats.NewMessages + stats.SkippedOld + stats.SkippedEmpty
	if accounted != stats.TotalFetched {
		t.Errorf("stats do not add up: new(%d) + skipped_old(%d) + skipped_empty(%d) = %d, want %d",
			stats.NewMessages, stats.SkippedOld, stats.SkippedEmpty, accounted, stats.TotalFetched)
	}
}

// helper to create int pointer
func intPtr(i int) *int {
	return &i
}
