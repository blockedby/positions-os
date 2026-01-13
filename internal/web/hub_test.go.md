# hub_test.go

Unit tests for WebSocket hub.

## Test Cases

| Test | Covers |
|------|--------|
| Hub registration/unregistration | Client lifecycle |
| Broadcast | Message delivery to all clients |
| Client read/write pump | Goroutine management |
