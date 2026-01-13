# Task 01: Environment Cleanup

### 📝 What

Remove all traces of `TG_SESSION_STRING` from the application configuration and environment variables.

### 🎯 Why

To eliminate manual session handling and force the system to migrate to the database-only session model. This prevents "session cloning" errors and security leaks.

### 🛠 How

1.  **Code**: Identify and remove `TGSessionStr` field from `internal/config/config.go`.
2.  **Config**: Update `config.Load()` to stop looking for `TG_SESSION_STRING`.
3.  **Docker**: Remove `TG_SESSION_STRING: ${TG_SESSION_STRING}` from `docker-compose.yml`.
4.  **Env**: Remove `TG_SESSION_STRING` from `.env.example`.

### ✅ Acceptance Criteria

- [ ] Application compiles successfully.
- [ ] `config.Config` struct no longer contains `TGSessionStr`.
- [ ] Running `docker-compose up` doesn't pass the session string to the container.
- [ ] Logs show no warnings about missing session strings on startup.
