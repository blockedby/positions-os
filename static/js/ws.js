function connectHelper() {
  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  const ws = new WebSocket(`${protocol}//${window.location.host}/ws`);

  ws.onopen = function () {
    console.log("Connected to WebSocket");
    // Reset retry delay or something
  };

  ws.onmessage = function (event) {
    // We expect JSON messages to trigger HTMX updates
    // Format: { "type": "job_update", "job_id": "...", "status": "..." }
    // Or simple event triggering
    try {
      const msg = JSON.parse(event.data);
      console.log("Received WS message:", msg);

      // Dispatch event for HTMX
      if (msg.event) {
        document.body.dispatchEvent(new Event(msg.event));
      }

      // We can also trigger specific element updates if needed
    } catch (e) {
      console.error("Error parsing WS message:", e);
    }
  };

  ws.onclose = function () {
    console.log("WebSocket closed. Reconnecting...");
    setTimeout(connectHelper, 3000);
  };

  ws.onerror = function (err) {
    console.error("WebSocket error:", err);
    ws.close();
  };
}

// Start connection
document.addEventListener("DOMContentLoaded", connectHelper);
