package web

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Config holds server configuration
type Config struct {
	Port         int
	StaticDir    string
	TemplatesDir string
}

// Server represents the HTTP server
type Server struct {
	router     *chi.Mux
	httpServer *http.Server
	config     *Config
	listener   net.Listener
	hub        *Hub // WebSocket Hub
}

// NewServer creates a new HTTP server
func NewServer(cfg *Config, repo interface{}, hub interface{}) *Server {
	router := chi.NewRouter()

	srv := &Server{
		router: router,
		config: cfg,
	}

	if h, ok := hub.(*Hub); ok {
		srv.hub = h
	}

	srv.setupMiddleware()
	srv.setupRoutes()

	return srv
}

func (s *Server) setupMiddleware() {
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(middleware.Timeout(30 * time.Second))
	s.router.Use(middleware.Compress(5))
}

func (s *Server) setupRoutes() {
	// Static files
	if s.config.StaticDir != "" {
		fileServer := http.FileServer(http.Dir(s.config.StaticDir))
		s.router.Handle("/static/*", http.StripPrefix("/static/", fileServer))
	}

	// WebSocket
	if s.hub != nil {
		s.router.Get("/ws", func(w http.ResponseWriter, r *http.Request) {
			ServeWs(s.hub, w, r)
		})
	}

	// Health endpoint
	s.router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","version":"dev"}`))
	})
}

// Start starts the HTTP server
func (s *Server) Start() error {
	// Create listener
	addr := fmt.Sprintf(":%d", s.config.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	s.listener = listener

	s.httpServer = &http.Server{
		Handler: s.router,
	}

	return s.httpServer.Serve(listener)
}

// Stop gracefully stops the server
func (s *Server) Stop(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// BaseURL returns the server's base URL
func (s *Server) BaseURL() string {
	if s.listener != nil {
		return fmt.Sprintf("http://%s", s.listener.Addr().String())
	}
	return fmt.Sprintf("http://localhost:%d", s.config.Port)
}

// RegisterPagesHandler registers page handlers
func (s *Server) RegisterPagesHandler(handler interface{}) {
	type pagesHandler interface {
		Dashboard(w http.ResponseWriter, r *http.Request)
		Jobs(w http.ResponseWriter, r *http.Request)
		Settings(w http.ResponseWriter, r *http.Request)
		JobDetail(w http.ResponseWriter, r *http.Request)
		JobRow(w http.ResponseWriter, r *http.Request)
		StatsCards(w http.ResponseWriter, r *http.Request)
		RecentJobs(w http.ResponseWriter, r *http.Request)
	}

	if h, ok := handler.(pagesHandler); ok {
		s.router.Get("/", h.Dashboard)
		s.router.Get("/jobs", h.Jobs)
		s.router.Get("/jobs/{id}", h.JobDetail)
		s.router.Get("/settings", h.Settings)

		s.router.Get("/partials/jobs/row/{id}", h.JobRow)
		s.router.Get("/partials/stats-cards", h.StatsCards)
		s.router.Get("/partials/recent-jobs", h.RecentJobs)
	}
}

// RegisterJobsHandler registers jobs API handlers
func (s *Server) RegisterJobsHandler(handler interface{}) {
	type jobsHandler interface {
		List(w http.ResponseWriter, r *http.Request)
		GetByID(w http.ResponseWriter, r *http.Request)
		UpdateStatus(w http.ResponseWriter, r *http.Request)
	}

	if h, ok := handler.(jobsHandler); ok {
		s.router.Route("/api/v1/jobs", func(r chi.Router) {
			r.Get("/", h.List)
			r.Get("/{id}", h.GetByID)
			r.Patch("/{id}/status", h.UpdateStatus)
		})
	}
}

// RegisterTargetsHandler registers targets API handlers
func (s *Server) RegisterTargetsHandler(handler interface{}) {
	type targetsHandler interface {
		List(w http.ResponseWriter, r *http.Request)
		Create(w http.ResponseWriter, r *http.Request)
		Delete(w http.ResponseWriter, r *http.Request)
		Update(w http.ResponseWriter, r *http.Request)
	}

	if h, ok := handler.(targetsHandler); ok {
		s.router.Route("/api/v1/targets", func(r chi.Router) {
			r.Get("/", h.List)
			r.Post("/", h.Create)
			r.Delete("/{id}", h.Delete)
			r.Put("/{id}", h.Update)
		})
	}
}

// RegisterStatsHandler registers stats API handlers
func (s *Server) RegisterStatsHandler(handler interface{}) {
	type statsHandler interface {
		GetStats(w http.ResponseWriter, r *http.Request)
	}

	if h, ok := handler.(statsHandler); ok {
		s.router.Get("/api/v1/stats", h.GetStats)
	}
}

// RegisterCollectorHandler registers collector API handlers
func (s *Server) RegisterCollectorHandler(handler interface{}) {
	type collectorHandler interface {
		StartScrape(w http.ResponseWriter, r *http.Request)
		StopScrape(w http.ResponseWriter, r *http.Request)
		Status(w http.ResponseWriter, r *http.Request)
	}

	if h, ok := handler.(collectorHandler); ok {
		s.router.Route("/api/v1/scrape", func(r chi.Router) {
			r.Post("/telegram", h.StartScrape)
			r.Delete("/current", h.StopScrape)
			r.Get("/status", h.Status)
		})
	}
}

// RegisterAuthHandler registers auth API handlers
func (s *Server) RegisterAuthHandler(handler interface{}) {
	type authHandler interface {
		StartQR(w http.ResponseWriter, r *http.Request)
	}

	if h, ok := handler.(authHandler); ok {
		s.router.Route("/api/v1/auth", func(r chi.Router) {
			r.Post("/qr", h.StartQR)
		})
	}
}
