package repository

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/blockedby/positions-os/internal/logger"
	"github.com/blockedby/positions-os/internal/models"
)

// This test file follows TDD Red-Green-Refactor pattern
// RED: Write failing test first
// GREEN: Write minimal code to pass
// REFACTOR: Improve while keeping tests green

// TestApplicationsRepository_NewApplicationsRepository tests the constructor
func TestApplicationsRepository_NewApplicationsRepository(t *testing.T) {
	repo := NewApplicationsRepository(nil, &logger.Logger{})
	assert.NotNil(t, repo, "NewApplicationsRepository should return non-nil")
	assert.NotNil(t, repo.log, "Repository should have a logger")
}

// Integration tests (require real database)
// Set INTEGRATION_TEST=1 DATABASE_URL=postgres://... to run these

func TestApplicationsRepository_Integration(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration test; set INTEGRATION_TEST=1 to run")
	}

	// Setup database connection
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

	repo := NewApplicationsRepository(pool, log)

	// Clean up any existing test data
	testJobID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	_, _ = pool.Exec(ctx, "DELETE FROM job_applications WHERE job_id = $1", testJobID)
	_, _ = pool.Exec(ctx, "DELETE FROM jobs WHERE id = $1", testJobID)

	// Create a test job first (foreign key constraint)
	_, err = pool.Exec(ctx, `
		INSERT INTO jobs (id, target_id, external_id, raw_content, status)
		VALUES ($1, $2, $3, $4, $5)
	`, testJobID, uuid.New(), "test-ext", "test content", "INTERESTED")
	require.NoError(t, err, "Failed to create test job")

	t.Run("Create", func(t *testing.T) {
		app := newTestApplication(testJobID)
		err := repo.Create(ctx, app)
		require.NoError(t, err, "Create should succeed")
		assert.NotEqual(t, uuid.Nil, app.ID, "ID should be set")
		assert.NotEqual(t, 0, app.CreatedAt.Unix(), "CreatedAt should be set")
	})

	t.Run("GetByID", func(t *testing.T) {
		app := newTestApplication(testJobID)
		err := repo.Create(ctx, app)
		require.NoError(t, err)

		fetched, err := repo.GetByID(ctx, app.ID)
		require.NoError(t, err, "GetByID should succeed")
		require.NotNil(t, fetched, "Application should exist")
		assert.Equal(t, app.ID, fetched.ID, "ID should match")
		assert.Equal(t, testJobID, fetched.JobID, "JobID should match")
		assert.Equal(t, models.DeliveryStatusPending, fetched.DeliveryStatus, "Status should be PENDING")
	})

	t.Run("GetByID_NotFound", func(t *testing.T) {
		fetched, err := repo.GetByID(ctx, uuid.New())
		require.NoError(t, err, "GetByID should succeed even when not found")
		assert.Nil(t, fetched, "Application should be nil when not found")
	})

	t.Run("GetByJobID", func(t *testing.T) {
		app1 := newTestApplication(testJobID)
		app2 := newTestApplication(testJobID)
		require.NoError(t, repo.Create(ctx, app1))
		require.NoError(t, repo.Create(ctx, app2))

		apps, err := repo.GetByJobID(ctx, testJobID)
		require.NoError(t, err, "GetByJobID should succeed")
		assert.GreaterOrEqual(t, len(apps), 2, "Should have at least 2 applications")
	})

	t.Run("UpdateDeliveryStatus", func(t *testing.T) {
		app := newTestApplication(testJobID)
		require.NoError(t, repo.Create(ctx, app))

		err := repo.UpdateDeliveryStatus(ctx, app.ID, models.DeliveryStatusSent)
		require.NoError(t, err, "UpdateDeliveryStatus should succeed")

		fetched, err := repo.GetByID(ctx, app.ID)
		require.NoError(t, err)
		assert.Equal(t, models.DeliveryStatusSent, fetched.DeliveryStatus, "Status should be SENT")
	})

	t.Run("UpdateRecipient", func(t *testing.T) {
		app := newTestApplication(testJobID)
		require.NoError(t, repo.Create(ctx, app))

		newRecipient := "@updated_recruiter"
		err := repo.UpdateRecipient(ctx, app.ID, newRecipient)
		require.NoError(t, err, "UpdateRecipient should succeed")

		fetched, err := repo.GetByID(ctx, app.ID)
		require.NoError(t, err)
		assert.Equal(t, &newRecipient, fetched.Recipient, "Recipient should be updated")
	})

	t.Run("MarkSent", func(t *testing.T) {
		app := newTestApplication(testJobID)
		require.NoError(t, repo.Create(ctx, app))

		err := repo.MarkSent(ctx, app.ID)
		require.NoError(t, err, "MarkSent should succeed")

		fetched, err := repo.GetByID(ctx, app.ID)
		require.NoError(t, err)
		assert.Equal(t, models.DeliveryStatusSent, fetched.DeliveryStatus, "Status should be SENT")
		assert.NotNil(t, fetched.SentAt, "SentAt should be set")
	})

	t.Run("ListPending", func(t *testing.T) {
		// Create a pending application
		app := newTestApplication(testJobID)
		require.NoError(t, repo.Create(ctx, app))

		// List pending
		apps, err := repo.ListPending(ctx, 10)
		require.NoError(t, err, "ListPending should succeed")
		assert.NotEmpty(t, apps, "Should have pending applications")
	})

	t.Run("UpdateTimestamps", func(t *testing.T) {
		app := newTestApplication(testJobID)
		require.NoError(t, repo.Create(ctx, app))

		err := repo.UpdateTimestamps(ctx, app.ID, "delivered_at", app.CreatedAt)
		require.NoError(t, err, "UpdateTimestamps should succeed")

		fetched, err := repo.GetByID(ctx, app.ID)
		require.NoError(t, err)
		assert.NotNil(t, fetched.DeliveredAt, "DeliveredAt should be set")
	})

	// Cleanup
	_, _ = pool.Exec(ctx, "DELETE FROM job_applications WHERE job_id = $1", testJobID)
	_, _ = pool.Exec(ctx, "DELETE FROM jobs WHERE id = $1", testJobID)
}

// Helper to create a test application
func newTestApplication(jobID uuid.UUID) *models.JobApplication {
	id := uuid.New()
	channel := models.DeliveryChannelTGDM
	return &models.JobApplication{
		ID:               id,
		JobID:            jobID,
		DeliveryChannel:  &channel,
		DeliveryStatus:   models.DeliveryStatusPending,
		Recipient:        stringPtr("@test_recruiter"),
		TailoredResumeMD: stringPtr("Test resume content"),
		CoverLetterMD:    stringPtr("Test cover letter"),
	}
}

func stringPtr(s string) *string {
	return &s
}
