package brain

import (
	"testing"
)

// TestLoadPrompts_LoadsResumePrompt
func TestLoadPrompts_LoadsResumePrompt(t *testing.T) {
	// Execute
	system, user, err := LoadResumePrompt()

	// Assert
	if err != nil {
		t.Errorf("LoadResumePrompt() error = %v", err)
	}
	if system == "" {
		t.Error("system prompt is empty")
	}
	if user == "" {
		t.Error("user prompt is empty")
	}
	// Check for placeholders
	if !contains(user, "{{JOB_DATA}}") {
		t.Error("user prompt missing {{JOB_DATA}} placeholder")
	}
	if !contains(user, "{{BASE_RESUME}}") {
		t.Error("user prompt missing {{BASE_RESUME}} placeholder")
	}
}

// TestLoadPrompts_LoadsCoverPrompt
func TestLoadPrompts_LoadsCoverPrompt(t *testing.T) {
	// Execute
	system, templates, err := LoadCoverPrompt()

	// Assert
	if err != nil {
		t.Errorf("LoadCoverPrompt() error = %v", err)
	}
	if system == "" {
		t.Error("system prompt is empty")
	}
	if len(templates) == 0 {
		t.Error("no templates loaded")
	}
	// Check for required templates
	required := []string{"formal_ru", "modern_ru", "professional_en"}
	for _, id := range required {
		if templates[id] == "" {
			t.Errorf("template %s is empty", id)
		}
	}
}

// contains is a helper for substring check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
