package api

import (
	"time"

	"github.com/blockedby/positions-os/internal/models"
	"github.com/blockedby/positions-os/internal/repository"
	"github.com/google/uuid"
)

// ============================================================================
// Common Types
// ============================================================================

// ErrorResponse represents an API error response.
type ErrorResponse struct {
	Error   string `json:"error" description:"Error message"`
	Details string `json:"details,omitempty" description:"Additional error details"`
}

// HealthResponse represents the health check response.
type HealthResponse struct {
	Status  string `json:"status" example:"ok" description:"Health status"`
	Version string `json:"version" example:"dev" description:"Application version"`
}

// ============================================================================
// Jobs Types
// ============================================================================

// JobResponse represents a job in API responses.
type JobResponse struct {
	ID             uuid.UUID              `json:"id" description:"Job unique identifier"`
	TargetID       uuid.UUID              `json:"target_id" description:"Scraping target ID"`
	ExternalID     string                 `json:"external_id" description:"External source ID (e.g., Telegram message ID)"`
	StructuredData map[string]interface{} `json:"structured_data" description:"LLM-extracted structured job data"`
	SourceURL      *string                `json:"source_url,omitempty" description:"Original job posting URL"`
	SourceDate     *time.Time             `json:"source_date,omitempty" description:"Original posting date"`
	Status         string                 `json:"status" description:"Job status: RAW, ANALYZED, INTERESTED, REJECTED, TAILORED, SENT, RESPONDED"`
	CreatedAt      time.Time              `json:"created_at" description:"Record creation timestamp"`
	UpdatedAt      time.Time              `json:"updated_at" description:"Last update timestamp"`
	AnalyzedAt     *time.Time             `json:"analyzed_at,omitempty" description:"When the job was analyzed by LLM"`
}

// JobsListRequest contains query parameters for listing jobs.
type JobsListRequest struct {
	Status    string `query:"status" description:"Filter by job status" example:"ANALYZED"`
	Tech      string `query:"tech" description:"Filter by technology" example:"go"`
	Query     string `query:"q" description:"Full-text search query"`
	SalaryMin int    `query:"salary_min" description:"Minimum salary filter"`
	SalaryMax int    `query:"salary_max" description:"Maximum salary filter"`
	Page      int    `query:"page" default:"1" description:"Page number (1-indexed)"`
	Limit     int    `query:"limit" default:"50" description:"Items per page (max 100)"`
}

// JobsListResponse contains paginated list of jobs.
type JobsListResponse struct {
	Jobs  []JobResponse `json:"jobs" description:"List of jobs"`
	Total int           `json:"total" description:"Total number of matching jobs"`
	Page  int           `json:"page" description:"Current page number"`
	Limit int           `json:"limit" description:"Items per page"`
	Pages int           `json:"pages" description:"Total number of pages"`
}

// JobGetRequest contains path parameters for getting a single job.
type JobGetRequest struct {
	ID uuid.UUID `path:"id" description:"Job ID"`
}

// JobUpdateStatusRequest contains the request body for updating job status.
type JobUpdateStatusRequest struct {
	ID     uuid.UUID `path:"id" description:"Job ID"`
	Status string    `json:"status" validate:"required,oneof=RAW ANALYZED INTERESTED REJECTED TAILORED SENT RESPONDED" description:"New job status"`
}

// JobsBulkDeleteRequest contains the request body for bulk deleting jobs.
type JobsBulkDeleteRequest struct {
	IDs []uuid.UUID `json:"ids" validate:"required,min=1,max=100" description:"Job IDs to delete (max 100)"`
}

// JobsBulkDeleteResponse contains the response after bulk deleting jobs.
type JobsBulkDeleteResponse struct {
	Deleted int `json:"deleted" description:"Number of jobs deleted"`
}

// ============================================================================
// Targets Types
// ============================================================================

