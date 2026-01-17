// Package api provides HTTP handlers for the REST API.
package api

import (
	"context"
	"errors"
	"strconv"

	"github.com/blockedby/positions-os/internal/dispatcher"
	"github.com/blockedby/positions-os/internal/models"
	"github.com/blockedby/positions-os/internal/repository"
	"github.com/blockedby/positions-os/internal/telegram"
	"github.com/go-fuego/fuego"
	"github.com/google/uuid"
)

// ============================================================================
// Health
// ============================================================================

func (s *Server) healthCheck(c fuego.ContextNoBody) (HealthResponse, error) {
	return HealthResponse{
		Status:  "ok",
		Version: "dev",
	}, nil
}

// ============================================================================
// Jobs Handlers
// ============================================================================

func (s *Server) listJobs(c fuego.ContextNoBody) (JobsListResponse, error) {
	// Parse query parameters manually
	page := parseIntWithDefault(c.QueryParam("page"), 1)
	limit := parseIntWithDefault(c.QueryParam("limit"), 50)
	status := c.QueryParam("status")
	tech := c.QueryParam("tech")
	query := c.QueryParam("q")
	salaryMin := parseIntWithDefault(c.QueryParam("salary_min"), 0)
	salaryMax := parseIntWithDefault(c.QueryParam("salary_max"), 0)

	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	filter := repository.JobFilter{
		Status:    status,
		Tech:      tech,
		Query:     query,
		SalaryMin: salaryMin,
		SalaryMax: salaryMax,
		Page:      page,
		Limit:     limit,
	}

	jobs, total, err := s.deps.JobsRepo.List(c.Context(), filter)
	if err != nil {
		return JobsListResponse{}, fuego.InternalServerError{Detail: err.Error()}
	}

	// Calculate total pages
	pages := (total + limit - 1) / limit
	if pages < 1 {
		pages = 1
	}

	return JobsListResponse{
		Jobs:  JobsFromRepo(jobs),
		Total: total,
		Page:  page,
		Limit: limit,
		Pages: pages,
	}, nil
}

func (s *Server) getJob(c fuego.ContextNoBody) (JobResponse, error) {
	idStr := c.PathParam("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return JobResponse{}, fuego.BadRequestError{Detail: "Invalid job ID"}
	}

	job, err := s.deps.JobsRepo.GetByID(c.Context(), id)
	if err != nil {
		return JobResponse{}, fuego.InternalServerError{Detail: err.Error()}
	}
	if job == nil {
		return JobResponse{}, fuego.NotFoundError{Detail: "Job not found"}
	}

	return JobFromRepo(job), nil
}

func (s *Server) updateJobStatus(c fuego.ContextWithBody[JobUpdateStatusRequest]) (any, error) {
	idStr := c.PathParam("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fuego.BadRequestError{Detail: "Invalid job ID"}
	}

	body, err := c.Body()
	if err != nil {
		return nil, fuego.BadRequestError{Detail: err.Error()}
	}

	// Validate status
	validStatuses := map[string]bool{
		"RAW": true, "ANALYZED": true, "INTERESTED": true,
		"REJECTED": true, "TAILORED": true, "SENT": true, "RESPONDED": true,
	}
	if !validStatuses[body.Status] {
		return nil, fuego.BadRequestError{Detail: "Invalid status"}
	}

	if err := s.deps.JobsRepo.UpdateStatus(c.Context(), id, body.Status); err != nil {
		return nil, fuego.InternalServerError{Detail: err.Error()}
	}

	// Notify WebSocket clients
	if s.deps.Hub != nil {
		s.deps.Hub.Broadcast(map[string]interface{}{
			"type":   "job.updated",
			"job_id": id.String(),
			"status": body.Status,
		})
	}

	return map[string]string{"status": "updated"}, nil
}

