package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Job represents a job posting
type Job struct {
	ID             uuid.UUID
	TargetID       uuid.UUID
	ExternalID     string
	ContentHash    *string
	RawContent     string
	StructuredData map[string]interface{}
	SourceURL      *string
	SourceDate     *time.Time
	TgMessageID    *int64
	TgTopicID      *int64
	Status         string // RAW, ANALYZED, REJECTED, INTERESTED, TAILORED, SENT, RESPONDED
	CreatedAt      time.Time
	UpdatedAt      time.Time
	AnalyzedAt     *time.Time
}

// IsValidStatus checks if job status is valid
func (j *Job) IsValidStatus() bool {
	valid := map[string]bool{
		"RAW": true, "ANALYZED": true, "REJECTED": true,
		"INTERESTED": true, "TAILORED": true, "SENT": true, "RESPONDED": true,
	}
	return valid[j.Status]
}

// IsNew checks if job is in RAW state
func (j *Job) IsNew() bool {
	return j.Status == "RAW"
}

// ComputeHash computes sha256 hash of raw content
func (j *Job) ComputeHash() string {
	h := sha256.Sum256([]byte(j.RawContent))
	return hex.EncodeToString(h[:])
}

// JobsRepository handles jobs table operations
type JobsRepository struct {
	pool *pgxpool.Pool
}

// NewJobsRepository creates a new jobs repository
func NewJobsRepository(pool *pgxpool.Pool) *JobsRepository {
	return &JobsRepository{pool: pool}
}

// Create creates a new job
func (r *JobsRepository) Create(ctx context.Context, j *Job) error {
	// compute hash if not set
	if j.ContentHash == nil || *j.ContentHash == "" {
		hash := j.ComputeHash()
		j.ContentHash = &hash
	}

	err := r.pool.QueryRow(ctx, `
		INSERT INTO jobs (target_id, external_id, content_hash, raw_content, 
		                  source_url, source_date, tg_message_id, tg_topic_id, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`, j.TargetID, j.ExternalID, j.ContentHash, j.RawContent,
		j.SourceURL, j.SourceDate, j.TgMessageID, j.TgTopicID, j.Status,
	).Scan(&j.ID, &j.CreatedAt, &j.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create job: %w", err)
	}
	return nil
}

// Exists checks if a job with given target_id and external_id exists
func (r *JobsRepository) Exists(ctx context.Context, targetID uuid.UUID, externalID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM jobs WHERE target_id = $1 AND external_id = $2)
	`, targetID, externalID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check job exists: %w", err)
	}
	return exists, nil
}

// GetByExternalID returns a job by target and external ID
func (r *JobsRepository) GetByExternalID(ctx context.Context, targetID uuid.UUID, externalID string) (*Job, error) {
	var j Job
	err := r.pool.QueryRow(ctx, `
		SELECT id, target_id, external_id, content_hash, raw_content,
		       structured_data, source_url, source_date, tg_message_id, tg_topic_id,
		       status, created_at, updated_at, analyzed_at
		FROM jobs
		WHERE target_id = $1 AND external_id = $2
	`, targetID, externalID).Scan(
		&j.ID, &j.TargetID, &j.ExternalID, &j.ContentHash, &j.RawContent,
		&j.StructuredData, &j.SourceURL, &j.SourceDate, &j.TgMessageID, &j.TgTopicID,
		&j.Status, &j.CreatedAt, &j.UpdatedAt, &j.AnalyzedAt,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("get job by external id: %w", err)
	}
	return &j, nil
}

// GetByStatus returns jobs with given status
func (r *JobsRepository) GetByStatus(ctx context.Context, status string, limit int) ([]Job, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, target_id, external_id, content_hash, raw_content,
		       structured_data, source_url, source_date, tg_message_id, tg_topic_id,
		       status, created_at, updated_at, analyzed_at
		FROM jobs
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, status, limit)
	if err != nil {
		return nil, fmt.Errorf("get jobs by status: %w", err)
	}
	defer rows.Close()

	var jobs []Job
	for rows.Next() {
		var j Job
		if err := rows.Scan(
			&j.ID, &j.TargetID, &j.ExternalID, &j.ContentHash, &j.RawContent,
			&j.StructuredData, &j.SourceURL, &j.SourceDate, &j.TgMessageID, &j.TgTopicID,
			&j.Status, &j.CreatedAt, &j.UpdatedAt, &j.AnalyzedAt,
		); err != nil {
			return nil, fmt.Errorf("scan job: %w", err)
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}

// CountByStatus returns count of jobs with given status
func (r *JobsRepository) CountByStatus(ctx context.Context, status string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM jobs WHERE status = $1
	`, status).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count jobs by status: %w", err)
	}
	return count, nil
}

// UpdateStatus updates job status
func (r *JobsRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE jobs SET status = $2, updated_at = NOW() WHERE id = $1
	`, id, status)
	if err != nil {
		return fmt.Errorf("update job status: %w", err)
	}
	return nil
}

// GetByID returns a job by ID
func (r *JobsRepository) GetByID(ctx context.Context, id uuid.UUID) (*Job, error) {
	var j Job
	err := r.pool.QueryRow(ctx, `
		SELECT id, target_id, external_id, content_hash, raw_content,
		       structured_data, source_url, source_date, tg_message_id, tg_topic_id,
		       status, created_at, updated_at, analyzed_at
		FROM jobs
		WHERE id = $1
	`, id).Scan(
		&j.ID, &j.TargetID, &j.ExternalID, &j.ContentHash, &j.RawContent,
		&j.StructuredData, &j.SourceURL, &j.SourceDate, &j.TgMessageID, &j.TgTopicID,
		&j.Status, &j.CreatedAt, &j.UpdatedAt, &j.AnalyzedAt,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("get job by id: %w", err)
	}
	return &j, nil
}

// UpdateStructuredData updates job structured data and sets status to ANALYZED
func (r *JobsRepository) UpdateStructuredData(ctx context.Context, id uuid.UUID, data map[string]interface{}) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE jobs
		SET structured_data = $2,
		    status = 'ANALYZED',
		    updated_at = NOW(),
		    analyzed_at = NOW()
		WHERE id = $1
	`, id, data)
	if err != nil {
		return fmt.Errorf("update structured data: %w", err)
	}
	return nil
}
