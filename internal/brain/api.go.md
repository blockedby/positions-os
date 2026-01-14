# api.go

HTTP handlers for Brain API endpoints.

## Type

- **`Handler`** — Handles brain API requests

## Interfaces

| Interface | Methods | Description |
|-----------|---------|-------------|
| `Repository` | `GetByID(id)`, `UpdateBrainOutputs(id, resumePath, coverText)` | Data layer |
| `BrainService` | `PrepareJob(jobID) -> (*PipelineResult, error)` | Service layer |

## Functions

| Function | Returns | Description |
|----------|---------|-------------|
| `NewHandler(repo, svc) *Handler` | Handler | Creates new handler |
| `RegisterRoutes(r, h)` | - | Registers brain routes on router |

## HTTP Handlers

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/jobs/{id}/prepare` | Triggers tailoring pipeline (async) |
| GET | `/api/v1/jobs/{id}/documents` | Returns document URLs and status |
| GET | `/api/v1/jobs/{id}/documents/resume.pdf` | Downloads resume PDF |

## Request/Response Types

### PrepareJobResponse
```go
type PrepareJobResponse struct {
    Status     string `json:"status"`      // "processing"
    WSChannel  string `json:"ws_channel"`  // "brain.{job_id}"
    Message    string `json:"message,omitempty"`
}
```

### DocumentsResponse
```go
type DocumentsResponse struct {
    ResumeURL   string `json:"resume_url,omitempty"`
    CoverLetter string `json:"cover_letter,omitempty"`
    Status      string `json:"status"`
    PreparedAt  string `json:"prepared_at,omitempty"`
}
```

## Dependencies

- `github.com/go-chi/chi/v5` — HTTP routing
- `github.com/google/uuid` — UUID parsing

## Acceptance Status

- [x] POST /prepare returns 202 with ws_channel
- [x] POST /prepare validates job status (INTERESTED only)
- [x] GET /documents returns resume_url and cover_letter
- [x] GET /documents returns 404 when not tailored
- [x] Download endpoint serves PDF with correct headers
- [x] All handlers logged

## Important Notes

- PrepareJob runs asynchronously — returns immediately with ws_channel
- Only INTERESTED jobs can be prepared (returns 400 otherwise)
- Cover letter is TEXT in response, not downloadable
