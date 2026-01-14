package brain

import (
	"context"
	"fmt"
	"strings"

	"github.com/blockedby/positions-os/internal/logger"
)

// Storage defines the storage interface for brain operations.
type Storage interface {
	LoadBaseResume() (string, error)
	SaveTailoredResume(jobID, content string) error
	SaveCoverLetter(jobID, content string) error
}

// LLM defines the LLM interface for brain operations.
type LLM interface {
	TailorResume(ctx context.Context, baseResume, jobData string) (string, error)
	GenerateCover(ctx context.Context, jobData, tailoredResume, templateID string) (string, error)
}

// Broadcaster defines the interface for sending WebSocket events.
type Broadcaster interface {
	Broadcast(event interface{})
}

// Service orchestrates the resume tailoring pipeline.
type Service struct {
	storage    Storage
	llm        LLM
	pdf        Renderer
	storageDir string
	broadcaster Broadcaster // Optional: for WebSocket events
}

// PipelineResult contains the output paths from the pipeline.
type PipelineResult struct {
	ResumeMDPath   string // Path to tailored resume markdown
	ResumePDFPath  string // Path to resume PDF (for attachment)
	CoverLetterMD  string // Cover letter content (for email/message)
}

// NewService creates a new brain service.
func NewService(storage Storage, llm LLM, pdf Renderer) *Service {
	return &Service{
		storage: storage,
		llm:     llm,
		pdf:     pdf,
	}
}

// SetBroadcaster sets the WebSocket event broadcaster.
func (s *Service) SetBroadcaster(b Broadcaster) {
	s.broadcaster = b
}

// emitProgress sends a progress event via WebSocket if broadcaster is set.
func (s *Service) emitProgress(jobID, step string, progress int, message string) {
	if s.broadcaster != nil {
		evt := map[string]interface{}{
			"type": "brain.progress",
			"payload": map[string]interface{}{
				"job_id":   jobID,
				"step":     step,
				"progress": progress,
				"message":  message,
			},
		}
		s.broadcaster.Broadcast(evt)
	}
}

// emitComplete sends a completion event via WebSocket if broadcaster is set.
func (s *Service) emitComplete(jobID, resumeURL, coverLetter string) {
	if s.broadcaster != nil {
		evt := map[string]interface{}{
			"type": "brain.completed",
			"payload": map[string]interface{}{
				"job_id":       jobID,
				"resume_url":   resumeURL,
				"cover_letter": coverLetter,
			},
		}
		s.broadcaster.Broadcast(evt)
	}
}

// emitError sends an error event via WebSocket if broadcaster is set.
func (s *Service) emitError(jobID, step, errMsg string) {
	if s.broadcaster != nil {
		evt := map[string]interface{}{
			"type": "brain.error",
			"payload": map[string]interface{}{
				"job_id": jobID,
				"step":   step,
				"error":  errMsg,
			},
		}
		s.broadcaster.Broadcast(evt)
	}
}

// SetStorageDir sets the storage directory for the service.
func (s *Service) SetStorageDir(dir string) {
	s.storageDir = dir
}

