package handlers

import (
	"context"

	"github.com/blockedby/positions-os/internal/telegram"
)

// TelegramClient defines the interface required by AuthHandler
type TelegramClient interface {
	StartQR(ctx context.Context, onQRCode func(url string)) error
	GetStatus() telegram.Status
}
