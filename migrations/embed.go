// Package migrations embeds database migration files for use by services.
package migrations

import "embed"

// FS contains all migration SQL files.
//
//go:embed *.sql
var FS embed.FS
