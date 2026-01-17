package dispatcher

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/blockedby/positions-os/internal/logger"
)

// TestReadTracker_NewReadTracker tests the constructor
func TestReadTracker_NewReadTracker(t *testing.T) {
	log := &logger.Logger{}

	// Test that ReadTracker can be created
	rt := &ReadTracker{
		repo:         nil,
		messageToApp: make(map[int64]uuid.UUID),
		log:          log,
	}

	assert.NotNil(t, rt, "ReadTracker should be non-nil")
	assert.NotNil(t, rt.messageToApp, "ReadTracker should have a message map")
	assert.NotNil(t, rt.log, "ReadTracker should have a log")
}

// TestReadTracker_RegisterSentMessage tests message registration
func TestReadTracker_RegisterSentMessage(t *testing.T) {
	log := &logger.Logger{}
	rt := &ReadTracker{
		messageToApp: make(map[int64]uuid.UUID),
		log:          log,
	}

	msgID := int64(12345)
	appID := uuid.New()

	rt.RegisterSentMessage(msgID, appID)

	retrievedID, found := rt.GetApplicationID(msgID)
	assert.True(t, found, "Message should be found")
	assert.Equal(t, appID, retrievedID, "Application ID should match")
	assert.Equal(t, 1, rt.Count(), "Count should be 1")
}

// TestReadTracker_GetApplicationID_NotFound tests getting non-existent message
func TestReadTracker_GetApplicationID_NotFound(t *testing.T) {
	log := &logger.Logger{}
	rt := &ReadTracker{
		messageToApp: make(map[int64]uuid.UUID),
		log:          log,
	}

	_, found := rt.GetApplicationID(999)
	assert.False(t, found, "Non-existent message should not be found")
}

// TestReadTracker_UnregisterMessage tests unregistering a message
func TestReadTracker_UnregisterMessage(t *testing.T) {
	log := &logger.Logger{}
	rt := &ReadTracker{
		messageToApp: make(map[int64]uuid.UUID),
		log:          log,
	}

	msgID := int64(12345)
	appID := uuid.New()

	rt.RegisterSentMessage(msgID, appID)
	assert.Equal(t, 1, rt.Count(), "Count should be 1")

	rt.UnregisterMessage(msgID)
	assert.Equal(t, 0, rt.Count(), "Count should be 0 after unregister")

	_, found := rt.GetApplicationID(msgID)
	assert.False(t, found, "Message should not be found after unregister")
}

// TestReadTracker_Clear tests clearing all messages
func TestReadTracker_Clear(t *testing.T) {
	log := &logger.Logger{}
	rt := &ReadTracker{
		messageToApp: make(map[int64]uuid.UUID),
		log:          log,
	}

	// Register multiple messages
	for i := 0; i < 5; i++ {
		rt.RegisterSentMessage(int64(i), uuid.New())
	}
	assert.Equal(t, 5, rt.Count(), "Count should be 5")

	rt.Clear()
	assert.Equal(t, 0, rt.Count(), "Count should be 0 after clear")
}

// TestReadTracker_OnMessageRead_UnknownMessage tests handling read for unknown message
func TestReadTracker_OnMessageRead_UnknownMessage(t *testing.T) {
	log := &logger.Logger{}
	rt := &ReadTracker{
		messageToApp: make(map[int64]uuid.UUID),
		log:          log,
	}

	err := rt.OnMessageRead(context.Background(), 12345, 99999)
	assert.NoError(t, err, "OnMessageRead should succeed even for unknown messages")
}
