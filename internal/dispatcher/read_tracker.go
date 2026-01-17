package dispatcher

import (
	"context"
	"fmt"
	"sync"

	"github.com/blockedby/positions-os/internal/logger"
	"github.com/google/uuid"
)

// ReadTracker handles Telegram read receipt detection
// Maps Telegram message IDs to application IDs for automatic status updates
type ReadTracker struct {
	repo         *DeliveryTracker
	messageToApp map[int64]uuid.UUID // Telegram msg ID → App ID
	mu           sync.RWMutex
	log          *logger.Logger
}

// NewReadTracker creates a new read receipt tracker
func NewReadTracker(tracker *DeliveryTracker, log *logger.Logger) *ReadTracker {
	return &ReadTracker{
		repo:         tracker,
		messageToApp: make(map[int64]uuid.UUID),
		log:          log,
	}
}

// OnMessageRead handles updateReadHistoryOutbox from Telegram
// This is called when the peer reads our messages
func (rt *ReadTracker) OnMessageRead(ctx context.Context, peerUserID int64, maxMsgID int64) error {
	rt.mu.RLock()
	appID, found := rt.messageToApp[maxMsgID]
	rt.mu.RUnlock()

	if !found {
		// Not our message (or already processed)
		return nil
	}

	// Automatically mark as READ
	err := rt.repo.UpdateStatus(ctx, appID, StatusRead)
	if err != nil {
		return fmt.Errorf("update status to READ: %w", err)
	}

	// Clean up mapping
	rt.mu.Lock()
	delete(rt.messageToApp, maxMsgID)
	rt.mu.Unlock()

	rt.log.Info().
		Int64("peer_user_id", peerUserID).
		Int64("max_msg_id", maxMsgID).
		Str("app_id", appID.String()).
		Msg("message read by recipient")

	return nil
}

// RegisterSentMessage stores mapping for later read detection
// Call this after successfully sending a message via Telegram
func (rt *ReadTracker) RegisterSentMessage(msgID int64, appID uuid.UUID) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.messageToApp[msgID] = appID

	rt.log.Debug().
		Int64("msg_id", msgID).
		Str("app_id", appID.String()).
		Msg("registered sent message for read tracking")
}

// UnregisterMessage removes a message from tracking
// Useful if delivery failed or message was deleted
func (rt *ReadTracker) UnregisterMessage(msgID int64) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	delete(rt.messageToApp, msgID)

	rt.log.Debug().
		Int64("msg_id", msgID).
		Msg("unregistered message from read tracking")
}

// GetApplicationID returns the application ID for a given message ID
func (rt *ReadTracker) GetApplicationID(msgID int64) (uuid.UUID, bool) {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	appID, found := rt.messageToApp[msgID]
	return appID, found
}

// Clear removes all message mappings
// Useful for cleanup or reset
func (rt *ReadTracker) Clear() {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.messageToApp = make(map[int64]uuid.UUID)
	rt.log.Info().Msg("cleared all read tracking mappings")
}

// Count returns the number of tracked messages
func (rt *ReadTracker) Count() int {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	return len(rt.messageToApp)
}
