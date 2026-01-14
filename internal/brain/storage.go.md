# storage.go

Storage layer for resume operations — loads base resume and saves tailored outputs.

## Functions

- **`LoadBaseResume(storagePath string) -> (string, error)`** — Reads `storagePath/resume.md`
- **`SaveTailoredResume(storagePath, jobID, content string) -> error`** — Saves to `storagePath/outputs/{jobID}/resume_tailored.md`

## Constants

| Constant | Value | Purpose |
|----------|-------|---------|
| `BaseResumeFilename` | `resume.md` | Base resume source file |
| `TailoredResumeFilename` | `resume_tailored.md` | Tailored output filename |
| `OutputsDir` | `outputs` | Generated files directory |

## Acceptance Status

- [x] `LoadBaseResume()` returns file contents
- [x] `LoadBaseResume()` returns error when file not found
- [x] `SaveTailoredResume()` creates directory if needed
- [x] `SaveTailoredResume()` saves file with correct content
- [x] All operations logged (info level)
