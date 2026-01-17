// Package telegram provides Telegram MTProto client wrapper.
package telegram

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/blockedby/positions-os/internal/logger"
	"github.com/celestix/gotgproto"
	"github.com/gotd/td/tg"
)

// Client wraps gotgproto client and provides high-level telegram operations.
// It is now resilient and uses the Manager to access the underlying protocol client.
type Client struct {
	manager     *Manager
	rateLimiter *RateLimiter
	log         *logger.Logger
}

// NewClient creates a new telegram client wrapper using the Manager.
func NewClient(manager *Manager) *Client {
	return &Client{
		manager:     manager,
		rateLimiter: DefaultRateLimiter(),
		log:         logger.Get(),
	}
}

// Close stops the client via the manager.
func (c *Client) Close() {
	if c.manager != nil {
		c.manager.Stop()
	}
}

// GetStatus returns the current status of the telegram client.
func (c *Client) GetStatus() Status {
	return c.manager.GetStatus()
}

// StartQR starts the QR login flow effectively proxying to the manager.
func (c *Client) StartQR(ctx context.Context, onQRCode func(url string)) error {
	return c.manager.StartQR(ctx, onQRCode)
}

// IsQRInProgress returns true if a QR login flow is currently in progress.
func (c *Client) IsQRInProgress() bool {
	return c.manager.IsQRInProgress()
}

// CancelQR cancels any ongoing QR login flow.
func (c *Client) CancelQR() {
	c.manager.CancelQR()
}

// getProto returns the current protocol client if available.
func (c *Client) getProto() (*gotgproto.Client, error) {
	proto := c.manager.GetClient()
	if proto == nil {
		return nil, fmt.Errorf("telegram client not authorized")
	}
	return proto, nil
}

// API returns the raw tg.Client for direct API calls.
func (c *Client) API() (*tg.Client, error) {
	proto, err := c.getProto()
	if err != nil {
		return nil, err
	}
	return proto.API(), nil
}

// ResolveChannel resolves channel username to Channel info
// username can be with or without @ prefix
func (c *Client) ResolveChannel(ctx context.Context, username string) (*Channel, error) {
	// strip @ prefix if present
	username = strings.TrimPrefix(username, "@")

	c.log.Debug().Str("username", username).Msg("telegram: waiting for rate limiter")
	if err := c.rateLimiter.Wait(ctx); err != nil {
		c.log.Error().Err(err).Msg("telegram: rate limiter wait failed")
		return nil, err
	}

	c.log.Info().Str("username", username).Msg("telegram: resolving channel username")
	api, err := c.API()
	if err != nil {
		return nil, err
	}
	resolved, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
		Username: username,
	})
	if err != nil {
		if wait := c.checkFloodWait(err); wait > 0 {
			c.log.Warn().Int("wait_seconds", wait).Msg("telegram: FLOOD_WAIT detected, updating rate limiter")
			c.rateLimiter.SetFloodWait(wait)
		}
		c.log.Error().Err(err).Str("username", username).Msg("telegram: failed to resolve username")
		return nil, fmt.Errorf("resolve username %s: %w", username, err)
	}

	if len(resolved.Chats) == 0 {
		return nil, fmt.Errorf("channel not found: %s", username)
	}

	ch, ok := resolved.Chats[0].(*tg.Channel)
	if !ok {
		return nil, fmt.Errorf("not a channel: %s", username)
	}

	api, err = c.API()
	if err != nil {
		return nil, fmt.Errorf("get api: %w", err)
	}
	fullCh, err := api.ChannelsGetFullChannel(ctx, &tg.InputChannel{
		ChannelID:  ch.ID,
		AccessHash: ch.AccessHash,
	})
	if err != nil {
		return nil, fmt.Errorf("get full channel: %w", err)
	}

	chFull, ok := fullCh.FullChat.(*tg.ChannelFull)
	if !ok {
		return nil, fmt.Errorf("unexpected channel type")
	}

	// forum flag is at position 30 in ChannelFull flags
	isForum := chFull.Flags.Has(30)

	return &Channel{
		ID:         ch.ID,
		AccessHash: ch.AccessHash,
		Username:   username,
		Title:      ch.Title,
		IsForum:    isForum,
	}, nil
}

// ChannelExists checks if channel username exists and is accessible
func (c *Client) ChannelExists(ctx context.Context, username string) (bool, error) {
	_, err := c.ResolveChannel(ctx, username)
	if err != nil {
		// check if it's a "not found" error
		if strings.Contains(err.Error(), "not found") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// GetMessages fetches messages from a channel
// offsetID: start from this message id (0 = newest messages)
// limit: max number of messages to fetch (max 100)
func (c *Client) GetMessages(ctx context.Context, channel *Channel, offsetID int, limit int) ([]Message, error) {
	if limit > 100 {
		limit = 100 // telegram api limit
	}

	c.log.Debug().Int64("channel_id", channel.ID).Int("offset_id", offsetID).Int("limit", limit).Msg("telegram: waiting for rate limiter before GetMessages")
	if err := c.rateLimiter.Wait(ctx); err != nil {
		c.log.Error().Err(err).Msg("telegram: rate limiter wait failed")
		return nil, err
	}

	c.log.Info().Int64("channel_id", channel.ID).Int("offset_id", offsetID).Int("limit", limit).Msg("telegram: calling MessagesGetHistory API")
	api, err := c.API()
	if err != nil {
		return nil, err
	}
	history, err := api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
		Peer: &tg.InputPeerChannel{
			ChannelID:  channel.ID,
			AccessHash: channel.AccessHash,
		},
		OffsetID: offsetID,
		Limit:    limit,
	})
	if err != nil {
		if wait := c.checkFloodWait(err); wait > 0 {
			c.log.Warn().Int("wait_seconds", wait).Msg("telegram: FLOOD_WAIT detected in GetMessages, updating rate limiter")
			c.rateLimiter.SetFloodWait(wait)
		}
		c.log.Error().Err(err).Int("offset_id", offsetID).Msg("telegram: MessagesGetHistory failed")
		return nil, fmt.Errorf("get history: %w", err)
	}

	return c.extractMessages(history, channel)
}

