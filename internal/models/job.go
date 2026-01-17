package models

import (
	"time"

	"github.com/google/uuid"
)

// JobStatus represents the processing status of a job posting.
type JobStatus string

// JobStatus constants define the possible states of a job posting.
const (
	JobStatusRaw             JobStatus = "RAW"
	JobStatusAnalyzed        JobStatus = "ANALYZED"
	JobStatusRejected        JobStatus = "REJECTED"
	JobStatusInterested      JobStatus = "INTERESTED"
	JobStatusTailored        JobStatus = "TAILORED"
	JobStatusTailoredApproved JobStatus = "TAILORED_APPROVED"
	JobStatusSent            JobStatus = "SENT"
	JobStatusResponded       JobStatus = "RESPONDED"
)

// Job represents a job posting from any source.
type Job struct {
	ID       uuid.UUID `json:"id" db:"id"`
	TargetID uuid.UUID `json:"target_id" db:"target_id"`

	// identification
	ExternalID  string `json:"external_id" db:"external_id"`
	ContentHash string `json:"content_hash" db:"content_hash"`

	// content
	RawContent     string   `json:"raw_content" db:"raw_content"`
	StructuredData *JobData `json:"structured_data" db:"structured_data"`

	// source metadata
	SourceURL  *string    `json:"source_url,omitempty" db:"source_url"`
	SourceDate *time.Time `json:"source_date,omitempty" db:"source_date"`

	// telegram specific
	TGMessageID *int64 `json:"tg_message_id,omitempty" db:"tg_message_id"`
	TGTopicID   *int64 `json:"tg_topic_id,omitempty" db:"tg_topic_id"`

	// status
	Status JobStatus `json:"status" db:"status"`

	// timestamps
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`
	AnalyzedAt *time.Time `json:"analyzed_at,omitempty" db:"analyzed_at"`
}

// JobData represents structured data extracted by llm.
type JobData struct {
	Title           *string  `json:"title,omitempty"`
	Description     *string  `json:"description,omitempty"`
	SalaryMin       *int     `json:"salary_min,omitempty"`
	SalaryMax       *int     `json:"salary_max,omitempty"`
	Currency        *string  `json:"currency,omitempty"`
	Location        *string  `json:"location,omitempty"`
	IsRemote        bool     `json:"is_remote"`
	Language        string   `json:"language"`
	Technologies    []string `json:"technologies"`
	ExperienceYears *int     `json:"experience_years,omitempty"`
	Company         *string  `json:"company,omitempty"`
	Contacts        []string `json:"contacts"`
}
