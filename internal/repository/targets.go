package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ScrapingTarget represents a scraping source
type ScrapingTarget struct {
	ID            uuid.UUID
	Name          string
	Type          string // TG_CHANNEL, TG_GROUP, TG_FORUM, HH_SEARCH, LINKEDIN_SEARCH
	URL           string
	TgAccessHash  *int64
	TgChannelID   *int64
	Metadata      map[string]interface{}
	LastScrapedAt *time.Time
	LastMessageID *int64
	IsActive      bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// valid target types
var validTargetTypes = map[string]bool{
	"TG_CHANNEL":      true,
	"TG_GROUP":        true,
	"TG_FORUM":        true,
	"HH_SEARCH":       true,
	"LINKEDIN_SEARCH": true,
}

// IsValid checks if target has valid type
func (t *ScrapingTarget) IsValid() bool {
	return validTargetTypes[t.Type]
}

// IsTelegram checks if target is a telegram source
func (t *ScrapingTarget) IsTelegram() bool {
	return strings.HasPrefix(t.Type, "TG_")
}

// IsForum checks if target is a telegram forum
func (t *ScrapingTarget) IsForum() bool {
	return t.Type == "TG_FORUM"
}

// TargetsRepository handles scraping_targets table operations
type TargetsRepository struct {
	pool *pgxpool.Pool
}

// NewTargetsRepository creates a new targets repository
func NewTargetsRepository(pool *pgxpool.Pool) *TargetsRepository {
	return &TargetsRepository{pool: pool}
}

// GetByID returns a target by ID
func (r *TargetsRepository) GetByID(ctx context.Context, id uuid.UUID) (*ScrapingTarget, error) {
	var t ScrapingTarget
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, type, url, tg_access_hash, tg_channel_id, 
		       metadata, last_scraped_at, last_message_id, is_active, 
		       created_at, updated_at
		FROM scraping_targets
		WHERE id = $1
	`, id).Scan(
		&t.ID, &t.Name, &t.Type, &t.URL, &t.TgAccessHash, &t.TgChannelID,
		&t.Metadata, &t.LastScrapedAt, &t.LastMessageID, &t.IsActive,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("get target by id: %w", err)
	}
	return &t, nil
}

// GetByURL returns a target by URL (channel username)
func (r *TargetsRepository) GetByURL(ctx context.Context, url string) (*ScrapingTarget, error) {
	// normalize url - strip @ prefix
	url = strings.TrimPrefix(url, "@")

	var t ScrapingTarget
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, type, url, tg_access_hash, tg_channel_id, 
		       metadata, last_scraped_at, last_message_id, is_active, 
		       created_at, updated_at
		FROM scraping_targets
		WHERE url = $1 OR url = '@' || $1
	`, url).Scan(
		&t.ID, &t.Name, &t.Type, &t.URL, &t.TgAccessHash, &t.TgChannelID,
		&t.Metadata, &t.LastScrapedAt, &t.LastMessageID, &t.IsActive,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("get target by url: %w", err)
	}
	return &t, nil
}

// GetActive returns all active targets
func (r *TargetsRepository) GetActive(ctx context.Context) ([]ScrapingTarget, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, type, url, tg_access_hash, tg_channel_id, 
		       metadata, last_scraped_at, last_message_id, is_active, 
		       created_at, updated_at
		FROM scraping_targets
		WHERE is_active = true
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("get active targets: %w", err)
	}
	defer rows.Close()

	var targets []ScrapingTarget
	for rows.Next() {
		var t ScrapingTarget
		if err := rows.Scan(
			&t.ID, &t.Name, &t.Type, &t.URL, &t.TgAccessHash, &t.TgChannelID,
			&t.Metadata, &t.LastScrapedAt, &t.LastMessageID, &t.IsActive,
			&t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan target: %w", err)
		}
		targets = append(targets, t)
	}
	return targets, nil
}

// Create creates a new target
func (r *TargetsRepository) Create(ctx context.Context, t *ScrapingTarget) error {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO scraping_targets (name, type, url, tg_access_hash, tg_channel_id, metadata, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`, t.Name, t.Type, t.URL, t.TgAccessHash, t.TgChannelID, t.Metadata, t.IsActive).Scan(
		&t.ID, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create target: %w", err)
	}
	return nil
}

// UpdateLastScraped updates the last scraped timestamp and message ID
func (r *TargetsRepository) UpdateLastScraped(ctx context.Context, id uuid.UUID, messageID int64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE scraping_targets
		SET last_scraped_at = NOW(), last_message_id = $2, updated_at = NOW()
		WHERE id = $1
	`, id, messageID)
	if err != nil {
		return fmt.Errorf("update last scraped: %w", err)
	}
	return nil
}

// UpdateTelegramInfo updates telegram-specific fields
func (r *TargetsRepository) UpdateTelegramInfo(ctx context.Context, id uuid.UUID, channelID, accessHash int64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE scraping_targets
		SET tg_channel_id = $2, tg_access_hash = $3, updated_at = NOW()
		WHERE id = $1
	`, id, channelID, accessHash)
	if err != nil {
		return fmt.Errorf("update telegram info: %w", err)
	}
	return nil
}
