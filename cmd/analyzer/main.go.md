# main.go

Analyzer service entry point — processes jobs via LLM and NATS.

- Subscribes to NATS `jobs.new` events
- Loads LLM prompts from `docs/prompts/job-extraction.xml`
- Runs consumer for background processing
