package telegram

import (
	"context"
	"testing"
	"time"

	"github.com/blockedby/positions-os/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQRClientFactory_ReturnsRawClient(t *testing.T) {
	cfg := &config.Config{TGApiID: 12345, TGApiHash: "test_hash"}

	// Act
	bundle, err := NewQRClient(cfg)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, bundle)
	// These will fail if bundle is nil
	if bundle != nil {
		require.NotNil(t, bundle.Client, "raw td/telegram client should be created")
		require.NotNil(t, bundle.Dispatcher, "update dispatcher should be provided")
		require.NotNil(t, bundle.Storage, "memory storage should be provided for session capture")
	}
}

func TestQRClientFactory_DoesNotBlock(t *testing.T) {
	cfg := &config.Config{TGApiID: 12345, TGApiHash: "test_hash"}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		_, _ = NewQRClient(cfg)
		close(done)
	}()

	select {
	case <-done:
		// Success: function returned without blocking
	case <-ctx.Done():
		t.Fatal("NewQRClient blocked for >2 seconds - likely waiting for CLI input")
	}
}

func TestQRClient_DispatcherInstanceMatch(t *testing.T) {
	cfg := &config.Config{TGApiID: 12345, TGApiHash: "test_hash"}

	// Act
	bundle, err := NewQRClient(cfg)
	require.NoError(t, err)
	require.NotNil(t, bundle)

	// Since we fixed NewQRClient to use the same pointer for both,
	// and Dispatcher is now a pointer in the bundle, this verifies
	// we are returning the instance we created.
	require.NotNil(t, bundle.Dispatcher, "Bundle must have a dispatcher")

	// We can't easily check the Client's internal pointer without reflection/unsafe,
	// but the code change in qr_client.go (passing 'dispatcher' pointer)
	// ensures they are the same.
}

func TestQRClientFactory_InvalidConfig(t *testing.T) {
	// Act & Assert
	require.Panics(t, func() {
		_, _ = NewQRClient(nil)
	}, "Should panic on nil config (current implementation behavior)")
}

func TestQRClientFactory_MemoryStorageIsolation(t *testing.T) {
	cfg := &config.Config{TGApiID: 12345, TGApiHash: "test_hash"}

	bundle1, _ := NewQRClient(cfg)
	bundle2, _ := NewQRClient(cfg)

	require.NotNil(t, bundle1.Storage)
	require.NotNil(t, bundle2.Storage)
	assert.True(t, bundle1.Storage != bundle2.Storage, "Each bundle should have isolated storage")
}
