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

// NewJobUpdatedEvent creates a JSON message for job updates
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

// NewHTMXJobUpdateEvent returns an OOB swap trigger (if we were sending HTML directly over WS)
// Or better, sends a trigger for HTMX to refresh specific parts.
// When using hx-ws, we can send HTML for OOB swaps.
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
