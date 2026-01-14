package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/blockedby/positions-os/internal/logger"
	"github.com/blockedby/positions-os/internal/models"
)

// ApplicationsRepository handles job applications CRUD operations
type ApplicationsRepository struct {
	pool *pgxpool.Pool
	log  *logger.Logger
}

// NewApplicationsRepository creates a new applications repository
func NewApplicationsRepository(pool *pgxpool.Pool, log *logger.Logger) *ApplicationsRepository {
	return &ApplicationsRepository{
		pool: pool,
		log:  log,
	}
}

// Create creates a new job application record
func (r *ApplicationsRepository) Create(ctx context.Context, app *models.JobApplication) error {
	// Generate ID if not set
	if app.ID == uuid.Nil {
		app.ID = uuid.New()
	}

	// Set default status if not set
	if app.DeliveryStatus == "" {
		app.DeliveryStatus = models.DeliveryStatusPending
	}

	err := r.pool.QueryRow(ctx, `
		INSERT INTO job_applications (
			id, job_id, delivery_channel, delivery_status, recipient,
			tailored_resume_md, cover_letter_md, resume_pdf_path, cover_letter_pdf_path
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING created_at, updated_at
	`, app.ID, app.JobID, app.DeliveryChannel, app.DeliveryStatus, app.Recipient,
		app.TailoredResumeMD, app.CoverLetterMD, app.ResumePDFPath, app.CoverLetterPDFPath,
	).Scan(&app.CreatedAt, &app.UpdatedAt)

	if err != nil {
		return fmt.Errorf("create application: %w", err)
	}

	r.log.Info().
		Str("id", app.ID.String()).
		Str("job_id", app.JobID.String()).
		Str("status", string(app.DeliveryStatus)).
		Msg("created application")

	return nil
}

// GetByID returns a single application by ID
func (r *ApplicationsRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.JobApplication, error) {
	var app models.JobApplication
	var channel, recipient *string
	var resumeMD, coverMD, resumePath, coverPath *string
	var response *string
	var sentAt, deliveredAt, readAt, responseAt *time.Time

	err := r.pool.QueryRow(ctx, `
		SELECT id, job_id, delivery_channel, delivery_status, recipient,
		       tailored_resume_md, cover_letter_md, resume_pdf_path, cover_letter_pdf_path,
		       sent_at, delivered_at, read_at, response_received_at, recruiter_response,
		       created_at, updated_at, version
		FROM job_applications
		WHERE id = $1
	`, id).Scan(
		&app.ID, &app.JobID, &channel, &app.DeliveryStatus, &recipient,
		&resumeMD, &coverMD, &resumePath, &coverPath,
		&sentAt, &deliveredAt, &readAt, &responseAt, &response,
		&app.CreatedAt, &app.UpdatedAt, &app.Version,
	)

	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("get application by id: %w", err)
	}

	// Convert string pointers to appropriate types
	app.Recipient = recipient
	app.TailoredResumeMD = resumeMD
	app.CoverLetterMD = coverMD
	app.ResumePDFPath = resumePath
	app.CoverLetterPDFPath = coverPath
	app.RecruiterResponse = response
	app.SentAt = sentAt
	app.DeliveredAt = deliveredAt
	app.ReadAt = readAt
	app.ResponseReceivedAt = responseAt

	// Convert channel string to DeliveryChannel
	if channel != nil {
		ch := models.DeliveryChannel(*channel)
		app.DeliveryChannel = &ch
	}

	return &app, nil
}

// GetByJobID returns all applications for a job
func (r *ApplicationsRepository) GetByJobID(ctx context.Context, jobID uuid.UUID) ([]*models.JobApplication, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, job_id, delivery_channel, delivery_status, recipient,
		       tailored_resume_md, cover_letter_md, resume_pdf_path, cover_letter_pdf_path,
		       sent_at, delivered_at, read_at, response_received_at, recruiter_response,
		       created_at, updated_at, version
		FROM job_applications
		WHERE job_id = $1
		ORDER BY created_at DESC
	`, jobID)
	if err != nil {
		return nil, fmt.Errorf("get applications by job id: %w", err)
	}
	defer rows.Close()

	var apps []*models.JobApplication
	for rows.Next() {
		var app models.JobApplication
		var channel, recipient *string
		var resumeMD, coverMD, resumePath, coverPath *string
		var response *string
		var sentAt, deliveredAt, readAt, responseAt *time.Time

		err := rows.Scan(
			&app.ID, &app.JobID, &channel, &app.DeliveryStatus, &recipient,
			&resumeMD, &coverMD, &resumePath, &coverPath,
			&sentAt, &deliveredAt, &readAt, &responseAt, &response,
			&app.CreatedAt, &app.UpdatedAt, &app.Version,
		)
		if err != nil {
			return nil, fmt.Errorf("scan application: %w", err)
		}

		app.Recipient = recipient
		app.TailoredResumeMD = resumeMD
		app.CoverLetterMD = coverMD
		app.ResumePDFPath = resumePath
		app.CoverLetterPDFPath = coverPath
		app.RecruiterResponse = response
		app.SentAt = sentAt
		app.DeliveredAt = deliveredAt
		app.ReadAt = readAt
		app.ResponseReceivedAt = responseAt

		if channel != nil {
			ch := models.DeliveryChannel(*channel)
			app.DeliveryChannel = &ch
		}

		apps = append(apps, &app)
	}

	return apps, nil
}

// UpdateDeliveryStatus updates the delivery status of an application
func (r *ApplicationsRepository) UpdateDeliveryStatus(ctx context.Context, id uuid.UUID, status models.DeliveryStatus) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE job_applications
		SET delivery_status = $2, updated_at = NOW()
		WHERE id = $1
	`, id, status)
	if err != nil {
		return fmt.Errorf("update delivery status: %w", err)
	}

	r.log.Info().
		Str("id", id.String()).
		Str("status", string(status)).
		Msg("updated delivery status")

	return nil
}

