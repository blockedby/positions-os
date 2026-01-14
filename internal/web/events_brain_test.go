package web

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

// TestBrainStartedEvent_CreatesValidJSON
func TestBrainStartedEvent_CreatesValidJSON(t *testing.T) {
	jobID := uuid.New()
	event := BrainStartedEvent(jobID)

	var wsEvent WSEvent
	if err := json.Unmarshal(event, &wsEvent); err != nil {
		t.Fatal(err)
	}

	if wsEvent.Type != EventBrainStarted {
		t.Errorf("expected type %s, got %s", EventBrainStarted, wsEvent.Type)
	}

	payload, ok := wsEvent.Payload.(map[string]interface{})
	if !ok {
		t.Fatal("payload is not a map")
	}

	if payload["job_id"] != jobID.String() {
		t.Errorf("expected job_id %s, got %v", jobID.String(), payload["job_id"])
	}
}

// TestBrainProgressEvent_CreatesValidJSON
func TestBrainProgressEvent_CreatesValidJSON(t *testing.T) {
	jobID := uuid.New()
	event := BrainProgressEvent(jobID, "tailoring", 50, "Adapting resume...")

	var wsEvent WSEvent
	if err := json.Unmarshal(event, &wsEvent); err != nil {
		t.Fatal(err)
	}

	if wsEvent.Type != EventBrainProgress {
		t.Errorf("expected type %s, got %s", EventBrainProgress, wsEvent.Type)
	}

	payload, ok := wsEvent.Payload.(map[string]interface{})
	if !ok {
		t.Fatal("payload is not a map")
	}

	if payload["job_id"] != jobID.String() {
		t.Errorf("expected job_id %s, got %v", jobID.String(), payload["job_id"])
	}
	if payload["step"] != "tailoring" {
		t.Errorf("expected step tailoring, got %v", payload["step"])
	}
	if payload["progress"] != float64(50) {
		t.Errorf("expected progress 50, got %v", payload["progress"])
	}
	if payload["message"] != "Adapting resume..." {
		t.Errorf("expected message 'Adapting resume...', got %v", payload["message"])
	}
}

// TestBrainCompletedEvent_CreatesValidJSON
func TestBrainCompletedEvent_CreatesValidJSON(t *testing.T) {
	jobID := uuid.New()
	resumeURL := "/api/v1/jobs/" + jobID.String() + "/documents/resume.pdf"
	event := BrainCompletedEvent(jobID, resumeURL, "Dear Hiring Manager,")

	var wsEvent WSEvent
	if err := json.Unmarshal(event, &wsEvent); err != nil {
		t.Fatal(err)
	}

	if wsEvent.Type != EventBrainCompleted {
		t.Errorf("expected type %s, got %s", EventBrainCompleted, wsEvent.Type)
	}

	payload, ok := wsEvent.Payload.(map[string]interface{})
	if !ok {
		t.Fatal("payload is not a map")
	}

	if payload["job_id"] != jobID.String() {
		t.Errorf("expected job_id %s, got %v", jobID.String(), payload["job_id"])
	}
	if payload["resume_url"] != resumeURL {
		t.Errorf("expected resume_url %s, got %v", resumeURL, payload["resume_url"])
	}
	if payload["cover_letter"] != "Dear Hiring Manager," {
		t.Errorf("expected cover_letter 'Dear Hiring Manager,', got %v", payload["cover_letter"])
	}
}

// TestBrainErrorEvent_CreatesValidJSON
func TestBrainErrorEvent_CreatesValidJSON(t *testing.T) {
	jobID := uuid.New()
	event := BrainErrorEvent(jobID, "tailoring", "LLM timeout")

	var wsEvent WSEvent
	if err := json.Unmarshal(event, &wsEvent); err != nil {
		t.Fatal(err)
	}

	if wsEvent.Type != EventBrainError {
		t.Errorf("expected type %s, got %s", EventBrainError, wsEvent.Type)
	}

	payload, ok := wsEvent.Payload.(map[string]interface{})
	if !ok {
		t.Fatal("payload is not a map")
	}

	if payload["job_id"] != jobID.String() {
		t.Errorf("expected job_id %s, got %v", jobID.String(), payload["job_id"])
	}
	if payload["step"] != "tailoring" {
		t.Errorf("expected step tailoring, got %v", payload["step"])
	}
	if payload["error"] != "LLM timeout" {
		t.Errorf("expected error 'LLM timeout', got %v", payload["error"])
	}
}
