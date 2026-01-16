package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"
)

// Server represents the Fuego API server.
type Server struct {
	fuego *fuego.Server
	deps  *Dependencies
	port  int
}

// Dependencies contains all service dependencies.
type Dependencies struct {
	JobsRepo          JobsRepository
	TargetsRepo       TargetsRepository
	StatsRepo         StatsRepository
	ApplicationsRepo  ApplicationsRepository
	TelegramClient    TelegramClient
	CollectorService  CollectorService
	DispatcherService DispatcherService
	Hub               HubBroadcaster
}

// Config holds API server configuration.
type Config struct {
	Port        int
	Title       string
	Description string
	Version     string
}

// NewServer creates a new Fuego API server.
func NewServer(cfg *Config, deps *Dependencies) *Server {
	s := fuego.NewServer(
		fuego.WithAddr(fmt.Sprintf(":%d", cfg.Port)),
		fuego.WithEngineOptions(
			fuego.WithOpenAPIConfig(fuego.OpenAPIConfig{
				PrettyFormatJSON: true,
				JSONFilePath:     "openapi.json",
				SwaggerURL:       "/docs",
				SpecURL:          "/openapi.json",
				UIHandler: func(specURL string) http.Handler {
					return ScalarHandler(specURL, cfg.Title, cfg.Description)
				},
			}),
		),
	)

	// Set OpenAPI info
	s.OpenAPI.Description().Info.Title = cfg.Title
	s.OpenAPI.Description().Info.Description = cfg.Description
	s.OpenAPI.Description().Info.Version = cfg.Version

	// Add Chi middleware (Fuego is net/http compatible)
	fuego.Use(s, middleware.RequestID)
	fuego.Use(s, middleware.RealIP)
	fuego.Use(s, middleware.Logger)
	fuego.Use(s, middleware.Recoverer)

	srv := &Server{
		fuego: s,
		deps:  deps,
		port:  cfg.Port,
	}

	srv.registerRoutes()

	return srv
}

