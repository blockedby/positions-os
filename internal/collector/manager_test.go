package collector

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/blockedby/positions-os/internal/telegram"
	"github.com/google/uuid"
)

// MockScraper for testing
type MockScraper struct {
	Called         bool
	Opts           ScrapeOptions
	Delay          time.Duration
	TopicsToReturn []telegram.Topic
}

func (m *MockScraper) Scrape(ctx context.Context, opts ScrapeOptions) (*ScrapeResult, error) {
	m.Called = true
	m.Opts = opts
	if m.Delay > 0 {
		select {
		case <-time.After(m.Delay):
		case <-ctx.Done():
		}
	}
	return &ScrapeResult{}, nil
}

func (m *MockScraper) ListTopics(ctx context.Context, channelURL string) ([]telegram.Topic, error) {
	return m.TopicsToReturn, nil
}

// GetTelegramStatus stub
func (m *MockScraper) GetTelegramStatus() telegram.Status {
	return telegram.StatusReady
}

// test manager start
func TestScrapeManager_Start(t *testing.T) {
	t.Run("starts job successfully", func(t *testing.T) {
		mockScraper := &MockScraper{}
		manager := NewScrapeManager(mockScraper)

		job, err := manager.Start(context.Background(), ScrapeOptions{
			Channel: "test_channel",
			Limit:   100,
		})

		if err != nil {
			t.Fatalf("Start() unexpected error: %v", err)
		}
		if job == nil {
			t.Fatal("Start() returned nil job")
		}
		if job.ID == uuid.Nil {
			t.Error("job.ID should not be nil")
		}
		if job.Options.Channel != "test_channel" {
			t.Errorf("job.Options.Channel = %s, want test_channel", job.Options.Channel)
		}

		// give goroutine time to run
		time.Sleep(10 * time.Millisecond)
		if !mockScraper.Called {
			t.Error("Scraper.Scrape was not called")
		}
		if mockScraper.Opts.Channel != "test_channel" {
			t.Errorf("Scraper received channel %s, want test_channel", mockScraper.Opts.Channel)
		}

		// cleanup
		manager.Stop()
	})

	t.Run("returns error when already running", func(t *testing.T) {
		manager := NewScrapeManager(&MockScraper{})

		// start first job
		_, err := manager.Start(context.Background(), ScrapeOptions{
			Channel: "first",
		})
		if err != nil {
			t.Fatalf("first Start() unexpected error: %v", err)
		}

		// try to start second
		_, err = manager.Start(context.Background(), ScrapeOptions{
			Channel: "second",
		})
		if err != ErrAlreadyRunning {
			t.Errorf("second Start() error = %v, want ErrAlreadyRunning", err)
		}

		// cleanup
		manager.Stop()
	})
}

// test manager stop
func TestScrapeManager_Stop(t *testing.T) {
	t.Run("stops running job", func(t *testing.T) {
		manager := NewScrapeManager(&MockScraper{})

		_, err := manager.Start(context.Background(), ScrapeOptions{
			Channel: "test",
		})
		if err != nil {
			t.Fatalf("Start() error: %v", err)
		}

		// verify job is running
		if manager.Current() == nil {
			t.Fatal("Current() should return job before stop")
		}

		// stop
		manager.Stop()

		// give a bit of time for cleanup
		time.Sleep(10 * time.Millisecond)

		// verify job is stopped
		if manager.Current() != nil {
			t.Error("Current() should return nil after stop")
		}
	})

	t.Run("safe to call when not running", func(t *testing.T) {
		manager := NewScrapeManager(&MockScraper{})

		// should not panic
		manager.Stop()
		manager.Stop()
		manager.Stop()
	})
}

// test manager current
func TestScrapeManager_Current(t *testing.T) {
	t.Run("returns nil when not running", func(t *testing.T) {
		manager := NewScrapeManager(&MockScraper{})

		if manager.Current() != nil {
			t.Error("Current() should return nil when not running")
		}
	})

	t.Run("returns job when running", func(t *testing.T) {
		manager := NewScrapeManager(&MockScraper{})

		job, _ := manager.Start(context.Background(), ScrapeOptions{
			Channel: "test",
		})

		current := manager.Current()
		if current == nil {
			t.Fatal("Current() should return job when running")
		}
		if current.ID != job.ID {
			t.Error("Current() should return the same job")
		}

		manager.Stop()
	})
}

// test concurrent access
func TestScrapeManager_ConcurrentAccess(t *testing.T) {
	manager := NewScrapeManager(&MockScraper{})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			manager.Start(context.Background(), ScrapeOptions{})
			manager.Current()
			manager.Stop()
		}()
	}
	wg.Wait()
	// if we get here without panic, test passes
}
