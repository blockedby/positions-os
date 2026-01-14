package brain

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadBaseResume_RedisFileNotFound
// RED: Test fails because storage.go doesn't exist yet
func TestLoadBaseResume_ReadsFile(t *testing.T) {
	// Setup: Create temp directory with resume.md
	tmpDir := t.TempDir()
	resumePath := filepath.Join(tmpDir, "resume.md")
	expectedContent := "# John Doe\nSenior Go Developer"

	err := os.WriteFile(resumePath, []byte(expectedContent), 0644)
	if err != nil {
		t.Fatalf("failed to setup test file: %v", err)
	}

	// Execute
	content, err := LoadBaseResume(tmpDir)

	// Assert
	if err != nil {
		t.Errorf("LoadBaseResume() error = %v", err)
	}
	if content != expectedContent {
		t.Errorf("LoadBaseResume() = %q, want %q", content, expectedContent)
	}
}

// TestSaveTailoredResume_CreatesFile
func TestSaveTailoredResume_SavesFile(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	jobID := "test-job-123"
	content := "# Tailored Resume\nCustom content"

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

// TestLoadBaseResume_FileNotFound
func TestLoadBaseResume_FileNotFound(t *testing.T) {
	// Setup: Empty directory
	tmpDir := t.TempDir()

	// Execute
	_, err := LoadBaseResume(tmpDir)

	// Assert - should return error
	if err == nil {
		t.Error("LoadBaseResume() expected error for missing file, got nil")
	}
}
