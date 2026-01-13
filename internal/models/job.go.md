# job.go

Core job entity and status enum.

**JobStatus** values:
- `RAW` — Newly scraped, unprocessed
- `ANALYZED` — LLM extraction complete
- `REJECTED` — Not interested
- `INTERESTED` — Worth pursuing
- `TAILORED` — Resume customized
- `SENT` — Application sent
- `RESPONDED` — Received reply

**Job** struct fields:
- IDs: `ID`, `TargetID`, `ExternalID`
- Content: `RawContent`, `StructuredData` (JobData)
- Source metadata: `SourceURL`, `SourceDate`
- Telegram: `TGMessageID`, `TGTopicID`
- Status tracking: `Status`, timestamps

**JobData** (LLM extracted):
- Title, description, company
- Salary: min, max, currency
- Location, is_remote
- Technologies, experience_years
- Contacts
