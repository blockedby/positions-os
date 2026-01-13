# database.go

PostgreSQL connection management with dual-layer support.

- Wraps both `pgxpool.Pool` and `gorm.DB` for different use cases
- `New()` creates connection pool and GORM instance from DATABASE_URL
- `Ping()` checks database connectivity
- `Close()` shuts down the connection pool
- Used by repository layer for SQL operations
