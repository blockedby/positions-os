# auth_interface.go

Telegram client interface for auth handler — defines contract for Telegram operations.

## Interface

```go
type TelegramClient interface {
    StartQR(ctx context.Context, onQRCode func(url string)) error
    GetStatus() telegram.Status
    IsQRInProgress() bool
    CancelQR()
}
```

## Methods

- **StartQR()** — Initiates QR login flow, calls `onQRCode` callback with QR URL
- **GetStatus()** — Returns current connection status
- **IsQRInProgress()** — Checks if QR flow is currently running
- **CancelQR()** — Cancels any ongoing QR login flow

## Implementation

Implemented by `telegram.Client` which wraps `telegram.Manager`.
