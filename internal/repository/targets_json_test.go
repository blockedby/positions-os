package repository

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestScrapingTargetJSONTags(t *testing.T) {
	target := ScrapingTarget{
		ID:       uuid.New(),
		Name:     "Test",
		Type:     "TG_CHANNEL",
		URL:      "@test",
		IsActive: true,
	}
	data, err := json.Marshal(target)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	jsonStr := string(data)
	t.Logf("JSON output: %s", jsonStr)

	// Check that JSON uses lowercase keys
	if strings.Contains(jsonStr, `"ID"`) {
		t.Errorf("JSON contains 'ID' instead of 'id': %s", jsonStr)
	}
	if strings.Contains(jsonStr, `"Name"`) {
		t.Errorf("JSON contains 'Name' instead of 'name': %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"id"`) {
		t.Errorf("JSON missing 'id' key: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"name"`) {
		t.Errorf("JSON missing 'name' key: %s", jsonStr)
	}
}
