package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/blockedby/positions-os/internal/repository"
	"github.com/blockedby/positions-os/internal/web"
)

// JobsHandler handles job-related requests
type JobsHandler struct {
	repo JobsRepository
	hub  *web.Hub
}

func NewJobsHandler(repo JobsRepository, hub *web.Hub) *JobsHandler {
	return &JobsHandler{
		repo: repo,
		hub:  hub,
	}
}

// ... List ... GetByID ...

func (h *JobsHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var payload struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate status
	dummy := repository.Job{Status: payload.Status}
	if !dummy.IsValidStatus() {
		http.Error(w, "Invalid status", http.StatusBadRequest)
		return
	}

	if err := h.repo.UpdateStatus(r.Context(), id, payload.Status); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Notify clients
	if h.hub != nil {
		h.hub.Broadcast(web.JobUpdatedEvent(id, payload.Status))
	}

	w.WriteHeader(http.StatusOK)
}

func (h *JobsHandler) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 50
	}

	salaryMin, _ := strconv.Atoi(r.URL.Query().Get("salary_min"))
	salaryMax, _ := strconv.Atoi(r.URL.Query().Get("salary_max"))

	filter := repository.JobFilter{
		Status:    r.URL.Query().Get("status"),
		Tech:      r.URL.Query().Get("tech"),
		Query:     r.URL.Query().Get("q"),
		SalaryMin: salaryMin,
		SalaryMax: salaryMax,
		Page:      page,
		Limit:     limit,
	}

	jobs, total, err := h.repo.List(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := struct {
		Jobs  []*repository.Job `json:"jobs"`
		Total int               `json:"total"`
		Page  int               `json:"page"`
		Limit int               `json:"limit"`
	}{
		Jobs:  jobs,
		Total: total,
		Page:  filter.Page,
		Limit: filter.Limit,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *JobsHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	job, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if job == nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}
