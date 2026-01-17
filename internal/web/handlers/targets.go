package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/blockedby/positions-os/internal/repository"
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
	"TG_GROUP":   true,
	"TG_FORUM":   true,
	"HH_SEARCH":  true,
}

type TargetsHandler struct {
	repo TargetsRepository
}

func NewTargetsHandler(repo TargetsRepository) *TargetsHandler {
	return &TargetsHandler{
		repo: repo,
	}
}

// List returns the list of targets.
func (h *TargetsHandler) List(w http.ResponseWriter, r *http.Request) {
	targets, err := h.repo.List(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Ensure we return empty array, not null
	if targets == nil {
		targets = []repository.ScrapingTarget{}
	}

	respondJSON(w, http.StatusOK, targets)
}

// CreateTargetRequest represents the JSON body for creating a target
type CreateTargetRequest struct {
	Name     string                 `json:"name"`
	Type     string                 `json:"type"`
	URL      string                 `json:"url"`
	IsActive *bool                  `json:"is_active,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

func (h *TargetsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateTargetRequest

	// Try JSON first (for React frontend)
	if r.Header.Get("Content-Type") == "application/json" {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid json: "+err.Error())
			return
		}
	} else {
		// Fallback to form values (for HTMX)
		req.Name = r.FormValue("name")
		req.Type = r.FormValue("type")
		req.URL = r.FormValue("url")
	}

	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Type == "" {
		respondError(w, http.StatusBadRequest, "type is required")
		return
	}
	if !validTargetTypes[req.Type] {
		respondError(w, http.StatusBadRequest, "invalid type: "+req.Type)
		return
	}
	if req.URL == "" {
		respondError(w, http.StatusBadRequest, "url is required")
		return
	}

	metadata := req.Metadata
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	t := &repository.ScrapingTarget{
		Name:     req.Name,
		Type:     req.Type,
		URL:      req.URL,
		IsActive: isActive,
		Metadata: metadata,
	}

	if err := h.repo.Create(r.Context(), t); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Return JSON for API requests
	respondJSON(w, http.StatusCreated, t)
}

// GetByID returns a single target by ID
func (h *TargetsHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	t, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if t == nil {
		respondError(w, http.StatusNotFound, "target not found")
		return
	}

	respondJSON(w, http.StatusOK, t)
}

func (h *TargetsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UpdateTargetRequest represents the JSON body for updating a target
type UpdateTargetRequest struct {
	Name     *string                `json:"name,omitempty"`
	URL      *string                `json:"url,omitempty"`
	IsActive *bool                  `json:"is_active,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

func (h *TargetsHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	t, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if t == nil {
		respondError(w, http.StatusNotFound, "target not found")
		return
	}

	// Try JSON first (for React frontend)
	if r.Header.Get("Content-Type") == "application/json" {
		var req UpdateTargetRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid json: "+err.Error())
			return
		}

		if req.Name != nil {
			t.Name = *req.Name
		}
		if req.URL != nil {
			t.URL = *req.URL
		}
		if req.IsActive != nil {
			t.IsActive = *req.IsActive
		}
		if req.Metadata != nil {
			t.Metadata = req.Metadata
		}
	} else {
		// Fallback to form values (for HTMX)
		if name := r.FormValue("name"); name != "" {
			t.Name = name
		}
		if typ := r.FormValue("type"); typ != "" {
			t.Type = typ
		}
		if url := r.FormValue("url"); url != "" {
			t.URL = url
		}
		isActive := r.FormValue("is_active") == "true" || r.FormValue("is_active") == "on"
		t.IsActive = isActive
	}

	if err := h.repo.Update(r.Context(), t); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, t)
}

// respondJSON is a helper function to respond with JSON
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		_ = err // Client disconnected
	}
}

// respondError is a helper function to respond with a JSON error
func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
