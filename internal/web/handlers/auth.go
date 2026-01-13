package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/blockedby/positions-os/internal/telegram"
	// "github.com/blockedby/positions-os/internal/web" // for Hub interface if we had one here, pass generic for now
)

// AuthHandler handles authentication related requests
type AuthHandler struct {
	client TelegramClient
	hub    HubBroadcaster // Interface for Hub
}

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

// StartQR initiates the QR code login flow
func (h *AuthHandler) StartQR(w http.ResponseWriter, r *http.Request) {
	// 1. Check current status
	if h.client.GetStatus() == telegram.StatusReady {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "already logged in"})
		return
	}

	// 2. Start QR flow in background
	// We create a background context or use one linked to server?
	// If request context dies, we might stop?
	// Usually auth flow should persist a bit or be tied to a session.
	// For simplicity, let's use a detached background context, OR keep it until successful.
	// Ideally, if the user leaves the page, we might want to cancel?
	// But Manager.StartQR takes a context.

	go func() {
		ctx := context.Background() // TODO: Handle cancellation
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
			// Broadcast error
			if h.hub != nil {
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
	json.NewEncoder(w).Encode(map[string]string{"status": "started"})
}
