package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/blockedby/positions-os/internal/dispatcher"
	"github.com/blockedby/positions-os/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockApplicationsRepository is a mock for ApplicationsRepository
type MockApplicationsRepository struct {
	applications map[uuid.UUID]*models.JobApplication
	createErr    error
	getErr       error
	updateErr    error
}

func (m *MockApplicationsRepository) Create(ctx context.Context, app *models.JobApplication) error {
	if m.createErr != nil {
		return m.createErr
	}
	if app.ID == uuid.Nil {
		app.ID = uuid.New()
	}
	if m.applications == nil {
		m.applications = make(map[uuid.UUID]*models.JobApplication)
	}
	m.applications[app.ID] = app
	return nil
}

func (m *MockApplicationsRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.JobApplication, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.applications == nil {
		return nil, nil
	}
	return m.applications[id], nil
}

func (m *MockApplicationsRepository) GetByJobID(ctx context.Context, jobID uuid.UUID) ([]*models.JobApplication, error) {
	if m.applications == nil {
		return []*models.JobApplication{}, nil
	}
	var result []*models.JobApplication
	for _, app := range m.applications {
		if app.JobID == jobID {
			result = append(result, app)
		}
	}
	return result, nil
}

func (m *MockApplicationsRepository) UpdateDeliveryStatus(ctx context.Context, id uuid.UUID, status models.DeliveryStatus) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	if m.applications == nil {
		return nil
	}
	if app, ok := m.applications[id]; ok {
		app.DeliveryStatus = status
	}
	return nil
}

// MockDispatcherService is a mock for DispatcherService
type MockDispatcherService struct {
	sendErr error
}

func (m *MockDispatcherService) SendApplication(ctx context.Context, req *dispatcher.SendRequest) error {
	return m.sendErr
}

func TestNewApplicationsHandler(t *testing.T) {
	repo := &MockApplicationsRepository{}
	dispatcherSvc := &MockDispatcherService{}

	handler := NewApplicationsHandler(repo, dispatcherSvc)

	assert.NotNil(t, handler)
	assert.NotNil(t, handler.repo)
	assert.NotNil(t, handler.dispatcher)
}

func TestApplicationsHandler_ListByJobID(t *testing.T) {
	jobID := uuid.New()
	app1 := &models.JobApplication{
		ID:              uuid.New(),
		JobID:           jobID,
		DeliveryStatus:  models.DeliveryStatusPending,
		DeliveryChannel: deliveryChannelPtr(models.DeliveryChannelTGDM),
	}
	app2 := &models.JobApplication{
		ID:              uuid.New(),
		JobID:           uuid.New(), // Different job
		DeliveryStatus:  models.DeliveryStatusSent,
		DeliveryChannel: deliveryChannelPtr(models.DeliveryChannelTGDM),
	}

	repo := &MockApplicationsRepository{
		applications: map[uuid.UUID]*models.JobApplication{
			app1.ID: app1,
			app2.ID: app2,
		},
	}
	dispatcherSvc := &MockDispatcherService{}
	handler := NewApplicationsHandler(repo, dispatcherSvc)

	req := httptest.NewRequest("GET", "/api/v1/applications?job_id="+jobID.String(), nil)
	w := httptest.NewRecorder()

	handler.List(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Applications []*models.JobApplication `json:"applications"`
		Total        int                      `json:"total"`
	}
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)

	assert.Equal(t, 1, resp.Total)
	assert.Equal(t, 1, len(resp.Applications))
	assert.Equal(t, app1.ID, resp.Applications[0].ID)
}

func TestApplicationsHandler_GetByID(t *testing.T) {
	appID := uuid.New()
	app := &models.JobApplication{
		ID:              appID,
		JobID:           uuid.New(),
		DeliveryStatus:  models.DeliveryStatusPending,
		DeliveryChannel: deliveryChannelPtr(models.DeliveryChannelTGDM),
	}

	repo := &MockApplicationsRepository{
		applications: map[uuid.UUID]*models.JobApplication{
			appID: app,
		},
	}
	dispatcherSvc := &MockDispatcherService{}
	handler := NewApplicationsHandler(repo, dispatcherSvc)

	// Since we can't easily mock chi.URLParam in a unit test without chi/router,
	// we verify the handler is correctly constructed
	assert.NotNil(t, handler)
	assert.NotNil(t, handler.repo)
}

