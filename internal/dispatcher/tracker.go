package dispatcher

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/blockedby/positions-os/internal/logger"
	"github.com/blockedby/positions-os/internal/models"
	"github.com/blockedby/positions-os/internal/repository"
	"github.com/blockedby/positions-os/internal/web"
	"github.com/google/uuid"
)

// DeliveryStatus represents the delivery state of an application
type DeliveryStatus = models.DeliveryStatus

const (
	StatusPending   DeliveryStatus = models.DeliveryStatusPending
	StatusSending   DeliveryStatus = "SENDING" // Additional transient status
	StatusSent      DeliveryStatus = models.DeliveryStatusSent
	StatusDelivered DeliveryStatus = models.DeliveryStatusDelivered
	StatusRead      DeliveryStatus = models.DeliveryStatusRead
	StatusFailed    DeliveryStatus = models.DeliveryStatusFailed
)

// DeliveryTracker is the concrete implementation of DeliveryTrackerInterface
type DeliveryTracker struct {
	repo *repository.ApplicationsRepository
	hub  *web.Hub
	log  *logger.Logger
}

// NewDeliveryTracker creates a new delivery tracker
func NewDeliveryTracker(repo *repository.ApplicationsRepository, hub *web.Hub, log *logger.Logger) *DeliveryTracker {
	return &DeliveryTracker{
		repo: repo,
		hub:  hub,
		log:  log,
	}
}

