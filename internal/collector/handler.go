package collector

import (
	"encoding/json"
	"net/http"
	"time"
)

// Handler handles HTTP requests for collector service
type Handler struct {
	manager *ScrapeManager
}

// NewHandler creates a new handler with the given manager
func NewHandler(manager *ScrapeManager) *Handler {
	return &Handler{manager: manager}
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
