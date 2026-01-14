package brain

import (
	"embed"
	"fmt"
	"regexp"
	"strings"
)

//go:embed docs/prompts/*.xml
var promptsFS embed.FS

const (
	// Prompt file names (relative to embedded FS)
	resumePromptFile  = "docs/prompts/resume-tailoring.xml"
	coverPromptFile   = "docs/prompts/cover-letter.xml"
)

// LoadResumePrompt loads the resume tailoring prompt from embedded FS.
// Returns (systemPrompt, userPrompt, error).
func LoadResumePrompt() (string, string, error) {
	data, err := promptsFS.ReadFile(resumePromptFile)
	if err != nil {
		return "", "", fmt.Errorf("read resume prompt: %w", err)
	}

	system, user, err := parseXMLPrompt(string(data))
	if err != nil {
		return "", "", fmt.Errorf("parse resume prompt: %w", err)
	}

	return system, user, nil
}

// LoadCoverPrompt loads the cover letter prompt from embedded FS.
// Returns (systemPrompt, templates map, error).
func LoadCoverPrompt() (string, map[string]string, error) {
	data, err := promptsFS.ReadFile(coverPromptFile)
	if err != nil {
		return "", nil, fmt.Errorf("read cover prompt: %w", err)
	}

	system, templates, err := parseXMLPromptWithTemplates(string(data))
	if err != nil {
		return "", nil, fmt.Errorf("parse cover prompt: %w", err)
	}

	return system, templates, nil
}

// parseXMLPrompt extracts system and user prompts from XML.
func parseXMLPrompt(xml string) (string, string, error) {
	system := extractTagContent(xml, "system")
	user := extractTagContent(xml, "user")
	return system, user, nil
}

// parseXMLPromptWithTemplates extracts system prompt and templates from XML.
func parseXMLPromptWithTemplates(xml string) (string, map[string]string, error) {
	system := extractTagContent(xml, "system")
	templates := extractTemplates(xml)
	return system, templates, nil
}

// extractTagContent extracts text content from an XML tag (multiline).
func extractTagContent(xml, tagName string) string {
	re := regexp.MustCompile(`(?s)<` + tagName + `>(.*?)</` + tagName + `>`)
	matches := re.FindStringSubmatch(xml)
	if len(matches) < 2 {
		return ""
	}
	return strings.TrimSpace(matches[1])
}

// extractTemplates extracts all template definitions from XML (multiline).
func extractTemplates(xml string) map[string]string {
	re := regexp.MustCompile(`(?s)<template id="([^"]+)">(.*?)</template>`)
	matches := re.FindAllStringSubmatch(xml, -1)

	templates := make(map[string]string)
	for _, match := range matches {
		if len(match) >= 3 {
			id := match[1]
			content := strings.TrimSpace(match[2])
			templates[id] = content
		}
	}
	return templates
}