func (s *Server) bulkDeleteJobs(c fuego.ContextWithBody[JobsBulkDeleteRequest]) (JobsBulkDeleteResponse, error) {
	body, err := c.Body()
	if err != nil {
		return JobsBulkDeleteResponse{}, fuego.BadRequestError{Detail: err.Error()}
	}

	if len(body.IDs) == 0 {
		return JobsBulkDeleteResponse{}, fuego.BadRequestError{Detail: "No job IDs provided"}
	}

	if len(body.IDs) > 100 {
		return JobsBulkDeleteResponse{}, fuego.BadRequestError{Detail: "Cannot delete more than 100 jobs at once"}
	}

	deleted, err := s.deps.JobsRepo.BulkDelete(c.Context(), body.IDs)
	if err != nil {
		return JobsBulkDeleteResponse{}, fuego.InternalServerError{Detail: err.Error()}
	}

	// Notify WebSocket clients
	if s.deps.Hub != nil {
		s.deps.Hub.Broadcast(map[string]interface{}{
			"type":    "jobs.deleted",
			"count":   deleted,
			"job_ids": body.IDs,
		})
	}

	return JobsBulkDeleteResponse{Deleted: deleted}, nil
}

// ============================================================================
// Targets Handlers
// ============================================================================

func (s *Server) listTargets(c fuego.ContextNoBody) (TargetsListResponse, error) {
	targets, err := s.deps.TargetsRepo.List(c.Context())
	if err != nil {
		return TargetsListResponse{}, fuego.InternalServerError{Detail: err.Error()}
	}

	return TargetsListResponse{
		Targets: TargetsFromRepo(targets),
		Total:   len(targets),
	}, nil
}

func (s *Server) createTarget(c fuego.ContextWithBody[TargetCreateRequest]) (TargetResponse, error) {
	body, err := c.Body()
	if err != nil {
		return TargetResponse{}, fuego.BadRequestError{Detail: err.Error()}
	}

	// Validate type
	validTypes := map[string]bool{
		"TG_CHANNEL": true,
		"TG_FORUM":   true,
		"HH_SEARCH":  true,
	}
	if !validTypes[body.Type] {
		return TargetResponse{}, fuego.BadRequestError{Detail: "Invalid target type"}
	}

	target := &repository.ScrapingTarget{
		Name:     body.Name,
		Type:     body.Type,
		URL:      body.URL,
		IsActive: true,
		Metadata: body.Metadata,
	}

	if err := s.deps.TargetsRepo.Create(c.Context(), target); err != nil {
		return TargetResponse{}, fuego.InternalServerError{Detail: err.Error()}
	}

	return TargetFromRepo(target), nil
}

func (s *Server) getTarget(c fuego.ContextNoBody) (TargetResponse, error) {
	idStr := c.PathParam("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return TargetResponse{}, fuego.BadRequestError{Detail: "Invalid target ID"}
	}

	target, err := s.deps.TargetsRepo.GetByID(c.Context(), id)
	if err != nil {
		return TargetResponse{}, fuego.InternalServerError{Detail: err.Error()}
	}
	if target == nil {
		return TargetResponse{}, fuego.NotFoundError{Detail: "Target not found"}
	}

	return TargetFromRepo(target), nil
}

func (s *Server) updateTarget(c fuego.ContextWithBody[TargetUpdateRequest]) (TargetResponse, error) {
	idStr := c.PathParam("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return TargetResponse{}, fuego.BadRequestError{Detail: "Invalid target ID"}
	}

	target, err := s.deps.TargetsRepo.GetByID(c.Context(), id)
	if err != nil {
		return TargetResponse{}, fuego.InternalServerError{Detail: err.Error()}
	}
	if target == nil {
		return TargetResponse{}, fuego.NotFoundError{Detail: "Target not found"}
	}

	body, err := c.Body()
	if err != nil {
		return TargetResponse{}, fuego.BadRequestError{Detail: err.Error()}
	}

	// Update fields if provided
	if body.Name != "" {
		target.Name = body.Name
	}
	if body.Type != "" {
		target.Type = body.Type
	}
	if body.URL != "" {
		target.URL = body.URL
	}
	if body.IsActive != nil {
		target.IsActive = *body.IsActive
	}
	if body.Metadata != nil {
		target.Metadata = body.Metadata
	}

	if err := s.deps.TargetsRepo.Update(c.Context(), target); err != nil {
		return TargetResponse{}, fuego.InternalServerError{Detail: err.Error()}
	}

	return TargetFromRepo(target), nil
}

