package dispatcher

import (
	"context"
	"errors"
	"testing"

	"github.com/blockedby/positions-os/internal/logger"
	"github.com/blockedby/positions-os/internal/models"
	"github.com/celestix/gotgproto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestNewTelegramSender_RedPhase tests that the sender can be created with proper rate limiting.
// This is a RED phase test - it will fail until we implement TelegramSender.
func TestNewTelegramSender_RedPhase(t *testing.T) {
	// This test will fail until we implement TelegramSender
	mockClient := &gotgproto.Client{}
	mockTracker := &mockDeliveryTracker{}
	mockRepo := &mockApplicationsRepository{}
	mockReadTracker := &mockReadTracker{}
	log := logger.Get()

	sender := NewTelegramSender(mockClient, mockTracker, mockRepo, mockReadTracker, log)

	// Assertions that will fail until implementation exists
	assert.NotNil(t, sender, "NewTelegramSender should return a non-nil sender")
	assert.NotNil(t, sender.LimiterForTest(), "Rate limiter should be initialized")
}

// Mock interfaces for testing (will be replaced by real interfaces from Thread A)
type mockDeliveryTracker struct{}

func (m *mockDeliveryTracker) TrackStart(ctx context.Context, appID uuid.UUID) error   { return nil }
func (m *mockDeliveryTracker) TrackSuccess(ctx context.Context, appID uuid.UUID) error { return nil }
func (m *mockDeliveryTracker) TrackFailure(ctx context.Context, appID uuid.UUID, err error) error {
	return nil
}

type mockApplicationsRepository struct{}

func (m *mockApplicationsRepository) Create(ctx context.Context, app *models.JobApplication) error {
	return nil
}
func (m *mockApplicationsRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.JobApplication, error) {
	return nil, nil
}
func (m *mockApplicationsRepository) GetByJobID(ctx context.Context, jobID uuid.UUID) ([]*models.JobApplication, error) {
	return nil, nil
}

type mockReadTracker struct{}

func (m *mockReadTracker) RegisterSentMessage(msgID int64, appID uuid.UUID) {}

// TestResolveUsername_StripsAtPrefix tests that @ prefix is stripped.
func TestResolveUsername_StripsAtPrefix(t *testing.T) {
	mockClient := &gotgproto.Client{}
	sender := NewTelegramSender(mockClient, nil, nil, nil, logger.Get())

	// Test that @ prefix is handled
	tests := []struct {
		name     string
		username string
		expected string
	}{
		{"with @", "@recruiter", "recruiter"},
		{"without @", "recruiter", "recruiter"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sender.stripAtPrefix(tt.username)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestResolveUsername_ReturnsErrorForEmpty tests that empty username returns error.
func TestResolveUsername_ReturnsErrorForEmpty(t *testing.T) {
	mockClient := &gotgproto.Client{}
	sender := NewTelegramSender(mockClient, nil, nil, nil, logger.Get())

	_, err := sender.ResolveUsername(context.Background(), "")
	assert.Error(t, err, "Empty username should return error")
}

// TestUploadAndSend_ValidatesInputs tests that inputs are validated before sending.
func TestUploadAndSend_ValidatesInputs(t *testing.T) {
	mockClient := &gotgproto.Client{}
	sender := NewTelegramSender(mockClient, nil, nil, nil, logger.Get())

	// Test empty recipient
	err := sender.UploadAndSend(context.Background(), "", "cover letter", "/path/to/resume.pdf")
	assert.Error(t, err, "Empty recipient should return error")

	// Test empty text
	err = sender.UploadAndSend(context.Background(), "@recruiter", "", "/path/to/resume.pdf")
	assert.Error(t, err, "Empty text should return error")

	// Test empty PDF path
	err = sender.UploadAndSend(context.Background(), "@recruiter", "cover letter", "")
	assert.Error(t, err, "Empty PDF path should return error")
}

// Mock JobApplication for testing SendApplication
type mockJobApplication struct {
	id              string
	coverLetterMD   string
	resumePDFPath   string
	recipient       string
	deliveryChannel string
	deliveryStatus  string
}

func (m *mockJobApplication) GetID() string            { return m.id }
func (m *mockJobApplication) GetCoverLetterMD() string { return m.coverLetterMD }
func (m *mockJobApplication) GetResumePDFPath() string { return m.resumePDFPath }
func (m *mockJobApplication) GetRecipient() string     { return m.recipient }

// TestSendApplication_ValidatesApplication tests that SendApplication validates the application.
func TestSendApplication_ValidatesApplication(t *testing.T) {
	mockClient := &gotgproto.Client{}
	mockTracker := &mockDeliveryTracker{}
	mockRepo := &mockApplicationsRepository{}
	sender := NewTelegramSender(mockClient, mockTracker, mockRepo, nil, logger.Get())

	// Test with nil UUID
	err := sender.SendApplication(context.Background(), uuid.Nil, "@recruiter")
	assert.Error(t, err, "Nil UUID should return error")
}

// TestIsFloodWait tests detection of FLOOD_WAIT errors.
func TestIsFloodWait(t *testing.T) {
	sender := NewTelegramSender(nil, nil, nil, nil, logger.Get())

	// Test FLOOD_WAIT error string
	tests := []struct {
		name     string
		errMsg   string
		expected bool
		waitSec  int
	}{
		{"flood wait 30", "FLOOD_WAIT_30", true, 30},
		{"flood wait with prefix", "rpc error: code 420: FLOOD_WAIT_15", true, 15},
		{"normal error", "user not found", false, 0},
		{"empty error", "", false, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wait := sender.isFloodWait(errors.New(tt.errMsg))
			if tt.expected {
				assert.Equal(t, tt.waitSec, wait, "Should extract wait seconds")
			} else {
				assert.Equal(t, 0, wait, "Should return 0 for non-flood wait errors")
			}
		})
	}
}