// TargetResponse represents a scraping target in API responses.
type TargetResponse struct {
	ID        uuid.UUID              `json:"id" description:"Target unique identifier"`
	Name      string                 `json:"name" description:"Human-readable target name"`
	Type      string                 `json:"type" description:"Target type: TG_CHANNEL, TG_FORUM, HH_SEARCH"`
	URL       string                 `json:"url" description:"Target URL or channel username"`
	IsActive  bool                   `json:"is_active" description:"Whether target is actively scraped"`
	Metadata  map[string]interface{} `json:"metadata,omitempty" description:"Additional target configuration"`
	CreatedAt time.Time              `json:"created_at" description:"Record creation timestamp"`
	UpdatedAt time.Time              `json:"updated_at" description:"Last update timestamp"`
}

// TargetsListResponse contains list of scraping targets.
type TargetsListResponse struct {
	Targets []TargetResponse `json:"targets" description:"List of scraping targets"`
	Total   int              `json:"total" description:"Total number of targets"`
}

// TargetCreateRequest contains the request body for creating a target.
type TargetCreateRequest struct {
	Name     string                 `json:"name" validate:"required" description:"Human-readable target name"`
	Type     string                 `json:"type" validate:"required,oneof=TG_CHANNEL TG_FORUM HH_SEARCH" description:"Target type"`
	URL      string                 `json:"url" validate:"required" description:"Target URL or channel username"`
	Metadata map[string]interface{} `json:"metadata,omitempty" description:"Additional target configuration (e.g., topic_ids for TG_FORUM)"`
}

// TargetGetRequest contains path parameters for getting a single target.
type TargetGetRequest struct {
	ID uuid.UUID `path:"id" description:"Target ID"`
}

// TargetUpdateRequest contains the request body for updating a target.
type TargetUpdateRequest struct {
	ID       uuid.UUID              `path:"id" description:"Target ID"`
	Name     string                 `json:"name,omitempty" description:"Human-readable target name"`
	Type     string                 `json:"type,omitempty" validate:"omitempty,oneof=TG_CHANNEL TG_FORUM HH_SEARCH" description:"Target type"`
	URL      string                 `json:"url,omitempty" description:"Target URL or channel username"`
	IsActive *bool                  `json:"is_active,omitempty" description:"Whether target is actively scraped"`
	Metadata map[string]interface{} `json:"metadata,omitempty" description:"Additional target configuration"`
}

// TargetDeleteRequest contains path parameters for deleting a target.
type TargetDeleteRequest struct {
	ID uuid.UUID `path:"id" description:"Target ID"`
}

// ============================================================================
// Stats Types
// ============================================================================

// StatsResponse contains job statistics (matches DashboardStats).
type StatsResponse struct {
	TotalJobs      int `json:"total_jobs" description:"Total number of jobs"`
	AnalyzedJobs   int `json:"analyzed_jobs" description:"Jobs that have been analyzed"`
	InterestedJobs int `json:"interested_jobs" description:"Jobs marked as INTERESTED"`
	RejectedJobs   int `json:"rejected_jobs" description:"Jobs marked as REJECTED"`
	TodayJobs      int `json:"today_jobs" description:"Jobs added today"`
	ActiveTargets  int `json:"active_targets" description:"Number of active scraping targets"`
}

// ============================================================================
// Scraping Types
// ============================================================================

// ScrapeStartRequest contains the request body for starting a scrape.
type ScrapeStartRequest struct {
	Channel  string  `json:"channel" validate:"required" description:"Telegram channel username (e.g., @golang_jobs)"`
	Limit    int     `json:"limit" default:"100" description:"Maximum messages to scrape (max 10000)"`
	Until    *string `json:"until,omitempty" description:"Scrape until this date (ISO 8601)"`
	TopicIDs []int64 `json:"topic_ids,omitempty" description:"Forum topic IDs to scrape (for TG_FORUM type)"`
}

