package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/blockedby/positions-os/internal/database"
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

func TestJobsRepository_List(t *testing.T) {
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

	// Create a target
	targetID := uuid.New()
	_, err = db.Pool.Exec(ctx, "INSERT INTO scraping_targets (id, name, url, type, is_active, created_at, updated_at) VALUES ($1, $2, $3, 'TG_CHANNEL', true, NOW(), NOW())", targetID, "Test Channel 2", "http://t.me/test2")
	requireNoError(t, err)

	// Helper to create job
	createJob := func(extID, status, raw, title string, salary string, tech []string) {
		job := &Job{
			TargetID:   targetID,
			ExternalID: extID,
			RawContent: raw,
			Status:     status,
			SourceDate: func() *time.Time { t := time.Now(); return &t }(),
			StructuredData: map[string]interface{}{
				"title":        title,
				"salary":       salary,
				"technologies": tech,
			},
		}

		err := repo.Create(ctx, job)
		requireNoError(t, err)

		// For numeric salary filter test
		if salary != "" {
			job.StructuredData["salary_min"] = 100000
		}

		err = repo.UpdateStructuredData(ctx, job.ID, job.StructuredData)
		requireNoError(t, err)

		if status != "ANALYZED" {
			err = repo.UpdateStatus(ctx, job.ID, status)
			requireNoError(t, err)
		}
	}

	createJob("j1", "RAW", "Go developer needed", "Go Dev", "", []string{"go"})
	createJob("j2", "ANALYZED", "Python developer", "Python Dev", "100k", []string{"python"})
	createJob("j3", "INTERESTED", "Senior Go", "Senior Go", "200k", []string{"go", "k8s"})

	// Test 1: List All
	jobs, total, err := repo.List(ctx, JobFilter{Limit: 10})
	requireNoError(t, err)
	if total != 3 {
		t.Errorf("expected 3 jobs, got %d", total)
	}
	if len(jobs) != 3 {
		t.Errorf("expected 3 jobs in page, got %d", len(jobs))
	}

	// Test 2: Filter by Status
	jobs, total, err = repo.List(ctx, JobFilter{Status: "RAW", Limit: 10})
	requireNoError(t, err)
	if total != 1 {
		t.Errorf("expected 1 RAW job, got %d", total)
	}
	if jobs[0].ExternalID != "j1" {
		t.Errorf("expected job j1, got %s", jobs[0].ExternalID)
	}

	// Test 3: Filter by Text Search (Query)
	jobs, total, err = repo.List(ctx, JobFilter{Query: "Senior", Limit: 10})
	requireNoError(t, err)
	if total != 1 {
		t.Errorf("expected 1 'Senior' job, got %d", total)
	}
	if jobs[0].ExternalID != "j3" {
		t.Errorf("expected job j3, got %s", jobs[0].ExternalID)
	}

	// Test 4: Pagination
	jobs, total, err = repo.List(ctx, JobFilter{Limit: 1, Page: 1})
	requireNoError(t, err)
	if total != 3 {
		t.Errorf("expected total 3, got %d", total)
	}
	if len(jobs) != 1 {
		t.Errorf("expected 1 job on page 1, got %d", len(jobs))
	}
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJobsRepository_GetExistingMessageIDs(t *testing.T) {
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
	targetID := uuid.New()

	// Create target
	_, err = db.Pool.Exec(ctx,
		"INSERT INTO scraping_targets (id, name, url, type, is_active, created_at, updated_at) VALUES ($1, $2, $3, 'TG_CHANNEL', true, NOW(), NOW())",
		targetID, "Test Channel for MessageIDs", "http://t.me/msgidtest")
	requireNoError(t, err)

	// Create jobs with different message IDs
	messageIDs := []int64{1001, 1005, 1010}
	for i, msgID := range messageIDs {
		job := &Job{
			TargetID:    targetID,
			ExternalID:  string(rune('a' + i)),
			RawContent:  "test content",
			Status:      "RAW",
			TgMessageID: &msgID,
		}
		err = repo.Create(ctx, job)
		requireNoError(t, err)
	}

	// Create a job without message ID (should not be returned)
	jobNoMsgID := &Job{
		TargetID:   targetID,
		ExternalID: "no-msg-id",
		RawContent: "no message id",
		Status:     "RAW",
	}
	err = repo.Create(ctx, jobNoMsgID)
	requireNoError(t, err)

	// Test GetExistingMessageIDs
	existingIDs, err := repo.GetExistingMessageIDs(ctx, targetID)
	requireNoError(t, err)

	if len(existingIDs) != 3 {
		t.Errorf("expected 3 message IDs, got %d", len(existingIDs))
	}

	// Verify all expected IDs are present
	idSet := make(map[int64]bool)
	for _, id := range existingIDs {
		idSet[id] = true
	}

	for _, expectedID := range messageIDs {
		if !idSet[expectedID] {
			t.Errorf("expected message ID %d to be in result", expectedID)
		}
	}

	// Test with non-existent target (should return empty slice)
	nonExistentTarget := uuid.New()
	emptyIDs, err := repo.GetExistingMessageIDs(ctx, nonExistentTarget)
	requireNoError(t, err)

	if len(emptyIDs) != 0 {
		t.Errorf("expected empty slice for non-existent target, got %d items", len(emptyIDs))
	}
}