// TailorResumePipeline runs the full tailoring pipeline:
// 1. Load base resume
// 2. Tailor resume via LLM
// 3. Generate cover letter via LLM
// 4. Save markdown outputs
// 5. Render PDFs
func (s *Service) TailorResumePipeline(ctx context.Context, jobID string, jobData map[string]string) (*PipelineResult, error) {
	logger.Info("starting tailoring pipeline for job: " + jobID)

	result := &PipelineResult{}
	var multiErr []error

	// Emit started event (0%)
	s.emitProgress(jobID, "started", 0, "Starting tailoring pipeline")

	// Step 1: Load base resume (0-10%)
	logger.Info("loading base resume")
	baseResume, err := s.storage.LoadBaseResume()
	if err != nil {
		logger.Error("failed to load base resume", err)
		s.emitError(jobID, "load", "Failed to load base resume")
		return nil, fmt.Errorf("load base resume: %w", err)
	}

	// Format job data for LLM
	jobDataStr := formatJobData(jobData)

	// Step 2: Tailor resume (10-50%)
	logger.Info("tailoring resume via LLM")
	s.emitProgress(jobID, "tailoring", 25, "Adapting resume to job requirements")
	tailoredResume, err := s.llm.TailorResume(ctx, baseResume, jobDataStr)
	if err != nil {
		logger.Error("failed to tailor resume", err)
		s.emitError(jobID, "tailoring", "Failed to tailor resume")
		return nil, fmt.Errorf("tailor resume: %w", err)
	}

	// Save tailored resume markdown
	if err := s.storage.SaveTailoredResume(jobID, tailoredResume); err != nil {
		logger.Error("failed to save tailored resume", err)
		s.emitError(jobID, "save", "Failed to save resume")
		return nil, fmt.Errorf("save tailored resume: %w", err)
	}
	result.ResumeMDPath = fmt.Sprintf("storage/outputs/%s/resume_tailored.md", jobID)

	// Step 3: Generate cover letter (50-75%)
	logger.Info("generating cover letter via LLM")
	s.emitProgress(jobID, "cover_letter", 50, "Generating cover letter")
	templateID := selectCoverTemplate(jobData)
	coverLetter, err := s.llm.GenerateCover(ctx, jobDataStr, tailoredResume, templateID)
	if err != nil {
		logger.Error("failed to generate cover letter", err)
		s.emitError(jobID, "cover_letter", "Failed to generate cover letter")
		return nil, fmt.Errorf("generate cover letter: %w", err)
	}

	// Save cover letter content
	if err := s.storage.SaveCoverLetter(jobID, coverLetter); err != nil {
		logger.Error("failed to save cover letter", err)
		s.emitError(jobID, "save", "Failed to save cover letter")
		return nil, fmt.Errorf("save cover letter: %w", err)
	}
	result.CoverLetterMD = coverLetter

	// Step 4: Render resume PDF only (75-100%)
	logger.Info("rendering resume PDF")
	s.emitProgress(jobID, "pdf_rendering", 75, "Creating PDF document")
	resumePDFData := extractResumeData(tailoredResume)
	resumePath, err := s.pdf.RenderResume(ctx, jobID, resumePDFData)
	if err != nil {
		logger.Error("failed to render resume PDF", err)
		s.emitError(jobID, "pdf_rendering", "Failed to create PDF")
		multiErr = append(multiErr, fmt.Errorf("resume PDF: %w", err))
	} else {
		result.ResumePDFPath = resumePath
	}

	// If we have errors, return partial result
	if len(multiErr) > 0 {
		if result.ResumePDFPath == "" {
			return result, fmt.Errorf("PDF rendering failed: %v", multiErr)
		}
	}

	// Emit completion event (100%)
	s.emitProgress(jobID, "complete", 100, "Tailoring complete")
	s.emitComplete(jobID, "/api/v1/jobs/"+jobID+"/documents/resume.pdf", result.CoverLetterMD)

	logger.Info("tailoring pipeline complete for job: " + jobID)
	return result, nil
}

// formatJobData converts job data map to formatted string for LLM.
func formatJobData(data map[string]string) string {
	if data == nil {
		return ""
	}
	var parts []string
	for k, v := range data {
		parts = append(parts, fmt.Sprintf("%s: %s", k, v))
	}
	return strings.Join(parts, "\n")
}

// selectCoverTemplate chooses a template based on job data.
func selectCoverTemplate(jobData map[string]string) string {
	if jobData == nil {
		return "professional_en"
	}
	lang := jobData["language"]
	if lang == "ru" || lang == "russian" {
		return "formal_ru"
	}
	return "professional_en"
}

// extractResumeData extracts structured data from markdown resume.
func extractResumeData(markdown string) map[string]string {
	data := make(map[string]string)
	lines := strings.Split(markdown, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "# ") {
			data["name"] = strings.TrimPrefix(line, "# ")
		}
	}
	if data["name"] == "" {
		data["name"] = "Your Name"
	}
	data["title"] = "Professional Resume"
	data["summary"] = "Summary"
	data["skills"] = "Skills"
	data["experience"] = "Work Experience"
	data["education"] = "Education"
	return data
}