func (s *Server) deleteTarget(c fuego.ContextNoBody) (any, error) {
	idStr := c.PathParam("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fuego.BadRequestError{Detail: "Invalid target ID"}
	}

	if err := s.deps.TargetsRepo.Delete(c.Context(), id); err != nil {
		return nil, fuego.InternalServerError{Detail: err.Error()}
	}

	return map[string]string{"status": "deleted"}, nil
}

// ============================================================================
// Stats Handlers
// ============================================================================

func (s *Server) getStats(c fuego.ContextNoBody) (StatsResponse, error) {
	stats, err := s.deps.StatsRepo.GetStats(c.Context())
	if err != nil {
		return StatsResponse{}, fuego.InternalServerError{Detail: err.Error()}
	}

	return StatsResponse{
		TotalJobs:      stats.TotalJobs,
		AnalyzedJobs:   stats.AnalyzedJobs,
		InterestedJobs: stats.InterestedJobs,
		RejectedJobs:   stats.RejectedJobs,
		TodayJobs:      stats.TodayJobs,
		ActiveTargets:  stats.ActiveTargets,
	}, nil
}

// ============================================================================
// Scraping Handlers
// ============================================================================

func (s *Server) startScrape(c fuego.ContextWithBody[ScrapeStartRequest]) (ScrapeStartResponse, error) {
	if s.deps.CollectorService == nil {
		return ScrapeStartResponse{}, fuego.InternalServerError{Detail: "Collector service not available"}
	}

	body, err := c.Body()
	if err != nil {
		return ScrapeStartResponse{}, fuego.BadRequestError{Detail: err.Error()}
	}

	if body.Channel == "" {
		return ScrapeStartResponse{}, fuego.BadRequestError{Detail: "Channel is required"}
	}

	limit := body.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 10000 {
		limit = 10000
	}

	// Start scrape in background
	go func() {
		ctx := context.Background()
		_ = s.deps.CollectorService.StartScrape(ctx, body.Channel, limit, body.TopicIDs)
	}()

	return ScrapeStartResponse{
		Status:  "started",
		Message: "Scraping started for " + body.Channel,
	}, nil
}

func (s *Server) stopScrape(c fuego.ContextNoBody) (any, error) {
	if s.deps.CollectorService == nil {
		return nil, fuego.InternalServerError{Detail: "Collector service not available"}
	}

	if err := s.deps.CollectorService.StopScrape(); err != nil {
		return nil, fuego.InternalServerError{Detail: err.Error()}
	}

	return map[string]string{"status": "stopped"}, nil
}

func (s *Server) getScrapeStatus(c fuego.ContextNoBody) (ScrapeStatusResponse, error) {
	if s.deps.CollectorService == nil {
		return ScrapeStatusResponse{}, nil
	}

	isRunning := s.deps.CollectorService.IsRunning()
	target, progress, newJobs := s.deps.CollectorService.Status()

	var targetPtr *string
	if target != "" {
		targetPtr = &target
	}

	return ScrapeStatusResponse{
		IsRunning: isRunning,
		Target:    targetPtr,
		Progress:  progress,
		NewJobs:   newJobs,
	}, nil
}

// ============================================================================
// Auth Handlers
// ============================================================================

