package integration

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/blockedby/positions-os/internal/config"
	"github.com/blockedby/positions-os/internal/telegram"
	"github.com/celestix/gotgproto"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestTelegramAuth_EmptyDB_StatusUnauthorized(t *testing.T) {
	// We'll skip if INTEGRATION_TEST is not set, or just run it since it uses in-memory DB
	if os.Getenv("INTEGRATION_TEST") == "" {
		t.Skip("Skipping integration test; set INTEGRATION_TEST=1")
	}

	// Arrange - fresh in-memory DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Create sessions table (empty) - using the production schema
	db.Exec("CREATE TABLE sessions (version integer primary key, data blob)")

	cfg := &config.Config{
		TGApiID:   12345,
		TGApiHash: "test_hash",
	}

	m := telegram.NewManager(cfg, db)

	// Act
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = m.Init(ctx)

	// Assert
	require.NoError(t, err, "Init should not return error")
	assert.Equal(t, telegram.StatusUnauthorized, m.GetStatus(),
		"Empty DB should result in UNAUTHORIZED status")
}

func TestTelegramAuth_SessionInDB_StatusReady(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") == "" {
		t.Skip("Skipping integration test; set INTEGRATION_TEST=1")
	}

	// Arrange
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	db.Exec("CREATE TABLE sessions (version integer primary key, data blob)")
	// Seed a mock session
	db.Exec("INSERT INTO sessions (version, data) VALUES (1, ?)",
		[]byte(`{"DC":2,"AuthKey":"dGVzdA=="}`))

	cfg := &config.Config{
		TGApiID:   12345,
		TGApiHash: "test_hash",
	}

	m := telegram.NewManager(cfg, db)

	// Mock the client factory to avoid network calls
	m.SetClientFactory(func(ctx context.Context, cfg *config.Config, db *gorm.DB) (*gotgproto.Client, error) {
		return &gotgproto.Client{}, nil // Mock success
	})

	// Act
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = m.Init(ctx)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, telegram.StatusReady, m.GetStatus(),
		"Session in DB should result in READY status")
}

func TestTelegramAuth_InvalidSession_FallbackUnauthorized(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") == "" {
		t.Skip("Skipping integration test; set INTEGRATION_TEST=1")
	}

	// Arrange
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	db.Exec("CREATE TABLE sessions (version integer primary key, data blob)")
	// Seed a corrupted/invalid session
	db.Exec("INSERT INTO sessions (version, data) VALUES (1, ?)",
		[]byte(`invalid-json-garbage`))

	cfg := &config.Config{
		TGApiID:   12345,
		TGApiHash: "test_hash",
	}

	m := telegram.NewManager(cfg, db)

	// Mock the client factory to return error on invalid session
	m.SetClientFactory(func(ctx context.Context, cfg *config.Config, db *gorm.DB) (*gotgproto.Client, error) {
		return nil, errors.New("invalid session data")
	})

	// Act
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = m.Init(ctx)

	// Assert
	require.NoError(t, err, "Init should not return error on factory failure")
	assert.Equal(t, telegram.StatusUnauthorized, m.GetStatus(),
		"Invalid session should fallback to UNAUTHORIZED status")
}

func TestTelegramAuth_SessionPersistence_Restart(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") == "" {
		t.Skip("Skipping integration test; set INTEGRATION_TEST=1")
	}

	// 1. Setup shared DB
	// We use a file-based sqlite for "restart" simulation, or just don't close the in-memory connection
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	db.Exec("CREATE TABLE sessions (version integer primary key, data blob)")

	cfg := &config.Config{TGApiID: 12345, TGApiHash: "test_hash"}

	// 2. Initial state: Unauthorized
	m1 := telegram.NewManager(cfg, db)
	require.NoError(t, m1.Init(context.Background()))
	assert.Equal(t, telegram.StatusUnauthorized, m1.GetStatus())

	// 3. Simulate successful login by saving session directly
	// (In real life this happens inside StartQR)
	sessionData := []byte(`{"DC":2,"Addr":"1.2.3.4:443","AuthKey":"dGVzdA=="}`)
	db.Exec("INSERT INTO sessions (version, data) VALUES (1, ?)", sessionData)

	// 4. "Restart" - Create new manager instance
	m2 := telegram.NewManager(cfg, db)
	// Mock factory to avoid network
	m2.SetClientFactory(func(ctx context.Context, cfg *config.Config, db *gorm.DB) (*gotgproto.Client, error) {
		return &gotgproto.Client{}, nil
	})

	err = m2.Init(context.Background())

	// 5. Assert: Status should be READY now
	require.NoError(t, err)
	assert.Equal(t, telegram.StatusReady, m2.GetStatus(), "Session should persist across restarts")
}
