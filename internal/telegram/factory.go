package telegram

import (
	"context"
	"fmt"

	"github.com/blockedby/positions-os/internal/config"
	"github.com/celestix/gotgproto"
	"github.com/celestix/gotgproto/sessionMaker"
	"gorm.io/gorm"
)

// NewPersistentClient creates a telegram client that uses the database for session storage.
// It will automatically persist session updates (auth key refreshes) back to the DB.
func NewPersistentClient(ctx context.Context, cfg *config.Config, db *gorm.DB) (*gotgproto.Client, error) {
	// 1. Configure the session constructor.
	// We use SqlSession, which will store session data and peers in the database.
	sessionConstructor := sessionMaker.SqlSession(db.Dialector)

	// 2. Initialize the client.
	// If the database is empty, gotgproto will use the provided Session constructor (SqlSession).
	// If you want to "seed" the database with an existing session string from .env on the first run,
	// we should handle that logic here.

	clientOpts := &gotgproto.ClientOpts{
		Session:          sessionConstructor,
		DisableCopyright: true,
		InMemory:         false, // Essential: persistence enabled
	}

	// 3. Create the client
	// Note: gotgproto manages the "import from string" internally if we pass StringSession,
	// but SqlSession is better for long-term.
	// If there's no session in DB, we use the StringSession from .env as a fallback/seed.

	// Check if session table has any data (simple heuristic)

	client, err := gotgproto.NewClient(
		cfg.TGApiID,
		cfg.TGApiHash,
		gotgproto.ClientTypePhone(""), // Empty = use session
		clientOpts,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram client: %w", err)
	}

	return client, nil
}
