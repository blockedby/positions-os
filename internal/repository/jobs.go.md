# jobs.go

Job repository — CRUD and filtering operations.

**Queries:**
- `Create()` — Insert new job
- `GetByID()` — Fetch single job
- `UpdateStructuredData()` — Save LLM results
- `List()` — Filter by status, salary, tech, full-text
- `UpdateStatus()` — Change job status

**JobFilter** options:
- Status equality
- Salary range (min/max)
- Technology search in structured_data
- Full-text query
- Pagination (page, limit)
- Sorting (sort, order)