// UpdateRecipient updates the recipient of an application
func (r *ApplicationsRepository) UpdateRecipient(ctx context.Context, id uuid.UUID, recipient string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE job_applications
		SET recipient = $2, updated_at = NOW()
		WHERE id = $1
	`, id, recipient)
	if err != nil {
		return fmt.Errorf("update recipient: %w", err)
	}

	r.log.Info().
		Str("id", id.String()).
		Str("recipient", recipient).
		Msg("updated recipient")

	return nil
}

// MarkSent marks an application as sent
func (r *ApplicationsRepository) MarkSent(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	_, err := r.pool.Exec(ctx, `
		UPDATE job_applications
		SET delivery_status = 'SENT',
		    sent_at = $2,
		    updated_at = NOW()
		WHERE id = $1
	`, id, now)
	if err != nil {
		return fmt.Errorf("mark sent: %w", err)
	}

	r.log.Info().
		Str("id", id.String()).
		Time("sent_at", now).
		Msg("marked application as sent")

	return nil
}

// ListPending returns applications pending delivery
func (r *ApplicationsRepository) ListPending(ctx context.Context, limit int) ([]*models.JobApplication, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, job_id, delivery_channel, delivery_status, recipient,
		       tailored_resume_md, cover_letter_md, resume_pdf_path, cover_letter_pdf_path,
		       sent_at, delivered_at, read_at, response_received_at, recruiter_response,
		       created_at, updated_at, version
		FROM job_applications
		WHERE delivery_status = 'PENDING'
		ORDER BY created_at ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list pending: %w", err)
	}
	defer rows.Close()

	var apps []*models.JobApplication
	for rows.Next() {
		var app models.JobApplication
		var channel, recipient *string
		var resumeMD, coverMD, resumePath, coverPath *string
		var response *string
		var sentAt, deliveredAt, readAt, responseAt *time.Time

		err := rows.Scan(
			&app.ID, &app.JobID, &channel, &app.DeliveryStatus, &recipient,
			&resumeMD, &coverMD, &resumePath, &coverPath,
			&sentAt, &deliveredAt, &readAt, &responseAt, &response,
			&app.CreatedAt, &app.UpdatedAt, &app.Version,
		)
		if err != nil {
			return nil, fmt.Errorf("scan application: %w", err)
		}

		app.Recipient = recipient
		app.TailoredResumeMD = resumeMD
		app.CoverLetterMD = coverMD
		app.ResumePDFPath = resumePath
		app.CoverLetterPDFPath = coverPath
		app.RecruiterResponse = response
		app.SentAt = sentAt
		app.DeliveredAt = deliveredAt
		app.ReadAt = readAt
		app.ResponseReceivedAt = responseAt

		if channel != nil {
			ch := models.DeliveryChannel(*channel)
			app.DeliveryChannel = &ch
		}

		apps = append(apps, &app)
	}

	return apps, nil
}

// UpdateTimestamps updates the tracking timestamps (delivered_at, read_at, etc.)
func (r *ApplicationsRepository) UpdateTimestamps(ctx context.Context, id uuid.UUID, timestampType string, value time.Time) error {
	var column string
	switch timestampType {
	case "delivered_at":
		column = "delivered_at"
	case "read_at":
		column = "read_at"
	case "response_received_at":
		column = "response_received_at"
	default:
		return fmt.Errorf("invalid timestamp type: %s", timestampType)
	}

	_, err := r.pool.Exec(ctx, `
		UPDATE job_applications
		SET `+column+` = $2, updated_at = NOW()
		WHERE id = $1
	`, id, value)
	if err != nil {
		return fmt.Errorf("update timestamp %s: %w", timestampType, err)
	}

	return nil
}