func (s *Server) getAuthStatus(c fuego.ContextNoBody) (AuthStatusResponse, error) {
	if s.deps.TelegramClient == nil {
		return AuthStatusResponse{
			Status:       "DISCONNECTED",
			IsReady:      false,
			QRInProgress: false,
		}, nil
	}

	status := s.deps.TelegramClient.GetStatus()
	return AuthStatusResponse{
		Status:       string(status),
		IsReady:      status == telegram.StatusReady,
		QRInProgress: s.deps.TelegramClient.IsQRInProgress(),
	}, nil
}

func (s *Server) startQRAuth(c fuego.ContextNoBody) (AuthQRStartResponse, error) {
	if s.deps.TelegramClient == nil {
		return AuthQRStartResponse{}, fuego.InternalServerError{Detail: "Telegram client not available"}
	}

	if s.deps.TelegramClient.GetStatus() == telegram.StatusReady {
		return AuthQRStartResponse{}, fuego.BadRequestError{Detail: "Already logged in"}
	}

	if s.deps.TelegramClient.IsQRInProgress() {
		return AuthQRStartResponse{Status: "already in progress"}, nil
	}

	// Start QR flow in background
	go func() {
		ctx := context.Background()
		err := s.deps.TelegramClient.StartQR(ctx, func(url string) {
			if s.deps.Hub != nil {
				s.deps.Hub.Broadcast(map[string]string{
					"type": "tg_qr",
					"url":  url,
				})
			}
		})

		if err != nil && !errors.Is(err, context.Canceled) && s.deps.Hub != nil {
			s.deps.Hub.Broadcast(map[string]string{
				"type":    "error",
				"message": err.Error(),
			})
			return
		}

		if err == nil && s.deps.Hub != nil {
			s.deps.Hub.Broadcast(map[string]string{
				"type": "tg_auth_success",
			})
		}
	}()

	return AuthQRStartResponse{Status: "started"}, nil
}

// ============================================================================
// Applications Handlers
// ============================================================================

func (s *Server) listApplications(c fuego.ContextNoBody) (ApplicationsListResponse, error) {
	jobIDStr := c.QueryParam("job_id")
	if jobIDStr == "" {
		return ApplicationsListResponse{}, fuego.BadRequestError{Detail: "job_id query parameter is required"}
	}

	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		return ApplicationsListResponse{}, fuego.BadRequestError{Detail: "Invalid job_id format"}
	}

	apps, err := s.deps.ApplicationsRepo.GetByJobID(c.Context(), jobID)
	if err != nil {
		return ApplicationsListResponse{}, fuego.InternalServerError{Detail: err.Error()}
	}

	return ApplicationsListResponse{
		Applications: ApplicationsFromModel(apps),
		Total:        len(apps),
	}, nil
}

func (s *Server) createApplication(c fuego.ContextWithBody[ApplicationCreateRequest]) (ApplicationResponse, error) {
	body, err := c.Body()
	if err != nil {
		return ApplicationResponse{}, fuego.BadRequestError{Detail: err.Error()}
	}

	channel := models.DeliveryChannel(body.DeliveryChannel)
	app := &models.JobApplication{
		ID:                 uuid.New(),
		JobID:              body.JobID,
		TailoredResumeMD:   body.TailoredResume,
		CoverLetterMD:      body.CoverLetter,
		ResumePDFPath:      body.ResumePDFPath,
		CoverLetterPDFPath: body.CoverPDFPath,
		DeliveryChannel:    &channel,
		DeliveryStatus:     models.DeliveryStatusPending,
	}

	if err := s.deps.ApplicationsRepo.Create(c.Context(), app); err != nil {
		return ApplicationResponse{}, fuego.InternalServerError{Detail: err.Error()}
	}

	return ApplicationFromModel(app), nil
}

