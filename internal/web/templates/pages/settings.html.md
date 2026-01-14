# settings.html

Settings page — Telegram authentication and target management.

## Features

- **Telegram Connection** card with status display
- **QR Code** display for Telegram login with localStorage persistence
- **Add Target** form for new scraping targets
- **Targets List** with inline editing

## Telegram Authentication Flow

### Initial State
- Checks `/api/v1/scrape/status` on load via `init()` function
- Shows "Connected" / "Disconnected" status
- Hides "Connect Telegram" button if already connected
- **NEW:** Attempts to restore previously saved QR code from localStorage

### QR Login
1. User clicks "Connect Telegram" button (or QR restored from localStorage on page load)
2. Button disabled for 3 seconds (debouncing)
3. QR code appears with 30-second countdown (shows remaining time if restored)
4. WebSocket receives `tg_qr` events → QR saved to localStorage and displayed
5. User scans with Telegram app
6. WebSocket receives `tg_auth_success` → Status updated, QR hidden, localStorage cleared

### QR Code Persistence (`QR_PERSISTENCE` module)

**Storage Keys:**
- `tg_qr_url` — The QR code login URL
- `tg_qr_timestamp` — Unix timestamp when QR was generated

**Methods:**
```javascript
QR_PERSISTENCE.save(url)           // Save QR URL with current timestamp
QR_PERSISTENCE.load()              // Load QR if not expired (30s), returns {url, ageSeconds} or null
QR_PERSISTENCE.clear()             // Remove QR from localStorage
```

**Behavior:**
- QR codes are saved to localStorage when received via WebSocket
- On page load, saved QR is displayed if less than 30 seconds old
- Expired QRs (>30s) are automatically cleared on load
- Successfully authenticated or error → localStorage cleared

### QR Code Protection

- **Deduplication:** `lastQRUrl` tracking ignores duplicate QR tokens
- **Persistence:** localStorage saves QR across page reloads
- **Auto-expiry:** QR codes expire after 30 seconds, timer shows remaining time
- **Clean on success/error:** localStorage cleared when auth completes or fails

### Custom Event Listeners

```javascript
document.addEventListener("tg_qr", (e) => {
    const url = e.detail.url;
    if (url === lastQRUrl) return;  // Skip duplicates
    QR_PERSISTENCE.save(url);        // Save to localStorage
    displayQR(url, 30);              // Display with full timer
});

document.addEventListener("tg_auth_success", () => {
    QR_PERSISTENCE.clear();          // Clear saved QR
    // Hide QR, show "Connected"...
});

document.addEventListener("tg_auth_error", (e) => {
    QR_PERSISTENCE.clear();          // Clear saved QR
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
| QR lost on page reload | **NEW:** localStorage persistence with expiry check |

## External Scripts

- `qrcode.min.js` — QR code rendering library (loaded from unpkg.com)