// GetTopics returns list of forum topics for a channel
// returns empty list if channel is not a forum
func (c *Client) GetTopics(ctx context.Context, channel *Channel) ([]Topic, error) {
	if !channel.IsForum {
		return []Topic{}, nil
	}

	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, err
	}

	api, err := c.API()
	if err != nil {
		return nil, err
	}
	result, err := api.MessagesGetForumTopics(ctx, &tg.MessagesGetForumTopicsRequest{
		Peer: &tg.InputPeerChannel{
			ChannelID:  channel.ID,
			AccessHash: channel.AccessHash,
		},
		Limit: 100, // fetch up to 100 topics
	})
	if err != nil {
		if wait := c.checkFloodWait(err); wait > 0 {
			c.rateLimiter.SetFloodWait(wait)
		}
		return nil, fmt.Errorf("get forum topics: %w", err)
	}

	// result is *tg.MessagesForumTopics
	topics := result

	var out []Topic
	for _, t := range topics.Topics {
		topic, ok := t.(*tg.ForumTopic)
		if !ok {
			continue
		}

		out = append(out, Topic{
			ID:         topic.ID,
			Title:      topic.Title,
			TopMessage: topic.TopMessage,
			Closed:     topic.Closed,
			Pinned:     topic.Pinned,
		})
	}

	return out, nil
}

// GetTopicMessages fetches messages from a specific forum topic
func (c *Client) GetTopicMessages(ctx context.Context, channel *Channel, topicID int, offsetID int, limit int) ([]Message, error) {
	if limit > 100 {
		limit = 100
	}

	api, err := c.API()
	if err != nil {
		return nil, err
	}
	result, err := api.MessagesGetReplies(ctx, &tg.MessagesGetRepliesRequest{
		Peer: &tg.InputPeerChannel{
			ChannelID:  channel.ID,
			AccessHash: channel.AccessHash,
		},
		MsgID:    topicID, // topic id is the message id
		OffsetID: offsetID,
		Limit:    limit,
	})
	if err != nil {
		if wait := c.checkFloodWait(err); wait > 0 {
			c.rateLimiter.SetFloodWait(wait)
		}
		return nil, fmt.Errorf("get topic messages: %w", err)
	}

	return c.extractMessages(result, channel)
}

// extractMessages converts telegram message response to our Message type
func (c *Client) extractMessages(messagesClass tg.MessagesMessagesClass, channel *Channel) ([]Message, error) {
	var messages []Message

	switch h := messagesClass.(type) {
	case *tg.MessagesChannelMessages:
		for _, msg := range h.Messages {
			if m := c.parseMessage(msg, channel); m != nil {
				messages = append(messages, *m)
			}
		}
	case *tg.MessagesMessages:
		for _, msg := range h.Messages {
			if m := c.parseMessage(msg, channel); m != nil {
				messages = append(messages, *m)
			}
		}
	}

	return messages, nil
}

// parseMessage converts a single telegram message to our Message type
func (c *Client) parseMessage(msg tg.MessageClass, channel *Channel) *Message {
	m, ok := msg.(*tg.Message)
	if !ok {
		return nil
	}

	// extract topic id from reply header if it's a forum message
	var topicID *int
	if m.ReplyTo != nil {
		if replyHeader, ok := m.ReplyTo.(*tg.MessageReplyHeader); ok {
			if replyHeader.ForumTopic {
				tid := replyHeader.ReplyToMsgID
				topicID = &tid
			}
		}
	}

	return &Message{
		ID:        m.ID,
		ChannelID: channel.ID,
		Text:      m.Message,
		Date:      time.Unix(int64(m.Date), 0),
		TopicID:   topicID,
		Views:     m.Views,
		Forwards:  m.Forwards,
	}
}

// checkFloodWait checks if error is a FLOOD_WAIT error and returns wait seconds
func (c *Client) checkFloodWait(err error) int {
	if err == nil {
		return 0
	}

	// gotgproto/gotd errors are usually wrapped
	// we check for specific error string as it's the most reliable way
	// without deep coupling to gotd/tg definition of FloodWait
	str := err.Error()
	if strings.Contains(str, "FLOOD_WAIT_") {
		// format is usually FLOOD_WAIT_X where X is seconds
		var seconds int
		// try to parse from string, e.g. "rpc error: code 420: FLOOD_WAIT_15"
		parts := strings.Split(str, "FLOOD_WAIT_")
		if len(parts) > 1 {
			// take the number part
			numStr := strings.TrimSpace(parts[1])
			// sometimes it has " (caused by...)" or other suffix, simple scan
			_, _ = fmt.Sscanf(numStr, "%d", &seconds)
			return seconds
		}
	}
	return 0
}
