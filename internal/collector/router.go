package collector

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter creates a new chi router with all collector endpoints
func NewRouter(handler *Handler) http.Handler {
	r := chi.NewRouter()

	// middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	// health check
	r.Get("/health", handler.Health)

	// api v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// scraping endpoints
		r.Post("/scrape/telegram", handler.StartScrape)
		r.Delete("/scrape/current", handler.StopScrape)
		r.Get("/scrape/status", handler.Status)

		// targets endpoints (placeholder for now)
		r.Get("/targets", func(w http.ResponseWriter, r *http.Request) {
			respondJSON(w, http.StatusOK, []interface{}{})
		})
		r.Post("/targets", func(w http.ResponseWriter, r *http.Request) {
			respondError(w, http.StatusNotImplemented, "not implemented yet")
		})
	})

	return r
}
