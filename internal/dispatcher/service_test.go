package dispatcher

import (
	"context"
	"errors"
	"testing"

	"github.com/blockedby/positions-os/internal/logger"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock EmailSender for testing
type mockEmailSenderForService struct{}

func (m *mockEmailSenderForService) SendApplication(ctx context.Context, appID uuid.UUID, recipient string) error {
	return nil
}

// TestNewDispatcherService tests that the service can be created.
func TestNewDispatcherService(t *testing.T) {
	mockTgSender := &mockTelegramSenderForService{}
	mockEmailSender := &mockEmailSenderForService{}
	mockTracker := &mockDeliveryTracker{}
	mockRepo := &mockApplicationsRepository{}
	log := logger.Get()

	service := NewDispatcherService(mockTgSender, mockEmailSender, mockTracker, mockRepo, log)

	assert.NotNil(t, service, "Service should be created")
	assert.NotNil(t, service.tgSender, "TG sender should be set")
	assert.NotNil(t, service.emailSender, "Email sender should be set")
}

// TestSendApplication_TGDM_RoutesCorrectly tests TG_DM routing.
func TestSendApplication_TGDM_RoutesCorrectly(t *testing.T) {
	mockTgSender := &mockTelegramSenderForService{
		sendApplicationFunc: func(ctx context.Context, appID uuid.UUID, recipient string) error {
			return errors.New("TG send failed (expected)")
		},
	}
	mockEmailSender := &mockEmailSenderForService{}
	mockTracker := &mockDeliveryTracker{}
	mockRepo := &mockApplicationsRepository{}
	log := logger.Get()

	service := NewDispatcherService(mockTgSender, mockEmailSender, mockTracker, mockRepo, log)

	jobID := uuid.New()
	req := &SendRequest{
		JobID:     jobID,
		Channel:   "TG_DM",
		Recipient: "@recruiter",
	}

	err := service.SendApplication(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "TG send failed")
}

// TestSendApplication_EMAIL_NotImplemented tests email is not implemented.
func TestSendApplication_EMAIL_NotImplemented(t *testing.T) {
	mockTgSender := &mockTelegramSenderForService{}
	mockTracker := &mockDeliveryTracker{}
	mockRepo := &mockApplicationsRepository{}
	log := logger.Get()

	// Create service with nil email sender to test "not configured" path
	service := &DispatcherService{
		tgSender: mockTgSender,
		// emailSender is nil to test not configured case
		tracker: mockTracker,
		repo:    mockRepo,
		log:     log,
	}

	jobID := uuid.New()
	req := &SendRequest{
		JobID:     jobID,
		Channel:   "EMAIL",
		Recipient: "recruiter@example.com",
	}

	err := service.SendApplication(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

// TestSendApplication_InvalidChannel tests invalid channel.
func TestSendApplication_InvalidChannel(t *testing.T) {
	service := &DispatcherService{
		tgSender:    &mockTelegramSenderForService{},
		emailSender: &mockEmailSenderForService{},
		tracker:     &mockDeliveryTracker{},
		repo:        &mockApplicationsRepository{},
		log:         logger.Get(),
	}

	jobID := uuid.New()
	req := &SendRequest{
		JobID:     jobID,
		Channel:   "INVALID",
		Recipient: "@recruiter",
	}

	err := service.SendApplication(context.Background(), req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported channel")
}

// TestSendApplication_ValidatesRequest tests request validation.
func TestSendApplication_ValidatesRequest(t *testing.T) {
	service := &DispatcherService{
		tgSender:    &mockTelegramSenderForService{},
		emailSender: &mockEmailSenderForService{},
		tracker:     &mockDeliveryTracker{},
		repo:        &mockApplicationsRepository{},
		log:         logger.Get(),
	}

	tests := []struct {
		name        string
		req         *SendRequest
		expectedErr string
	}{
		{
			name:        "nil request",
			req:         nil,
			expectedErr: "request cannot be nil",
		},
		{
			name: "empty job ID",
			req: &SendRequest{
				JobID:     uuid.Nil,
				Channel:   "TG_DM",
				Recipient: "@recruiter",
			},
			expectedErr: "job ID cannot be empty",
		},
		{
			name: "empty recipient",
			req: &SendRequest{
				JobID:     uuid.New(),
				Channel:   "TG_DM",
				Recipient: "",
			},
			expectedErr: "recipient cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.SendApplication(context.Background(), tt.req)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

// Mock TelegramSender for testing - implements interface that can be used by service
type mockTelegramSenderForService struct {
	sendApplicationFunc func(ctx context.Context, appID uuid.UUID, recipient string) error
}

func (m *mockTelegramSenderForService) SendApplication(ctx context.Context, appID uuid.UUID, recipient string) error {
	if m.sendApplicationFunc != nil {
		return m.sendApplicationFunc(ctx, appID, recipient)
	}
	return nil
}
