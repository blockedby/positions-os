package brain

import (
	"context"
	"fmt"
	"testing"
)

// Mock implementations for testing
type mockStorage struct {
	baseResume   string
	loadErr      error
	saveErr      error
	savedJobs    map[string]string
	savedCovers  map[string]string
}

func (m *mockStorage) LoadBaseResume() (string, error) {
	return m.baseResume, m.loadErr
}

func (m *mockStorage) SaveTailoredResume(jobID, content string) error {
	if m.savedJobs == nil {
		m.savedJobs = make(map[string]string)
	}
	m.savedJobs[jobID] = content
	return m.saveErr
}

func (m *mockStorage) SaveCoverLetter(jobID, content string) error {
	if m.savedCovers == nil {
		m.savedCovers = make(map[string]string)
	}
	m.savedCovers[jobID] = content
	return m.saveErr
}

type mockLLM struct {
	tailoredResume string
	coverLetter    string
	tailorErr      error
	coverErr       error
}

func (m *mockLLM) TailorResume(ctx context.Context, baseResume, jobData string) (string, error) {
	return m.tailoredResume, m.tailorErr
}

func (m *mockLLM) GenerateCover(ctx context.Context, jobData, tailoredResume, templateID string) (string, error) {
	return m.coverLetter, m.coverErr
}

type mockPDFRenderer struct {
	resumePath string
	resumeErr  error
}

func (m *mockPDFRenderer) RenderResume(ctx context.Context, jobID string, data map[string]string) (string, error) {
	return m.resumePath, m.resumeErr
}

// TestService_TailorResumePipeline_FullFlow
func TestService_TailorResumePipeline_FullFlow(t *testing.T) {
	// Setup
	storage := &mockStorage{
		baseResume:  "# Base Resume\nSenior Developer",
		savedJobs:   make(map[string]string),
		savedCovers: make(map[string]string),
	}
	llm := &mockLLM{
		tailoredResume: "# Tailored Resume\nGo Developer for this job",
		coverLetter:    "Dear Hiring Manager,\nI'm interested...",
	}
	pdf := &mockPDFRenderer{
		resumePath: "/outputs/job-123/resume.pdf",
	}

	svc := NewService(storage, llm, pdf)
	ctx := context.Background()

	jobData := map[string]string{
		"title":   "Go Developer",
		"company": "TechCorp",
	}

	// Execute
	result, err := svc.TailorResumePipeline(ctx, "job-123", jobData)

	// Assert
	if err != nil {
		t.Errorf("TailorResumePipeline() error = %v", err)
	}
	if result.ResumePDFPath == "" {
		t.Error("ResumePDFPath is empty")
	}
	if result.CoverLetterMD == "" {
		t.Error("CoverLetterMD is empty")
	}

	// Verify storage was called
	if storage.savedJobs["job-123"] == "" {
		t.Error("tailored resume not saved")
	}
	if storage.savedCovers["job-123"] == "" {
		t.Error("cover letter not saved")
	}
}

// TestService_TailorResumePipeline_LLMErrors_Propagated
func TestService_TailorResumePipeline_LLMErrors_Propagated(t *testing.T) {
	// Setup
	storage := &mockStorage{
		baseResume: "# Base Resume",
	}
	llm := &mockLLM{
		tailorErr: fmt.Errorf("LLM timeout"),
	}
	pdf := &mockPDFRenderer{}

	svc := NewService(storage, llm, pdf)
	ctx := context.Background()

	// Execute
	_, err := svc.TailorResumePipeline(ctx, "job-123", nil)

	// Assert
	if err == nil {
		t.Error("expected error from LLM, got nil")
	}
}

// TestService_TailorResumePipeline_PDFError_SavesMarkdown
func TestService_TailorResumePipeline_PDFError_SavesMarkdown(t *testing.T) {
	// Setup
	storage := &mockStorage{
		baseResume:   "# Base Resume",
		savedJobs:    make(map[string]string),
		savedCovers:  make(map[string]string),
	}
	llm := &mockLLM{
		tailoredResume: "# Tailored",
		coverLetter:    "# Cover",
	}
	pdf := &mockPDFRenderer{
		resumeErr: fmt.Errorf("Chrome not available"),
	}

	svc := NewService(storage, llm, pdf)
	ctx := context.Background()

	// Execute
	result, err := svc.TailorResumePipeline(ctx, "job-123", nil)

	// Assert - should save markdown even if PDF fails
	if err == nil {
		t.Error("expected PDF error to be returned")
	}
	if storage.savedJobs["job-123"] == "" {
		t.Error("markdown should be saved even if PDF fails")
	}
	// Markdown paths should still be populated
	if result.ResumeMDPath == "" {
		t.Error("ResumeMDPath should be set even if PDF fails")
	}
	if result.CoverLetterMD == "" {
		t.Error("CoverLetterMD should be set even if PDF fails")
	}
}
