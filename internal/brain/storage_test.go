package brain

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveTailoredResume_SavesFile(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	jobID := "test-job-123"
	content := "# John Doe\n\nSenior Go Developer..."

	// Execute
	err := SaveTailoredResume(tmpDir, jobID, content)

	// Assert
	if err != nil {
		t.Errorf("SaveTailoredResume() error = %v", err)
	}

	// Verify file exists
	expectedPath := filepath.Join(tmpDir, "outputs", jobID, "resume_tailored.md")
	data, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Errorf("saved file not found: %v", err)
	}

	if string(data) != content {
		t.Errorf("file content = %q, want %q", string(data), content)
	}
}

func TestSaveCoverLetter_SavesFile(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	jobID := "test-job-456"
	content := "Dear Hiring Manager,\n\nI am excited to apply..."

	// Execute
	err := SaveCoverLetter(tmpDir, jobID, content)

	// Assert
	if err != nil {
		t.Errorf("SaveCoverLetter() error = %v", err)
	}

	// Verify file exists
	expectedPath := filepath.Join(tmpDir, "outputs", jobID, "cover_letter.md")
	data, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Errorf("saved file not found: %v", err)
	}

	if string(data) != content {
		t.Errorf("file content = %q, want %q", string(data), content)
	}
}
