package brain

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/blockedby/positions-os/internal/logger"
)

const (
	// BaseResumeFilename is the name of the base resume file
	BaseResumeFilename = "resume.md"
	// TailoredResumeFilename is the name of the tailored resume output
	TailoredResumeFilename = "resume_tailored.md"
	// CoverLetterFilename is the name of the cover letter output
	CoverLetterFilename = "cover_letter.md"
	// OutputsDir is the directory for generated files
	OutputsDir = "outputs"
)

// LoadBaseResume reads the base resume from storage directory.
// Returns the content and an error if the file doesn't exist.
func LoadBaseResume(storagePath string) (string, error) {
	logger.Info("loading base resume from " + storagePath)

	resumePath := filepath.Join(storagePath, BaseResumeFilename)

	content, err := os.ReadFile(resumePath)
	if err != nil {
		logger.Error("failed to load base resume", err)
		return "", fmt.Errorf("base resume not found at %s", resumePath)
	}

	logger.Info("base resume loaded successfully")
	return string(content), nil
}

// SaveTailoredResume saves the tailored resume for a specific job.
// Creates the outputs/{job_id} directory if it doesn't exist.
func SaveTailoredResume(storagePath, jobID, content string) error {
	logger.Info("saving tailored resume for job: " + jobID)

	outputDir := filepath.Join(storagePath, OutputsDir, jobID)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		logger.Error("failed to create output directory", err)
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	outputPath := filepath.Join(outputDir, TailoredResumeFilename)

	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		logger.Error("failed to save tailored resume", err)
		return fmt.Errorf("failed to save tailored resume: %w", err)
	}

	logger.Info("tailored resume saved successfully")
	return nil
}

// SaveCoverLetter saves the cover letter for a specific job.
func SaveCoverLetter(storagePath, jobID, content string) error {
	logger.Info("saving cover letter for job: " + jobID)

	outputDir := filepath.Join(storagePath, OutputsDir, jobID)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		logger.Error("failed to create output directory", err)
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	outputPath := filepath.Join(outputDir, CoverLetterFilename)
	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		logger.Error("failed to save cover letter", err)
		return fmt.Errorf("failed to save cover letter: %w", err)
	}

	logger.Info("cover letter saved successfully")
	return nil
}

// FileStorage implements Storage interface using the file system.
type FileStorage struct {
	StoragePath string
}

// LoadBaseResume implements Storage.
func (f *FileStorage) LoadBaseResume() (string, error) {
	return LoadBaseResume(f.StoragePath)
}

// SaveTailoredResume implements Storage.
func (f *FileStorage) SaveTailoredResume(jobID, content string) error {
	return SaveTailoredResume(f.StoragePath, jobID, content)
}

// SaveCoverLetter implements Storage.
func (f *FileStorage) SaveCoverLetter(jobID, content string) error {
	return SaveCoverLetter(f.StoragePath, jobID, content)
}
