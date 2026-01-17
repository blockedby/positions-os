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
	ID            uuid.UUID              `json:"id"`
	Name          string                 `json:"name"`
	Type          string                 `json:"type"` // TG_CHANNEL, TG_GROUP, TG_FORUM, HH_SEARCH, LINKEDIN_SEARCH
	URL           string                 `json:"url"`
	TgAccessHash  *int64                 `json:"tg_access_hash,omitempty"`
	TgChannelID   *int64                 `json:"tg_channel_id,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	LastScrapedAt *time.Time             `json:"last_scraped_at,omitempty"`
	LastMessageID *int64                 `json:"last_message_id,omitempty"`
	IsActive      bool                   `json:"is_active"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
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
			return nil, ErrNotFound
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
			return nil, ErrNotFound
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

// List returns all targets (active and inactive)
func (r *TargetsRepository) List(ctx context.Context) ([]ScrapingTarget, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, type, url, tg_access_hash, tg_channel_id, 
		       metadata, last_scraped_at, last_message_id, is_active, 
		       created_at, updated_at
		FROM scraping_targets
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("list targets: %w", err)
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

// Update updates a target
func (r *TargetsRepository) Update(ctx context.Context, t *ScrapingTarget) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE scraping_targets
		SET name = $2, type = $3, url = $4, metadata = $5, is_active = $6, updated_at = NOW()
		WHERE id = $1
	`, t.ID, t.Name, t.Type, t.URL, t.Metadata, t.IsActive)
	if err != nil {
		return fmt.Errorf("update target: %w", err)
	}
	return nil
}

// Delete deletes a target
func (r *TargetsRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM scraping_targets WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete target: %w", err)
	}
	return nil
}
