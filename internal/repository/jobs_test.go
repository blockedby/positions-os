package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// test job status validation
func TestJob_IsValidStatus(t *testing.T) {
	validStatuses := []string{"RAW", "ANALYZED", "REJECTED", "INTERESTED", "TAILORED", "TAILORED_APPROVED", "SENT", "RESPONDED"}

	for _, status := range validStatuses {
		job := Job{Status: status}
		if !job.IsValidStatus() {
			t.Errorf("status %s should be valid", status)
		}
	}

	invalidJob := Job{Status: "INVALID"}
	if invalidJob.IsValidStatus() {
		t.Error("invalid status should not be valid")
	}
}

// test job is new check
func TestJob_IsNew(t *testing.T) {
	newJob := Job{Status: "RAW"}
	if !newJob.IsNew() {
		t.Error("job with RAW status should be new")
	}

	analyzedJob := Job{Status: "ANALYZED"}
	if analyzedJob.IsNew() {
		t.Error("job with ANALYZED status should not be new")
	}
}

// test job content hash
func TestJob_ComputeHash(t *testing.T) {
	job := Job{RawContent: "test content"}
	hash := job.ComputeHash()

	if hash == "" {
		t.Error("ComputeHash() should return non-empty string")
	}

	// same content = same hash
	job2 := Job{RawContent: "test content"}
	if job2.ComputeHash() != hash {
		t.Error("same content should produce same hash")
	}

	// different content = different hash
	job3 := Job{RawContent: "different content"}
	if job3.ComputeHash() == hash {
		t.Error("different content should produce different hash")
	}
}

// test BrainOutput fields exist in Job struct
func TestJob_HasBrainOutputFields(t *testing.T) {
	job := Job{
		TailoredResumePath: "/path/to/resume.pdf",
		CoverLetterText:    "Dear Hiring Manager...",
	}

	if job.TailoredResumePath != "/path/to/resume.pdf" {
		t.Errorf("TailoredResumePath = %q, want %q", job.TailoredResumePath, "/path/to/resume.pdf")
	}

	if job.CoverLetterText != "Dear Hiring Manager..." {
		t.Errorf("CoverLetterText = %q, want %q", job.CoverLetterText, "Dear Hiring Manager...")
	}
}

// test valid status transitions
func TestJob_CanTransitionTo(t *testing.T) {
	tests := []struct {
		from string
		to   string
		want bool
	}{
		// Valid transitions from RAW
		{"RAW", "ANALYZED", true},
		{"RAW", "REJECTED", true},

		// Invalid: can't skip ANALYZED to go to INTERESTED
		{"RAW", "INTERESTED", false},
		{"RAW", "TAILORED", false},
		{"RAW", "SENT", false},
		{"RAW", "RESPONDED", false},

		// Valid transitions from ANALYZED
		{"ANALYZED", "INTERESTED", true},
		{"ANALYZED", "REJECTED", true},

		// Valid transitions from INTERESTED
		{"INTERESTED", "TAILORED", true},
		{"INTERESTED", "REJECTED", true},

		// Valid transitions from TAILORED
		{"TAILORED", "TAILORED_APPROVED", true}, // NEW: can approve after tailoring
		{"TAILORED", "SENT", false},             // UPDATE: can't skip approval
		{"TAILORED", "REJECTED", true},

		// Valid transitions from TAILORED_APPROVED (NEW)
		{"TAILORED_APPROVED", "SENT", true},
		{"TAILORED_APPROVED", "REJECTED", true},
		{"TAILORED_APPROVED", "RAW", true},

		// Invalid: can't skip TAILORED_APPROVED
		{"TAILORED", "RESPONDED", false},
		{"TAILORED_APPROVED", "RESPONDED", false},

		// Valid transitions from SENT
		{"SENT", "RESPONDED", true},
		{"SENT", "REJECTED", true},

		// Always allow re-analysis
		{"ANALYZED", "RAW", true},
		{"INTERESTED", "RAW", true},
		{"REJECTED", "RAW", true},
	}

	for _, tt := range tests {
		t.Run(tt.from+"->"+tt.to, func(t *testing.T) {
			job := Job{Status: tt.from}
			got := job.CanTransitionTo(tt.to)
			if got != tt.want {
				t.Errorf("CanTransitionTo(%s) = %v, want %v", tt.to, got, tt.want)
			}
		})
	}
}

// TestJobsRepository_GetExistingMessageIDs_Interface verifies the method exists
func TestJobsRepository_GetExistingMessageIDs_Interface(t *testing.T) {
	// Verify the method signature exists on JobsRepository
	var _ interface {
		GetExistingMessageIDs(ctx context.Context, targetID uuid.UUID) ([]int64, error)
	} = (*JobsRepository)(nil)
}

// TestJobsRepository_UpdateBrainOutputs_MethodExists verifies the UpdateBrainOutputs method exists
// with the correct signature. This is a compile-time check.
func TestJobsRepository_UpdateBrainOutputs_MethodExists(t *testing.T) {
	// This test verifies that the UpdateBrainOutputs method exists with the correct signature.
	// If this compiles, the method exists.
	var repo *JobsRepository
	_ = repo.UpdateBrainOutputs
}