// StatusChangeEvent represents a status change event
type StatusChangeEvent struct {
	Type           string    `json:"type"`
	ApplicationID  string    `json:"application_id"`
	PreviousStatus string    `json:"previous_status,omitempty"`
	CurrentStatus  string    `json:"current_status"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ProgressEvent represents a progress update during sending
type ProgressEvent struct {
	Type          string `json:"type"`
	ApplicationID string `json:"application_id"`
	Step          string `json:"step"`
	Progress      int    `json:"progress"`
	Message       string `json:"message,omitempty"`
}

// FailureEvent represents a delivery failure
type FailureEvent struct {
	Type          string `json:"type"`
	ApplicationID string `json:"application_id"`
	Error         string `json:"error"`
	Retryable     bool   `json:"retryable"`
}

// TrackStart marks delivery as started (PENDING → SENDING)
func (t *DeliveryTracker) TrackStart(ctx context.Context, appID uuid.UUID) error {
	app, err := t.repo.GetByID(ctx, appID)
	if err != nil {
		return fmt.Errorf("get application: %w", err)
	}
	if app == nil {
		return fmt.Errorf("application not found: %s", appID)
	}

	previousStatus := app.DeliveryStatus

	// Update to SENDING
	err = t.repo.UpdateDeliveryStatus(ctx, appID, StatusSending)
	if err != nil {
		return fmt.Errorf("update status to SENDING: %w", err)
	}

	// Broadcast WebSocket event
	t.hub.Broadcast(StatusChangeEvent{
		Type:           "dispatcher.status_changed",
		ApplicationID:  appID.String(),
		PreviousStatus: string(previousStatus),
		CurrentStatus:  string(StatusSending),
		UpdatedAt:      time.Now(),
	})

	t.log.Info().
		Str("app_id", appID.String()).
		Str("from", string(previousStatus)).
		Str("to", string(StatusSending)).
		Msg("delivery status changed")

	return nil
}

// TrackSuccess marks delivery as successful (SENDING → SENT)
func (t *DeliveryTracker) TrackSuccess(ctx context.Context, appID uuid.UUID) error {
	app, err := t.repo.GetByID(ctx, appID)
	if err != nil {
		return fmt.Errorf("get application: %w", err)
	}
	if app == nil {
		return fmt.Errorf("application not found: %s", appID)
	}

	previousStatus := app.DeliveryStatus

	// Mark as sent (sets sent_at timestamp)
	err = t.repo.MarkSent(ctx, appID)
	if err != nil {
		return fmt.Errorf("mark sent: %w", err)
	}

	// Broadcast WebSocket event
	t.hub.Broadcast(StatusChangeEvent{
		Type:           "dispatcher.status_changed",
		ApplicationID:  appID.String(),
		PreviousStatus: string(previousStatus),
		CurrentStatus:  string(StatusSent),
		UpdatedAt:      time.Now(),
	})

	t.log.Info().
		Str("app_id", appID.String()).
		Str("from", string(previousStatus)).
		Str("to", string(StatusSent)).
		Msg("delivery status changed")

	return nil
}

// TrackFailure marks delivery as failed (any → FAILED)
// Stores error message in recruiter_response field
func (t *DeliveryTracker) TrackFailure(ctx context.Context, appID uuid.UUID, err error) error {
	app, getErr := t.repo.GetByID(ctx, appID)
	if getErr != nil {
		return fmt.Errorf("get application: %w", getErr)
	}
	if app == nil {
		return fmt.Errorf("application not found: %s", appID)
	}

	previousStatus := app.DeliveryStatus
	errorMsg := err.Error()

	// Update status to FAILED
	updateErr := t.repo.UpdateDeliveryStatus(ctx, appID, StatusFailed)
	if updateErr != nil {
		return fmt.Errorf("update status to FAILED: %w", updateErr)
	}

	// TODO: Store error in recruiter_response field

	// Broadcast WebSocket event
	t.hub.Broadcast(FailureEvent{
		Type:          "dispatcher.failed",
		ApplicationID: appID.String(),
		Error:         errorMsg,
		Retryable:     isRetryable(err),
	})

	t.log.Error().
		Str("app_id", appID.String()).
		Str("from", string(previousStatus)).
		Str("to", string(StatusFailed)).
		Str("error", errorMsg).
		Msg("delivery failed")

	return nil
}

// TrackProgress reports intermediate progress (during SENDING)
func (t *DeliveryTracker) TrackProgress(ctx context.Context, appID uuid.UUID, step string, progress int) error {
	t.hub.Broadcast(ProgressEvent{
		Type:          "dispatcher.progress",
		ApplicationID: appID.String(),
		Step:          step,
		Progress:      progress,
	})

	t.log.Debug().
		Str("app_id", appID.String()).
		Str("step", step).
		Int("progress", progress).
		Msg("delivery progress")

	return nil
}

// UpdateStatus manually updates status (for user actions)
func (t *DeliveryTracker) UpdateStatus(ctx context.Context, appID uuid.UUID, status DeliveryStatus) error {
	app, err := t.repo.GetByID(ctx, appID)
	if err != nil {
		return fmt.Errorf("get application: %w", err)
	}
	if app == nil {
		return fmt.Errorf("application not found: %s", appID)
	}

	if !t.ValidateTransition(app.DeliveryStatus, status) {
		return fmt.Errorf("invalid status transition: %s → %s", app.DeliveryStatus, status)
	}

	previousStatus := app.DeliveryStatus

	err = t.repo.UpdateDeliveryStatus(ctx, appID, status)
	if err != nil {
		return fmt.Errorf("update status: %w", err)
	}

	// Broadcast WebSocket event
	t.hub.Broadcast(StatusChangeEvent{
		Type:           "dispatcher.status_changed",
		ApplicationID:  appID.String(),
		PreviousStatus: string(previousStatus),
		CurrentStatus:  string(status),
		UpdatedAt:      time.Now(),
	})

	t.log.Info().
		Str("app_id", appID.String()).
		Str("from", string(previousStatus)).
		Str("to", string(status)).
		Msg("delivery status changed")

	return nil
}

// GetStatus returns current status of an application
func (t *DeliveryTracker) GetStatus(ctx context.Context, appID uuid.UUID) (DeliveryStatus, error) {
	app, err := t.repo.GetByID(ctx, appID)
	if err != nil {
		return "", fmt.Errorf("get application: %w", err)
	}
	if app == nil {
		return "", fmt.Errorf("application not found: %s", appID)
	}

	return app.DeliveryStatus, nil
}

// ValidateTransition checks if status transition is valid
func (t *DeliveryTracker) ValidateTransition(from, to DeliveryStatus) bool {
	// Valid transitions:
	// PENDING → SENDING
	// SENDING → SENT | FAILED
	// SENT → DELIVERED
	// DELIVERED → READ
	// READ → RESPONDED (not implemented yet)
	// Any → FAILED (error cases)

	validTransitions := map[DeliveryStatus][]DeliveryStatus{
		StatusPending:   {StatusSending, StatusFailed},
		StatusSending:   {StatusSent, StatusFailed},
		StatusSent:      {StatusDelivered, StatusFailed},
		StatusDelivered: {StatusRead},
		StatusRead:      {}, // Could add RESPONDED later
		StatusFailed:    {}, // Terminal state
	}

	allowed, exists := validTransitions[from]
	if !exists {
		return false
	}

	for _, valid := range allowed {
		if valid == to {
			return true
		}
	}
	return false
}

// isRetryable checks if an error is retryable
func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()

	// FloodWait errors are retryable
	if strings.Contains(errMsg, "FLOOD_WAIT") || strings.Contains(errMsg, "FloodWait") {
		return true
	}

	// Network timeouts are retryable
	if strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "connection") {
		return true
	}

	return false
}
