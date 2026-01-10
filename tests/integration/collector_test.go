package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/blockedby/positions-os/internal/collector"
	"github.com/blockedby/positions-os/internal/database"
	"github.com/blockedby/positions-os/internal/logger"
	"github.com/blockedby/positions-os/internal/repository"
	"github.com/blockedby/positions-os/internal/telegram"
)

// MockTGClient mocks telegram client
type MockTGClient struct {
	Channel  *telegram.Channel
	Messages []telegram.Message
}

func (m *MockTGClient) ResolveChannel(ctx context.Context, username string) (*telegram.Channel, error) {
	if m.Channel == nil {
		return nil, fmt.Errorf("channel not found")
	}
	return m.Channel, nil
}

func (m *MockTGClient) GetMessages(ctx context.Context, channel *telegram.Channel, offsetID int, limit int) ([]telegram.Message, error) {
	if offsetID > 0 {
		// simulate end of history for test simplicity
		return []telegram.Message{}, nil
	}
	return m.Messages, nil
}

func (m *MockTGClient) GetTopics(ctx context.Context, channel *telegram.Channel) ([]telegram.Topic, error) {
	return []telegram.Topic{}, nil
}

// MockPublisher mocks event publisher
type MockPublisher struct {
	Events []collector.JobNewEvent
}

func (m *MockPublisher) PublishJobNew(ctx context.Context, event collector.JobNewEvent) error {
	m.Events = append(m.Events, event)
	return nil
}

func TestEndToEnd_Scraping(t *testing.T) {
	// this test requires database
	if os.Getenv("INTEGRATION_TEST") == "" {
		t.Skip("Skipping integration test; set INTEGRATION_TEST=1 to run (WARNING: wipes database)")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	// setup logger
	logger.Init("debug", "")
	log := logger.Get()

	// connect to db
	db, err := database.New(context.Background(), dbURL)
	if err != nil {
		t.Fatalf("failed to connect to db: %v", err)
	}
	defer db.Close()

	// cleanup db
	dropTables(t, db)
	runMigrations(t, db)

	// init repos
	targetsRepo := repository.NewTargetsRepository(db.Pool)
	jobsRepo := repository.NewJobsRepository(db.Pool)
	rangesRepo := repository.NewRangesRepository(db.Pool)

	// prepare mock data
	channelID := int64(123456)
	accessHash := int64(789012)
	channel := &telegram.Channel{
		ID:         channelID,
		AccessHash: accessHash,
		Username:   "test_channel",
		Title:      "Test Channel",
	}

	msgs := []telegram.Message{
		{
			ID:        100,
			ChannelID: channelID,
			Text:      "Job 1 #golang",
			Date:      time.Now().Add(-1 * time.Hour),
		},
		{
			ID:        101,
			ChannelID: channelID,
			Text:      "Job 2 #python",
			Date:      time.Now(),
		},
	}

	tgClient := &MockTGClient{
		Channel:  channel,
		Messages: msgs,
	}

	publisher := &MockPublisher{}

	// create service
	svc := collector.NewService(
		tgClient,
		targetsRepo,
		jobsRepo,
		rangesRepo,
		publisher,
		log,
	)

	// run scrape
	ctx := context.Background()
	opts := collector.ScrapeOptions{
		Channel: "@easy_python_job",
		Limit:   10,
	}

	result, err := svc.Scrape(ctx, opts)
	if err != nil {
		t.Fatalf("Scrape() error: %v", err)
	}

	// verify result
	if result.NewJobs != 2 {
		t.Errorf("NewJobs = %d, want 2", result.NewJobs)
	}
	if result.Errors != 0 {
		t.Errorf("Errors = %d, want 0", result.Errors)
	}

	// verify db state

	// target created?
	target, err := targetsRepo.GetByURL(ctx, "easy_python_job")
	if err != nil {
		t.Fatalf("GetByURL error: %v", err)
	}
	if target == nil {
		t.Fatal("Target should be created")
	}
	if *target.TgChannelID != channelID {
		t.Errorf("Target TgChannelID = %d, want %d", *target.TgChannelID, channelID)
	}

	// jobs created?
	job1, err := jobsRepo.GetByExternalID(ctx, target.ID, "100")
	if err != nil {
		t.Fatalf("GetByExternalID(100) error: %v", err)
	}
	if job1 == nil {
		t.Error("Job 100 should exist")
	}

	job2, err := jobsRepo.GetByExternalID(ctx, target.ID, "101")
	if err != nil {
		t.Fatalf("GetByExternalID(101) error: %v", err)
	}
	if job2 == nil {
		t.Error("Job 101 should exist")
	}

	// parsed range updated?
	// rangesRepo doesnt have Get, but we can check if running Scrape again produces 0 new jobs

	tgClient.Messages = []telegram.Message{
		msgs[0], // old
		msgs[1], // old
		{ // new
			ID:        102,
			ChannelID: channelID,
			Text:      "Job 3 #rust",
			Date:      time.Now(),
		},
	}

	result2, err := svc.Scrape(ctx, opts)
	if err != nil {
		t.Fatalf("Scrape() 2nd run error: %v", err)
	}

	if result2.NewJobs != 1 {
		t.Errorf("2nd run NewJobs = %d, want 1 (new msg only)", result2.NewJobs)
	}
	if result2.SkippedOld != 2 {
		t.Errorf("2nd run SkippedOld = %d, want 2", result2.SkippedOld)
	}

	// events published?
	if len(publisher.Events) != 3 { // 2 from first run + 1 from second
		t.Errorf("Publisher events = %d, want 3", len(publisher.Events))
	}
}

func dropTables(t *testing.T, db *database.DB) {
	ctx := context.Background()
	// drops tables related to this test
	_, err := db.Pool.Exec(ctx, `
		DROP TABLE IF EXISTS job_applications CASCADE;
		DROP TABLE IF EXISTS job_listings CASCADE;
		DROP TABLE IF EXISTS parsed_ranges CASCADE;
		DROP TABLE IF EXISTS jobs CASCADE;
		DROP TABLE IF EXISTS scraping_targets CASCADE;
		DROP TYPE IF EXISTS job_status CASCADE;
		DROP TYPE IF EXISTS scraping_target_type CASCADE;
	`)
	if err != nil {
		t.Fatalf("failed to drop tables: %v", err)
	}
}

func runMigrations(t *testing.T, db *database.DB) {
	// naive migration runner - assumes run from project root or checks paths
	// for integration tests running from project root via go test ./tests/integration
	// the CWD is usually the package directory.

	files := []string{
		"../../migrations/0001_create_scraping_targets.up.sql",
		"../../migrations/0002_create_jobs.up.sql",
		"../../migrations/0005_create_parsed_ranges.up.sql",
	}

	ctx := context.Background()
	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("failed to read migration %s: %v. CWD: %s", f, err, os.Getenv("CD"))
		}
		_, err = db.Pool.Exec(ctx, string(content))
		if err != nil {
			t.Fatalf("failed to run migration %s: %v", f, err)
		}
	}
}