func (s *Server) getApplication(c fuego.ContextNoBody) (ApplicationResponse, error) {
	idStr := c.PathParam("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return ApplicationResponse{}, fuego.BadRequestError{Detail: "Invalid application ID"}
	}

	app, err := s.deps.ApplicationsRepo.GetByID(c.Context(), id)
	if err != nil {
		return ApplicationResponse{}, fuego.InternalServerError{Detail: err.Error()}
	}
	if app == nil {
		return ApplicationResponse{}, fuego.NotFoundError{Detail: "Application not found"}
	}

	return ApplicationFromModel(app), nil
}

func (s *Server) sendApplication(c fuego.ContextWithBody[ApplicationSendRequest]) (ApplicationSendResponse, error) {
	idStr := c.PathParam("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return ApplicationSendResponse{}, fuego.BadRequestError{Detail: "Invalid application ID"}
	}

	app, err := s.deps.ApplicationsRepo.GetByID(c.Context(), id)
	if err != nil {
		return ApplicationSendResponse{}, fuego.InternalServerError{Detail: err.Error()}
	}
	if app == nil {
		return ApplicationSendResponse{}, fuego.NotFoundError{Detail: "Application not found"}
	}

	if app.ResumePDFPath == nil || *app.ResumePDFPath == "" {
		return ApplicationSendResponse{}, fuego.BadRequestError{Detail: "Application missing resume PDF"}
	}

	body, err := c.Body()
	if err != nil {
		return ApplicationSendResponse{}, fuego.BadRequestError{Detail: err.Error()}
	}

	if body.Recipient == "" {
		return ApplicationSendResponse{}, fuego.BadRequestError{Detail: "Recipient is required"}
	}

	// Determine channel
	channel := "TG_DM"
	if app.DeliveryChannel != nil {
		switch *app.DeliveryChannel {
		case models.DeliveryChannelTGDM:
			channel = "TG_DM"
		case models.DeliveryChannelEmail:
			channel = "EMAIL"
		case models.DeliveryChannelHH:
			channel = "HH"
		}
	}

	dispatchReq := &dispatcher.SendRequest{
		JobID:     app.JobID,
		Channel:   channel,
		Recipient: body.Recipient,
	}

	if err := s.deps.DispatcherService.SendApplication(c.Context(), dispatchReq); err != nil {
		return ApplicationSendResponse{}, fuego.InternalServerError{Detail: "Failed to send: " + err.Error()}
	}

	return ApplicationSendResponse{
		Status:  "sent",
		Message: "Application sent successfully",
	}, nil
}

func (s *Server) updateDeliveryStatus(c fuego.ContextWithBody[ApplicationUpdateDeliveryRequest]) (any, error) {
	idStr := c.PathParam("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fuego.BadRequestError{Detail: "Invalid application ID"}
	}

	body, err := c.Body()
	if err != nil {
		return nil, fuego.BadRequestError{Detail: err.Error()}
	}

	// Validate status
	validStatuses := map[string]models.DeliveryStatus{
		"PENDING":   models.DeliveryStatusPending,
		"SENT":      models.DeliveryStatusSent,
		"DELIVERED": models.DeliveryStatusDelivered,
		"READ":      models.DeliveryStatusRead,
		"FAILED":    models.DeliveryStatusFailed,
	}

	status, ok := validStatuses[body.Status]
	if !ok {
		return nil, fuego.BadRequestError{Detail: "Invalid delivery status"}
	}

	// Check application exists
	app, err := s.deps.ApplicationsRepo.GetByID(c.Context(), id)
	if err != nil {
		return nil, fuego.InternalServerError{Detail: err.Error()}
	}
	if app == nil {
		return nil, fuego.NotFoundError{Detail: "Application not found"}
	}

	if err := s.deps.ApplicationsRepo.UpdateDeliveryStatus(c.Context(), id, status); err != nil {
		return nil, fuego.InternalServerError{Detail: err.Error()}
	}

	return map[string]string{"status": "updated"}, nil
}

// Helper to parse int with default
func parseIntWithDefault(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return v
}
