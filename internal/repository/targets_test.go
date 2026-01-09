package repository

import (
	"testing"
)

// test target type validation
func TestScrapingTarget_IsValid(t *testing.T) {
	validTypes := []string{"TG_CHANNEL", "TG_GROUP", "TG_FORUM", "HH_SEARCH", "LINKEDIN_SEARCH"}

	for _, typ := range validTypes {
		target := ScrapingTarget{
			Name:     "Test",
			Type:     typ,
			URL:      "@test",
			IsActive: true,
		}
		if !target.IsValid() {
			t.Errorf("target with type %s should be valid", typ)
		}
	}

	// invalid type
	invalidTarget := ScrapingTarget{
		Name: "Test",
		Type: "INVALID",
		URL:  "@test",
	}
	if invalidTarget.IsValid() {
		t.Error("target with invalid type should not be valid")
	}
}

// test telegram target check
func TestScrapingTarget_IsTelegram(t *testing.T) {
	tests := []struct {
		typ       string
		wantTg    bool
		wantForum bool
	}{
		{"TG_CHANNEL", true, false},
		{"TG_GROUP", true, false},
		{"TG_FORUM", true, true},
		{"HH_SEARCH", false, false},
		{"LINKEDIN_SEARCH", false, false},
	}

	for _, tt := range tests {
		target := ScrapingTarget{Type: tt.typ}
		if target.IsTelegram() != tt.wantTg {
			t.Errorf("type %s: IsTelegram() = %v, want %v", tt.typ, target.IsTelegram(), tt.wantTg)
		}
		if target.IsForum() != tt.wantForum {
			t.Errorf("type %s: IsForum() = %v, want %v", tt.typ, target.IsForum(), tt.wantForum)
		}
	}
}
