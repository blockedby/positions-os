package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/blockedby/positions-os/internal/models"
	"github.com/blockedby/positions-os/internal/repository"
	"github.com/blockedby/positions-os/internal/telegram"
)

// Mock implementations for testing

type mockJobsRepo struct {
	jobs  []*repository.Job
	total int
}

func (m *mockJobsRepo) List(ctx context.Context, filter repository.JobFilter) ([]*repository.Job, int, error) {
	return m.jobs, m.total, nil
}

func (m *mockJobsRepo) GetByID(ctx context.Context, id uuid.UUID) (*repository.Job, error) {
	for _, j := range m.jobs {
		if j.ID == id {
			return j, nil
		}
	}
	return nil, nil
}

func (m *mockJobsRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	return nil
}

type mockTargetsRepo struct {
	targets []repository.ScrapingTarget
}

func (m *mockTargetsRepo) List(ctx context.Context) ([]repository.ScrapingTarget, error) {
	return m.targets, nil
}

func (m *mockTargetsRepo) Create(ctx context.Context, t *repository.ScrapingTarget) error {
	t.ID = uuid.New()
	return nil
}

func (m *mockTargetsRepo) GetByID(ctx context.Context, id uuid.UUID) (*repository.ScrapingTarget, error) {
	for _, t := range m.targets {
		if t.ID == id {
			return &t, nil
		}
	}
	return nil, nil
}

func (m *mockTargetsRepo) Update(ctx context.Context, t *repository.ScrapingTarget) error {
	return nil
}

func (m *mockTargetsRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

type mockStatsRepo struct {
	stats *repository.DashboardStats
}

func (m *mockStatsRepo) GetStats(ctx context.Context) (*repository.DashboardStats, error) {
	return m.stats, nil
}

type mockApplicationsRepo struct{}

func (m *mockApplicationsRepo) Create(ctx context.Context, app *models.JobApplication) error {
	return nil
}

func (m *mockApplicationsRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.JobApplication, error) {
	return nil, nil
}

func (m *mockApplicationsRepo) GetByJobID(ctx context.Context, jobID uuid.UUID) ([]*models.JobApplication, error) {
	return nil, nil
}

func (m *mockApplicationsRepo) UpdateDeliveryStatus(ctx context.Context, id uuid.UUID, status models.DeliveryStatus) error {
	return nil
}

type mockTelegramClient struct {
	status       telegram.Status
	qrInProgress bool
}

func (m *mockTelegramClient) GetStatus() telegram.Status {
	return m.status
}

func (m *mockTelegramClient) IsQRInProgress() bool {
	return m.qrInProgress
}

func (m *mockTelegramClient) StartQR(ctx context.Context, onURL func(string)) error {
	return nil
}

func TestNewServer(t *testing.T) {
	cfg := &Config{
		Port:        8080,
		Title:       "Test API",
		Description: "Test API Description",
		Version:     "1.0.0",
	}

	deps := &Dependencies{
		JobsRepo:         &mockJobsRepo{},
		TargetsRepo:      &mockTargetsRepo{},
		StatsRepo:        &mockStatsRepo{stats: &repository.DashboardStats{}},
		ApplicationsRepo: &mockApplicationsRepo{},
		TelegramClient:   &mockTelegramClient{status: telegram.StatusReady},
	}

	srv := NewServer(cfg, deps)
	if srv == nil {
		t.Fatal("expected server to be created")
	}
	if srv.fuego == nil {
		t.Fatal("expected fuego server to be initialized")
	}
}

func TestHealthEndpoint(t *testing.T) {
	cfg := &Config{
		Port:        8080,
		Title:       "Test API",
		Description: "Test",
		Version:     "1.0.0",
	}

	deps := &Dependencies{
		JobsRepo:         &mockJobsRepo{},
		TargetsRepo:      &mockTargetsRepo{},
		StatsRepo:        &mockStatsRepo{stats: &repository.DashboardStats{}},
		ApplicationsRepo: &mockApplicationsRepo{},
		TelegramClient:   &mockTelegramClient{status: telegram.StatusReady},
	}

	srv := NewServer(cfg, deps)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	srv.fuego.Mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("expected status 'ok', got '%s'", resp.Status)
	}
}

func TestListJobsEndpoint(t *testing.T) {
	jobID := uuid.New()
	targetID := uuid.New()

	cfg := &Config{
		Port:        8080,
		Title:       "Test API",
		Description: "Test",
		Version:     "1.0.0",
	}

	deps := &Dependencies{
		JobsRepo: &mockJobsRepo{
			jobs: []*repository.Job{
				{
					ID:             jobID,
					TargetID:       targetID,
					ExternalID:     "ext-1",
					Status:         "ANALYZED",
					StructuredData: map[string]interface{}{"title": "Go Developer"},
				},
			},
			total: 1,
		},
		TargetsRepo:      &mockTargetsRepo{},
		StatsRepo:        &mockStatsRepo{stats: &repository.DashboardStats{}},
		ApplicationsRepo: &mockApplicationsRepo{},
		TelegramClient:   &mockTelegramClient{status: telegram.StatusReady},
	}

	srv := NewServer(cfg, deps)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/jobs/", nil)
	w := httptest.NewRecorder()

	srv.fuego.Mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp JobsListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Total != 1 {
		t.Errorf("expected total 1, got %d", resp.Total)
	}

	if len(resp.Jobs) != 1 {
		t.Errorf("expected 1 job, got %d", len(resp.Jobs))
	}
}

func TestStatsEndpoint(t *testing.T) {
	cfg := &Config{
		Port:        8080,
		Title:       "Test API",
		Description: "Test",
		Version:     "1.0.0",
	}

	deps := &Dependencies{
		JobsRepo:    &mockJobsRepo{},
		TargetsRepo: &mockTargetsRepo{},
		StatsRepo: &mockStatsRepo{
			stats: &repository.DashboardStats{
				TotalJobs:      100,
				AnalyzedJobs:   50,
				InterestedJobs: 10,
				RejectedJobs:   20,
				TodayJobs:      5,
				ActiveTargets:  3,
			},
		},
		ApplicationsRepo: &mockApplicationsRepo{},
		TelegramClient:   &mockTelegramClient{status: telegram.StatusReady},
	}

	srv := NewServer(cfg, deps)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil)
	w := httptest.NewRecorder()

	srv.fuego.Mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp StatsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.TotalJobs != 100 {
		t.Errorf("expected TotalJobs 100, got %d", resp.TotalJobs)
	}

	if resp.ActiveTargets != 3 {
		t.Errorf("expected ActiveTargets 3, got %d", resp.ActiveTargets)
	}
}

func TestAuthStatusEndpoint(t *testing.T) {
	cfg := &Config{
		Port:        8080,
		Title:       "Test API",
		Description: "Test",
		Version:     "1.0.0",
	}

	deps := &Dependencies{
		JobsRepo:         &mockJobsRepo{},
		TargetsRepo:      &mockTargetsRepo{},
		StatsRepo:        &mockStatsRepo{stats: &repository.DashboardStats{}},
		ApplicationsRepo: &mockApplicationsRepo{},
		TelegramClient: &mockTelegramClient{
			status:       telegram.StatusReady,
			qrInProgress: false,
		},
	}

	srv := NewServer(cfg, deps)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/status", nil)
	w := httptest.NewRecorder()

	srv.fuego.Mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp AuthStatusResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.IsReady {
		t.Error("expected IsReady to be true")
	}

	if resp.QRInProgress {
		t.Error("expected QRInProgress to be false")
	}
}
