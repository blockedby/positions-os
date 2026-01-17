package web

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer_Starts(t *testing.T) {
	cfg := &Config{Port: 0} // random port
	srv := NewServer(cfg, nil, nil)

	go func() { _ = srv.Start() }()
	defer func() { _ = srv.Stop(context.Background()) }()

	// wait for server to be ready
	require.Eventually(t, func() bool {
		resp, err := http.Get(srv.BaseURL() + "/health")
		if err != nil {
			return false
		}
		defer func() { _ = resp.Body.Close() }()
		return resp.StatusCode == 200
	}, 2*time.Second, 100*time.Millisecond)
}

func TestServer_ServesStatic(t *testing.T) {
	// Create test static files
	staticDir := filepath.Join(t.TempDir(), "static")
	cssDir := filepath.Join(staticDir, "css")
	require.NoError(t, os.MkdirAll(cssDir, 0755))

	cssContent := "body { background: #0d1117; }"
	require.NoError(t, os.WriteFile(filepath.Join(cssDir, "style.css"), []byte(cssContent), 0644))

	jsDir := filepath.Join(staticDir, "js")
	require.NoError(t, os.MkdirAll(jsDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(jsDir, "ws.js"), []byte("console.log('ws')"), 0644))

	cfg := &Config{Port: 0, StaticDir: staticDir}
	srv := NewServer(cfg, nil, nil)

	go func() { _ = srv.Start() }()
	defer func() { _ = srv.Stop(context.Background()) }()

	// Wait for server
	time.Sleep(50 * time.Millisecond)

	resp, err := http.Get(srv.BaseURL() + "/static/css/style.css")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "text/css")

	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, cssContent, string(body))

	// Check JS
	respJS, err := http.Get(srv.BaseURL() + "/static/js/ws.js")
	require.NoError(t, err)
	defer func() { _ = respJS.Body.Close() }()
	assert.Equal(t, http.StatusOK, respJS.StatusCode)
}

func TestServer_HealthEndpoint(t *testing.T) {
	cfg := &Config{Port: 0}
	srv := NewServer(cfg, nil, nil)

	go func() { _ = srv.Start() }()
	defer func() { _ = srv.Stop(context.Background()) }()

	time.Sleep(50 * time.Millisecond)

	resp, err := http.Get(srv.BaseURL() + "/health")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var health struct {
		Status  string `json:"status"`
		Version string `json:"version"`
	}
	err = json.NewDecoder(resp.Body).Decode(&health)
	require.NoError(t, err)
	assert.Equal(t, "ok", health.Status)
	assert.NotEmpty(t, health.Version)
}

func TestServer_WebSocket(t *testing.T) {
	cfg := &Config{Port: 0}

	// Create hub
	hub := NewHub()
	go hub.Run()

	srv := NewServer(cfg, nil, hub)
	go func() { _ = srv.Start() }()
	defer func() { _ = srv.Stop(context.Background()) }()

	// Wait for server to start
	time.Sleep(50 * time.Millisecond)

	// Build WS URL
	u := url.URL{Scheme: "ws", Host: srv.listener.Addr().String(), Path: "/ws"}

	// Connect
	c, wsResp, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.NoError(t, err)
	defer func() { _ = c.Close() }()
	if wsResp != nil && wsResp.Body != nil {
		defer func() { _ = wsResp.Body.Close() }()
	}
}

func TestServer_RegisterJobsHandler(t *testing.T) {
	cfg := &Config{Port: 0}
	srv := NewServer(cfg, nil, nil)

	// mock handler
	handler := &mockJobsAPIHandler{}
	srv.RegisterJobsHandler(handler)

	go func() { _ = srv.Start() }()
	defer func() { _ = srv.Stop(context.Background()) }()
	time.Sleep(50 * time.Millisecond)

	resp, err := http.Get(srv.BaseURL() + "/api/v1/jobs")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// mockJobsAPIHandler implements jobsHandler interface for testing
type mockJobsAPIHandler struct{}

func (h *mockJobsAPIHandler) List(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
func (h *mockJobsAPIHandler) GetByID(_ http.ResponseWriter, _ *http.Request)      {}
func (h *mockJobsAPIHandler) UpdateStatus(_ http.ResponseWriter, _ *http.Request) {}

// mockBrainHandler implements brainHandler interface for testing
type mockBrainHandler struct{}

func (m *mockBrainHandler) PrepareJob(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (m *mockBrainHandler) GetDocuments(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (m *mockBrainHandler) DownloadResume(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func TestServer_RegisterBrainHandler(t *testing.T) {
	// Create server with minimal config
	cfg := &Config{Port: 0, StaticDir: ""}
	server := NewServer(cfg, nil, nil)

	// Register mock handler
	mockHandler := &mockBrainHandler{}
	server.RegisterBrainHandler(mockHandler)

	// Test that POST /api/v1/jobs/{id}/prepare route exists
	req := httptest.NewRequest("POST", "/api/v1/jobs/test-id/prepare", nil)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	// Should not be 404 (route exists)
	if w.Code == http.StatusNotFound {
		t.Error("POST /api/v1/jobs/{id}/prepare should be registered")
	}

	// Test that GET /api/v1/jobs/{id}/documents route exists
	req2 := httptest.NewRequest("GET", "/api/v1/jobs/test-id/documents", nil)
	w2 := httptest.NewRecorder()
	server.router.ServeHTTP(w2, req2)

	if w2.Code == http.StatusNotFound {
		t.Error("GET /api/v1/jobs/{id}/documents should be registered")
	}
}
