// Package migrator handles database schema migrations using golang-migrate.
package migrator

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// Migrator manages database migrations.
type Migrator struct {
	migrationsFS fs.FS
}

// NewWithFS creates a new Migrator with the given filesystem.
// The fs should contain .sql migration files.
func NewWithFS(migrationsFS fs.FS) (*Migrator, error) {
	if migrationsFS == nil {
		return nil, errors.New("migrationsFS cannot be nil")
	}

	return &Migrator{
		migrationsFS: migrationsFS,
	}, nil
}

// Up runs all pending migrations.
func (m *Migrator) Up(ctx context.Context, databaseURL string) error {
	if databaseURL == "" {
		return errors.New("database URL cannot be empty")
	}

	// Create source driver from embedded filesystem
	sourceDriver, err := iofs.New(m.migrationsFS, ".")
	if err != nil {
		return fmt.Errorf("create iofs source: %w", err)
	}

	// Create migrate instance
	migrator, err := migrate.NewWithSourceInstance("iofs", sourceDriver, databaseURL)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer migrator.Close()

	// Run migrations
	if err := migrator.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			// No migrations to run - this is fine
			return nil
		}
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}

// Version returns the current migration version and dirty state.
func (m *Migrator) Version(ctx context.Context, databaseURL string) (version uint, dirty bool, err error) {
	if databaseURL == "" {
		return 0, false, errors.New("database URL cannot be empty")
	}

	sourceDriver, err := iofs.New(m.migrationsFS, ".")
	if err != nil {
		return 0, false, fmt.Errorf("create iofs source: %w", err)
	}

	migrator, err := migrate.NewWithSourceInstance("iofs", sourceDriver, databaseURL)
	if err != nil {
		return 0, false, fmt.Errorf("create migrator: %w", err)
	}
	defer migrator.Close()

	version, dirty, err = migrator.Version()
	if err != nil {
		if errors.Is(err, migrate.ErrNilVersion) {
			// No migrations have been run yet
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("get version: %w", err)
	}

	return version, dirty, nil
}
