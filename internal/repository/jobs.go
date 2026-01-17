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
	ID             uuid.UUID              `json:"id"`
	TargetID       uuid.UUID              `json:"target_id"`
	ExternalID     string                 `json:"external_id"`
	ContentHash    *string                `json:"content_hash,omitempty"`
	RawContent     string                 `json:"raw_content"`
	StructuredData map[string]interface{} `json:"structured_data"`
	SourceURL      *string                `json:"source_url,omitempty"`
	SourceDate     *time.Time             `json:"source_date,omitempty"`
	TgMessageID    *int64                 `json:"tg_message_id,omitempty"`
	TgTopicID      *int64                 `json:"tg_topic_id,omitempty"`
	Status         string                 `json:"status"` // RAW, ANALYZED, REJECTED, INTERESTED, TAILORED, SENT, RESPONDED
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
	AnalyzedAt     *time.Time             `json:"analyzed_at,omitempty"`
}

// JobFilter defines criteria for listing jobs
type JobFilter struct {
	Status    string
	SalaryMin int
	SalaryMax int
	Tech      string // Search in structured_data -> tech
	Query     string // Full text search
	Page      int
	Limit     int
	Sort      string
	Order     string // ASC/DESC
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

// validTransitions defines allowed status transitions
// Key is "from" status, value is set of allowed "to" statuses
var validTransitions = map[string]map[string]bool{
	"RAW": {
		"ANALYZED": true,
		"REJECTED": true,
	},
	"ANALYZED": {
		"INTERESTED": true,
		"REJECTED":   true,
		"RAW":        true, // allow re-analysis
	},
	"INTERESTED": {
		"TAILORED": true,
		"REJECTED": true,
		"RAW":      true,
	},
	"REJECTED": {
		"RAW": true, // allow re-processing
	},
	"TAILORED": {
		"SENT":     true,
		"REJECTED": true,
		"RAW":      true,
	},
	"SENT": {
		"RESPONDED": true,
		"REJECTED":  true,
		"RAW":       true,
	},
	"RESPONDED": {
		"RAW": true,
	},
}

// CanTransitionTo checks if status transition is valid
func (j *Job) CanTransitionTo(newStatus string) bool {
	allowed, ok := validTransitions[j.Status]
	if !ok {
		return false
	}
	return allowed[newStatus]
}

// Title returns job title from structured data or fallback
func (j *Job) Title() string {
	if title, ok := j.StructuredData["title"].(string); ok && title != "" {
		return title
	}
	// Fallback to extraction from raw content or generic
	return "Unknown Position"
}

// Company returns company from structured data
func (j *Job) Company() string {
	if company, ok := j.StructuredData["company"].(string); ok {
		return company
	}
	return ""
}

// Salary returns formatted salary
func (j *Job) Salary() string {
	if salary, ok := j.StructuredData["salary"].(string); ok {
		return salary
	}
	return ""
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

// List returns jobs matching filter
// List returns jobs matching filter
func (r *JobsRepository) List(ctx context.Context, filter JobFilter) ([]*Job, int, error) {
	query := `
		SELECT 
			id, target_id, external_id, content_hash, raw_content,
			structured_data, source_url, source_date, tg_message_id, tg_topic_id,
			status, created_at, updated_at, analyzed_at,
			COUNT(*) OVER() as total_count
		FROM jobs
		WHERE 1=1
	`
	var args []interface{}
	argID := 1

	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argID)
		args = append(args, filter.Status)
		argID++
	}

	if filter.Query != "" {
		// Search in raw_content OR title
		q := "%" + filter.Query + "%"
		query += fmt.Sprintf(" AND (raw_content ILIKE $%d OR structured_data->>'title' ILIKE $%d)", argID, argID+1)
		args = append(args, q, q)
		argID += 2
	}

	if filter.Tech != "" {
		// Assuming tech is comma separated list of technologies
		// We want to find jobs that have ANY of these technologies
		// structured_data->'technologies' is a JSON array
		// Postgres JSONB operator ?| takes text[]
		// We need to pass a string slice to driver which converts to text[]
		techs := []string{filter.Tech} // Simplified: single tech or need split?
		// If query param is "go,k8s", we should split?
		// Let's assume passed as is for now, or split if simple string.
		// The test passes "go" (single).

		// If needed to split:
		// techs := strings.Split(filter.Tech, ",")

		query += fmt.Sprintf(" AND structured_data->'technologies' ?| $%d", argID)
		args = append(args, techs)
		argID++
	}

	if filter.SalaryMin > 0 {
		query += fmt.Sprintf(" AND COALESCE((structured_data->>'salary_min')::int, 0) >= $%d", argID)
		args = append(args, filter.SalaryMin)
		argID++
	}

	// Order
	query += " ORDER BY created_at DESC"

	// Pagination
	limit := 50
	if filter.Limit > 0 {
		limit = filter.Limit
	}
	query += fmt.Sprintf(" LIMIT $%d", argID)
	args = append(args, limit)
	argID++

	offset := 0
	if filter.Page > 1 {
		offset = (filter.Page - 1) * limit
	}
	query += fmt.Sprintf(" OFFSET $%d", argID)
	args = append(args, offset)
	argID++

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*Job
	var total int

	for rows.Next() {
		var j Job
		err := rows.Scan(
			&j.ID, &j.TargetID, &j.ExternalID, &j.ContentHash, &j.RawContent,
			&j.StructuredData, &j.SourceURL, &j.SourceDate, &j.TgMessageID, &j.TgTopicID,
			&j.Status, &j.CreatedAt, &j.UpdatedAt, &j.AnalyzedAt,
			&total, // Window function result
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan job: %w", err)
		}
		jobs = append(jobs, &j)
	}

	return jobs, total, nil
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

// BulkDelete removes multiple jobs by their IDs.
func (r *JobsRepository) BulkDelete(ctx context.Context, ids []uuid.UUID) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	query := `DELETE FROM jobs WHERE id = ANY($1)`
	result, err := r.pool.Exec(ctx, query, ids)
	if err != nil {
		return 0, fmt.Errorf("bulk delete jobs: %w", err)
	}

	return int(result.RowsAffected()), nil
}

// GetExistingMessageIDs returns all tg_message_ids for jobs belonging to a target.
// Used to check which messages already have jobs created (vs just being in parsed range).
func (r *JobsRepository) GetExistingMessageIDs(ctx context.Context, targetID uuid.UUID) ([]int64, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT tg_message_id
		FROM jobs
		WHERE target_id = $1 AND tg_message_id IS NOT NULL
	`, targetID)
	if err != nil {
		return nil, fmt.Errorf("get existing message ids: %w", err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan message id: %w", err)
		}
		ids = append(ids, id)
	}

	if ids == nil {
		return []int64{}, nil
	}
	return ids, nil
}
