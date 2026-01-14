package dispatcher

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/blockedby/positions-os/internal/logger"
	"github.com/blockedby/positions-os/internal/repository"
	"github.com/blockedby/positions-os/internal/telegram"
)

// TelegramSender handles sending job applications via Telegram DM.
// This is a stub for Thread A - full implementation in Thread B.
type TelegramSender struct {
	repo        *repository.ApplicationsRepository
	tracker     *DeliveryTracker
	readTracker *ReadTracker
	tgManager   *telegram.Manager
	log         *logger.Logger
}

// NewTelegramSender creates a new Telegram sender.
// The tgManager is used to send messages and receive read receipts.
func NewTelegramSender(
	repo *repository.ApplicationsRepository,
	tracker *DeliveryTracker,
	readTracker *ReadTracker,
	tgManager *telegram.Manager,
	log *logger.Logger,
) *TelegramSender {
	sender := &TelegramSender{
		repo:        repo,
		tracker:     tracker,
		readTracker: readTracker,
		tgManager:   tgManager,
		log:         log,
	}

	// Wire up read receipt callback
	tgManager.SetReadReceiptCallback(readTracker.OnMessageRead)

	return sender
}

// SendApplication sends an application via Telegram DM.
// This is a stub implementation - Thread B will implement actual sending.
func (s *TelegramSender) SendApplication(ctx context.Context, appID uuid.UUID) error {
	// TODO (Thread B): Implement actual Telegram sending:
	// 1. Call tracker.TrackStart(ctx, appID)
	// 2. Get application from repo
	// 3. Verify delivery channel is TG_DM
	// 4. Verify recipient is set
	// 5. Parse recipient (@username or user ID)
	// 6. Upload PDF files (resume, cover letter)
	// 7. Send files via Telegram DM
	// 8. Get message ID from response
	// 9. Call readTracker.RegisterSentMessage(msgID, appID)
	// 10. Call tracker.TrackSuccess(ctx, appID)
	// OR call tracker.TrackFailure on error

	return fmt.Errorf("telegram sender not yet implemented: see Thread B")
}

// GetReadTracker returns the read tracker for testing purposes.
func (s *TelegramSender) GetReadTracker() *ReadTracker {
	return s.readTracker
}

// GetTracker returns the delivery tracker for testing purposes.
func (s *TelegramSender) GetTracker() *DeliveryTracker {
	return s.tracker
}