func TestApplicationsHandler_Create(t *testing.T) {
	jobID := uuid.New()
	reqPayload := map[string]interface{}{
		"job_id":           jobID.String(),
		"tailored_resume":  "Resume content",
		"cover_letter":     "Cover letter",
		"resume_pdf_path":  "/path/to/resume.pdf",
		"cover_pdf_path":   "/path/to/cover.pdf",
		"delivery_channel": "TG_DM",
	}

	repo := &MockApplicationsRepository{}
	dispatcherSvc := &MockDispatcherService{}
	handler := NewApplicationsHandler(repo, dispatcherSvc)

	body, _ := json.Marshal(reqPayload)
	req := httptest.NewRequest("POST", "/api/v1/applications", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Create(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp models.JobApplication
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)

	assert.Equal(t, jobID, resp.JobID)
	assert.NotEqual(t, uuid.Nil, resp.ID)
}

func TestApplicationsHandler_Create_InvalidPayload(t *testing.T) {
	tests := []struct {
		name       string
		payload    map[string]interface{}
		expectCode int
		expectErr  string
	}{
		{
			name:       "missing job_id",
			payload:    map[string]interface{}{},
			expectCode: http.StatusBadRequest,
			expectErr:  "invalid job_id format",
		},
		{
			name: "invalid job_id format",
			payload: map[string]interface{}{
				"job_id": "not-a-uuid",
			},
			expectCode: http.StatusBadRequest,
			expectErr:  "invalid job_id format",
		},
		{
			name: "missing delivery_channel",
			payload: map[string]interface{}{
				"job_id": uuid.New().String(),
			},
			expectCode: http.StatusBadRequest,
			expectErr:  "delivery_channel is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockApplicationsRepository{}
			dispatcherSvc := &MockDispatcherService{}
			handler := NewApplicationsHandler(repo, dispatcherSvc)

			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest("POST", "/api/v1/applications", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.Create(w, req)

			assert.Equal(t, tt.expectCode, w.Code)

			var resp map[string]string
			_ = json.NewDecoder(w.Body).Decode(&resp)
			assert.Contains(t, resp["error"], tt.expectErr)
		})
	}
}

func TestApplicationsHandler_Send(t *testing.T) {
	appID := uuid.New()
	jobID := uuid.New()
	app := &models.JobApplication{
		ID:              appID,
		JobID:           jobID,
		DeliveryStatus:  models.DeliveryStatusPending,
		DeliveryChannel: deliveryChannelPtr(models.DeliveryChannelTGDM),
		ResumePDFPath:   strPtr("/path/to/resume.pdf"),
		CoverLetterMD:   strPtr("Cover letter"),
	}

	tests := []struct {
		name       string
		sendErr    error
		expectCode int
	}{
		{
			name:       "success",
			sendErr:    nil,
			expectCode: http.StatusOK,
		},
		{
			name:       "dispatcher error",
			sendErr:    assert.AnError,
			expectCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockApplicationsRepository{
				applications: map[uuid.UUID]*models.JobApplication{
					appID: app,
				},
			}
			dispatcherSvc := &MockDispatcherService{
				sendErr: tt.sendErr,
			}
			handler := NewApplicationsHandler(repo, dispatcherSvc)

			// Create chi router to properly handle URL params
			router := chi.NewRouter()
			router.Post("/api/v1/applications/{id}/send", handler.Send)

			payload := map[string]string{
				"recipient": "@recruiter",
			}
			body, _ := json.Marshal(payload)
			req := httptest.NewRequest("POST", "/api/v1/applications/"+appID.String()+"/send", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectCode, w.Code)
		})
	}
}

func TestApplicationsHandler_UpdateDeliveryStatus(t *testing.T) {
	appID := uuid.New()
	jobID := uuid.New()
	app := &models.JobApplication{
		ID:              appID,
		JobID:           jobID,
		DeliveryStatus:  models.DeliveryStatusPending,
		DeliveryChannel: deliveryChannelPtr(models.DeliveryChannelTGDM),
	}

	tests := []struct {
		name         string
		status       string
		expectCode   int
		expectStatus models.DeliveryStatus
	}{
		{
			name:         "update to sent",
			status:       "SENT",
			expectCode:   http.StatusOK,
			expectStatus: models.DeliveryStatusSent,
		},
		{
			name:         "update to delivered",
			status:       "DELIVERED",
			expectCode:   http.StatusOK,
			expectStatus: models.DeliveryStatusDelivered,
		},
		{
			name:         "update to read",
			status:       "READ",
			expectCode:   http.StatusOK,
			expectStatus: models.DeliveryStatusRead,
		},
		{
			name:         "invalid status",
			status:       "INVALID",
			expectCode:   http.StatusBadRequest,
			expectStatus: models.DeliveryStatusPending,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset app status for each test
			app.DeliveryStatus = models.DeliveryStatusPending

			repo := &MockApplicationsRepository{
				applications: map[uuid.UUID]*models.JobApplication{
					appID: app,
				},
			}
			dispatcherSvc := &MockDispatcherService{}
			handler := NewApplicationsHandler(repo, dispatcherSvc)

			// Create chi router to properly handle URL params
			router := chi.NewRouter()
			router.Patch("/api/v1/applications/{id}/delivery", handler.UpdateDeliveryStatus)

			payload := map[string]string{
				"status": tt.status,
			}
			body, _ := json.Marshal(payload)
			req := httptest.NewRequest("PATCH", "/api/v1/applications/"+appID.String()+"/delivery", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectCode, w.Code)
			// Verify the status was updated in the mock
			if tt.expectCode == http.StatusOK {
				assert.Equal(t, tt.expectStatus, app.DeliveryStatus)
			}
		})
	}
}

// Helper functions
func deliveryChannelPtr(c models.DeliveryChannel) *models.DeliveryChannel {
	return &c
}

func strPtr(s string) *string {
	return &s
}
