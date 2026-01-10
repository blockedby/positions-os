package llm

import (
	"encoding/xml"
	"fmt"
	"os"
	"strings"
)

// PromptConfig represents a prompt loaded from an XML file.
// It contains the system prompt and the user prompt template.
type PromptConfig struct {
	XMLName xml.Name `xml:"prompt"`
	System  string   `xml:"system"`
	User    string   `xml:"user"`
}

// LoadPrompt reads and parses a prompt configuration from an XML file.
func LoadPrompt(filepath string) (*PromptConfig, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("read prompt file: %w", err)
	}

	var config PromptConfig
	if err := xml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse prompt xml: %w", err)
	}

	return &config, nil
}

// BuildUserPrompt replaces {{RAW_CONTENT}} in the user prompt template with the actual content.
func (p *PromptConfig) BuildUserPrompt(rawContent string) string {
	return strings.ReplaceAll(p.User, "{{RAW_CONTENT}}", rawContent)
}
