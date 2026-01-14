package dispatcher

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// EmailSender stub - not implemented in Phase 5
// Email functionality is deferred until after Telegram DM is stable
type EmailSender struct {
	// Future fields (add when implementing):
	// smtpConfig *SMTPConfig
	// repo       *repository.ApplicationsRepository
	// hub        *web.Hub
	// log        *logger.Logger
}

// NewEmailSender creates a new (stub) email sender
func NewEmailSender() *EmailSender {
	return &EmailSender{}
}

// SendApplication returns "not implemented" error
// This stub ensures the API is ready but clearly indicates email is not available
func (s *EmailSender) SendApplication(ctx context.Context, appID uuid.UUID, recipient string) error {
	return fmt.Errorf("email sender not implemented: use TG_DM channel instead")
}

// SendApplicationWithContent is a placeholder for future implementation
func (s *EmailSender) SendApplicationWithContent(ctx context.Context, recipient string, subject string, body string, attachments []string) error {
	return fmt.Errorf("email sender not implemented: use TG_DM channel instead")
}
