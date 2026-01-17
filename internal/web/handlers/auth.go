package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/blockedby/positions-os/internal/telegram"
)

// AuthHandler handles authentication related requests
type AuthHandler struct {
	client TelegramClient
	hub    HubBroadcaster // Interface for Hub
}

// HubBroadcaster defines the interface for broadcasting messages to connected clients.
type HubBroadcaster interface {
	Broadcast(message interface{})
}

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler(client TelegramClient, hub HubBroadcaster) *AuthHandler {
	return &AuthHandler{
		client: client,
		hub:    hub,
	}
}

// GetStatus returns the current Telegram authentication status
func (h *AuthHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	status := h.client.GetStatus()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":         string(status),
		"is_ready":       status == telegram.StatusReady,
		"qr_in_progress": h.client.IsQRInProgress(),
	}); err != nil {
		_ = err // Client disconnected
	}
}

// StartQR initiates the QR code login flow
func (h *AuthHandler) StartQR(w http.ResponseWriter, r *http.Request) {
	// 1. Check current status
	if h.client.GetStatus() == telegram.StatusReady {
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": "already logged in"}); err != nil {
			_ = err // Client disconnected
		}
		return
	}

	// 2. Check if QR flow is already in progress
	if h.client.IsQRInProgress() {
		w.WriteHeader(http.StatusAccepted)
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "already in progress"}); err != nil {
			_ = err // Client disconnected
		}
		return
	}

	// 3. Start QR flow in background
	go func() {
		ctx := context.Background()
		err := h.client.StartQR(ctx, func(url string) {
			// Send QR code to WebSocket
			if h.hub != nil {
				h.hub.Broadcast(map[string]string{
					"type": "tg_qr",
					"url":  url,
				})
			}
		})

		if err != nil {
			// Broadcast error (but not for context cancellation, which is normal)
			if err != context.Canceled && h.hub != nil {
				h.hub.Broadcast(map[string]string{
					"type":    "error",
					"message": err.Error(),
				})
			}
			return
		}

		// Broadcast success on nil error
		if h.hub != nil {
			h.hub.Broadcast(map[string]string{
				"type": "tg_auth_success",
			})
		}
	}()

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "started"}); err != nil {
		_ = err // Client disconnected
	}
}
