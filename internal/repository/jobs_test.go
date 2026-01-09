package repository

import (
	"testing"
)

// test job status validation
func TestJob_IsValidStatus(t *testing.T) {
	validStatuses := []string{"RAW", "ANALYZED", "REJECTED", "INTERESTED", "TAILORED", "SENT", "RESPONDED"}

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
