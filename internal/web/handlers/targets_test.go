package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/blockedby/positions-os/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockTargetsRepository struct {
	mock.Mock
}

func (m *MockTargetsRepository) List(ctx context.Context) ([]repository.ScrapingTarget, error) {
	args := m.Called(ctx)
	return args.Get(0).([]repository.ScrapingTarget), args.Error(1)
}

func (m *MockTargetsRepository) Create(ctx context.Context, t *repository.ScrapingTarget) error {
	args := m.Called(ctx, t)
	return args.Error(0)
}

func (m *MockTargetsRepository) Update(ctx context.Context, t *repository.ScrapingTarget) error {
	args := m.Called(ctx, t)
	return args.Error(0)
}

func (m *MockTargetsRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockTargetsRepository) GetByID(ctx context.Context, id uuid.UUID) (*repository.ScrapingTarget, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.ScrapingTarget), args.Error(1)
}

func setupTargetsHandler(t *testing.T, repo TargetsRepository) *TargetsHandler {
	return NewTargetsHandler(repo)
}

func TestTargetsHandler_List(t *testing.T) {
	mockRepo := new(MockTargetsRepository)
	handler := setupTargetsHandler(t, mockRepo)

	targets := []repository.ScrapingTarget{
		{ID: uuid.New(), Name: "Go Jobs", Type: "TG_CHANNEL"},
		{ID: uuid.New(), Name: "Python Jobs", Type: "TG_CHANNEL"},
	}
	mockRepo.On("List", mock.Anything).Return(targets, nil)

	req := httptest.NewRequest("GET", "/targets", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var result []repository.ScrapingTarget
	err := json.NewDecoder(rec.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	mockRepo.AssertExpectations(t)
}

func TestTargetsHandler_Create(t *testing.T) {
	mockRepo := new(MockTargetsRepository)
	handler := setupTargetsHandler(t, mockRepo)

	mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(t *repository.ScrapingTarget) bool {
		return t.Name == "Go Jobs" && t.Type == "TG_CHANNEL"
	})).Return(nil)

	form := url.Values{}
	form.Add("name", "Go Jobs")
	form.Add("type", "TG_CHANNEL")
	form.Add("url", "@golang_jobs")

	req := httptest.NewRequest("POST", "/targets", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()
	handler.Create(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var result repository.ScrapingTarget
	err := json.NewDecoder(rec.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "Go Jobs", result.Name)

	mockRepo.AssertExpectations(t)
}

func TestTargetsHandler_Create_Validation(t *testing.T) {
	mockRepo := new(MockTargetsRepository)
	handler := setupTargetsHandler(t, mockRepo)

	tests := []struct {
		name    string
		form    url.Values
		wantErr string
	}{
		{"missing name", url.Values{"type": {"TG_CHANNEL"}, "url": {"@test"}}, "name is required"},
		{"missing type", url.Values{"name": {"Test"}, "url": {"@test"}}, "type is required"},
		{"invalid type", url.Values{"name": {"Test"}, "type": {"INVALID"}, "url": {"@test"}}, "invalid type"},
		{"missing url", url.Values{"name": {"Test"}, "type": {"TG_CHANNEL"}}, "url is required"},
		{"forum without topics", url.Values{"name": {"Test"}, "type": {"TG_FORUM"}, "url": {"@test"}}, "topic_ids required for TG_FORUM"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/targets", strings.NewReader(tt.form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rec := httptest.NewRecorder()

			handler.Create(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Contains(t, rec.Body.String(), tt.wantErr)
		})
	}
}

func TestTargetsHandler_Delete(t *testing.T) {
	mockRepo := new(MockTargetsRepository)
	handler := setupTargetsHandler(t, mockRepo)

	id := uuid.New()
	mockRepo.On("Delete", mock.Anything, id).Return(nil)

	r := chi.NewRouter()
	r.Delete("/targets/{id}", handler.Delete)

	req := httptest.NewRequest("DELETE", "/targets/"+id.String(), nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	mockRepo.AssertExpectations(t)
}

func TestTargetsHandler_Update(t *testing.T) {
	mockRepo := new(MockTargetsRepository)
	handler := setupTargetsHandler(t, mockRepo)

	id := uuid.New()
	target := &repository.ScrapingTarget{
		ID:       id,
		Name:     "Old Name",
		IsActive: true,
	}

	mockRepo.On("GetByID", mock.Anything, id).Return(target, nil)
	mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(t *repository.ScrapingTarget) bool {
		return t.ID == id && t.Name == "New Name" && !t.IsActive
	})).Return(nil)

	r := chi.NewRouter()
	r.Put("/targets/{id}", handler.Update)

	form := url.Values{}
	form.Add("name", "New Name")
	form.Add("type", "TG_CHANNEL")
	form.Add("url", "@new")
	form.Add("is_active", "false")

	req := httptest.NewRequest("PUT", "/targets/"+id.String(), strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	mockRepo.AssertExpectations(t)
}
