package llm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPrompt(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "llm_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Define valid XML content
	validXML := `
<prompt>
    <system>You are a helpful assistant.</system>
    <user>Analyze this content: {{RAW_CONTENT}}</user>
</prompt>`
	validFile := filepath.Join(tmpDir, "valid.xml")
	if err := os.WriteFile(validFile, []byte(validXML), 0644); err != nil {
		t.Fatalf("Failed to write valid XML file: %v", err)
	}

	// Define invalid XML content
	invalidXML := `<prompt><system>Unclosed tag`
	invalidFile := filepath.Join(tmpDir, "invalid.xml")
	if err := os.WriteFile(invalidFile, []byte(invalidXML), 0644); err != nil {
		t.Fatalf("Failed to write invalid XML file: %v", err)
	}

	tests := []struct {
		name      string
		filepath  string
		wantError bool
		wantSys   string
		wantUser  string
	}{
		{
			name:      "Valid XML",
			filepath:  validFile,
			wantError: false,
			wantSys:   "You are a helpful assistant.",
			wantUser:  "Analyze this content: {{RAW_CONTENT}}",
		},
		{
			name:      "Invalid XML",
			filepath:  invalidFile,
			wantError: true,
		},
		{
			name:      "Non-existent File",
			filepath:  filepath.Join(tmpDir, "nonexistent.xml"),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt, err := LoadPrompt(tt.filepath)
			if (err != nil) != tt.wantError {
				t.Errorf("LoadPrompt() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError {
				if prompt.System != tt.wantSys {
					t.Errorf("System prompt = %q, want %q", prompt.System, tt.wantSys)
				}
				if prompt.User != tt.wantUser {
					t.Errorf("User prompt = %q, want %q", prompt.User, tt.wantUser)
				}
			}
		})
	}
}

func TestBuildUserPrompt(t *testing.T) {
	p := &PromptConfig{
		User: "Analyze this: {{RAW_CONTENT}}",
	}

	raw := "some raw job description"
	expected := "Analyze this: some raw job description"
	got := p.BuildUserPrompt(raw)

	if got != expected {
		t.Errorf("BuildUserPrompt() = %q, want %q", got, expected)
	}
}
