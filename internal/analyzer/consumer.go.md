# consumer.go

NATS message consumer for job analysis events.

- Subscribes to `jobs.new` subject on `jobs` stream
- Durable consumer name: `analyzer_processor`
- Delegates actual processing to `Processor.ProcessJob()`
- Returns nil for malformed messages (Ack + skip)
- Returns error for processing failures (Nak + retry)
