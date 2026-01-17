// Package web provides WebSocket event types and formatting.
package web

import (
	"encoding/json"

	"github.com/google/uuid"
)

// WebSocket event types
const (
	EventJobNew      = "job.new"
	EventJobUpdated  = "job.updated"
	EventScrapeStart = "scrape.start"
	EventScrapeEnd   = "scrape.end"

	// Brain events
	EventBrainStarted   = "brain.started"
	EventBrainProgress  = "brain.progress"
	EventBrainCompleted = "brain.completed"
	EventBrainError     = "brain.error"
)

// WSEvent represents a structured WebSocket message
type WSEvent struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// JobUpdatedPayload is the payload for EventJobUpdated
type JobUpdatedPayload struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"`
}

// JobUpdatedEvent creates a JSON message for job updates.
func JobUpdatedEvent(jobID uuid.UUID, status string) []byte {
	evt := WSEvent{
		Type: EventJobUpdated,
		Payload: JobUpdatedPayload{
			JobID:  jobID.String(),
			Status: status,
		},
	}
	b, _ := json.Marshal(evt)
	return b
}

// JobRowUpdateHTML returns an OOB swap trigger for HTMX.
// When using hx-ws, this can send HTML for OOB swaps or JSON to trigger client-side events.
func JobRowUpdateHTML(jobID uuid.UUID, status string) []byte {
	// For now returning JSON as per previous plan, but HTMX WS extension usually expects
	// HTML for OOB swaps or JSON to trigger client-side events.
	// Let's assume we want to trigger a client-side event.
	// If we simply broadcast JSON, the `htmx-ws` extension might not process it as OOB.
	// We should probably send HTML if we want OOB.
	// BUT the plan says: "NotifyNewJob sends HTML snippet for OOB swap".

	// Let's implement helper for simple JSON event first, as implemented in UpdateStatus handler plan.
	return JobUpdatedEvent(jobID, status)
}

// Brain event payloads

// BrainStartedPayload is the payload for EventBrainStarted
type BrainStartedPayload struct {
	JobID string `json:"job_id"`
}

// BrainProgressPayload is the payload for EventBrainProgress
type BrainProgressPayload struct {
	JobID    string `json:"job_id"`
	Step     string `json:"step"`              // tailoring, cover_letter, pdf_rendering
	Progress int    `json:"progress"`          // 0-100
	Message  string `json:"message,omitempty"` // Human-readable message
}

// BrainCompletedPayload is the payload for EventBrainCompleted
type BrainCompletedPayload struct {
	JobID       string `json:"job_id"`
	ResumeURL   string `json:"resume_url"`
	CoverLetter string `json:"cover_letter,omitempty"`
}

// BrainErrorPayload is the payload for EventBrainError
type BrainErrorPayload struct {
	JobID string `json:"job_id"`
	Step  string `json:"step,omitempty"` // Where the error occurred
	Error string `json:"error"`
}

// BrainStartedEvent creates a JSON message for brain processing started
func BrainStartedEvent(jobID uuid.UUID) []byte {
	evt := WSEvent{
		Type: EventBrainStarted,
		Payload: BrainStartedPayload{
			JobID: jobID.String(),
		},
	}
	b, _ := json.Marshal(evt)
	return b
}

// BrainProgressEvent creates a JSON message for brain processing progress
func BrainProgressEvent(jobID uuid.UUID, step string, progress int, message string) []byte {
	evt := WSEvent{
		Type: EventBrainProgress,
		Payload: BrainProgressPayload{
			JobID:    jobID.String(),
			Step:     step,
			Progress: progress,
			Message:  message,
		},
	}
	b, _ := json.Marshal(evt)
	return b
}

// BrainCompletedEvent creates a JSON message for brain processing completed
func BrainCompletedEvent(jobID uuid.UUID, resumeURL, coverLetter string) []byte {
	evt := WSEvent{
		Type: EventBrainCompleted,
		Payload: BrainCompletedPayload{
			JobID:       jobID.String(),
			ResumeURL:   resumeURL,
			CoverLetter: coverLetter,
		},
	}
	b, _ := json.Marshal(evt)
	return b
}

// BrainErrorEvent creates a JSON message for brain processing error
func BrainErrorEvent(jobID uuid.UUID, step, errMsg string) []byte {
	evt := WSEvent{
		Type: EventBrainError,
		Payload: BrainErrorPayload{
			JobID: jobID.String(),
			Step:  step,
			Error: errMsg,
		},
	}
	b, _ := json.Marshal(evt)
	return b
}
