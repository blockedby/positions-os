package telegram

import (
	"context"
	"errors"
	"testing"

	"github.com/blockedby/positions-os/internal/config"
	"github.com/celestix/gotgproto"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// TestManager_StartQR_UsesQRFactory verifies that StartQR uses the injectable QRClientFactory.
func TestManager_StartQR_UsesQRFactory(t *testing.T) {
	// 1. Setup
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}
	// Migrate sessions table
	if err := db.Exec("CREATE TABLE sessions (id text primary key, session text)").Error; err != nil {
		t.Fatalf("failed to create sessions table: %v", err)
	}

	cfg := &config.Config{
		TGApiID:   12345,
		TGApiHash: "test_hash",
	}

	m := NewManager(cfg, db)

	// 2. Define a sentinel error
	mockErr := errors.New("mock factory called")

	// 3. Inject the QR mock factory
	qrCalled := false
	m.SetQRClientFactory(func(cfg *config.Config) (*QRClientBundle, error) {
		qrCalled = true
		return nil, mockErr
	})

	// Inject regular factory to ensure it's NOT called
	regularCalled := false
	m.SetClientFactory(func(ctx context.Context, cfg *config.Config, db *gorm.DB) (*gotgproto.Client, error) {
		regularCalled = true
		return nil, errors.New("regular factory called")
	})

	// 4. Act
	err = m.StartQR(context.Background(), func(url string) {})

	// 5. Assert
	if !qrCalled {
		t.Error("StartQR did NOT call the QRClientFactory")
	}
	if regularCalled {
		t.Error("StartQR SHOULD NOT call the regular ClientFactory")
	}
	if !errors.Is(err, mockErr) {
		// StartQR returns "create QR client: mock factory called"
		// errors.Is handles wrapped errors
		t.Errorf("StartQR returned wrong error. Expected wrapped '%v', got '%v'", mockErr, err)
	}
}
