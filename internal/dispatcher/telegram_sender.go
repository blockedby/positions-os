package dispatcher

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/celestix/gotgproto"
	"github.com/google/uuid"
	"github.com/gotd/td/tg"
	"golang.org/x/time/rate"

	"github.com/blockedby/positions-os/internal/logger"
	"github.com/blockedby/positions-os/internal/models"
)

// DeliveryTrackerInterface defines the interface for tracking delivery status.
// This will be provided by Thread A (Task 2.x).
type DeliveryTrackerInterface interface {
	TrackStart(ctx context.Context, appID uuid.UUID) error
	TrackSuccess(ctx context.Context, appID uuid.UUID) error
	TrackFailure(ctx context.Context, appID uuid.UUID, err error) error
}

// ApplicationsRepository defines the interface for application data access.
// This will be provided by Thread A (Task 1.x).
type ApplicationsRepository interface {
	Create(ctx context.Context, app *models.JobApplication) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.JobApplication, error)
	GetByJobID(ctx context.Context, jobID uuid.UUID) ([]*models.JobApplication, error)
}

// ReadTrackerInterface defines the interface for read receipt detection.
// This will be provided by Thread A (Task 2.5.x).
type ReadTrackerInterface interface {
	RegisterSentMessage(msgID int64, appID uuid.UUID)
}

// TelegramSender handles sending job applications via Telegram DM.
type TelegramSender struct {
	client      *gotgproto.Client
	tracker     DeliveryTrackerInterface
	repo        ApplicationsRepository
	readTracker ReadTrackerInterface
	limiter     *rate.Limiter // 1 request per 10 seconds
	log         *logger.Logger
}

// NewTelegramSender creates a new TelegramSender with rate limiting.
func NewTelegramSender(
	client *gotgproto.Client,
	tracker DeliveryTrackerInterface,
	repo ApplicationsRepository,
	readTracker ReadTrackerInterface,
	log *logger.Logger,
) *TelegramSender {
	return &TelegramSender{
		client:      client,
		tracker:     tracker,
		repo:        repo,
		readTracker: readTracker,
		limiter:     rate.NewLimiter(rate.Every(10*time.Second), 1),
		log:         log,
	}
}

// LimiterForTest exposes the rate limiter for testing.
func (s *TelegramSender) LimiterForTest() *rate.Limiter {
	return s.limiter
}

// ResolveUsername resolves a Telegram username to an InputPeerUser.
// The username can be with or without @ prefix.
func (s *TelegramSender) ResolveUsername(ctx context.Context, username string) (*tg.InputPeerUser, error) {
	if username == "" {
		return nil, errors.New("username cannot be empty")
	}

	// Strip @ if present
	username = s.stripAtPrefix(username)

	// Use contacts.ResolveUsername API
	result, err := s.client.API().ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
		Username: username,
	})
	if err != nil {
		return nil, fmt.Errorf("resolve username %s: %w", username, err)
	}

	if len(result.Users) == 0 {
		return nil, fmt.Errorf("user not found: %s", username)
	}

	user, ok := result.Users[0].(*tg.User)
	if !ok {
		return nil, fmt.Errorf("not a user: %s", username)
	}

	return &tg.InputPeerUser{
		UserID:     user.ID,
		AccessHash: user.AccessHash,
	}, nil
}

// stripAtPrefix removes the @ prefix from a username if present.
func (s *TelegramSender) stripAtPrefix(username string) string {
	return strings.TrimPrefix(username, "@")
}

// uploadFile uploads a file to Telegram servers and returns the uploaded file info.
func (s *TelegramSender) uploadFile(ctx context.Context, path string) (*tg.InputFile, error) {
	// Read file content
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	api := s.client.API()
	fileID := time.Now().UnixNano()

	const chunkSize = 512 * 1024 // 512KB chunks
	var totalBytes int
	var parts int

	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}
		chunk := data[i:end]

		_, err := api.UploadSaveFilePart(ctx, &tg.UploadSaveFilePartRequest{
			FileID:   fileID,
			FilePart: parts,
			Bytes:    chunk,
		})
		if err != nil {
			return nil, fmt.Errorf("upload part %d: %w", parts, err)
		}

		totalBytes += len(chunk)
		parts++
	}

	return &tg.InputFile{
		ID:          fileID,
		Parts:       parts,
		Name:        filepath.Base(path),
		MD5Checksum: "", // Optional
	}, nil
}

