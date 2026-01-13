# settings.html

Settings page — Telegram authentication and target management.

## Features

- **Telegram Connection** card with status display
- **QR Code** display for Telegram login
- **Add Target** form for new scraping targets
- **Targets List** with inline editing

## Telegram Authentication Flow

### Initial State
- Checks `/api/v1/scrape/status` on load
- Shows "Connected" / "Disconnected" status
- Hides "Connect Telegram" button if already connected

### QR Login
1. User clicks "Connect Telegram" button
2. Button disabled for 3 seconds (debouncing)
3. QR code appears with 30-second countdown
4. WebSocket receives `tg_qr` events → QR displayed
5. User scans with Telegram app
6. WebSocket receives `tg_auth_success` → Status updated, QR hidden

### QR Code Protection

- **Deduplication:** `lastQRUrl` tracking ignores duplicate QR tokens
- **Auto-regeneration:** Gotd library regenerates expired QRs automatically
- **No manual refresh:** QR expires after 30 seconds, user must retry

### Custom Event Listeners

```javascript
document.addEventListener("tg_qr", (e) => {
    const url = e.detail.url;
    if (url === lastQRUrl) return;  // Skip duplicates
    // Display QR code...
});

document.addEventListener("tg_auth_success", () => {
    // Hide QR, show "Connected"...
});

document.addEventListener("tg_auth_error", (e) => {
    // Show error...
});
```

## Changes from Original

| Issue | Fix |
|-------|-----|
| QR loop on page load | Removed `hx-trigger="load"` from button |
| Duplicate WebSocket | Removed local WS, use global from layout.html |
| Duplicate QR displays | Track `lastQRUrl` to ignore repeats |
| Multiple concurrent requests | 3-second button debounce |

## External Scripts

- `qrcode.min.js` — QR code rendering library
