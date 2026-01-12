package handlers

import (
	"context"
	"net/http"

	"github.com/blockedby/positions-os/internal/repository"
	"github.com/blockedby/positions-os/internal/web"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type TargetsRepository interface {
	List(ctx context.Context) ([]repository.ScrapingTarget, error)
	Create(ctx context.Context, t *repository.ScrapingTarget) error
	Update(ctx context.Context, t *repository.ScrapingTarget) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*repository.ScrapingTarget, error)
}

var validTargetTypes = map[string]bool{
	"TG_CHANNEL": true,
	"TG_FORUM":   true,
	"HH_SEARCH":  true,
}

type TargetsHandler struct {
	repo      TargetsRepository
	templates *web.TemplateEngine
}

func NewTargetsHandler(repo TargetsRepository, templates *web.TemplateEngine) *TargetsHandler {
	return &TargetsHandler{
		repo:      repo,
		templates: templates,
	}
}

// List returns the list of targets.
// If HTMX request, could return rows? Or typically API returns JSON.
// Let's support both or stick to one. The plan says API routes.
// But for UI we need HTML.
// Let's implement partial rendering if HX-Request is generic, or specific endpoints.
func (h *TargetsHandler) List(w http.ResponseWriter, r *http.Request) {
	targets, err := h.repo.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		data := map[string]interface{}{
			"Targets": targets,
		}
		// We might need a specific template for just the rows
		h.templates.RenderContent(w, "targets-list", data)
		return
	}

	// JSON fallback
	// ... implementation ...
}

func (h *TargetsHandler) Create(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	typ := r.FormValue("type")
	url := r.FormValue("url")

	if name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if typ == "" {
		http.Error(w, "type is required", http.StatusBadRequest)
		return
	}
	if !validTargetTypes[typ] {
		http.Error(w, "invalid type", http.StatusBadRequest)
		return
	}
	if url == "" {
		http.Error(w, "url is required", http.StatusBadRequest)
		return
	}

	metadata := make(map[string]interface{})
	if typ == "TG_FORUM" {
		topics := r.FormValue("topic_ids")
		if topics == "" {
			http.Error(w, "topic_ids required for TG_FORUM", http.StatusBadRequest)
			return
		}
		metadata["topic_ids"] = topics
	}

	t := &repository.ScrapingTarget{
		Name:     name,
		Type:     typ,
		URL:      url,
		IsActive: true,
		Metadata: metadata,
	}

	if err := h.repo.Create(r.Context(), t); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the new row HTML
	if r.Header.Get("HX-Request") == "true" {
		h.templates.RenderPartial(w, "target-row", map[string]interface{}{"Target": t})
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *TargetsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK) // Empty response removes element in HTMX if swap is outerHTML
}

func (h *TargetsHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	t, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if t == nil {
		http.NotFound(w, r)
		return
	}

	// Update fields
	t.Name = r.FormValue("name")
	t.Type = r.FormValue("type")
	t.URL = r.FormValue("url")

	isActive := r.FormValue("is_active") == "true" || r.FormValue("is_active") == "on"
	t.IsActive = isActive

	if err := h.repo.Update(r.Context(), t); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		h.templates.RenderPartial(w, "target-row", map[string]interface{}{"Target": t})
		return
	}

	w.WriteHeader(http.StatusOK)
}
