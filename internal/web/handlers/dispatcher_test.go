package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDispatcherHandler(t *testing.T) {
	handler := NewDispatcherHandler()

	assert.NotNil(t, handler)
	assert.NotNil(t, handler.log)
}

func TestDispatcherHandler_Status(t *testing.T) {
	handler := NewDispatcherHandler()

	req := httptest.NewRequest("GET", "/api/v1/dispatcher/status", nil)
	w := httptest.NewRecorder()

	handler.Status(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp StatusResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	assert.NoError(t, err)

	assert.Equal(t, "healthy", resp.Status)
	assert.True(t, resp.TelegramAvailable)
	assert.False(t, resp.EmailAvailable)
	assert.Equal(t, 0, resp.ActiveSends)
	assert.Equal(t, 0.1, resp.RateLimitPerSecond)
	assert.Equal(t, "dev", resp.Version)
}

func TestDispatcherHandler_DispatcherStatus(t *testing.T) {
	handler := NewDispatcherHandler()

	req := httptest.NewRequest("GET", "/api/v1/dispatcher/status", nil)
	w := httptest.NewRecorder()

	handler.DispatcherStatus(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp StatusResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	assert.NoError(t, err)

	assert.Equal(t, "healthy", resp.Status)
}
