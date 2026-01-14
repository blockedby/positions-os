# service.go

Service orchestrates the resume tailoring pipeline.

## Type

- **`Service`** — Orchestrates the full tailoring pipeline

## Interfaces

| Interface | Methods | Description |
|-----------|---------|-------------|
| `Storage` | `LoadBaseResume()`, `SaveTailoredResume()`, `SaveCoverLetter()` | Storage operations |
| `LLM` | `TailorResume()`, `GenerateCover()` | LLM operations |
| `Renderer` | `RenderResume()` | PDF rendering |
| `Broadcaster` | `Broadcast(event)` | WebSocket event broadcasting (optional) |

## Functions

| Function | Returns | Description |
|----------|---------|-------------|
| `NewService(storage, llm, pdf) *Service` | Service | Creates new service |
| `SetBroadcaster(b Broadcaster)` | - | Sets WebSocket event broadcaster |
| `TailorResumePipeline(ctx, jobID, jobData) -> *PipelineResult, error` | Result | Runs full pipeline |

## PipelineResult

| Field | Type | Description |
|-------|------|-------------|
| `ResumeMDPath` | string | Path to tailored resume markdown |
| `ResumePDFPath` | string | Path to resume PDF (for attachment) |
| `CoverLetterMD` | string | Cover letter content (for email/message) |

## Pipeline Steps (with WebSocket events)

| Step | Progress | Event |
|------|----------|-------|
| Started | 0% | `brain.progress` with step="started" |
| Load base resume | 0-10% | (logged only) |
| Tailor resume | 25% | `brain.progress` with step="tailoring" |
| Generate cover letter | 50% | `brain.progress` with step="cover_letter" |
| Render PDF | 75% | `brain.progress` with step="pdf_rendering" |
| Complete | 100% | `brain.progress` + `brain.completed` |

## Acceptance Status

- [x] Storage integration
- [x] LLM integration
- [x] Cover letter generated as TEXT (not PDF)
- [x] Resume PDF rendered
- [x] All operations logged
- [x] WebSocket progress events emitted
- [x] Tests pass (19/19)

## Important Notes

**Cover letter is TEXT, not PDF!** Cover letters are meant for email/messages,
not as PDF attachments. Only the resume gets rendered to PDF for attachment.

**WebSocket events** are emitted at each pipeline step if a Broadcaster is set.
Events include progress updates (0%, 25%, 50%, 75%, 100%) and completion/resume URL.
