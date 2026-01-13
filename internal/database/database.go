// package database provides postgresql connection management.
package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// DB wraps a postgresql connection pool and GORM instance.
type DB struct {
	Pool *pgxpool.Pool
	GORM *gorm.DB
}

// New creates a new database connection pool and GORM instance.
func New(ctx context.Context, databaseURL string) (*DB, error) {
	// 1. Initialize pgxpool
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	// 2. Initialize GORM (reusing the same URL for now)
	gormDB, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open gorm: %w", err)
	}

	return &DB{
		Pool: pool,
		GORM: gormDB,
	}, nil
}

// Close closes the database connection pool.
func (db *DB) Close() {
	db.Pool.Close()
	// GORM doesn't explicitly need closing if it's using the pooled driver,
	// but sql.DB should be closed if we had one.
}

// Ping checks if the database is reachable.
func (db *DB) Ping(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}
