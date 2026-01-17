package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/blockedby/positions-os/internal/repository"
	"github.com/blockedby/positions-os/internal/web"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockJobsRepository is a mock for JobsRepository
type MockJobsRepository struct {
	mock.Mock
}

// Ensure MockJobsRepository implements the interface expected by JobsHandler (to be defined)
// For now we just implement the method we need for the test

func (m *MockJobsRepository) List(ctx context.Context, filter repository.JobFilter) ([]*repository.Job, int, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]*repository.Job), args.Int(1), args.Error(2)
}

func (m *MockJobsRepository) GetByID(ctx context.Context, id uuid.UUID) (*repository.Job, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.Job), args.Error(1)
}

func TestJobsAPI_List(t *testing.T) {
	mockRepo := new(MockJobsRepository)

	// Sample data
	jobs := []*repository.Job{
		{
			ID:         uuid.New(),
			ExternalID: "job-1",
			Status:     "RAW",
			CreatedAt:  time.Now(),
		},
		{
			ID:         uuid.New(),
			ExternalID: "job-2",
			Status:     "INTERESTED",
			CreatedAt:  time.Now(),
		},
	}

	// Expectation
	mockRepo.On("List", mock.Anything, mock.MatchedBy(func(f repository.JobFilter) bool {
		return f.Limit == 50 // Default limit check
	})).Return(jobs, 2, nil)

	handler := NewJobsHandler(mockRepo, nil)

	req := httptest.NewRequest("GET", "/api/v1/jobs", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Jobs  []*repository.Job `json:"jobs"`
		Total int               `json:"total"`
		Page  int               `json:"page"`
		Limit int               `json:"limit"`
	}
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)

	assert.Equal(t, 2, resp.Total)
	assert.Len(t, resp.Jobs, 2)
	assert.Equal(t, "job-1", resp.Jobs[0].ExternalID)
}

