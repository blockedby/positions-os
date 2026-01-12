package handlers

import (
	"net/http"

	"github.com/blockedby/positions-os/internal/repository"
	"github.com/blockedby/positions-os/internal/web"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// PagesHandler handles HTML page requests
type PagesHandler struct {
	templates *web.TemplateEngine
	jobsRepo  JobsRepository
	statsRepo StatsRepository
}

// NewPagesHandler creates a new pages handler
func NewPagesHandler(templates *web.TemplateEngine, jobsRepo JobsRepository, statsRepo StatsRepository) *PagesHandler {
	return &PagesHandler{
		templates: templates,
		jobsRepo:  jobsRepo,
		statsRepo: statsRepo,
	}
}

// Dashboard renders the dashboard page
func (h *PagesHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	stats, err := h.statsRepo.GetStats(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":      "Dashboard",
		"ActivePage": "dashboard",
		"Stats":      stats,
	}

	if r.Header.Get("HX-Request") == "true" {
		if err := h.templates.RenderContent(w, "dashboard", data); err != nil {
			http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	if err := h.templates.Render(w, "dashboard", data); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// Jobs renders the jobs list page
func (h *PagesHandler) Jobs(w http.ResponseWriter, r *http.Request) {
	// Parse basic filters from query params
	filter := repository.JobFilter{
		Status: r.URL.Query().Get("status"),
		Query:  r.URL.Query().Get("q"),
	}
	// ... (add other filters as needed)

	jobs, total, err := h.jobsRepo.List(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":      "Jobs",
		"ActivePage": "jobs",
		"Jobs":       jobs,
		"Total":      total,
	}

	if r.Header.Get("HX-Request") == "true" {
		if err := h.templates.RenderContent(w, "jobs", data); err != nil {
			http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	if err := h.templates.Render(w, "jobs", data); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// Settings renders the settings page
func (h *PagesHandler) Settings(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title":      "Settings",
		"ActivePage": "settings",
	}

	if r.Header.Get("HX-Request") == "true" {
		if err := h.templates.RenderContent(w, "settings", data); err != nil {
			http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	if err := h.templates.Render(w, "settings", data); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// JobDetail renders the job detail panel
func (h *PagesHandler) JobDetail(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	job, err := h.jobsRepo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// For HTMX request (from side panel), render just the content
	if r.Header.Get("HX-Request") == "true" {
		h.templates.RenderContent(w, "job-panel", map[string]interface{}{"Job": job})
		return
	}

	// For full page load, maybe redirect to jobs list or render full page?
	// For now, let's redirect to /jobs (or we could render jobs page with open panel)
	http.Redirect(w, r, "/jobs", http.StatusSeeOther)
}

// JobRow renders a single job row for HTMX updates
func (h *PagesHandler) JobRow(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	job, err := h.jobsRepo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if job == nil {
		http.NotFound(w, r)
		return
	}

	// Always render as partial content
	h.templates.RenderContent(w, "job-row", map[string]interface{}{"Job": job})
}

// StatsCards renders the statistics cards for the dashboard
func (h *PagesHandler) StatsCards(w http.ResponseWriter, r *http.Request) {
	stats, err := h.statsRepo.GetStats(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.templates.RenderContent(w, "stats-cards", map[string]interface{}{"Stats": stats})
}

// RecentJobs renders the most recent jobs for the dashboard
func (h *PagesHandler) RecentJobs(w http.ResponseWriter, r *http.Request) {
	filter := repository.JobFilter{
		Limit: 5,
	}

	jobs, _, err := h.jobsRepo.List(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.templates.RenderContent(w, "recent-jobs", map[string]interface{}{"Jobs": jobs})
}
