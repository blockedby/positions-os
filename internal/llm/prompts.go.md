# prompts.go

Prompt templates for LLM job data extraction.

- `PromptConfig` holds system and user prompt templates
- `BuildUserPrompt()` renders user template with `{{RAW_CONTENT}}` variable
- System prompt defines JSON schema for extraction
