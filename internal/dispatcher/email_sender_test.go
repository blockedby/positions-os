package dispatcher

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestEmailSender_NewEmailSender tests the constructor
func TestEmailSender_NewEmailSender(t *testing.T) {
	sender := NewEmailSender()
	assert.NotNil(t, sender, "NewEmailSender should return non-nil")
}

// TestEmailSender_SendApplication tests that email sender returns not implemented error
func TestEmailSender_SendApplication(t *testing.T) {
	sender := NewEmailSender()
	appID := uuid.New()

	err := sender.SendApplication(context.Background(), appID, "test@example.com")
	assert.Error(t, err, "SendApplication should return error")
	assert.Contains(t, err.Error(), "not implemented", "Error should mention not implemented")
}

// TestEmailSender_SendApplicationWithContent tests that content send returns not implemented error
func TestEmailSender_SendApplicationWithContent(t *testing.T) {
	sender := NewEmailSender()

	err := sender.SendApplicationWithContent(
		context.Background(),
		"test@example.com",
		"Test Subject",
		"Test Body",
		[]string{"resume.pdf"},
	)
	assert.Error(t, err, "SendApplicationWithContent should return error")
	assert.Contains(t, err.Error(), "not implemented", "Error should mention not implemented")
}
