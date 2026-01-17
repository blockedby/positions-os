package web

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Config holds server configuration
type Config struct {
	Port      int
	StaticDir string // For PDF templates and other static assets
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
func NewServer(cfg *Config, _ interface{}, hub interface{}) *Server {
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
	// SPA static files serving
	if s.config.StaticDir != "" {
		distDir := s.config.StaticDir + "/dist"

		// Serve assets directory
		assetsFS := http.FileServer(http.Dir(distDir + "/assets"))
		s.router.Handle("/assets/*", http.StripPrefix("/assets/", assetsFS))

		// Also keep legacy static serving for PDF templates
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
	s.router.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"status":"ok","version":"dev"}`)); err != nil {
			_ = err // Client disconnected
		}
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
		Handler:           s.router,
		ReadHeaderTimeout: 10 * time.Second,
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
		GetByID(w http.ResponseWriter, r *http.Request)
		Delete(w http.ResponseWriter, r *http.Request)
		Update(w http.ResponseWriter, r *http.Request)
	}

	if h, ok := handler.(targetsHandler); ok {
		s.router.Route("/api/v1/targets", func(r chi.Router) {
			r.Get("/", h.List)
			r.Post("/", h.Create)
			r.Get("/{id}", h.GetByID)
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
		GetStatus(w http.ResponseWriter, r *http.Request)
		StartQR(w http.ResponseWriter, r *http.Request)
	}

	if h, ok := handler.(authHandler); ok {
		s.router.Route("/api/v1/auth", func(r chi.Router) {
			r.Get("/status", h.GetStatus)
			r.Post("/qr", h.StartQR)
		})
		// Also register under /telegram for backwards compatibility
		s.router.Get("/api/v1/telegram/status", h.GetStatus)
	}
}

// RegisterApplicationsHandler registers applications API handlers
func (s *Server) RegisterApplicationsHandler(handler interface{}) {
	type applicationsHandler interface {
		List(w http.ResponseWriter, r *http.Request)
		GetByID(w http.ResponseWriter, r *http.Request)
		Create(w http.ResponseWriter, r *http.Request)
		Send(w http.ResponseWriter, r *http.Request)
		UpdateDeliveryStatus(w http.ResponseWriter, r *http.Request)
	}

	if h, ok := handler.(applicationsHandler); ok {
		s.router.Route("/api/v1/applications", func(r chi.Router) {
			r.Get("/", h.List)
			r.Post("/", h.Create)
			r.Get("/{id}", h.GetByID)
			r.Post("/{id}/send", h.Send)
			r.Patch("/{id}/delivery", h.UpdateDeliveryStatus)
		})
	}
}

// RegisterDispatcherHandler registers dispatcher service status handler
func (s *Server) RegisterDispatcherHandler(handler interface{}) {
	type dispatcherHandler interface {
		Status(w http.ResponseWriter, r *http.Request)
	}

	if h, ok := handler.(dispatcherHandler); ok {
		s.router.Get("/api/v1/dispatcher/status", h.Status)
	}
}

// RegisterBrainHandler registers brain service handlers for job preparation.
// The brain service handles the AI-powered document generation workflow:
//   - PrepareJob: Initiates resume tailoring and cover letter generation
//   - GetDocuments: Returns generated document metadata for a job
//   - DownloadResume: Serves the generated PDF resume
func (s *Server) RegisterBrainHandler(handler interface{}) {
	// brainHandler defines the interface for brain service HTTP handlers.
	// Implementations should handle document generation and retrieval
	// for the job application workflow.
	type brainHandler interface {
		// PrepareJob initiates the document generation pipeline for a job.
		// POST /api/v1/jobs/{id}/prepare
		// Returns status and WebSocket channel for progress updates.
		PrepareJob(w http.ResponseWriter, r *http.Request)

		// GetDocuments returns metadata about generated documents for a job.
		// GET /api/v1/jobs/{id}/documents
		// Returns JSON with resume and cover letter paths/content.
		GetDocuments(w http.ResponseWriter, r *http.Request)

		// DownloadResume serves the generated PDF resume file.
		// GET /api/v1/jobs/{id}/documents/resume.pdf
		// Returns application/pdf content.
		DownloadResume(w http.ResponseWriter, r *http.Request)
	}

	if h, ok := handler.(brainHandler); ok {
		s.router.Post("/api/v1/jobs/{id}/prepare", h.PrepareJob)
		s.router.Get("/api/v1/jobs/{id}/documents", h.GetDocuments)
		s.router.Get("/api/v1/jobs/{id}/documents/resume.pdf", h.DownloadResume)
	}
}

// Router returns the underlying Chi router for external route mounting.
func (s *Server) Router() *chi.Mux {
	return s.router
}

// SetupSPAFallback adds SPA fallback routing. Call this after all API routes are registered.
func (s *Server) SetupSPAFallback() {
	if s.config.StaticDir == "" {
		return
	}

	distDir := filepath.Join(s.config.StaticDir, "dist")
	indexPath := filepath.Join(distDir, "index.html")

	// Check if index.html exists
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		return
	}

	// Serve index.html for SPA routes
	s.router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		// Only serve index.html for non-API, non-asset routes
		path := r.URL.Path
		if strings.HasPrefix(path, "/api/") ||
			strings.HasPrefix(path, "/assets/") ||
			strings.HasPrefix(path, "/static/") ||
			path == "/ws" ||
			path == "/health" {
			http.NotFound(w, r)
			return
		}

		// Serve index.html for SPA routes
		http.ServeFile(w, r, indexPath)
	})
}
