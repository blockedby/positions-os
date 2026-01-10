# Telegram Session Invalidation Report for Positions-OS

## 🚨 Core Problem Analysis

Based on your logs (exit status `0xc000013a`, "Enter Phone Number" prompt) and web research on `gotd/gotgproto` beta issues, here are the most likely causes for your constant unlogin events.

### 1. ⚠️ Critical: TDesktop Session Conflict (Most Likely)

Your logs show you authorized using **Option 1: Use Telegram Desktop Session**.

```
detected 2 telegram desktop session(s) at: D:\Telegram Desktop\tdata
enter choice [1]: 1
```

**The Problem:** When you convert a TDesktop session to a Go session, `gotgproto` might be either:

1.  **Cloning** the session exact parameters (App Version, Device Model, etc.). If you run **BOTH** Telegram Desktop and the Collector simultaneously with the exact same session identity, Telegram's server anti-abuse system detects two identical instances from the same IP/Device and frequently kills the "older" one (or the bot) to prevent "session hijacking".
2.  **Session File Locking**: If TDesktop is running, it locks the `tdata` files. If the auth tool tried to read them while locked, the generated string might be partial or corrupted, leading to a session that works for 5 minutes (until next re-keying) and then dies.

**Solution:**

- **Stop TDesktop** while generating the session string.
- OR, preferably, **Use Option 2 (SMS Code)** to generate a _fresh, independent session_ specifically for the bot. This creates a separate "scan" in the Active Sessions list which can coexist with your Desktop app.

### 2. beta Library Issues (`gotgproto`)

`gotgproto` is in beta. Research indicates it sometimes struggles with:

- **Session Persistence**: If the library updates the auth key (re-keying happens periodically) but fails to print/save the _updated_ session string back to your `.env` (which it cannot do automatically), the next time you restart, you are using an _old_ key. Telegram rejects old keys immediately.
- **"Zombie" State**: The logs show the app running for ~6 minutes (`01:53` -> `01:59`). This matches a typical key-refresh cycle. If the refresh fails or isn't saved, the session dies.

### 3. Rate Limiting "Flood Wait"

Even "without parsing", the library connects and fetches "State" (Updates).

- If your `App ID` / `App Hash` is new, Telegram treats it with suspicion.
- If the library tries to fetch "dialogs" or "peers" too aggressively on startup (e.g. `ResolveChannel` calls on init), you get hit by `FLOOD_WAIT`.
- **Fatal Error**: `gotd` often treats global flood waits as fatal errors, killing the connection.

### 4. "Enter Phone Number" Behavior

This text appearing in your logs means the `SessionString` provided in `.env` was rejected by Telegram (Invalid/Revoked) on startup or during a refresh. The library's default fallback behavior is "Interactive Login" -> it tries to ask you for a phone number on the console. Since you are running in a non-interactive mode (or just running `go run`), it crashes or hangs.

---

## 🛠 Action Plan / Fixes

### Step 1: Generate a Clean, Independent Session (Recommended)

Do not clone the Desktop session. Create a dedicated session for the bot.

1.  Run `go run cmd/tg-auth/main.go`.
2.  Choose **Option 2** (Authenticate with phone number).
3.  Enter your phone number.
4.  Enter the code sent to your Telegram App.
5.  **Copy the NEW session string** to your `.env`.

This separates your Bot's identity from your Desktop App's identity.

### Step 2: Disable Interactive Fallback

In your code (`internal/telegram/client.go` or `collector/main.go`), ensure you are NOT enabling interactive login if the session is dead. It's better to Crash explicitly than hang asking for input.
_Currently, `gotgproto` might be defaulting to interactive._

### Step 3: Check "Active Sessions" in Telegram

1.  Open Telegram on your Phone/Desktop.
2.  Go to **Settings -> Privacy and Security -> Devices**.
3.  Look for the session created by your bot (it might say "Go Application" or similar).
4.  If you see it appearing and disappearing, Telegram is actively killing it (likely due to Conflict #1).

### Step 4: Use the Rate Limiter (Already Implemented)

You correctly asked for this. The 2 RPS limit we added in Phase 2 should prevent `FLOOD_WAIT` bans once the session is stable.

---

**Summary:** The core problem is likely **Session Cloning conflict** with your main TDesktop app. Switch to SMS-based auth for the bot to fix it.
