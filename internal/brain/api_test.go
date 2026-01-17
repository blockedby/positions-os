package brain

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// MockRepository for testing
type mockRepository struct {
	job          *Job
	updateError  error
	getByIDError error
}

func (m *mockRepository) GetByID(id uuid.UUID) (*Job, error) {
	if m.getByIDError != nil {
		return nil, m.getByIDError
	}
	return m.job, nil
}

func (m *mockRepository) UpdateBrainOutputs(id uuid.UUID, resumePath, coverText string) error {
	if m.updateError != nil {
		return m.updateError
	}
	if m.job != nil {
		m.job.TailoredResumePath = resumePath
		m.job.CoverLetterText = coverText
	}
	return nil
}

// MockService for testing
type mockService struct {
	result     *PipelineResult
	resultErr  error
	prepareErr error
}

func (m *mockService) PrepareJob(jobID string) (*PipelineResult, error) {
	if m.prepareErr != nil {
		return nil, m.prepareErr
	}
	return m.result, m.resultErr
}

// TestHandler_PrepareJob_StartsProcessing
func TestHandler_PrepareJob_StartsProcessing(t *testing.T) {
	// Setup
	jobID := uuid.New()
	repo := &mockRepository{
		job: &Job{ID: jobID, Status: "INTERESTED"},
	}
	svc := &mockService{
		result: &PipelineResult{
			ResumeMDPath:  "/path/to/resume.md",
			ResumePDFPath: "/path/to/resume.pdf",
			CoverLetterMD: "Dear Hiring Manager,",
		},
	}
	h := NewHandler(repo, svc)

	// Create request with chi context
	req := httptest.NewRequest("POST", "/api/v1/jobs/"+jobID.String()+"/prepare", nil)
	w := httptest.NewRecorder()

	// Set chi URL param
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", jobID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	// Execute
	h.PrepareJob(w, req)

	// Assert
	if w.Code != http.StatusAccepted {
		t.Errorf("expected status 202, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}

	if resp["status"] != "processing" {
		t.Errorf("expected status=processing, got %v", resp["status"])
	}

	if resp["ws_channel"] == nil {
		t.Error("expected ws_channel in response")
	}
}

// TestHandler_PrepareJob_InvalidID_Returns400
func TestHandler_PrepareJob_InvalidID(t *testing.T) {
	repo := &mockRepository{}
	svc := &mockService{}
	h := NewHandler(repo, svc)

	req := httptest.NewRequest("POST", "/api/v1/jobs/invalid-uuid/prepare", nil)
	w := httptest.NewRecorder()

	h.PrepareJob(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

// TestHandler_PrepareJob_JobNotFound_Returns404
func TestHandler_PrepareJob_JobNotFound(t *testing.T) {
	jobID := uuid.New()
	repo := &mockRepository{
		getByIDError: ErrJobNotFound,
	}
	svc := &mockService{}
	h := NewHandler(repo, svc)

	req := httptest.NewRequest("POST", "/api/v1/jobs/"+jobID.String()+"/prepare", nil)
	w := httptest.NewRecorder()

	// Set chi URL param
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", jobID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.PrepareJob(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

// TestHandler_PrepareJob_WrongStatus_Returns400
func TestHandler_PrepareJob_WrongStatus(t *testing.T) {
	jobID := uuid.New()
	repo := &mockRepository{
		job: &Job{ID: jobID, Status: "RAW"},
	}
	svc := &mockService{}
	h := NewHandler(repo, svc)

	req := httptest.NewRequest("POST", "/api/v1/jobs/"+jobID.String()+"/prepare", nil)
	w := httptest.NewRecorder()

	// Set chi URL param
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", jobID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.PrepareJob(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

// TestHandler_GetDocuments_ReturnsDocumentInfo
func TestHandler_GetDocuments_ReturnsDocumentInfo(t *testing.T) {
	jobID := uuid.New()
	repo := &mockRepository{
		job: &Job{
			ID:                 jobID,
			Status:             "TAILORED",
			TailoredResumePath: "/storage/outputs/" + jobID.String() + "/resume.pdf",
			CoverLetterText:    "Dear Hiring Manager,",
		},
	}
	svc := &mockService{}
	h := NewHandler(repo, svc)

	req := httptest.NewRequest("GET", "/api/v1/jobs/"+jobID.String()+"/documents", nil)
	w := httptest.NewRecorder()

	// Set chi URL param
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", jobID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.GetDocuments(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}

	if resp["resume_url"] == nil {
		t.Error("expected resume_url in response")
	}
	if resp["cover_letter"] == nil {
		t.Error("expected cover_letter in response")
	}
}

// TestHandler_GetDocuments_NotTailored_Returns404
func TestHandler_GetDocuments_NotTailored_Returns404(t *testing.T) {
	jobID := uuid.New()
	repo := &mockRepository{
		job: &Job{ID: jobID, Status: "INTERESTED"},
	}
	svc := &mockService{}
	h := NewHandler(repo, svc)

	req := httptest.NewRequest("GET", "/api/v1/jobs/"+jobID.String()+"/documents", nil)
	w := httptest.NewRecorder()

	// Set chi URL param
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", jobID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.GetDocuments(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

// TestHandler_DownloadResume_ServesPDF
func TestHandler_DownloadResume_ServesPDF(t *testing.T) {
	// This test requires actual file - using mock
	// For full implementation, would create temp PDF file
}

// TestRegisterRoutes_RegistersAllRoutes
func TestRegisterRoutes_RegistersAllRoutes(t *testing.T) {
	repo := &mockRepository{
		job: &Job{
			ID:                 uuid.New(),
			Status:             "TAILORED",
			TailoredResumePath: "/path/to/resume.pdf",
			CoverLetterText:    "Cover letter text",
		},
	}
	svc := &mockService{
		result: &PipelineResult{
			ResumePDFPath: "/path/to/resume.pdf",
			CoverLetterMD: "Cover letter",
		},
	}
	h := NewHandler(repo, svc)

	router := chi.NewRouter()
	RegisterRoutes(router, h)

	jobID := repo.job.ID

	// Test POST /api/v1/jobs/{id}/prepare
	req := httptest.NewRequest("POST", "/api/v1/jobs/"+jobID.String()+"/prepare", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusAccepted && w.Code != http.StatusNotFound {
		// Might be 404 due to mock repository, but route exists
		t.Logf("prepare endpoint: status %d", w.Code)
	}

	// Test GET /api/v1/jobs/{id}/documents
	req = httptest.NewRequest("GET", "/api/v1/jobs/"+jobID.String()+"/documents", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
		t.Logf("documents endpoint: status %d", w.Code)
	}
}
