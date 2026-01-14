package brain

import (
	"context"
	"os"
	"testing"
)

// TestPDFRenderer_RenderResume_GeneratesPDF
// Note: This test requires chromedp/headless-shell or Chrome installed
func TestPDFRenderer_RenderResume_GeneratesPDF(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping PDF test in short mode")
	}

	// Setup
	tmpDir := t.TempDir()
	renderer := NewPDFRenderer(tmpDir)
	ctx := context.Background()

	resumeData := map[string]string{
		"name":        "John Doe",
		"title":       "Senior Go Developer",
		"experience":  "5+ years building scalable systems",
		"summary":     "Experienced developer",
		"skills":      "Go, Python, PostgreSQL",
		"education":   "BS Computer Science, State University, 2018",
	}

	// Execute
	pdfPath, err := renderer.RenderResume(ctx, "test-job", resumeData)

	// Assert
	if err != nil {
		t.Errorf("RenderResume() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		t.Errorf("PDF file not created at %s", pdfPath)
	}
}

