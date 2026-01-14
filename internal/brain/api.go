package brain

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/blockedby/positions-os/internal/logger"
)

// Common errors
var (
	ErrJobNotFound = fmt.Errorf("job not found")
	ErrInvalidID   = fmt.Errorf("invalid job ID")
)

// Repository defines the data layer interface for brain operations.
type Repository interface {
	GetByID(id uuid.UUID) (*BrainJob, error)
	UpdateBrainOutputs(id uuid.UUID, resumePath, coverText string) error
}

// BrainJob represents job data needed by brain handlers.
type BrainJob struct {
	ID                 uuid.UUID
	Status             string
	TailoredResumePath string
	CoverLetterText    string
	StructuredData     map[string]string
}

// BrainService defines the service interface for brain operations.
type BrainService interface {
	PrepareJob(jobID string) (*PipelineResult, error)
}

// Handler handles brain API requests.
type Handler struct {
	repo Repository
	svc  BrainService
}

// NewHandler creates a new brain handler.
func NewHandler(repo Repository, svc BrainService) *Handler {
	return &Handler{
		repo: repo,
		svc:  svc,
	}
}

// PrepareJobRequest represents the request body for prepare endpoint.
type PrepareJobRequest struct {
	// No body needed for now, job ID comes from URL
}

// PrepareJobResponse represents the response for prepare endpoint.
type PrepareJobResponse struct {
	Status     string `json:"status"`
	WSChannel  string `json:"ws_channel"`
	Message    string `json:"message,omitempty"`
}

// DocumentsResponse represents the response for documents endpoint.
type DocumentsResponse struct {
	ResumeURL    string `json:"resume_url,omitempty"`
	CoverLetter  string `json:"cover_letter,omitempty"`
	Status       string `json:"status"`
	PreparedAt   string `json:"prepared_at,omitempty"`
}

// PrepareJob triggers the resume tailoring pipeline for a job.
// POST /api/v1/jobs/{id}/prepare
func (h *Handler) PrepareJob(w http.ResponseWriter, r *http.Request) {
	logger.Info("prepare job request received")

	// Parse job ID
	idStr := chi.URLParam(r, "id")
	jobID, err := uuid.Parse(idStr)
	if err != nil {
		logger.Error("invalid job ID", err)
		http.Error(w, "Invalid job ID", http.StatusBadRequest)
		return
	}

	// Get job
	job, err := h.repo.GetByID(jobID)
	if err != nil {
		if err == ErrJobNotFound {
			http.NotFound(w, r)
			return
		}
		logger.Error("failed to get job", err)
		http.Error(w, "Failed to get job", http.StatusInternalServerError)
		return
	}

	// Validate status - only INTERESTED jobs can be prepared
	if job.Status != "INTERESTED" {
		logger.Info("invalid job status for prepare: " + job.Status)
		http.Error(w, "Job must be in INTERESTED status", http.StatusBadRequest)
		return
	}

	// Start async processing
	go h.processJob(jobID)

	// Return immediate response with WS channel
	resp := PrepareJobResponse{
		Status:    "processing",
		WSChannel: fmt.Sprintf("brain.%s", jobID.String()),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(resp)
	logger.Info("prepare job accepted: " + jobID.String())
}

// processJob runs the tailoring pipeline asynchronously.
func (h *Handler) processJob(jobID uuid.UUID) {
	logger.Info("processing job: " + jobID.String())

	result, err := h.svc.PrepareJob(jobID.String())
	if err != nil {
		logger.Error("job processing failed", err)
		// TODO: Send WebSocket error event
		return
	}

	// Update job with document paths
	if err := h.repo.UpdateBrainOutputs(jobID, result.ResumePDFPath, result.CoverLetterMD); err != nil {
		logger.Error("failed to update job outputs", err)
		return
	}

	logger.Info("job processing complete: " + jobID.String())
	// TODO: Send WebSocket complete event
}

// GetDocuments returns the generated documents for a job.
// GET /api/v1/jobs/{id}/documents
func (h *Handler) GetDocuments(w http.ResponseWriter, r *http.Request) {
	logger.Info("get documents request received")

	idStr := chi.URLParam(r, "id")
	jobID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid job ID", http.StatusBadRequest)
		return
	}

	job, err := h.repo.GetByID(jobID)
	if err != nil {
		if err == ErrJobNotFound {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "Failed to get job", http.StatusInternalServerError)
		return
	}

	// Check if documents have been generated
	if job.TailoredResumePath == "" {
		http.NotFound(w, r)
		return
	}

	resp := DocumentsResponse{
		ResumeURL:   "/api/v1/jobs/" + jobID.String() + "/documents/resume.pdf",
		CoverLetter: job.CoverLetterText,
		Status:      job.Status,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// DownloadResume serves the generated resume PDF.
// GET /api/v1/jobs/{id}/documents/resume.pdf
func (h *Handler) DownloadResume(w http.ResponseWriter, r *http.Request) {
	logger.Info("download resume request received")

	idStr := chi.URLParam(r, "id")
	jobID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid job ID", http.StatusBadRequest)
		return
	}

	job, err := h.repo.GetByID(jobID)
	if err != nil {
		if err == ErrJobNotFound {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "Failed to get job", http.StatusInternalServerError)
		return
	}

	if job.TailoredResumePath == "" {
		http.NotFound(w, r)
		return
	}

	// Serve the PDF file
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=resume_%s.pdf", jobID.String()))
	http.ServeFile(w, r, job.TailoredResumePath)
}

// RegisterRoutes registers all brain API routes.
func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/api/v1/jobs", func(r chi.Router) {
		r.Post("/{id}/prepare", h.PrepareJob)
		r.Get("/{id}/documents", h.GetDocuments)
		r.Get("/{id}/documents/resume.pdf", h.DownloadResume)
	})
}
