package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/blockedby/positions-os/internal/database"
	"github.com/google/uuid"
)

func TestJobsRepository_GetByID_UpdateStructuredData(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") == "" {
		t.Skip("Skipping integration test; set INTEGRATION_TEST=1 to run")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set")
	}

	ctx := context.Background()
	db, err := database.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("failed to connect to db: %v", err)
	}
	defer db.Close()

	setupSchema(t, db)

	repo := NewJobsRepository(db.Pool)

	// 1. Create a job
	targetID := uuid.New()
	// Needs scraping_targets entry. Type must be valid enum
	_, err = db.Pool.Exec(ctx, "INSERT INTO scraping_targets (id, name, url, type, is_active, created_at, updated_at) VALUES ($1, $2, $3, 'TG_CHANNEL', true, NOW(), NOW())", targetID, "Test Channel", "http://t.me/test")
	if err != nil {
		t.Fatalf("failed to create target: %v", err)
	}

	externalID := "ext-123"
	job := &Job{
		TargetID:   targetID,
		ExternalID: externalID,
		RawContent: "raw job content",
		Status:     "RAW",
		SourceDate: func() *time.Time { t := time.Now(); return &t }(),
	}

	err = repo.Create(ctx, job)
	if err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	// 2. GetByID
	fetchedJob, err := repo.GetByID(ctx, job.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if fetchedJob == nil {
		t.Fatalf("GetByID returned nil") // Should fail here because implementation returns nil, nil
	}
	if fetchedJob.ID != job.ID {
		t.Errorf("expected ID %v, got %v", job.ID, fetchedJob.ID)
	}
	if fetchedJob.RawContent != job.RawContent {
		t.Errorf("expected content %v, got %v", job.RawContent, fetchedJob.RawContent)
	}

	// 3. UpdateStructuredData
	structuredData := map[string]interface{}{
		"title":  "Software Engineer",
		"salary": "100k",
	}

	err = repo.UpdateStructuredData(ctx, job.ID, structuredData)
	if err != nil {
		t.Fatalf("UpdateStructuredData failed: %v", err)
	}

	// Verify update
	updatedJob, err := repo.GetByID(ctx, job.ID)
	if err != nil {
		t.Fatalf("GetByID after update failed: %v", err)
	}

	if updatedJob.Status != "ANALYZED" {
		t.Errorf("expected status ANALYZED, got %s", updatedJob.Status)
	}
	if updatedJob.AnalyzedAt == nil {
		t.Error("expected AnalyzedAt to be set")
	}

	val, ok := updatedJob.StructuredData["title"]
	if !ok || val != "Software Engineer" {
		t.Errorf("expected title 'Software Engineer', got %v", val)
	}
}

func setupSchema(t *testing.T, db *database.DB) {
	ctx := context.Background()

	// Cleanup
	_, _ = db.Pool.Exec(ctx, `
		DROP TABLE IF EXISTS job_applications CASCADE;
		DROP TABLE IF EXISTS job_listings CASCADE;
		DROP TABLE IF EXISTS parsed_ranges CASCADE;
		DROP TABLE IF EXISTS jobs CASCADE;
		DROP TABLE IF EXISTS scraping_targets CASCADE;
		DROP TYPE IF EXISTS job_status CASCADE;
		DROP TYPE IF EXISTS scraping_target_type CASCADE;
	`)

	// Migrations
	files := []string{
		"../../migrations/0001_create_scraping_targets.up.sql",
		"../../migrations/0002_create_jobs.up.sql",
	}

	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("failed to read migration %s: %v", f, err)
		}
		_, err = db.Pool.Exec(ctx, string(content))
		if err != nil {
			t.Fatalf("failed to run migration %s: %v", f, err)
		}
	}
}
