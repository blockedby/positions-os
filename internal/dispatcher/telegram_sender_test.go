package dispatcher

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/blockedby/positions-os/internal/logger"
	"github.com/blockedby/positions-os/internal/repository"
	"github.com/blockedby/positions-os/internal/telegram"
	"github.com/blockedby/positions-os/internal/web"
)

// TestTelegramSender_NewTelegramSender tests the constructor
func TestTelegramSender_NewTelegramSender(t *testing.T) {
	repo := repository.NewApplicationsRepository(nil, &logger.Logger{})
	hub := web.NewHub()
	log := &logger.Logger{}

	tracker := NewDeliveryTracker(repo, hub, log)
	readTracker := NewReadTracker(tracker, log)

	// Create a mock telegram manager (or use the real one without client)
	tgManager := &telegram.Manager{}

	sender := NewTelegramSender(repo, tracker, readTracker, tgManager, log)

	assert.NotNil(t, sender, "NewTelegramSender should return non-nil")
	assert.NotNil(t, sender.repo, "Sender should have a repo")
	assert.NotNil(t, sender.tracker, "Sender should have a tracker")
	assert.NotNil(t, sender.readTracker, "Sender should have a read tracker")
	assert.NotNil(t, sender.tgManager, "Sender should have a telegram manager")
}

// TestTelegramSender_SendApplication_NotImplemented tests that sending returns an error
func TestTelegramSender_SendApplication_NotImplemented(t *testing.T) {
	repo := repository.NewApplicationsRepository(nil, &logger.Logger{})
	hub := web.NewHub()
	log := &logger.Logger{}

	tracker := NewDeliveryTracker(repo, hub, log)
	readTracker := NewReadTracker(tracker, log)
	tgManager := &telegram.Manager{}

	sender := NewTelegramSender(repo, tracker, readTracker, tgManager, log)

	appID := uuid.New()
	err := sender.SendApplication(context.Background(), appID)

	assert.Error(t, err, "SendApplication should return error")
	assert.Contains(t, err.Error(), "not yet implemented", "Error should mention not implemented")
}

// TestTelegramSender_ReadReceiptCallback tests that read receipts are wired up
func TestTelegramSender_ReadReceiptCallback(t *testing.T) {
	repo := repository.NewApplicationsRepository(nil, &logger.Logger{})
	hub := web.NewHub()
	log := &logger.Logger{}

	tracker := NewDeliveryTracker(repo, hub, log)
	readTracker := NewReadTracker(tracker, log)
	tgManager := &telegram.Manager{}

	sender := NewTelegramSender(repo, tracker, readTracker, tgManager, log)

	// Verify read tracker is accessible
	assert.Equal(t, readTracker, sender.GetReadTracker(), "Read tracker should be accessible")
	assert.Equal(t, tracker, sender.GetTracker(), "Tracker should be accessible")
}