// ScrapeStartResponse contains the response after starting a scrape.
type ScrapeStartResponse struct {
	Status  string `json:"status" example:"started" description:"Scrape status"`
	Message string `json:"message" description:"Status message"`
}

// ScrapeStatusResponse contains the current scraping status.
type ScrapeStatusResponse struct {
	IsRunning bool    `json:"is_running" description:"Whether a scrape is currently running"`
	Target    *string `json:"target,omitempty" description:"Current scraping target"`
	Progress  int     `json:"progress" description:"Number of messages processed"`
	NewJobs   int     `json:"new_jobs" description:"Number of new jobs found"`
}

// ============================================================================
// Auth Types
// ============================================================================

// AuthStatusResponse contains Telegram authentication status.
type AuthStatusResponse struct {
	Status       string `json:"status" description:"Auth status: INITIALIZING, AWAITING_QR, READY, DISCONNECTED"`
	IsReady      bool   `json:"is_ready" description:"Whether Telegram client is ready"`
	QRInProgress bool   `json:"qr_in_progress" description:"Whether QR login flow is active"`
}

// AuthQRStartResponse contains the response after starting QR login.
type AuthQRStartResponse struct {
	Status string `json:"status" example:"started" description:"QR flow status"`
}

// ============================================================================
// Applications Types
// ============================================================================

// ApplicationResponse represents a job application in API responses.
type ApplicationResponse struct {
	ID                 uuid.UUID  `json:"id" description:"Application unique identifier"`
	JobID              uuid.UUID  `json:"job_id" description:"Associated job ID"`
	TailoredResumeMD   *string    `json:"tailored_resume_md,omitempty" description:"Tailored resume in Markdown"`
	CoverLetterMD      *string    `json:"cover_letter_md,omitempty" description:"Cover letter in Markdown"`
	ResumePDFPath      *string    `json:"resume_pdf_path,omitempty" description:"Path to generated resume PDF"`
	CoverLetterPDFPath *string    `json:"cover_letter_pdf_path,omitempty" description:"Path to generated cover letter PDF"`
	DeliveryChannel    *string    `json:"delivery_channel,omitempty" description:"Delivery channel: TG_DM, EMAIL, HH_RESPONSE"`
	DeliveryStatus     string     `json:"delivery_status" description:"Delivery status: PENDING, SENT, DELIVERED, READ, FAILED"`
	Recipient          *string    `json:"recipient,omitempty" description:"Delivery recipient"`
	SentAt             *time.Time `json:"sent_at,omitempty" description:"When the application was sent"`
	CreatedAt          time.Time  `json:"created_at" description:"Record creation timestamp"`
	UpdatedAt          time.Time  `json:"updated_at" description:"Last update timestamp"`
}

// ApplicationsListRequest contains query parameters for listing applications.
type ApplicationsListRequest struct {
	JobID uuid.UUID `query:"job_id" validate:"required" description:"Filter by job ID"`
}

// ApplicationsListResponse contains list of applications.
type ApplicationsListResponse struct {
	Applications []ApplicationResponse `json:"applications" description:"List of applications"`
	Total        int                   `json:"total" description:"Total number of applications"`
}

// ApplicationGetRequest contains path parameters for getting a single application.
type ApplicationGetRequest struct {
	ID uuid.UUID `path:"id" description:"Application ID"`
}

// ApplicationCreateRequest contains the request body for creating an application.
type ApplicationCreateRequest struct {
	JobID           uuid.UUID `json:"job_id" validate:"required" description:"Associated job ID"`
	TailoredResume  *string   `json:"tailored_resume,omitempty" description:"Tailored resume in Markdown"`
	CoverLetter     *string   `json:"cover_letter,omitempty" description:"Cover letter in Markdown"`
	ResumePDFPath   *string   `json:"resume_pdf_path,omitempty" description:"Path to resume PDF"`
	CoverPDFPath    *string   `json:"cover_pdf_path,omitempty" description:"Path to cover letter PDF"`
	DeliveryChannel string    `json:"delivery_channel" validate:"required,oneof=TG_DM EMAIL HH_RESPONSE" description:"Delivery channel"`
}