func (s *Server) registerRoutes() {
	// Health check
	fuego.Get(s.fuego, "/health", s.healthCheck,
		option.Summary("Health Check"),
		option.Description("Returns the health status of the API"),
		option.Tags("System"),
	)

	// Jobs API
	jobsGroup := fuego.Group(s.fuego, "/api/v1/jobs",
		option.Tags("Jobs"),
	)

	fuego.Get(jobsGroup, "/", s.listJobs,
		option.Summary("List Jobs"),
		option.Description("Returns a paginated list of jobs with optional filtering"),
		option.Query("status", "Filter by job status (RAW, ANALYZED, INTERESTED, REJECTED, TAILORED, SENT, RESPONDED)"),
		option.Query("tech", "Filter by technology"),
		option.Query("q", "Full-text search query"),
		option.Query("salary_min", "Minimum salary filter"),
		option.Query("salary_max", "Maximum salary filter"),
		option.Query("page", "Page number (1-indexed, default: 1)"),
		option.Query("limit", "Items per page (default: 50, max: 100)"),
	)

	fuego.Get(jobsGroup, "/{id}", s.getJob,
		option.Summary("Get Job"),
		option.Description("Returns a single job by ID"),
	)

	fuego.Patch(jobsGroup, "/{id}/status", s.updateJobStatus,
		option.Summary("Update Job Status"),
		option.Description("Updates the status of a job"),
	)

	fuego.Delete(jobsGroup, "/", s.bulkDeleteJobs,
		option.Summary("Bulk delete jobs"),
		option.Description("Delete multiple jobs by their IDs (max 100)"),
	)

	// Targets API
	targetsGroup := fuego.Group(s.fuego, "/api/v1/targets",
		option.Tags("Targets"),
	)

	fuego.Get(targetsGroup, "/", s.listTargets,
		option.Summary("List Targets"),
		option.Description("Returns all scraping targets"),
	)

	fuego.Post(targetsGroup, "/", s.createTarget,
		option.Summary("Create Target"),
		option.Description("Creates a new scraping target"),
	)

	fuego.Get(targetsGroup, "/{id}", s.getTarget,
		option.Summary("Get Target"),
		option.Description("Returns a single target by ID"),
	)

	fuego.Put(targetsGroup, "/{id}", s.updateTarget,
		option.Summary("Update Target"),
		option.Description("Updates an existing target"),
	)

	fuego.Delete(targetsGroup, "/{id}", s.deleteTarget,
		option.Summary("Delete Target"),
		option.Description("Deletes a target"),
	)

	// Stats API
	fuego.Get(s.fuego, "/api/v1/stats", s.getStats,
		option.Summary("Get Statistics"),
		option.Description("Returns job statistics and analytics"),
		option.Tags("Analytics"),
	)

	// Scraping API
	scrapeGroup := fuego.Group(s.fuego, "/api/v1/scrape",
		option.Tags("Scraping"),
	)

	fuego.Post(scrapeGroup, "/telegram", s.startScrape,
		option.Summary("Start Telegram Scrape"),
		option.Description("Starts scraping a Telegram channel or forum"),
	)

	fuego.Delete(scrapeGroup, "/current", s.stopScrape,
		option.Summary("Stop Current Scrape"),
		option.Description("Stops the currently running scrape"),
	)

	fuego.Get(scrapeGroup, "/status", s.getScrapeStatus,
		option.Summary("Get Scrape Status"),
		option.Description("Returns the status of the current scraping operation"),
	)

	// Auth API
	authGroup := fuego.Group(s.fuego, "/api/v1/auth",
		option.Tags("Authentication"),
	)

	fuego.Get(authGroup, "/status", s.getAuthStatus,
		option.Summary("Get Auth Status"),
		option.Description("Returns Telegram authentication status"),
	)

	fuego.Post(authGroup, "/qr", s.startQRAuth,
		option.Summary("Start QR Auth"),
		option.Description("Initiates Telegram QR code login flow"),
	)

	// Backwards compatibility
	fuego.Get(s.fuego, "/api/v1/telegram/status", s.getAuthStatus,
		option.Summary("Get Telegram Status (Legacy)"),
		option.Description("Legacy endpoint - use /api/v1/auth/status instead"),
		option.Tags("Authentication"),
		option.Deprecated(),
	)

	// Applications API
	appsGroup := fuego.Group(s.fuego, "/api/v1/applications",
		option.Tags("Applications"),
	)

	fuego.Get(appsGroup, "/", s.listApplications,
		option.Summary("List Applications"),
		option.Description("Returns applications filtered by job ID"),
		option.Query("job_id", "Filter by job ID (required)"),
	)

	fuego.Post(appsGroup, "/", s.createApplication,
		option.Summary("Create Application"),
		option.Description("Creates a new job application"),
	)

	fuego.Get(appsGroup, "/{id}", s.getApplication,
		option.Summary("Get Application"),
		option.Description("Returns a single application by ID"),
	)

	fuego.Post(appsGroup, "/{id}/send", s.sendApplication,
		option.Summary("Send Application"),
		option.Description("Sends an application via the configured delivery channel"),
	)

	fuego.Patch(appsGroup, "/{id}/delivery", s.updateDeliveryStatus,
		option.Summary("Update Delivery Status"),
		option.Description("Updates the delivery status of an application"),
	)
}

// Start starts the API server.
func (s *Server) Start() error {
	return s.fuego.Run()
}

// Stop gracefully stops the server.
func (s *Server) Stop(ctx context.Context) error {
	// Fuego uses net/http server internally
	return nil
}

// Mux returns the underlying ServeMux for mounting additional routes.
func (s *Server) Mux() *http.ServeMux {
	return s.fuego.Mux
}

// MountDocsOn mounts the OpenAPI documentation routes (/docs, /openapi.json)
// on a Chi router. This allows using Fuego's OpenAPI generation with an
// existing router.
func (s *Server) MountDocsOn(r interface {
	Get(pattern string, handlerFn http.HandlerFunc)
}, title, description string) {
	// Serve Scalar UI directly at /docs
	scalarHandler := ScalarHandler("/openapi.json", title, description)
	r.Get("/docs", func(w http.ResponseWriter, req *http.Request) {
		scalarHandler.ServeHTTP(w, req)
	})

	// Serve OpenAPI spec from Fuego's generated schema
	r.Get("/openapi.json", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		spec := s.fuego.OpenAPI.Description()
		if err := json.NewEncoder(w).Encode(spec); err != nil {
			http.Error(w, "Failed to encode OpenAPI spec", http.StatusInternalServerError)
		}
	})
}
