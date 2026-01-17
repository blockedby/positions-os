package telegram

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/blockedby/positions-os/internal/config"
)

func TestClient_API_UnauthorizedError(t *testing.T) {
	// Arrange
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	cfg := &config.Config{}
	manager := NewManager(cfg, db)
	// Manager status is INITIALIZING by default, then we don't Init it, so GetClient returns nil

	client := NewClient(manager)

	// Act
	api, err := client.API()

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "telegram client not authorized")
	assert.Nil(t, api)
}

func TestClient_ResolveChannel_UnauthorizedError(t *testing.T) {
	// Arrange
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	cfg := &config.Config{}
	manager := NewManager(cfg, db)
	client := NewClient(manager)

	// Act
	channel, err := client.ResolveChannel(context.Background(), "testchannel")

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "telegram client not authorized")
	assert.Nil(t, channel)
}
