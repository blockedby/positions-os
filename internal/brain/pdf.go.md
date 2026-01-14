# pdf.go

PDF rendering using chromedp — converts HTML templates to PDF files.

## Type

- **`PDFRenderer`** — Handles PDF generation from HTML templates
- **`Renderer`** — Interface for PDF rendering (used by Service)

## Functions

| Function | Returns | Description |
|----------|---------|-------------|
| `NewPDFRenderer(storagePath) *PDFRenderer` | Renderer | Creates new PDF renderer |
| `RenderResume(ctx, jobID, data) -> (path, error)` | PDF path | Generates resume PDF |

## Constants

| Constant | Value | Purpose |
|----------|-------|---------|
| `ResumePDFOutput` | `resume.pdf` | Resume PDF filename |
| `DefaultTimeout` | 30s | Chrome rendering timeout |

## HTML Templates

| Template | Variables |
|----------|-----------|
| `resume.html` | name, title, summary, skills, experience, education, date |

## Dependencies

- `github.com/chromedp/chromedp` — Headless Chrome control
- `github.com/chromedp/cdproto/page` — PDF generation

## Acceptance Status

- [x] HTML templates render with Go template engine
- [x] PDF generated via chromedp page.PrintToPDF
- [x] Output directory created automatically
- [x] Tests skip gracefully without Chrome (`-short` flag)
- [x] All operations logged

## Test Notes

Full PDF tests require Chrome/chromedp installed:
```bash
go test ./internal/brain/... -v -run TestPDFRenderer
```

Short mode (no Chrome required):
```bash
go test ./internal/brain/... -v -short -run TestPDFRenderer
```

## Important: Cover Letter Handling

**Cover letters are TEXT, not PDF!** Only the resume is rendered to PDF
for email attachment. Cover letters are generated as plain text for
email/messaging by the LLM service.
