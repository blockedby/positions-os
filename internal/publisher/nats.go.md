# nats.go

NATS event publisher for job events.

- Implements `collector.EventPublisher` interface
- `PublishJobNew()` publishes `JobNewEvent` to `jobs.new` subject
- JSON-encodes event before publishing
- Uses `NATSClient` interface for testability
