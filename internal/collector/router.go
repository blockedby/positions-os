package collector

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// NewRouter creates a new chi router with all collector endpoints
func NewRouter(handler *Handler) http.Handler {
	r := chi.NewRouter()

	// middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	// basic cors
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "OPTIONS", "DELETE", "PUT"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type"},
	}))

	// health check
	r.Get("/health", handler.Health)

	// api v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// scraping endpoints
		r.Post("/scrape/telegram", handler.StartScrape)
		r.Delete("/scrape/current", handler.StopScrape)
		r.Get("/scrape/status", handler.Status)

		// targets endpoints
		r.Get("/targets", handler.ListTargets)
		r.Post("/targets", handler.CreateTarget)

		// tools endpoints
		r.Get("/tools/telegram/topics", handler.ListForumTopics)
	})

	return r
}
