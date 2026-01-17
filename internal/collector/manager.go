package collector

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/blockedby/positions-os/internal/telegram"
)

// errors
var (
	ErrAlreadyRunning = errors.New("a scrape job is already running")
)

// ScrapeOptions holds options for a scrape job
type ScrapeOptions struct {
	TargetID uuid.UUID
	Channel  string
	Limit    int
	Until    *time.Time
	TopicIDs []int
}

// ScrapeJob represents an active scrape job
type ScrapeJob struct {
	ID        uuid.UUID
	TargetID  uuid.UUID
	StartedAt time.Time
	Options   ScrapeOptions
}

// Scraper defines the interface for scraping logic
type Scraper interface {
	Scrape(ctx context.Context, opts ScrapeOptions) (*ScrapeResult, error)
	ListTopics(ctx context.Context, channelURL string) ([]telegram.Topic, error)
	GetTelegramStatus() telegram.Status
}

// ScrapeManager manages active scrape jobs
// ensures only one job runs at a time
// thread-safe
type ScrapeManager struct {
	mu       sync.Mutex
	current  *ScrapeJob
	cancelFn context.CancelFunc
	scraper  Scraper
}

// NewScrapeManager creates a new scrape manager
func NewScrapeManager(scraper Scraper) *ScrapeManager {
	return &ScrapeManager{
		scraper: scraper,
	}
}

// Start starts a new scrape job
// returns ErrAlreadyRunning if a job is already running
func (m *ScrapeManager) Start(_ context.Context, opts ScrapeOptions) (*ScrapeJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.current != nil {
		return nil, ErrAlreadyRunning
	}

	// IMPORTANT: Use background context, NOT the HTTP request context!
	// The HTTP request context gets canceled when the handler returns,
	// which would immediately cancel our scrape job.
	// We create a new cancellable context from Background() so the job
	// continues running after the HTTP response is sent.
	scrapeCtx, cancel := context.WithCancel(context.Background())
	m.cancelFn = cancel

	job := &ScrapeJob{
		ID:        uuid.New(),
		TargetID:  opts.TargetID,
		StartedAt: time.Now(),
		Options:   opts,
	}
	m.current = job

	// run the actual scraping in a goroutine
	go m.run(scrapeCtx, job)

	return job, nil
}

// Stop stops the current scrape job
// safe to call when no job is running
func (m *ScrapeManager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cancelFn != nil {
		m.cancelFn()
		m.cancelFn = nil
	}
	m.current = nil
}

// Current returns the currently running job
// returns nil if no job is running
func (m *ScrapeManager) Current() *ScrapeJob {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.current
}

// run executes the scrape job
// this is called in a goroutine
func (m *ScrapeManager) run(ctx context.Context, job *ScrapeJob) {
	defer func() {
		m.mu.Lock()
		if m.current != nil && m.current.ID == job.ID {
			m.current = nil
			m.cancelFn = nil
		}
		m.mu.Unlock()
	}()

	// execute scraping
	if m.scraper != nil {
		_, _ = m.scraper.Scrape(ctx, job.Options)
		// errors are logged inside Scrape method usually
	}
}

// ListTopics delegates to scraper
func (m *ScrapeManager) ListTopics(ctx context.Context, channelURL string) ([]telegram.Topic, error) {
	if m.scraper == nil {
		return nil, errors.New("no scraper initialized")
	}
	return m.scraper.ListTopics(ctx, channelURL)
}

// GetTelegramStatus returns the current Telegram connection status
func (m *ScrapeManager) GetTelegramStatus() telegram.Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.scraper == nil {
		return "UNKNOWN"
	}
	return m.scraper.GetTelegramStatus()
}
