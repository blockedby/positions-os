# processor.go

Core LLM-based job analysis processor.

- Fetches raw job from database by JobID
- Builds prompt using configured system/user templates
- Calls LLM to extract structured data (title, salary, skills, etc.)
- Cleans JSON response (removes markdown code blocks)
- Updates job with `structured_data` in database
- Defines `LLMClient` and `JobsRepository` interfaces for dependency injection
