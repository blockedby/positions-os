package telegram

import (
	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"

	"github.com/blockedby/positions-os/internal/config"
)

// QRClientBundle contains all components needed for QR authentication
type QRClientBundle struct {
	Client     *telegram.Client
	Dispatcher tg.UpdateDispatcher
	Storage    *session.StorageMemory
}

// NewQRClient creates a raw td/telegram client suitable for QR authentication.
// Unlike gotgproto's NewClient, this does NOT attempt interactive CLI auth.
func NewQRClient(cfg *config.Config) (*QRClientBundle, error) {
	memStorage := &session.StorageMemory{}
	// Create dispatcher with initialized map to avoid "assignment to entry in nil map" panic
	dispatcher := tg.NewUpdateDispatcher()

	client := telegram.NewClient(cfg.TGApiID, cfg.TGApiHash, telegram.Options{
		SessionStorage: memStorage,
		UpdateHandler:  &dispatcher,
	})

	return &QRClientBundle{
		Client:     client,
		Dispatcher: dispatcher,
		Storage:    memStorage,
	}, nil
}
