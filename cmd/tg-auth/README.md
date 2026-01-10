# Telegram QR Authentication Tool

A simple command-line tool to generate Telegram session strings via QR code authentication.

## Features

✅ **QR Code Authentication** - Scan with your Telegram app  
✅ **Environment Variables** - Auto-loads from `.env` with confirmation  
✅ **Auto-Retry** - Automatically generates new QR if it expires  
✅ **Graceful Shutdown** - Press `Ctrl+C` to cancel  
✅ **Session Export** - Compatible with `gotgproto` format

## Prerequisites

1. **Telegram API Credentials**  
   Get your `API_ID` and `API_HASH` from [my.telegram.org](https://my.telegram.org/apps)

2. **Go Environment**  
   Go 1.21+ installed

## Setup

### 1. Add credentials to `.env` (optional)

```env
TG_API_ID=your_api_id
TG_API_HASH=your_api_hash
```

### 2. Build the tool

```bash
go build -o tg-auth ./cmd/tg-auth
```

## Usage

### Run the tool

```bash
./tg-auth
```

### Interactive Flow

1. **Confirm API Credentials** (if found in `.env`)

   ```
   Found TG_API_ID in .env: 12345678
   Press Y to accept or enter a different value: Y

   Found TG_API_HASH in .env: abc****xyz
   Press Y to accept or enter a different value: Y
   ```

2. **Scan QR Code**

   ```
   ╔═══════════════════════════════════════════════════════╗
   ║  SCAN THIS QR CODE WITH YOUR TELEGRAM APP            ║
   ║  Settings → Devices → Link Desktop Device            ║
   ║  Expires in: 30s                                      ║
   ╚═══════════════════════════════════════════════════════╝

   [QR CODE DISPLAYED HERE]

   Waiting for scan...
   ```

3. **Get Session String**

   ```
   ╔═══════════════════════════════════════════════════════╗
   ║           ✓ AUTHENTICATION SUCCESSFUL!               ║
   ╚═══════════════════════════════════════════════════════╝

   Logged in as: @your_username

   ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   YOUR SESSION STRING:
   ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   eyJWZXJzaW9uIjoxLCJEYXRhIjoi...
   ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

   Add this to your .env file as TG_SESSION_STRING

   ⚠️  KEEP THIS SECRET! It provides full access to your account.
   ```

4. **Add to `.env`**
   ```env
   TG_SESSION_STRING="eyJWZXJzaW9uIjoxLCJEYXRhIjoi..."
   ```

## Features Explained

### Auto-Retry on QR Expiration

If the QR code expires (30 seconds), a new one is automatically generated:

```
⚠️  QR code expired. Generating a new one...

[NEW QR CODE DISPLAYED]
```

### Graceful Shutdown

Press `Ctrl+C` at any time to cancel:

```
^C
Authentication cancelled by user.
```

### Environment Variable Confirmation

The tool asks for confirmation before using values from `.env`:

- Press `Y` or `Enter` to accept
- Type a different value to override

## Troubleshooting

### "QR auth failed"

- Make sure you're scanning with the **official Telegram app**
- Check that you have a stable internet connection
- Verify API credentials are correct

### "Invalid API ID"

- Make sure `TG_API_ID` is a number
- Get valid credentials from [my.telegram.org](https://my.telegram.org/apps)

### Session not working in your app

- Make sure you're using the **full base64 string** including quotes
- The format is compatible with `gotgproto` - see [telegram-qr-auth-guide.md](../../docs/telegram-qr-auth-guide.md)

## Security Notes

⚠️ **NEVER share your session string!**

- It provides **full access** to your Telegram account
- Anyone with this string can:
  - Read your messages
  - Send messages as you
  - Access all chats and channels
- Store it securely in `.env` files that are **never committed to git**
- Add `.env` to your `.gitignore`

## Technical Details

- Uses `gotd/td` for low-level Telegram API
- Exports sessions in `gotgproto` format
- Handles DC migration automatically
- Session data is stored in-memory during auth only

For implementation details, see:

- [telegram-qr-auth-guide.md](../../docs/telegram-qr-auth-guide.md) - QR auth guide
- [tg-auth-code-review.md](../../docs/tg-auth-code-review.md) - Code review
