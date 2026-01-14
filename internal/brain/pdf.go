package brain

import (
	"context"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/blockedby/positions-os/internal/logger"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

const (
	// ResumePDFOutput is the filename for resume PDF
	ResumePDFOutput = "resume.pdf"
	// DefaultTimeout for PDF rendering
	DefaultTimeout = 30 * time.Second
)

// Renderer defines the PDF rendering interface.
type Renderer interface {
	RenderResume(ctx context.Context, jobID string, data map[string]string) (string, error)
}

// PDFRenderer handles PDF generation using chromedp.
type PDFRenderer struct {
	templatesPath string
	storagePath   string
	timeout       time.Duration
}

// NewPDFRenderer creates a new PDF renderer.
// storagePath is where PDFs will be saved.
func NewPDFRenderer(storagePath string) *PDFRenderer {
	return &PDFRenderer{
		templatesPath: "static/pdf-templates",
		storagePath:   storagePath,
		timeout:       DefaultTimeout,
	}
}

// RenderResume generates a PDF resume from template data.
func (p *PDFRenderer) RenderResume(ctx context.Context, jobID string, data map[string]string) (string, error) {
	logger.Info("rendering resume PDF")

	// Add date if not present
	if data["date"] == "" {
		data["date"] = time.Now().Format("2006-01-02")
	}

	outputPath := filepath.Join(p.storagePath, OutputsDir, jobID, ResumePDFOutput)

	// Generate HTML
	html, err := p.renderTemplate("resume.html", data)
	if err != nil {
		logger.Error("failed to render resume HTML", err)
		return "", fmt.Errorf("render HTML: %w", err)
	}

	// Convert to PDF
	if err := p.htmlToPDF(ctx, html, outputPath); err != nil {
		logger.Error("failed to convert resume to PDF", err)
		return "", fmt.Errorf("HTML to PDF: %w", err)
	}

	logger.Info("resume PDF saved: " + outputPath)
	return outputPath, nil
}

// renderTemplate executes an HTML template with data.
func (p *PDFRenderer) renderTemplate(templateName string, data map[string]string) (string, error) {
	templatePath := filepath.Join(p.templatesPath, templateName)

	content, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("read template: %w", err)
	}

	tmpl, err := template.New(templateName).Parse(string(content))
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	// Execute template into a string builder
	var result strings.Builder
	if err := tmpl.Execute(&result, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return result.String(), nil
}

// htmlToPDF converts HTML to PDF using chromedp.
func (p *PDFRenderer) htmlToPDF(ctx context.Context, html, outputPath string) error {
	// Ensure output directory exists
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	// Create chromedp context with exec allocator options
	cctx, cancel := chromedp.NewExecAllocator(ctx,
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
	)
	defer cancel()

	// Create new browser context
	cctx, cancel = chromedp.NewContext(cctx)
	defer cancel()

	// Run PDF generation
	var pdfBuf []byte
	if err := chromedp.Run(cctx,
		chromedp.Navigate("data:text/html;charset=utf-8,"+html),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			pdfBuf, _, err = page.PrintToPDF().WithPrintBackground(true).Do(ctx)
			return err
		}),
	); err != nil {
		return fmt.Errorf("chromedp run: %w", err)
	}

	// Write PDF to file
	if err := os.WriteFile(outputPath, pdfBuf, 0644); err != nil {
		return fmt.Errorf("write PDF: %w", err)
	}

	return nil
}
