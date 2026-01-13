# client.go

NATS JetStream client for pub/sub messaging.

- `New()` creates connection and JetStream context
- `EnsureStream()` creates stream if not exists
- `Publish()` sends JSON-encoded messages to subjects
- `Subscribe()` creates durable consumer with explicit ACK policy
- Handler returns error for NAK (redelivery), nil for ACK
- `IsConnected()` checks connection status