// ApplicationSendRequest contains the request body for sending an application.
type ApplicationSendRequest struct {
	ID        uuid.UUID `path:"id" description:"Application ID"`
	Recipient string    `json:"recipient" validate:"required" description:"Delivery recipient (username or email)"`
}

// ApplicationSendResponse contains the response after sending an application.
type ApplicationSendResponse struct {
	Status  string `json:"status" example:"sent" description:"Send status"`
	Message string `json:"message" description:"Status message"`
}

// ApplicationUpdateDeliveryRequest contains the request body for updating delivery status.
type ApplicationUpdateDeliveryRequest struct {
	ID     uuid.UUID `path:"id" description:"Application ID"`
	Status string    `json:"status" validate:"required,oneof=PENDING SENT DELIVERED READ FAILED" description:"New delivery status"`
}

// ============================================================================
// Conversion Helpers
// ============================================================================

// JobFromRepo converts repository.Job to JobResponse.
func JobFromRepo(j *repository.Job) JobResponse {
	return JobResponse{
		ID:             j.ID,
		TargetID:       j.TargetID,
		ExternalID:     j.ExternalID,
		StructuredData: j.StructuredData,
		SourceURL:      j.SourceURL,
		SourceDate:     j.SourceDate,
		Status:         j.Status,
		CreatedAt:      j.CreatedAt,
		UpdatedAt:      j.UpdatedAt,
		AnalyzedAt:     j.AnalyzedAt,
	}
}

// JobsFromRepo converts slice of repository.Job to slice of JobResponse.
func JobsFromRepo(jobs []*repository.Job) []JobResponse {
	result := make([]JobResponse, len(jobs))
	for i, j := range jobs {
		result[i] = JobFromRepo(j)
	}
	return result
}

// TargetFromRepo converts repository.ScrapingTarget to TargetResponse.
func TargetFromRepo(t *repository.ScrapingTarget) TargetResponse {
	return TargetResponse{
		ID:        t.ID,
		Name:      t.Name,
		Type:      t.Type,
		URL:       t.URL,
		IsActive:  t.IsActive,
		Metadata:  t.Metadata,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}

// TargetsFromRepo converts slice of repository.ScrapingTarget to slice of TargetResponse.
func TargetsFromRepo(targets []repository.ScrapingTarget) []TargetResponse {
	result := make([]TargetResponse, len(targets))
	for i, t := range targets {
		result[i] = TargetFromRepo(&t)
	}
	return result
}

// ApplicationFromModel converts models.JobApplication to ApplicationResponse.
func ApplicationFromModel(a *models.JobApplication) ApplicationResponse {
	resp := ApplicationResponse{
		ID:                 a.ID,
		JobID:              a.JobID,
		TailoredResumeMD:   a.TailoredResumeMD,
		CoverLetterMD:      a.CoverLetterMD,
		ResumePDFPath:      a.ResumePDFPath,
		CoverLetterPDFPath: a.CoverLetterPDFPath,
		DeliveryStatus:     string(a.DeliveryStatus),
		Recipient:          a.Recipient,
		SentAt:             a.SentAt,
		CreatedAt:          a.CreatedAt,
		UpdatedAt:          a.UpdatedAt,
	}
	if a.DeliveryChannel != nil {
		ch := string(*a.DeliveryChannel)
		resp.DeliveryChannel = &ch
	}
	return resp
}

// ApplicationsFromModel converts slice of models.JobApplication to slice of ApplicationResponse.
func ApplicationsFromModel(apps []*models.JobApplication) []ApplicationResponse {
	result := make([]ApplicationResponse, len(apps))
	for i, a := range apps {
		result[i] = ApplicationFromModel(a)
	}
	return result
}
