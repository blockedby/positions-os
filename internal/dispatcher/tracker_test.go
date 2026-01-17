package dispatcher

import (
	"context"
	"os"
	"testing"

	"github.com/blockedby/positions-os/internal/logger"
	"github.com/blockedby/positions-os/internal/models"
	"github.com/blockedby/positions-os/internal/repository"
	"github.com/blockedby/positions-os/internal/web"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDeliveryTracker_NewDeliveryTracker tests the constructor
func TestDeliveryTracker_NewDeliveryTracker(t *testing.T) {
	repo := repository.NewApplicationsRepository(nil, &logger.Logger{})
	hub := web.NewHub()
	log := &logger.Logger{}

	tracker := NewDeliveryTracker(repo, hub, log)

	assert.NotNil(t, tracker, "NewDeliveryTracker should return non-nil")
	assert.NotNil(t, tracker.repo, "Tracker should have a repo")
	assert.NotNil(t, tracker.hub, "Tracker should have a hub")
	assert.NotNil(t, tracker.log, "Tracker should have a log")
}

// TestDeliveryTracker_ValidateTransition tests status transition validation
func TestDeliveryTracker_ValidateTransition(t *testing.T) {
	tracker := &DeliveryTracker{}

	tests := []struct {
		name     string
		from     DeliveryStatus
		to       DeliveryStatus
		expected bool
	}{
		{"Pending to Sending", StatusPending, StatusSending, true},
		{"Pending to Failed", StatusPending, StatusFailed, true},
		{"Sending to Sent", StatusSending, StatusSent, true},
		{"Sending to Failed", StatusSending, StatusFailed, true},
		{"Sent to Delivered", StatusSent, StatusDelivered, true},
		{"Delivered to Read", StatusDelivered, StatusRead, true},
		{"Sent to Read (invalid)", StatusSent, StatusRead, false},
		{"Pending to Sent (invalid)", StatusPending, StatusSent, false},
		{"Failed to Sent (invalid)", StatusFailed, StatusSent, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tracker.ValidateTransition(tt.from, tt.to)
			assert.Equal(t, tt.expected, result, "ValidateTransition result should match expected")
		})
	}
}

// TestDeliveryTracker_IsRetryable tests retryable error detection
func TestDeliveryTracker_IsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		errorMsg string
		expected bool
	}{
		{"FloodWait error", "FLOOD_WAIT: 30 seconds", true},
		{"FloodWait lowercase", "FloodWait error", true},
		{"Timeout error", "connection timeout", true},
		{"Connection error", "lost connection", true},
		{"User not found", "user not found", false},
		{"File not found", "file not found", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryable(&testError{msg: tt.errorMsg})
			assert.Equal(t, tt.expected, result, "isRetryable result should match expected")
		})
	}
}

// TestDeliveryTracker_Integration tests the tracker with real database
func TestDeliveryTracker_Integration(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration test; set INTEGRATION_TEST=1 to run")
	}

	ctx := context.Background()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/positions_os?sslmode=disable"
	}

	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err, "Failed to connect to database")
	defer pool.Close()

	log, err := logger.New("info", "")
	require.NoError(t, err, "Failed to create logger")

	repo := repository.NewApplicationsRepository(pool, log)
	hub := web.NewHub()
	tracker := NewDeliveryTracker(repo, hub, log)

	// Start hub in background
	go hub.Run()

	// Setup test data
	testJobID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	_, _ = pool.Exec(ctx, "DELETE FROM job_applications WHERE job_id = $1", testJobID)
	_, _ = pool.Exec(ctx, "DELETE FROM jobs WHERE id = $1", testJobID)

	// Cleanup when done
	defer func() {
		_, _ = pool.Exec(ctx, "DELETE FROM job_applications WHERE job_id = $1", testJobID)
		_, _ = pool.Exec(ctx, "DELETE FROM jobs WHERE id = $1", testJobID)
	}()

	// Create test job
	_, err = pool.Exec(ctx, `
		INSERT INTO jobs (id, target_id, external_id, raw_content, status)
		VALUES ($1, $2, $3, $4, $5)
	`, testJobID, uuid.New(), "test-ext-2", "test content", "INTERESTED")
	require.NoError(t, err)

	t.Run("TrackStart - PENDING to SENDING", func(t *testing.T) {
		app := createTestApp(testJobID)
		require.NoError(t, repo.Create(ctx, app))

		err := tracker.TrackStart(ctx, app.ID)
		require.NoError(t, err, "TrackStart should succeed")

		fetched, err := repo.GetByID(ctx, app.ID)
		require.NoError(t, err)
		assert.Equal(t, DeliveryStatus("SENDING"), fetched.DeliveryStatus, "Status should be SENDING")
	})

	t.Run("TrackSuccess - SENDING to SENT", func(t *testing.T) {
		app := createTestApp(testJobID)
		require.NoError(t, repo.Create(ctx, app))

		// First start sending
		require.NoError(t, tracker.TrackStart(ctx, app.ID))

		// Then mark as sent
		err := tracker.TrackSuccess(ctx, app.ID)
		require.NoError(t, err, "TrackSuccess should succeed")

		fetched, err := repo.GetByID(ctx, app.ID)
		require.NoError(t, err)
		assert.Equal(t, StatusSent, fetched.DeliveryStatus, "Status should be SENT")
		assert.NotNil(t, fetched.SentAt, "SentAt should be set")
	})

	t.Run("TrackFailure - any to FAILED", func(t *testing.T) {
		app := createTestApp(testJobID)
		require.NoError(t, repo.Create(ctx, app))

		testErr := assert.AnError
		err := tracker.TrackFailure(ctx, app.ID, testErr)
		require.NoError(t, err, "TrackFailure should succeed")

		fetched, err := repo.GetByID(ctx, app.ID)
		require.NoError(t, err)
		assert.Equal(t, StatusFailed, fetched.DeliveryStatus, "Status should be FAILED")
	})

	t.Run("UpdateStatus manual status change", func(t *testing.T) {
		app := createTestApp(testJobID)
		require.NoError(t, repo.Create(ctx, app))

		err := tracker.UpdateStatus(ctx, app.ID, StatusDelivered)
		// This should fail - invalid transition from PENDING to DELIVERED
		assert.Error(t, err, "UpdateStatus should fail for invalid transition")
	})

	t.Run("GetStatus", func(t *testing.T) {
		app := createTestApp(testJobID)
		require.NoError(t, repo.Create(ctx, app))

		status, err := tracker.GetStatus(ctx, app.ID)
		require.NoError(t, err, "GetStatus should succeed")
		assert.Equal(t, StatusPending, status, "Status should be PENDING")
	})
}

// Helper to create a test application
func createTestApp(jobID uuid.UUID) *models.JobApplication {
	id := uuid.New()
	channel := models.DeliveryChannelTGDM
	return &models.JobApplication{
		ID:              id,
		JobID:           jobID,
		DeliveryChannel: &channel,
		DeliveryStatus:  models.DeliveryStatusPending,
		Recipient:       stringPtr("@test_recruiter"),
	}
}

func stringPtr(s string) *string {
	return &s
}

// testError is a simple error for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