func TestJobsAPI_Filters(t *testing.T) {
	mockRepo := new(MockJobsRepository)
	handler := NewJobsHandler(mockRepo, nil)

	// Expectation with specific filters
	mockRepo.On("List", mock.Anything, mock.MatchedBy(func(f repository.JobFilter) bool {
		return f.Status == "INTERESTED" &&
			f.Tech == "go" &&
			f.SalaryMin == 100000 &&
			f.Query == "remote"
	})).Return([]*repository.Job{}, 0, nil)

	url := "/api/v1/jobs?status=INTERESTED&tech=go&salary_min=100000&q=remote"
	req := httptest.NewRequest("GET", url, nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	mockRepo.AssertExpectations(t)
}

func TestJobsAPI_GetByID(t *testing.T) {
	mockRepo := new(MockJobsRepository)
	handler := NewJobsHandler(mockRepo, nil)

	id := uuid.New()
	job := &repository.Job{ID: id, ExternalID: "job-1"}

	mockRepo.On("GetByID", mock.Anything, id).Return(job, nil)

	r := chi.NewRouter()
	r.Get("/api/v1/jobs/{id}", handler.GetByID)

	req := httptest.NewRequest("GET", "/api/v1/jobs/"+id.String(), nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp repository.Job
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, id, resp.ID)
}

func (m *MockJobsRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func TestJobsAPI_UpdateStatus(t *testing.T) {
	mockRepo := new(MockJobsRepository)
	handler := NewJobsHandler(mockRepo, nil)

	id := uuid.New()
	status := "INTERESTED"

	// Mock GetByID to return job in ANALYZED status (valid transition to INTERESTED)
	mockRepo.On("GetByID", mock.Anything, id).Return(&repository.Job{
		ID:     id,
		Status: "ANALYZED",
	}, nil)
	mockRepo.On("UpdateStatus", mock.Anything, id, status).Return(nil)

	r := chi.NewRouter()
	r.Patch("/api/v1/jobs/{id}/status", handler.UpdateStatus)

	payload := map[string]string{"status": status}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("PATCH", "/api/v1/jobs/"+id.String()+"/status", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestJobsAPI_StatusValidation(t *testing.T) {
	mockRepo := new(MockJobsRepository)
	handler := NewJobsHandler(mockRepo, nil)

	id := uuid.New()

	r := chi.NewRouter()
	r.Patch("/api/v1/jobs/{id}/status", handler.UpdateStatus)

	payload := map[string]string{"status": "INVALID_STATUS"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("PATCH", "/api/v1/jobs/"+id.String()+"/status", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestJobsAPI_Filtering(t *testing.T) {
	mockRepo := new(MockJobsRepository)
	handler := NewJobsHandler(mockRepo, nil)

	// Test Status Filter
	mockRepo.On("List", mock.Anything, mock.MatchedBy(func(f repository.JobFilter) bool {
		return f.Status == "ANALYZED"
	})).Return([]*repository.Job{}, 0, nil)

	req := httptest.NewRequest("GET", "/api/v1/jobs?status=ANALYZED", nil)
	rec := httptest.NewRecorder()
	handler.List(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Test Search Filter
	mockRepo.On("List", mock.Anything, mock.MatchedBy(func(f repository.JobFilter) bool {
		return f.Query == "Go"
	})).Return([]*repository.Job{}, 0, nil)

	req = httptest.NewRequest("GET", "/api/v1/jobs?q=Go", nil)
	rec = httptest.NewRecorder()
	handler.List(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	mockRepo.AssertExpectations(t)
}

func TestJobsAPI_PaginationDefault(t *testing.T) {
	mockRepo := new(MockJobsRepository)
	handler := NewJobsHandler(mockRepo, nil)

	// Expectation default page=1, limit=50
	mockRepo.On("List", mock.Anything, mock.MatchedBy(func(f repository.JobFilter) bool {
		return f.Page == 1 && f.Limit == 50
	})).Return([]*repository.Job{}, 0, nil)

	req := httptest.NewRequest("GET", "/api/v1/jobs?page=0&limit=-5", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	mockRepo.AssertExpectations(t)
}

func TestJobsHandler_UpdateStatus_InvalidTransition(t *testing.T) {
	mockRepo := new(MockJobsRepository)
	handler := NewJobsHandler(mockRepo, nil)

	// Job is in RAW status
	jobID := uuid.New()
	mockRepo.On("GetByID", mock.Anything, jobID).Return(&repository.Job{
		ID:     jobID,
		Status: "RAW",
	}, nil)

	// Try to transition RAW -> INTERESTED (invalid, must go through ANALYZED)
	body := `{"status": "INTERESTED"}`
	req := httptest.NewRequest("PATCH", "/api/v1/jobs/"+jobID.String()+"/status", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Add chi context
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", jobID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.UpdateStatus(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("UpdateStatus() status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
	if !strings.Contains(rr.Body.String(), "Invalid status transition") {
		t.Errorf("UpdateStatus() body = %s, want 'Invalid status transition'", rr.Body.String())
	}
}

func TestJobsAPI_UpdateStatus_Broadcasts(t *testing.T) {
	mockRepo := new(MockJobsRepository)

	// Setup real Hub
	hub := web.NewHub()
	go hub.Run()

	handler := NewJobsHandler(mockRepo, hub)

	id := uuid.New()
	status := "INTERESTED"

	// Mock GetByID to return job in ANALYZED status (valid transition to INTERESTED)
	mockRepo.On("GetByID", mock.Anything, id).Return(&repository.Job{
		ID:     id,
		Status: "ANALYZED",
	}, nil)
	mockRepo.On("UpdateStatus", mock.Anything, id, status).Return(nil)

	// Setup a test server
	r := chi.NewRouter()
	r.Patch("/api/v1/jobs/{id}/status", handler.UpdateStatus)
	r.Get("/ws", func(w http.ResponseWriter, r *http.Request) {
		web.ServeWs(hub, w, r)
	})

	srv := httptest.NewServer(r)
	defer srv.Close()

	// Connect WS client
	u := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	wsConn, wsResp, err := websocket.DefaultDialer.Dial(u, nil)
	require.NoError(t, err)
	defer func() { _ = wsConn.Close() }()
	if wsResp != nil && wsResp.Body != nil {
		defer func() { _ = wsResp.Body.Close() }()
	}

	// Allow time for registration
	time.Sleep(50 * time.Millisecond)

	// Perform Update
	payload := map[string]string{"status": status}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("PATCH", srv.URL+"/api/v1/jobs/"+id.String()+"/status", bytes.NewBuffer(body))

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Check WS message
	_ = wsConn.SetReadDeadline(time.Now().Add(time.Second))
	_, msg, err := wsConn.ReadMessage()
	require.NoError(t, err)

	assert.Contains(t, string(msg), "job.updated")
	assert.Contains(t, string(msg), status)
	assert.Contains(t, string(msg), id.String())
}
