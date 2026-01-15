# Фаза 3.5: Web UI Migration — Go Templates → React + Bun

## Обзор

Миграция с серверного рендеринга (Go Templates + HTMX) на современный SPA (React + Bun) для улучшения UX, developer experience и масштабируемости.

### Зачем мигрировать?

| Аспект          | Go Templates + HTMX               | React + Bun                        |
| --------------- | --------------------------------- | ---------------------------------- |
| Dev Experience  | Ограничен (шаблоны, мало libs)    | Богатая экосистема                 |
| Interactivity   | Ограничена HTMX атрибутами        | Полный контроль над UI             |
| State Management| Серверное состояние               | Client + Server sync               |
| Hot Reload      | Нет                               | Instant HMR                        |
| Type Safety     | Только на бэкенде                 | End-to-end TypeScript              |
| Performance     | Каждое действие = HTTP запрос     | Optimistic updates, local cache    |

---

## Tech Stack

### Core

| Компонент      | Выбор                        | Обоснование                           |
| -------------- | ---------------------------- | ------------------------------------- |
| Runtime        | **Bun**                      | 10-20x быстрее npm, встроенный bundler |
| Framework      | **React 19**                 | Latest: Compiler, Actions, Server Comps |
| Language       | **TypeScript 5.8**           | Type safety end-to-end                |
| Build          | **Bun Bundler**              | Zero-config, быстрый dev server        |
| Routing        | **React Router v7**          | Новое API с loaders/actions           |

### State & Data

| Компонент      | Выбор                        | Обоснование                           |
| -------------- | ---------------------------- | ------------------------------------- |
| Server State   | **TanStack Query v5**        | Caching, refetching, optimistic updates |
| Client State   | **Zustand**                  | Простой API, нет boilerplate          |
| Forms          | **React Hook Form**          | Минимальный ререндер, валидация        |
| Tables         | **TanStack Table v8**        | Virtualization, sorting, filtering    |

### UI & Styling

| Компонент      | Выбор                        | Обоснование                           |
| -------------- | ---------------------------- | ------------------------------------- |
| Components     | **shadcn/ui**                | Copy-paste, full customization        |
| Styling        | **Tailwind CSS v4**          | Utility-first, dark mode ready        |
| Icons          | **Lucide React**             | Tree-shakeable, consistent style       |
| Animations     | **Framer Motion**            | Declarative animations                 |

### Real-time

| Компонент      | Выбор                        | Обоснование                           |
| -------------- | ---------------------------- | ------------------------------------- |
| WebSocket      | **Native WebSocket + Hook**  | Простой API, автоматический reconnect  |
| Events         | **Event Emitter pattern**    | Pub/sub для real-time updates         |

### Code Quality

| Компонент      | Выбор                        | Обоснование                           |
| -------------- | ---------------------------- | ------------------------------------- |
| Linting        | **Biome**                    | 100x быстрее ESLint, форматирование    |
| Testing        | **Vitest**                   | Быстрее Jest, ESM-native               |
| E2E Testing    | **Playwright**               | Уже используется в проекте            |

---

## Архитектура

### Монорепо структура

```
positions-os/
├── cmd/
│   └── collector/          # Go backend (existing)
├── internal/              # Go backend (existing)
│   ├── web/
│   │   └── api/           # JSON API endpoints only
│   └── ...
├── web/                   # NEW: React frontend
│   ├── src/
│   │   ├── components/    # UI components
│   │   ├── features/      # Feature-based folders
│   │   ├── hooks/         # Custom hooks
│   │   ├── lib/           # Utilities, API client
│   │   ├── routes/        # Route definitions
│   │   ├── stores/        # Zustand stores
│   │   └── main.tsx       # Entry point
│   ├── public/
│   ├── index.html
│   ├── package.json
│   ├── tsconfig.json
│   └── vite.config.ts     # Or bun config
└── static/                # Served by Go backend
    └── assets/            # Built React app
```

---

## API Design

### Existing → New Mapping

| Old (HTMX)                          | New (React + JSON)                    |
| ----------------------------------- | ------------------------------------- |
| `GET /` → HTML                     | `GET /` → `index.html` (static)       |
| `GET /jobs` → HTML                 | `GET /api/v1/jobs` → JSON             |
| `GET /partials/jobs-table` → HTML  | `GET /api/v1/jobs?...` → JSON         |
| `PATCH /api/v1/jobs/:id/status`    | Same (already JSON)                   |
| WebSocket OOB swaps                | WebSocket JSON events                 |

---

## Migration Steps

### Phase 1: Setup

- [ ] Initialize Bun + React + TypeScript project in `web/`
- [ ] Setup Pico CSS with gently saving current layout
- [ ] Install and configure shadcn/ui
- [ ] Setup Biome for linting
- [ ] Configure Vitest for testing
- [ ] Setup CI/CD for builds

### Phase 2: Foundation

- [ ] Create layout components (Sidebar, Header)
- [ ] Setup React Router with routes
- [ ] Create API client with fetch wrapper
- [ ] Setup TanStack Query provider
- [ ] Create Zustand stores
- [ ] Implement WebSocket hook

### Phase 3: Dashboard

- [ ] Stats cards component
- [ ] Recent jobs component
- [ ] Connect to existing API

### Phase 4: Jobs Page

- [ ] Jobs filters component
- [ ] TanStack Table integration
- [ ] Job detail side panel
- [ ] Optimistic updates for status changes

### Phase 5: Settings

- [ ] Targets list component
- [ ] Target form with validation
- [ ] CRUD operations

### Phase 6: Integration

- [ ] Build pipeline
- [ ] Go server serves static files
- [ ] WebSocket integration testing
- [ ] E2E tests with Playwright

### Phase 7: Cutover

- [ ] Deploy to staging
- [ ] Final testing
- [ ] Production cutover
- [ ] Monitor and fix issues

---

## Build & Deployment

### Bun Config

```javascript
// web/bunfig.toml
[build]
publicDir = "public"
entrypoints = ["./src/main.tsx"]
outdir = "./dist"
target = "browser"
format = "esm"

[dev]
port = 3000
hmr = true
```

### Go Server Integration

```go
// internal/web/server.go
func (s *Server) setupRoutes() {
    // API routes
    s.router.Route("/api/v1", func(r chi.Router) {
        r.Get("/jobs", s.jobsHandler.List)
        // ...
    })

    // WebSocket
    s.router.Get("/ws", s.wsHandler.ServeWS)

    // Static file serving (React build)
    s.router.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
        if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/ws" {
            s.router.ServeHTTP(w, r)
            return
        }

        http.ServeFile(w, r, "./static/index.html")
    })
}
```

---

## Sources

- [Bun Docs - Build a React app](https://bun.com/docs/guides/ecosystem/react)
- [TanStack Table v8 Docs](https://tanstack.com/table/latest)
- [shadcn/ui Documentation](https://ui.shadcn.com)
- [React Router v7 Documentation](https://reactrouter.com)
- [TanStack Query Documentation](https://tanstack.com/query/latest)
