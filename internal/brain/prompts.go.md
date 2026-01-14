# prompts.go

Prompt template loader for LLM operations — loads embedded XML prompt templates.

## Functions

| Function | Returns | Description |
|----------|---------|-------------|
| `LoadResumePrompt()` | `(system, user, error)` | Loads resume tailoring prompt |
| `LoadCoverPrompt()` | `(system, templates, error)` | Loads cover letter prompt with templates |

## Internal Functions

- `parseXMLPrompt(xml)` — Extracts system/user sections
- `parseXMLPromptWithTemplates(xml)` — Extracts system + template map
- `extractTagContent(xml, tagName)` — Gets text from XML tag (multiline)
- `extractTemplates(xml)` — Extracts all `<template id="...">` blocks

## Embedded Files

| File | Purpose |
|------|---------|
| `docs/prompts/resume-tailoring.xml` | Resume adaptation prompt |
| `docs/prompts/cover-letter.xml` | Cover letter with 3 templates |

## Templates

| ID | Language | Style |
|----|----------|-------|
| `formal_ru` | Russian | Formal |
| `modern_ru` | Russian | Modern/casual |
| `professional_en` | English | Professional |

## Acceptance Status

- [x] XML prompts parsed correctly
- [x] System and user sections extracted
- [x] Templates map populated with all 3 templates
- [x] Placeholders preserved ({{JOB_DATA}}, {{BASE_RESUME}}, etc.)
