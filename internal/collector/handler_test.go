package collector

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/blockedby/positions-os/internal/telegram"
)

// test health endpoint
func TestHandler_Health(t *testing.T) {
	handler := NewHandler(NewScrapeManager(&MockScraper{}), nil)
	router := NewRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Health() status = %d, want %d", rec.Code, http.StatusOK)
	}
}

// test start scrape endpoint
func TestHandler_StartScrape(t *testing.T) {
	t.Run("returns 400 on empty request", func(t *testing.T) {
		handler := NewHandler(NewScrapeManager(&MockScraper{}), nil)
		router := NewRouter(handler)

		body := `{}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/scrape/telegram", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("StartScrape() status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("returns 400 on invalid json", func(t *testing.T) {
		handler := NewHandler(NewScrapeManager(&MockScraper{}), nil)
		router := NewRouter(handler)

		body := `not json`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/scrape/telegram", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("StartScrape() status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("returns 400 on negative limit", func(t *testing.T) {
		handler := NewHandler(NewScrapeManager(&MockScraper{}), nil)
		router := NewRouter(handler)

		body := `{"channel": "@test", "limit": -1}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/scrape/telegram", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("StartScrape() status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("returns 200 on valid request", func(t *testing.T) {
		handler := NewHandler(NewScrapeManager(&MockScraper{}), nil)
		router := NewRouter(handler)

		body := `{"channel": "@test_channel"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/scrape/telegram", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("StartScrape() status = %d, want %d", rec.Code, http.StatusOK)
		}

		var resp ScrapeResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.Status != "running" {
			t.Errorf("response status = %s, want running", resp.Status)
		}
	})

	t.Run("returns 409 when already running", func(t *testing.T) {
		manager := NewScrapeManager(&MockScraper{Delay: 100 * time.Millisecond})
		handler := NewHandler(manager, nil)
		router := NewRouter(handler)

		// start first job
		body := `{"channel": "@first"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/scrape/telegram", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("first request failed: %d", rec.Code)
		}

		// try second job
		body = `{"channel": "@second"}`
		req = httptest.NewRequest(http.MethodPost, "/api/v1/scrape/telegram", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusConflict {
			t.Errorf("second request status = %d, want %d", rec.Code, http.StatusConflict)
		}
	})
}

// test stop scrape endpoint
func TestHandler_StopScrape(t *testing.T) {
	t.Run("returns 200 even when not running", func(t *testing.T) {
		handler := NewHandler(NewScrapeManager(&MockScraper{}), nil)
		router := NewRouter(handler)

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/scrape/current", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("StopScrape() status = %d, want %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("stops running job", func(t *testing.T) {
		manager := NewScrapeManager(&MockScraper{})
		handler := NewHandler(manager, nil)
		router := NewRouter(handler)

		// start a job first
		body := `{"channel": "@test"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/scrape/telegram", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		// stop it
		req = httptest.NewRequest(http.MethodDelete, "/api/v1/scrape/current", nil)
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("StopScrape() status = %d, want %d", rec.Code, http.StatusOK)
		}

		// verify it stopped
		if manager.Current() != nil {
			t.Error("job should be stopped")
		}
	})
}

// test status endpoint
func TestHandler_Status(t *testing.T) {
	t.Run("returns no job when not running", func(t *testing.T) {
		handler := NewHandler(NewScrapeManager(&MockScraper{}), nil)
		router := NewRouter(handler)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/scrape/status", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Status() status = %d, want %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("returns telegram_status READY when connected", func(t *testing.T) {
		// RED Phase: This test expects telegram_status to be returned
		// Currently it will fail because the Status handler doesn't return telegram_status
		mockScraper := &MockScraper{
			// MockScraper already returns StatusReady from GetTelegramStatus()
		}
		handler := NewHandler(NewScrapeManager(mockScraper), nil)
		router := NewRouter(handler)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/scrape/status", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("Status() status = %d, want %d", rec.Code, http.StatusOK)
		}

		var resp map[string]interface{}
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		// This will fail because telegram_status is not returned
		telegramStatus, ok := resp["telegram_status"]
		if !ok {
			t.Error("RED FAIL: telegram_status key is missing from response. Current keys:", keys(resp))
		}
		if ok && telegramStatus != "READY" {
			t.Errorf("telegram_status = %v, want READY", telegramStatus)
		}
	})
}

func keys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// test list topics endpoint
func TestHandler_ListForumTopics(t *testing.T) {
	t.Run("returns topics with correct json keys", func(t *testing.T) {
		mockScraper := &MockScraper{
			TopicsToReturn: []telegram.Topic{
				{ID: 1, Title: "General"},
			},
		}
		handler := NewHandler(NewScrapeManager(mockScraper), nil)
		router := NewRouter(handler)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/tools/telegram/topics?channel=@test", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("ListForumTopics() status = %d, want %d", rec.Code, http.StatusOK)
		}

		var resp []map[string]interface{}
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(resp) == 0 {
			t.Fatal("response is empty")
		}

		topic := resp[0]
		if _, ok := topic["id"]; !ok {
			t.Error("JSON key 'id' missing (check case?)")
		}
		if _, ok := topic["title"]; !ok {
			t.Error("JSON key 'title' missing (check case?)")
		}
	})
}
