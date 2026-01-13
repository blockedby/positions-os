package telegram

import (
	"encoding/json"
	"fmt"

	"github.com/celestix/gotgproto/storage"
	"github.com/gotd/td/session"
)

// ConvertToGotgprotoSession converts gotd session.Data to gotgproto storage.Session.
// gotgproto expects the raw JSON bytes of session.Data in its storage.Session.Data field.
func ConvertToGotgprotoSession(data *session.Data) (*storage.Session, error) {
	if data == nil {
		return nil, fmt.Errorf("session data is nil")
	}

	dataJSON, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal session data: %w", err)
	}

	return &storage.Session{
		Version: storage.LatestVersion,
		Data:    dataJSON,
	}, nil
}
