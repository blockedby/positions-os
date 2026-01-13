# layout.html

Base template for all pages — defines HTML structure, sidebar, and global WebSocket.

## Features

- Semantic HTML with `data-theme="dark"` attribute
- Pico.css CSS framework
- HTMX library for dynamic interactions
- Global WebSocket connection with auto-reconnect

## WebSocket

**Endpoint:** `/ws`

**Message Handling:**
- `job.updated` — Refreshes job row via HTMX
- `tg_qr` — Dispatches `tg_qr` custom event for QR code
- `tg_auth_success` — Dispatches `tg_auth_success` custom event
- `error` — Dispatches `tg_auth_error` custom event

**Auto-Reconnect:**
- Reconnects after 2 seconds on connection close
- Logs connection state changes

## Custom Event Dispatching

For QR authentication flow, messages are dispatched as custom DOM events:

```javascript
document.dispatchEvent(new CustomEvent("tg_qr", { detail: msg }));
document.dispatchEvent(new Event("tg_auth_success"));
document.dispatchEvent(new CustomEvent("tg_auth_error", { detail: msg }));
```

This allows pages (like `settings.html`) to listen for specific events without duplicate WebSocket connections.

## Structure

```
<body>
  <div class="flex h-screen">
    {{ template "sidebar" }}
    <main id="main-content">
      {{ block "content" }}{{ end }}
    </main>
  </div>
  <div id="toast-container"></div>
</body>
```
