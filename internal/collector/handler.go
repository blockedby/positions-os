package collector

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/blockedby/positions-os/internal/repository"
)

// Handler handles HTTP requests for collector service
type Handler struct {
	manager     *ScrapeManager
	targetsRepo *repository.TargetsRepository
}

// NewHandler creates a new handler with the given manager
func NewHandler(manager *ScrapeManager, targetsRepo *repository.TargetsRepository) *Handler {
	return &Handler{
		manager:     manager,
		targetsRepo: targetsRepo,
	}
}

// Health handles GET /health
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// StartScrape handles POST /api/v1/scrape/telegram
func (h *Handler) StartScrape(w http.ResponseWriter, r *http.Request) {
	var req ScrapeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}

	// validate request
	if err := req.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// convert to options
	opts := ScrapeOptions{
		Channel:  req.Channel,
		Limit:    req.Limit,
		Until:    req.UntilTime(),
		TopicIDs: req.TopicIDs,
	}
	if req.TargetID != nil {
		opts.TargetID = *req.TargetID
	}

	// start scraping
	job, err := h.manager.Start(r.Context(), opts)
	if err != nil {
		if err == ErrAlreadyRunning {
			respondError(w, http.StatusConflict, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, ScrapeResponse{
		ScrapeID:  job.ID,
		Status:    "running",
		StartedAt: job.StartedAt,
		Target: TargetInfo{
			ID:      job.TargetID,
			Channel: job.Options.Channel,
		},
	})
}

// StopScrape handles DELETE /api/v1/scrape/current
func (h *Handler) StopScrape(w http.ResponseWriter, r *http.Request) {
	h.manager.Stop()
	respondJSON(w, http.StatusOK, map[string]string{
		"message": "scrape job stopped",
	})
}

// Status handles GET /api/v1/scrape/status
func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	current := h.manager.Current()
	if current == nil {
		respondJSON(w, http.StatusOK, map[string]string{
			"status": "idle",
			// We need to access manager's client status.
			// But ScrapeManager wrapper might not expose it directly or we need to access handlers.manager.GetClientStatus()?
			// ScrapeManager has 'service', 'service' has 'client'.
			// Let's assume we can get it or just leave it for now if complex.
			// For now, let's keep it simple and skip this enhancement to avoid breaking encapsulation too much,
			// OR we can add GetTelegramStatus to ScrapeManager.
			// Let's assume we can't easily get it here without refactoring ScrapeManager.
			// BUT the frontend expects "telegram_status".
			// Let's skip it here and handle it separate or just let the frontend default to 'Wait...'.
			// Actually, let's add it. ScrapeManager -> Service -> Client -> Manager -> Status.
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":     "running",
		"scrape_id":  current.ID.String(),
		"started_at": current.StartedAt.Format(time.RFC3339),
		"channel":    current.Options.Channel,
	})
}

// ListTargets handles GET /api/v1/targets
func (h *Handler) ListTargets(w http.ResponseWriter, r *http.Request) {
	targets, err := h.targetsRepo.GetActive(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, targets)
}

// CreateTargetRequest represents request body for creating a target
type CreateTargetRequest struct {
	Name string `json:"name"`
	Type string `json:"type"`
	URL  string `json:"url"`
}

// CreateTarget handles POST /api/v1/targets
func (h *Handler) CreateTarget(w http.ResponseWriter, r *http.Request) {
	var req CreateTargetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid json")
		return
	}

	if req.Name == "" || req.URL == "" {
		respondError(w, http.StatusBadRequest, "name and url are required")
		return
	}

	if req.Type == "" {
		req.Type = "TG_CHANNEL"
	}

	target := &repository.ScrapingTarget{
		Name:     req.Name,
		Type:     req.Type,
		URL:      req.URL,
		IsActive: true,
		Metadata: map[string]interface{}{},
	}

	if err := h.targetsRepo.Create(r.Context(), target); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, target)
}

// ListForumTopics handles GET /api/v1/tools/telegram/topics
func (h *Handler) ListForumTopics(w http.ResponseWriter, r *http.Request) {
	channel := r.URL.Query().Get("channel")
	if channel == "" {
		respondError(w, http.StatusBadRequest, "channel query param is required")
		return
	}

	topics, err := h.manager.ListTopics(r.Context(), channel)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, topics)
}

// helper functions

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{
		"error": message,
	})
}
