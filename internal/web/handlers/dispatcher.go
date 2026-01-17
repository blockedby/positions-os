package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/blockedby/positions-os/internal/logger"
)

// DispatcherHandler provides dispatcher service status information.
type DispatcherHandler struct {
	log *logger.Logger
}

// NewDispatcherHandler creates a new DispatcherHandler.
func NewDispatcherHandler() *DispatcherHandler {
	return &DispatcherHandler{
		log: logger.Get(),
	}
}

// StatusResponse represents the dispatcher service status.
type StatusResponse struct {
	// Service health status
	Status string `json:"status"` // "healthy", "degraded", "down"

	// Telegram sender availability
	TelegramAvailable bool `json:"telegram_available"`

	// Email sender availability
	EmailAvailable bool `json:"email_available"`

	// Active sends in progress
	ActiveSends int `json:"active_sends"`

	// Rate limiting info
	RateLimitPerSecond float64 `json:"rate_limit_per_second"`

	// Additional service info
	Version string `json:"version"`
}

// Status returns the current status of the dispatcher service.
// GET /api/v1/dispatcher/status
func (h *DispatcherHandler) Status(w http.ResponseWriter, r *http.Request) {
	// For now, return a static healthy status
	// In production, this would check actual service health
	resp := StatusResponse{
		Status:            "healthy",
		TelegramAvailable: true,  // TODO: Check actual Telegram client status
		EmailAvailable:    false, // TODO: Check email configuration
		ActiveSends:       0,     // TODO: Track active sends
		RateLimitPerSecond: 0.1,  // 1 request per 10 seconds
		Version:           "dev",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		_ = err // Client disconnected
	}
}

// DispatcherStatus is an alias for Status for clarity.
func (h *DispatcherHandler) DispatcherStatus(w http.ResponseWriter, r *http.Request) {
	h.Status(w, r)
}
