package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/blockedby/positions-os/internal/dispatcher"
	"github.com/blockedby/positions-os/internal/logger"
	"github.com/blockedby/positions-os/internal/models"
)

// ApplicationsRepository defines the interface for application data access.
type ApplicationsRepository interface {
	Create(ctx context.Context, app *models.JobApplication) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.JobApplication, error)
	GetByJobID(ctx context.Context, jobID uuid.UUID) ([]*models.JobApplication, error)
	UpdateDeliveryStatus(ctx context.Context, id uuid.UUID, status models.DeliveryStatus) error
}

// DispatcherService defines the interface for sending applications.
type DispatcherService interface {
	SendApplication(ctx context.Context, req *dispatcher.SendRequest) error
}

// ApplicationsHandler handles application-related HTTP requests.
type ApplicationsHandler struct {
	repo       ApplicationsRepository
	dispatcher DispatcherService
	log        *logger.Logger
}

// NewApplicationsHandler creates a new ApplicationsHandler.
func NewApplicationsHandler(repo ApplicationsRepository, dispatcher DispatcherService) *ApplicationsHandler {
	return &ApplicationsHandler{
		repo:       repo,
		dispatcher: dispatcher,
		log:        logger.Get(),
	}
}

// List returns applications filtered by job_id.
// GET /api/v1/applications?job_id={uuid}
func (h *ApplicationsHandler) List(w http.ResponseWriter, r *http.Request) {
	jobIDStr := r.URL.Query().Get("job_id")
	if jobIDStr == "" {
		http.Error(w, `{"error":"job_id query parameter is required"}`, http.StatusBadRequest)
		return
	}

	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid job_id format"}`, http.StatusBadRequest)
		return
	}

	applications, err := h.repo.GetByJobID(r.Context(), jobID)
	if err != nil {
		http.Error(w, `{"error":"failed to fetch applications"}`, http.StatusInternalServerError)
		return
	}

	resp := struct {
		Applications []*models.JobApplication `json:"applications"`
		Total        int                      `json:"total"`
	}{
		Applications: applications,
		Total:        len(applications),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GetByID returns a single application by ID.
// GET /api/v1/applications/{id}
func (h *ApplicationsHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid application ID"}`, http.StatusBadRequest)
		return
	}

	app, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"failed to fetch application"}`, http.StatusInternalServerError)
		return
	}
	if app == nil {
		http.Error(w, `{"error":"application not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(app)
}

