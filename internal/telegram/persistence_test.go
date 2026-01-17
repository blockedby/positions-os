package telegram

import (
	"context"
	"testing"
	"time"

	"github.com/blockedby/positions-os/internal/config"
	"github.com/celestix/gotgproto"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// Setup in-memory DB for testing
func setupTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	return db
}

type MockSession struct {
	SessionID string `gorm:"primaryKey"`
	Data      []byte
}

func (MockSession) TableName() string {
	return "sessions"
}

func TestDBSession(t *testing.T) {
	db := setupTestDB()
	// Manually migrate the session table to ensure it exists
	// In real app, gotgproto handles this, but here we want to maniplate it
	_ = db.AutoMigrate(&MockSession{})

	cfg := &config.Config{
		TGApiID:   12345,
		TGApiHash: "test_hash",
	}

	// We need to inject a client factory into Manager to avoid real network calls
	// But for the purpose of the "persistence check", we care about how Manager *decides* status based on DB/Client init.
	// If we can't change Manager internal easily, we might need to rely on NewPersistentClient behavior.

	// Assuming we modify Manager to allow checking DB state or we rely on NewPersistentClient failing/succeeding?
	// Let's assume we implement logic in Manager/Factory: "If DB empty, fail Init (return Unauthorized)"

	m := NewManager(cfg, db)

	// Override factory to prevent real connection attempt, we just want to test the logic flow
	// dependent on DB state if we implement that logic in Init before calling factory.
	// OR if we implement it inside factory.

	// Let's assume the logic is: Manager.Init checks DB. If empty -> Unauthorized.
	// If not empty -> call factory -> Ready.

	// Since we haven't implemented that logic yet, this test will FAIL (Red)
	// because current implementation just calls NewPersistentClient which tries to connect (and might hang or fail differently).
	// To make it deterministic, we'll need to control the behavior.

	// For now, let's write the test as if Manager handles this check.

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// 1. Empty DB -> Expect Unauthorized
	err := m.Init(ctx)
	assert.Nil(t, err)
	assert.Equal(t, StatusUnauthorized, m.GetStatus(), "Should be Unauthorized when DB is empty")

	// 2. Populated DB -> Expect Ready
	// Simulate a session
	db.Create(&MockSession{SessionID: "1", Data: []byte(`{"mock":"data"}`)})

	// We need to reset manager state effectively
	// But since we are mocking the connection part via expectation that logic exists:

	// NOTE: Real NewPersistentClient would try to connect here.
	// We MUST mock the client creation part to avoid network.
	m.SetClientFactory(func(ctx context.Context, cfg *config.Config, db *gorm.DB) (*gotgproto.Client, error) {
		// Mock success - return a stub client
		return &gotgproto.Client{}, nil
	})

	err = m.Init(ctx)
	assert.Nil(t, err)
	assert.Equal(t, StatusReady, m.GetStatus(), "Should be Ready when DB has session")
}

// MockClient to satisfy the interface (we need to define an interface for Client if we want full mocking,
// but Manager currently uses concrete *gotgproto.Client.
// We might need to abstract it or just return nil for the test if we are testing the status logic *before* client use).
//
// Issue: Manager uses `*gotgproto.Client`. We can't return a MockClient struct unless it embeds *gotgproto.Client
// or Manager works with an interface.
//
// Compromise for TDD step:
// The critical part of "Persistence Test" is that the Application *knows* to look in DB.
// Let's implement the `CheckSession` logic in Manager.