// UploadAndSend uploads a PDF file and sends it with a caption to a Telegram user.
// recipient: Telegram username (with or without @)
// text: Caption text (cover letter) to send with the file
// pdfPath: Path to the PDF file to upload
func (s *TelegramSender) UploadAndSend(ctx context.Context, recipient, text, pdfPath string) error {
	// Validate inputs
	if recipient == "" {
		return errors.New("recipient cannot be empty")
	}
	if text == "" {
		return errors.New("text cannot be empty")
	}
	if pdfPath == "" {
		return errors.New("pdfPath cannot be empty")
	}

	// Resolve username to peer
	peer, err := s.ResolveUsername(ctx, recipient)
	if err != nil {
		return fmt.Errorf("resolve username: %w", err)
	}

	// Wait for rate limiter (1 request per 10 seconds)
	if err := s.limiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limiter: %w", err)
	}

	// Upload PDF file
	uploadedFile, err := s.uploadFile(ctx, pdfPath)
	if err != nil {
		return fmt.Errorf("upload file: %w", err)
	}

	// Send media with caption
	media := &tg.InputMediaUploadedDocument{
		File:     uploadedFile,
		MimeType: "application/pdf",
		Attributes: []tg.DocumentAttributeClass{
			&tg.DocumentAttributeFilename{
				FileName: filepath.Base(pdfPath),
			},
		},
	}

	api := s.client.API()
	_, err = api.MessagesSendMedia(ctx, &tg.MessagesSendMediaRequest{
		Peer:    peer,
		Media:   media,
		Message: text,
	})
	if err != nil {
		return fmt.Errorf("send media: %w", err)
	}

	return nil
}

// SendApplication orchestrates the full send flow for a job application.
// It retrieves the application, tracks delivery, sends the resume, and updates status.
func (s *TelegramSender) SendApplication(ctx context.Context, appID uuid.UUID, recipient string) error {
	// Validate appID
	if appID == uuid.Nil {
		return errors.New("application ID cannot be empty")
	}

	// Get application from repository
	app, err := s.repo.GetByID(ctx, appID)
	if err != nil {
		return fmt.Errorf("get application: %w", err)
	}
	if app == nil {
		return fmt.Errorf("application not found: %s", appID)
	}

	// Prepare cover letter text
	coverLetter := ""
	if app.CoverLetterMD != nil {
		coverLetter = *app.CoverLetterMD
	}

	// Get resume PDF path
	resumePath := ""
	if app.ResumePDFPath != nil {
		resumePath = *app.ResumePDFPath
	}
	if resumePath == "" {
		return errors.New("resume PDF path not set for application")
	}

	// Track start
	if err := s.tracker.TrackStart(ctx, appID); err != nil {
		return fmt.Errorf("track start: %w", err)
	}

	// Send resume PDF
	if err := s.UploadAndSend(ctx, recipient, coverLetter, resumePath); err != nil {
		// Track failure
		_ = s.tracker.TrackFailure(ctx, appID, err)
		return fmt.Errorf("upload and send: %w", err)
	}

	// Register message for read detection (coordinate with Thread A Task 2.5.5)
	// TODO: This will be implemented when Thread A provides ReadTracker integration

	// Track success
	if err := s.tracker.TrackSuccess(ctx, appID); err != nil {
		return fmt.Errorf("track success: %w", err)
	}

	return nil
}

// isFloodWait checks if an error is a FLOOD_WAIT error and returns the wait seconds.
// Returns 0 if not a FLOOD_WAIT error.
func (s *TelegramSender) isFloodWait(err error) int {
	if err == nil {
		return 0
	}

	// Check for FLOOD_WAIT pattern in error string
	errStr := err.Error()
	if !strings.Contains(errStr, "FLOOD_WAIT") {
		return 0
	}

	// Extract wait seconds from "FLOOD_WAIT_X" pattern
	parts := strings.Split(errStr, "FLOOD_WAIT_")
	if len(parts) < 2 {
		return 0
	}

	// Parse the number (may have suffix like " (caused by...)")
	var waitSeconds int
	_, _ = fmt.Sscanf(parts[1], "%d", &waitSeconds)
	return waitSeconds
}
