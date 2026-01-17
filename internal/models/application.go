package models

import (
	"time"

	"github.com/google/uuid"
)

// DeliveryChannel represents how the application was sent.
type DeliveryChannel string

// DeliveryChannel constants define the supported channels for sending applications.
const (
	DeliveryChannelTGDM  DeliveryChannel = "TG_DM"
	DeliveryChannelEmail DeliveryChannel = "EMAIL"
	DeliveryChannelHH    DeliveryChannel = "HH_RESPONSE"
)

// DeliveryStatus represents the delivery state of an application.
type DeliveryStatus string

// DeliveryStatus constants define the possible states of application delivery.
const (
	DeliveryStatusPending   DeliveryStatus = "PENDING"
	DeliveryStatusSent      DeliveryStatus = "SENT"
	DeliveryStatusDelivered DeliveryStatus = "DELIVERED"
	DeliveryStatusRead      DeliveryStatus = "READ"
	DeliveryStatusFailed    DeliveryStatus = "FAILED"
)

// JobApplication represents a tailored resume and its delivery status.
type JobApplication struct {
	ID    uuid.UUID `json:"id" db:"id"`
	JobID uuid.UUID `json:"job_id" db:"job_id"`

	// generated content
	TailoredResumeMD *string `json:"tailored_resume_md,omitempty" db:"tailored_resume_md"`
	CoverLetterMD    *string `json:"cover_letter_md,omitempty" db:"cover_letter_md"`

	// generated files
	ResumePDFPath      *string `json:"resume_pdf_path,omitempty" db:"resume_pdf_path"`
	CoverLetterPDFPath *string `json:"cover_letter_pdf_path,omitempty" db:"cover_letter_pdf_path"`

	// delivery
	DeliveryChannel *DeliveryChannel `json:"delivery_channel,omitempty" db:"delivery_channel"`
	DeliveryStatus  DeliveryStatus   `json:"delivery_status" db:"delivery_status"`
	Recipient       *string          `json:"recipient,omitempty" db:"recipient"`

	// tracking
	SentAt             *time.Time `json:"sent_at,omitempty" db:"sent_at"`
	DeliveredAt        *time.Time `json:"delivered_at,omitempty" db:"delivered_at"`
	ReadAt             *time.Time `json:"read_at,omitempty" db:"read_at"`
	ResponseReceivedAt *time.Time `json:"response_received_at,omitempty" db:"response_received_at"`

	// recruiter response
	RecruiterResponse *string `json:"recruiter_response,omitempty" db:"recruiter_response"`

	// timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	Version   int       `json:"version" db:"version"`
}
