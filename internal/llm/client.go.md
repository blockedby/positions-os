# client.go

OpenAI-compatible LLM client for job analysis.

- Wrapper around `go-openai` library
- `ExtractJobData()` sends chat completion request
- Accepts system + user prompts for structured extraction
- Returns LLM response as JSON string
- Configurable: model, max tokens, temperature, timeout
- Base URL customizable for local/OpenAI-compatible APIs
