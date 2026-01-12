package web

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateEngine_Render(t *testing.T) {
	// Create test templates
	templateDir := t.TempDir()

	// Base layout
	layoutTmpl := `{{ define "layout" }}<!DOCTYPE html><html><body>{{ template "content" . }}</body></html>{{ end }}`
	require.NoError(t, writeFile(templateDir, "layout.html", layoutTmpl))

	// Content template
	require.NoError(t, os.MkdirAll(filepath.Join(templateDir, "pages"), 0755))
	contentTmpl := `{{ define "content" }}<h1>{{ .Title }}</h1>{{ end }}`
	require.NoError(t, writeFile(filepath.Join(templateDir, "pages"), "page.html", contentTmpl))

	engine := NewTemplateEngine(templateDir, false)
	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	data := map[string]interface{}{"Title": "Dashboard"}

	err := engine.Render(&buf, "page", data)
	require.NoError(t, err)

	html := buf.String()
	assert.Contains(t, html, "<!DOCTYPE html>")
	assert.Contains(t, html, "<h1>Dashboard</h1>")
}

func writeFile(dir, name, content string) error {
	return os.WriteFile(filepath.Join(dir, name), []byte(content), 0644)
}

func TestRealTemplates_Load(t *testing.T) {
	// This test verifies that the actual project templates are valid
	wd, err := os.Getwd()
	require.NoError(t, err)

	// Assuming test is run from internal/web or root
	// If run from internal/web, templates are in ./templates
	templatesDir := filepath.Join(wd, "templates")
	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		// If running from root, try internal/web/templates
		templatesDir = filepath.Join(wd, "internal", "web", "templates")
	}

	// Skip if we can't find the directory (e.g. CI environment without assets?)
	// But here we want to ensure they EXIST.
	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		t.Fatalf("Templates directory not found at %s", templatesDir)
	}

	engine := NewTemplateEngine(templatesDir, false)
	err = engine.Load()
	require.NoError(t, err, "Failed to load real templates")
}
