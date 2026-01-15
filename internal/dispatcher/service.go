package dispatcher

import (
	"context"
	"errors"
	"fmt"

	"github.com/blockedby/positions-os/internal/logger"
	"github.com/blockedby/positions-os/internal/models"
	"github.com/google/uuid"
)

// TelegramSenderInterface defines the interface for sending via Telegram.
// This allows mocking in tests.
type TelegramSenderInterface interface {
	SendApplication(ctx context.Context, appID uuid.UUID, recipient string) error
}

// DispatcherService is the main orchestrator for sending job applications.
// It routes to the appropriate sender based on the delivery channel.
type DispatcherService struct {
	tgSender    TelegramSenderInterface
	emailSender EmailSenderInterface
	tracker     DeliveryTrackerInterface
	repo        ApplicationsRepository
	log         *logger.Logger
}

// EmailSenderInterface defines the interface for sending applications via email.
// This is a stub from Thread A (Task E.1).
type EmailSenderInterface interface {
	SendApplication(ctx context.Context, appID uuid.UUID, recipient string) error
}

// NewDispatcherService creates a new DispatcherService.
func NewDispatcherService(
	tgSender TelegramSenderInterface,
	emailSender EmailSenderInterface,
	tracker DeliveryTrackerInterface,
	repo ApplicationsRepository,
	log *logger.Logger,
) *DispatcherService {
	return &DispatcherService{
		tgSender:    tgSender,
		emailSender: emailSender,
		tracker:     tracker,
		repo:        repo,
		log:         log,
	}
}

// SendRequest represents a request to send a job application.
type SendRequest struct {
	JobID     uuid.UUID `json:"job_id"`
	Channel   string    `json:"channel"`   // "TG_DM" or "EMAIL"
	Recipient string    `json:"recipient"`
}

// SendApplication routes the send request to the appropriate sender based on channel.
func (s *DispatcherService) SendApplication(ctx context.Context, req *SendRequest) error {
	// Validate request
	if req == nil {
		return errors.New("request cannot be nil")
	}
	if req.JobID == uuid.Nil {
		return errors.New("job ID cannot be empty")
	}
	if req.Recipient == "" {
		return errors.New("recipient cannot be empty")
	}

	// Route based on channel
	switch req.Channel {
	case "TG_DM":
		return s.SendViaTelegram(ctx, req.JobID, req.Recipient)
	case "EMAIL":
		if s.emailSender == nil {
			return errors.New("email sender not configured")
		}
		return s.emailSender.SendApplication(ctx, req.JobID, req.Recipient)
	default:
		return fmt.Errorf("unsupported channel: %s", req.Channel)
	}
}

// SendViaTelegram creates an application and sends it via Telegram.
func (s *DispatcherService) SendViaTelegram(ctx context.Context, jobID uuid.UUID, recipient string) error {
	// Create application record
	app := &models.JobApplication{
		ID:              uuid.New(),
		JobID:           jobID,
		DeliveryChannel: deliveryChannelPtr(models.DeliveryChannelTGDM),
		Recipient:       &recipient,
		DeliveryStatus:  models.DeliveryStatusPending,
	}

	if err := s.repo.Create(ctx, app); err != nil {
		return fmt.Errorf("create application: %w", err)
	}

	// Send via Telegram
	return s.tgSender.SendApplication(ctx, app.ID, recipient)
}

// Helper function to get pointer to DeliveryChannel
func deliveryChannelPtr(c models.DeliveryChannel) *models.DeliveryChannel {
	return &c
}
