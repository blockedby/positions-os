package handlers

import (
	"encoding/json"
	"net/http"
)

type StatsHandler struct {
	repo StatsRepository
}

func NewStatsHandler(repo StatsRepository) *StatsHandler {
	return &StatsHandler{repo: repo}
}

func (h *StatsHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.repo.GetStats(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		_ = err // Client disconnected
	}
}
