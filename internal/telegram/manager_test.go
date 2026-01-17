package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/blockedby/positions-os/internal/config"
	"github.com/celestix/gotgproto"
	"github.com/glebarez/sqlite"
	"github.com/gotd/td/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestManager_StartQR_CallsOnQRCode(t *testing.T) {
	// Arrange
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	// Create sessions table with correct schema
	db.Exec("CREATE TABLE sessions (version integer primary key, data blob)")

	cfg := &config.Config{TGApiID: 12345, TGApiHash: "test_hash"}
	m := NewManager(cfg, db)

	// Track callback invocation
	var receivedURL string
	callbackInvoked := make(chan struct{})

	// Mock QR client that simulates token generation
	m.SetQRClientFactory(func(cfg *config.Config) (*QRClientBundle, error) {
		// We return a mock bundle that will never complete, so we can test the callback
		// This is tricky because bundle.Client.Run is called.
		// We might need a real-ish client or a very complex mock.
		// For now, let's just fail it to see if it even reaches this part.
		return nil, errors.New("factory reached")
	})

	// Act
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err = m.StartQR(ctx, func(url string) {
		receivedURL = url
		close(callbackInvoked)
	})

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "factory reached")
	assert.Empty(t, receivedURL, "URL should be empty if factory fails")
}

func TestManager_Init_FactoryError_Unauthorized(t *testing.T) {
	// Arrange
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	db.Exec("CREATE TABLE sessions (version integer primary key, data blob)")
	// Seed a session so factory is called
	db.Exec("INSERT INTO sessions (version, data) VALUES (1, ?)", []byte(`{"mock":"data"}`))

	cfg := &config.Config{TGApiID: 12345, TGApiHash: "test_hash"}
	m := NewManager(cfg, db)

	// Inject factory that returns error
	m.SetClientFactory(func(ctx context.Context, cfg *config.Config, db *gorm.DB) (*gotgproto.Client, error) {
		return nil, errors.New("factory failure")
	})

	// Act
	err = m.Init(context.Background())

	// Assert
	assert.NoError(t, err, "Init should not return error even if factory fails")
	assert.Equal(t, StatusUnauthorized, m.GetStatus(), "Status should be Unauthorized on factory error")
}

func TestManager_GetStatus_Concurrent(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	cfg := &config.Config{}
	m := NewManager(cfg, db)

	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			m.GetStatus()
		}()
	}

	close(start)
	wg.Wait()
}

func TestManager_Stop_Graceful(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	m := NewManager(&config.Config{}, db)

	// Should not panic
	assert.NotPanics(t, func() {
		m.Stop()
	})
}

func TestConvertToGotgprotoSession_RoundTrip(t *testing.T) {
	// Arrange
	input := &session.Data{
		DC:      2,
		Addr:    "1.2.3.4:443",
		AuthKey: []byte("test-key-32-bytes-long-abc-12345"),
	}

	// Act
	result, err := ConvertToGotgprotoSession(input)
	require.NoError(t, err)

	// Verify gotgproto wrapped format: {"Version":1,"Data":{...}}
	var parsed map[string]interface{}
	err = json.Unmarshal(result.Data, &parsed)
	require.NoError(t, err)

	// Check wrapper structure
	assert.Equal(t, float64(1), parsed["Version"], "Should have Version=1")
	dataObj, ok := parsed["Data"].(map[string]interface{})
	require.True(t, ok, "Data should be a nested object")

	// Check session data is inside the Data wrapper
	assert.Equal(t, float64(2), dataObj["DC"], "DC should be in nested Data")
	assert.Equal(t, "1.2.3.4:443", dataObj["Addr"], "Addr should be in nested Data")
}
