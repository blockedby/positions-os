package telegram

import (
	"encoding/json"
	"fmt"

	"github.com/celestix/gotgproto/storage"
	"github.com/gotd/td/session"
)

// jsonData wraps session.Data for gotgproto's expected JSON format.
type jsonData struct {
	Version int         `json:"Version"`
	Data    session.Data `json:"Data"`
}

// ConvertToGotgprotoSession converts gotd session.Data to gotgproto storage.Session.
// gotgproto expects a wrapped JSON structure with Version and Data fields.
func ConvertToGotgprotoSession(data *session.Data) (*storage.Session, error) {
	if data == nil {
		return nil, fmt.Errorf("session data is nil")
	}

	// Wrap in the structure gotgproto expects
	wrapped := jsonData{
		Version: storage.LatestVersion,
		Data:    *data,
	}

	dataJSON, err := json.Marshal(wrapped)
	if err != nil {
		return nil, fmt.Errorf("marshal session data: %w", err)
	}

	return &storage.Session{
		Version: storage.LatestVersion,
		Data:    dataJSON,
	}, nil
}