// Create creates a new job application.
// POST /api/v1/applications
func (h *ApplicationsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		JobID           string    `json:"job_id"`
		TailoredResume  *string   `json:"tailored_resume,omitempty"`
		CoverLetter     *string   `json:"cover_letter,omitempty"`
		ResumePDFPath   *string   `json:"resume_pdf_path,omitempty"`
		CoverPDFPath    *string   `json:"cover_pdf_path,omitempty"`
		DeliveryChannel string    `json:"delivery_channel"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, `{"error":"invalid request payload"}`, http.StatusBadRequest)
		return
	}

	// Validate and parse job_id
	jobID, err := uuid.Parse(payload.JobID)
	if err != nil || payload.JobID == "" {
		http.Error(w, `{"error":"invalid job_id format"}`, http.StatusBadRequest)
		return
	}
	if jobID == uuid.Nil {
		http.Error(w, `{"error":"job_id is required"}`, http.StatusBadRequest)
		return
	}

	// Validate delivery_channel
	if payload.DeliveryChannel == "" {
		http.Error(w, `{"error":"delivery_channel is required"}`, http.StatusBadRequest)
		return
	}

	channel := models.DeliveryChannel(payload.DeliveryChannel)
	if channel != models.DeliveryChannelTGDM && channel != models.DeliveryChannelEmail && channel != models.DeliveryChannelHH {
		http.Error(w, `{"error":"invalid delivery_channel: must be TG_DM, EMAIL, or HH_RESPONSE"}`, http.StatusBadRequest)
		return
	}

	app := &models.JobApplication{
		ID:                uuid.New(),
		JobID:             jobID,
		TailoredResumeMD:  payload.TailoredResume,
		CoverLetterMD:     payload.CoverLetter,
		ResumePDFPath:     payload.ResumePDFPath,
		CoverLetterPDFPath: payload.CoverPDFPath,
		DeliveryChannel:   &channel,
		DeliveryStatus:    models.DeliveryStatusPending,
	}

	if err := h.repo.Create(r.Context(), app); err != nil {
		http.Error(w, `{"error":"failed to create application"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(app)
}

// SendRequest is the payload for sending an application.
type SendRequest struct {
	Recipient string `json:"recipient"` // e.g., "@recruiter" for Telegram, email for EMAIL channel
}

// Send sends an application via the configured channel.
// POST /api/v1/applications/{id}/send
func (h *ApplicationsHandler) Send(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid application ID"}`, http.StatusBadRequest)
		return
	}

	// Get application
	app, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"failed to fetch application"}`, http.StatusInternalServerError)
		return
	}
	if app == nil {
		http.Error(w, `{"error":"application not found"}`, http.StatusNotFound)
		return
	}

	// Validate application has required files
	if app.ResumePDFPath == nil || *app.ResumePDFPath == "" {
		http.Error(w, `{"error":"application missing resume PDF"}`, http.StatusBadRequest)
		return
	}

	// Parse request
	var req SendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request payload"}`, http.StatusBadRequest)
		return
	}

	// Validate recipient
	if req.Recipient == "" {
		http.Error(w, `{"error":"recipient is required"}`, http.StatusBadRequest)
		return
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

	// Send via dispatcher
	dispatchReq := &dispatcher.SendRequest{
		JobID:     app.JobID,
		Channel:   channel,
		Recipient: req.Recipient,
	}

	if err := h.dispatcher.SendApplication(r.Context(), dispatchReq); err != nil {
		http.Error(w, `{"error":"failed to send application: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "sent",
		"message": "application sent successfully",
	})
}

// UpdateDeliveryStatusRequest is the payload for updating delivery status.
type UpdateDeliveryStatusRequest struct {
	Status models.DeliveryStatus `json:"status"`
}

// UpdateDeliveryStatus updates the delivery status of an application.
// PATCH /api/v1/applications/{id}/delivery
func (h *ApplicationsHandler) UpdateDeliveryStatus(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid application ID"}`, http.StatusBadRequest)
		return
	}

	// Parse request
	var req UpdateDeliveryStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request payload"}`, http.StatusBadRequest)
		return
	}

	// Validate status
	validStatuses := map[models.DeliveryStatus]bool{
		models.DeliveryStatusPending:   true,
		models.DeliveryStatusSent:      true,
		models.DeliveryStatusDelivered: true,
		models.DeliveryStatusRead:      true,
		models.DeliveryStatusFailed:    true,
	}

	if !validStatuses[req.Status] {
		http.Error(w, `{"error":"invalid delivery status"}`, http.StatusBadRequest)
		return
	}

	// Check application exists
	app, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"failed to fetch application"}`, http.StatusInternalServerError)
		return
	}
	if app == nil {
		http.Error(w, `{"error":"application not found"}`, http.StatusNotFound)
		return
	}

	// Update status
	if err := h.repo.UpdateDeliveryStatus(r.Context(), id, req.Status); err != nil {
		http.Error(w, `{"error":"failed to update status"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "updated",
	})
}

// ListByJobID is an alias for List for clearer naming.
func (h *ApplicationsHandler) ListByJobID(w http.ResponseWriter, r *http.Request) {
	h.List(w, r)
}

// SendApplication is an alias for Send for clearer naming.
func (h *ApplicationsHandler) SendApplication(w http.ResponseWriter, r *http.Request) {
	h.Send(w, r)
}

var (
	ErrApplicationNotFound = errors.New("application not found")
	ErrInvalidRecipient    = errors.New("invalid recipient")
	ErrMissingResumePDF    = errors.New("missing resume PDF")
)
