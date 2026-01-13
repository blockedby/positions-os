# jobs_test.go

Unit tests for job repository business logic.

## Test Cases

| Test | Covers |
|------|--------|
| Job.IsValidStatus() | Valid status strings |
| Job.IsNew() | RAW status check |
| Job.Title() | Fallback to "Unknown Position" |
| Job.Company() | Structured data extraction |
| Job.Salary() | Salary formatting |
