# Фаза 3: Web UI — План Реализации

## Обзор

Web UI — это единый интерфейс для управления системой. На выходе:

- Просмотр и фильтрация вакансий
- Действия: Interested / Reject
- Управление источниками (targets)
- Real-time обновления через WebSocket
- Dashboard со статистикой

---

## Принятые решения

| Аспект      | Решение                   | Обоснование                            |
| ----------- | ------------------------- | -------------------------------------- |
| Frontend    | **HTMX + Go Templates**   | Нет Node.js, всё в Go                  |
| Архитектура | **Единый сервис**         | Collector + API + UI вместе            |
| Real-time   | **WebSocket**             | Двунаправленная связь                  |
| Пагинация   | **Server-side**           | С серверной сортировкой                |
| Фильтры     | **Комбинированные**       | Статус + технологии + зарплата + поиск |
| Детали      | **Side panel**            | Разворачивается справа от списка       |
| Actions     | **Кнопки в таблице**      | Быстрые действия без перехода          |
| Targets     | **Settings tab**          | Простые формы с комментариями          |
| Навигация   | **Single page + Sidebar** | Dashboard как одна из вкладок          |
| Тема        | **Dark only**             | Для разработчиков                      |

---

## Архитектура

```
┌──────────────────────────────────────────────────────────────────────┐
│                         Browser (HTMX)                                │
│  ┌────────────────────────────────────────────────────────────────┐  │
│  │  ┌──────────┐  ┌─────────────────────────────────────────────┐ │  │
│  │  │          │  │                                             │ │  │
│  │  │ Sidebar  │  │              Main Content                   │ │  │
│  │  │          │  │  ┌──────────────────┬───────────────────┐   │ │  │
│  │  │ Dashboard│  │  │                  │                   │   │ │  │
│  │  │ Jobs     │  │  │   Jobs Table     │   Detail Panel    │   │ │  │
│  │  │ Settings │  │  │   (filterable)   │   (expandable)    │   │ │  │
│  │  │          │  │  │                  │                   │   │ │  │
│  │  └──────────┘  │  └──────────────────┴───────────────────┘   │ │  │
│  └────────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────────┘
                               │
                               │ HTTP + WebSocket
                               ▼
┌──────────────────────────────────────────────────────────────────────┐
│                    Unified Service (Go)                              │
│  ┌────────────────────────────────────────────────────────────────┐  │
│  │                         HTTP Server                            │  │
│  │  ┌───────────────┐  ┌───────────────┐  ┌───────────────────┐  │  │
│  │  │  API Routes   │  │ HTML Routes   │  │  WebSocket Hub    │  │  │
│  │  │  /api/v1/*    │  │  /           │  │  /ws              │  │  │
│  │  └───────────────┘  └───────────────┘  └───────────────────┘  │  │
│  └────────────────────────────────────────────────────────────────┘  │
│  ┌────────────────────────────────────────────────────────────────┐  │
│  │                    Business Logic                              │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐            │  │
│  │  │  Collector  │  │  Repository │  │  Notifier   │            │  │
│  │  │  (scraper)  │  │  (DB ops)   │  │  (WS push)  │            │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘            │  │
│  └────────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────────┘
               │                    │                    │
               ▼                    ▼                    ▼
        ┌──────────┐         ┌──────────┐         ┌──────────┐
        │ Telegram │         │PostgreSQL│         │   NATS   │
        │   API    │         │          │         │          │
        └──────────┘         └──────────┘         └──────────┘
```

---

## 🎨 UI Layout

### Sidebar Navigation

```
┌─────────────────┐
│ 🏠 Dashboard    │  ← статистика, графики
├─────────────────┤
│ 💼 Jobs         │  ← список вакансий
├─────────────────┤
│ ⚙️ Settings     │  ← targets, config
└─────────────────┘
```

### Jobs Page Layout

```
┌─────────────────────────────────────────────────────────────────────────┐
│ Filters: [Status ▼] [Technologies ▼] [Salary: ___-___] [🔍 Search...]   │
├────────────────────────────────────────────┬────────────────────────────┤
│                                            │                            │
│  □ Title           Company    Salary   Act │  Selected Job Details      │
│  ─────────────────────────────────────────│                            │
│  ● Go Developer    Yandex     250-350k ✓✗ │  📋 Go Developer           │
│  ○ Backend Eng     TechCorp   $120k    ✓✗ │                            │
│  ○ Python Dev      Sber       200-300k ✓✗ │  Company: Yandex           │
│  ○ ...                                     │  Location: Remote          │
│                                            │  Salary: 250,000-350,000 ₽ │
│  [← 1 2 3 4 5 →]                          │                            │
│                                            │  Technologies:             │
│                                            │  [go] [postgresql] [k8s]   │
│                                            │                            │
│                                            │  --- Raw Content ---       │
│                                            │  Ищем Go разработчика...   │
│                                            │                            │
│                                            │  Contact: @recruiter       │
│                                            │                            │
│                                            │  [✓ Interested] [✗ Reject] │
└────────────────────────────────────────────┴────────────────────────────┘
```

### Settings Page

```
┌─────────────────────────────────────────────────────────────────────────┐
│ Settings                                                                 │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│ Scraping Targets                                                        │
│ ─────────────────────────────────────────────────────────────────────── │
│                                                                         │
│ Name: [Go Jobs Channel          ]  # человекочитаемое название          │
│ Type: [TG_CHANNEL ▼]               # тип источника                      │
│ URL:  [@golang_jobs             ]  # username канала                    │
│                                                                         │
│ Topic IDs: [15, 28              ]  # только для TG_FORUM               │
│                                     # ID топиков через запятую          │
│                                                                         │
│ Active: [✓]                        # парсить этот источник              │
│                                                                         │
│ [Save] [Delete]                                                         │
│                                                                         │
│ ─────────────────────────────────────────────────────────────────────── │
│ + Add New Target                                                        │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 🔌 WebSocket Integration

### Hub (Connection Manager)

```go
// internal/web/ws/hub.go
package ws

import (
    "sync"

    "github.com/gorilla/websocket"
)

// Hub manages WebSocket connections.
type Hub struct {
    // registered clients
    clients map[*Client]bool

    // inbound messages from clients
    broadcast chan []byte

    // register requests
    register chan *Client

    // unregister requests
    unregister chan *Client

    mu sync.RWMutex
}

// Client represents a WebSocket connection.
type Client struct {
    hub  *Hub
    conn *websocket.Conn
    send chan []byte
}

// NewHub creates a new Hub.
func NewHub() *Hub {
    return &Hub{
        broadcast:  make(chan []byte),
        register:   make(chan *Client),
        unregister: make(chan *Client),
        clients:    make(map[*Client]bool),
    }
}

// Run starts the hub's event loop.
func (h *Hub) Run() {
    for {
        select {
        case client := <-h.register:
            h.mu.Lock()
            h.clients[client] = true
            h.mu.Unlock()

        case client := <-h.unregister:
            h.mu.Lock()
            if _, ok := h.clients[client]; ok {
                delete(h.clients, client)
                close(client.send)
            }
            h.mu.Unlock()

        case message := <-h.broadcast:
            h.mu.RLock()
            for client := range h.clients {
                select {
                case client.send <- message:
                default:
                    close(client.send)
                    delete(h.clients, client)
                }
            }
            h.mu.RUnlock()
        }
    }
}

// Broadcast sends a message to all connected clients.
func (h *Hub) Broadcast(message []byte) {
    h.broadcast <- message
}
```

### Events

```go
// internal/web/ws/events.go
package ws

// Event types for WebSocket messages.
const (
    EventJobNew      = "job.new"      // новая вакансия добавлена
    EventJobUpdated  = "job.updated"  // вакансия обновлена (статус)
    EventScrapeStart = "scrape.start" // парсинг начался
    EventScrapeEnd   = "scrape.end"   // парсинг завершён
)

// WSEvent представляет событие для WebSocket.
type WSEvent struct {
    Type    string      `json:"type"`
    Payload interface{} `json:"payload"`
}

// JobNewPayload данные о новой вакансии.
type JobNewPayload struct {
    JobID string `json:"job_id"`
    Title string `json:"title"`
}

// JobUpdatedPayload данные об обновлении.
type JobUpdatedPayload struct {
    JobID  string `json:"job_id"`
    Status string `json:"status"`
}
```

### HTMX Integration

```html
<!-- templates/layout.html -->
<body hx-ext="ws" ws-connect="/ws">
  <!-- WebSocket подключается автоматически -->

  <!-- контент обновляется через OOB (Out of Band) swap -->
  <div id="jobs-table">
    <!-- таблица вакансий -->
  </div>

  <div id="notifications">
    <!-- уведомления появляются здесь -->
  </div>
</body>
```

```go
// при новой вакансии отправляем HTML snippet для OOB swap
func (h *Hub) NotifyNewJob(job *models.Job) {
    html := renderJobRow(job) // рендерим строку таблицы

    message := fmt.Sprintf(`
        <tr id="job-%s" hx-swap-oob="afterbegin:#jobs-tbody">
            %s
        </tr>
    `, job.ID, html)

    h.Broadcast([]byte(message))
}
```

---

## 📁 Структура файлов

```
positions-os/
├── cmd/
│   └── collector/
│       └── main.go              # Entry point (Collector + Web)
├── internal/
│   ├── web/
│   │   ├── server.go            # HTTP server setup
│   │   ├── router.go            # Chi router, routes
│   │   ├── handlers/
│   │   │   ├── pages.go         # HTML page handlers
│   │   │   ├── jobs.go          # Jobs API + partials
│   │   │   ├── targets.go       # Targets CRUD
│   │   │   └── scrape.go        # Scrape controls
│   │   ├── ws/
│   │   │   ├── hub.go           # WebSocket hub
│   │   │   ├── client.go        # WebSocket client
│   │   │   └── events.go        # Event types
│   │   └── templates/
│   │       ├── layout.html      # Base layout
│   │       ├── sidebar.html     # Navigation
│   │       ├── pages/
│   │       │   ├── dashboard.html
│   │       │   ├── jobs.html
│   │       │   └── settings.html
│   │       └── partials/
│   │           ├── job_row.html
│   │           ├── job_detail.html
│   │           ├── filter_bar.html
│   │           └── target_form.html
│   ├── collector/               # существующий код
│   └── repository/              # существующий код
├── static/
│   ├── css/
│   │   └── style.css            # Tailwind build (dark theme)
│   └── js/
│       └── htmx.min.js          # HTMX library
└── templates/                   # symlink to internal/web/templates
```

---

## 🎨 Tailwind CSS (Dark Only)

### tailwind.config.js

```javascript
module.exports = {
  content: ["./internal/web/templates/**/*.html"],
  darkMode: "class", // не используем, но оставляем
  theme: {
    extend: {
      colors: {
        // основные цвета
        "bg-primary": "#0d1117",
        "bg-secondary": "#161b22",
        "bg-tertiary": "#21262d",
        border: "#30363d",
        "text-primary": "#c9d1d9",
        "text-secondary": "#8b949e",
        accent: "#58a6ff",
        success: "#3fb950",
        danger: "#f85149",
        warning: "#d29922",
      },
    },
  },
  plugins: [],
};
```

### Base styles

```css
/* static/css/base.css */
body {
  @apply bg-bg-primary text-text-primary;
}

.sidebar {
  @apply bg-bg-secondary border-r border-border;
}

.card {
  @apply bg-bg-tertiary border border-border rounded-lg;
}

.btn-primary {
  @apply bg-accent hover:bg-accent/80 text-white px-4 py-2 rounded;
}

.btn-success {
  @apply bg-success hover:bg-success/80 text-white px-4 py-2 rounded;
}

.btn-danger {
  @apply bg-danger hover:bg-danger/80 text-white px-4 py-2 rounded;
}

.table-row {
  @apply border-b border-border hover:bg-bg-tertiary cursor-pointer;
}

.table-row.selected {
  @apply bg-bg-tertiary;
}
```

---

## 🔀 Routes

### HTML Routes

| Route           | Handler           | Description    |
| --------------- | ----------------- | -------------- |
| `GET /`         | `pages.Dashboard` | Dashboard page |
| `GET /jobs`     | `pages.Jobs`      | Jobs list page |
| `GET /settings` | `pages.Settings`  | Settings page  |

### API Routes

| Route                           | Handler             | Description        |
| ------------------------------- | ------------------- | ------------------ |
| `GET /api/v1/jobs`              | `jobs.List`         | Список с фильтрами |
| `GET /api/v1/jobs/:id`          | `jobs.Get`          | Детали вакансии    |
| `PATCH /api/v1/jobs/:id/status` | `jobs.UpdateStatus` | Изменить статус    |
| `GET /api/v1/targets`           | `targets.List`      | Список источников  |
| `POST /api/v1/targets`          | `targets.Create`    | Создать источник   |
| `PUT /api/v1/targets/:id`       | `targets.Update`    | Обновить           |
| `DELETE /api/v1/targets/:id`    | `targets.Delete`    | Удалить            |
| `POST /api/v1/scrape/telegram`  | `scrape.Start`      | Запустить парсинг  |
| `DELETE /api/v1/scrape/current` | `scrape.Stop`       | Остановить         |
| `GET /api/v1/scrape/status`     | `scrape.Status`     | Статус             |

### HTMX Partials

| Route                            | Returns | Description         |
| -------------------------------- | ------- | ------------------- |
| `GET /partials/jobs-table`       | HTML    | Таблица с фильтрами |
| `GET /partials/job-detail/:id`   | HTML    | Side panel          |
| `GET /partials/target-form/:id?` | HTML    | Форма target        |

### WebSocket

| Route     | Description        |
| --------- | ------------------ |
| `GET /ws` | WebSocket endpoint |

---

## 🔍 Фильтрация

### Query Parameters

```
GET /api/v1/jobs?status=ANALYZED&tech=go,postgresql&salary_min=200000&q=backend&sort=created_at&order=desc&page=1&limit=20
```

| Param        | Type   | Description                         |
| ------------ | ------ | ----------------------------------- |
| `status`     | string | RAW, ANALYZED, INTERESTED, REJECTED |
| `tech`       | string | Comma-separated technologies        |
| `salary_min` | int    | Минимальная зарплата                |
| `salary_max` | int    | Максимальная зарплата               |
| `q`          | string | Полнотекстовый поиск                |
| `sort`       | string | Поле сортировки                     |
| `order`      | string | asc / desc                          |
| `page`       | int    | Номер страницы                      |
| `limit`      | int    | Записей на странице                 |

### SQL Query Builder

```go
// internal/repository/jobs.go
func (r *JobsRepository) List(ctx context.Context, filter JobFilter) ([]models.Job, int, error) {
    var conditions []string
    var args []interface{}
    argNum := 1

    if filter.Status != "" {
        conditions = append(conditions, fmt.Sprintf("status = $%d", argNum))
        args = append(args, filter.Status)
        argNum++
    }

    if len(filter.Technologies) > 0 {
        // JSONB array contains
        conditions = append(conditions, fmt.Sprintf(
            "structured_data->'technologies' ?| $%d", argNum))
        args = append(args, pq.Array(filter.Technologies))
        argNum++
    }

    if filter.SalaryMin > 0 {
        conditions = append(conditions, fmt.Sprintf(
            "(structured_data->>'salary_min')::int >= $%d", argNum))
        args = append(args, filter.SalaryMin)
        argNum++
    }

    if filter.Query != "" {
        conditions = append(conditions, fmt.Sprintf(
            "raw_content ILIKE $%d", argNum))
        args = append(args, "%"+filter.Query+"%")
        argNum++
    }

    // build query...
}
```

---

## 🎯 Порядок реализации (TDD)

> **Принцип**: Для каждого этапа сначала пишем тесты, которые падают (Red), затем реализацию (Green), затем рефакторим (Refactor).

### Этап 1: Server Setup (3.1) [COMPLETED]

**Цель**: Создать базовый HTTP сервер с Chi router, middleware, статикой и шаблонизатором.

**Зависимости**:

- `github.com/go-chi/chi/v5`
- `github.com/go-chi/chi/v5/middleware`

**Файлы для создания**:

```
internal/web/
├── server.go           # Server struct, Start/Stop
├── router.go           # Chi router configuration
├── templates.go        # Template engine wrapper
└── server_test.go      # Integration tests
static/
├── css/style.css       # Compiled Tailwind
└── js/htmx.min.js      # HTMX library
```

---

#### 3.1.1 — HTTP Server Bootstrap

**Test**: `TestServer_Starts`

```go
func TestServer_Starts(t *testing.T) {
    cfg := &Config{Port: 0} // random port
    srv := NewServer(cfg, nil, nil)

    go srv.Start()
    defer srv.Stop(context.Background())

    // wait for server to be ready
    require.Eventually(t, func() bool {
        resp, err := http.Get(srv.BaseURL() + "/health")
        return err == nil && resp.StatusCode == 200
    }, 2*time.Second, 100*time.Millisecond)
}
```

**Implementation** (`internal/web/server.go`):

- `Server` struct с `*chi.Mux`, `*http.Server`, `Config`
- `NewServer(cfg, repo, hub)` — конструктор с DI
- `Start()` — запуск `http.ListenAndServe`
- `Stop(ctx)` — graceful shutdown
- `BaseURL()` — возвращает `http://localhost:{port}`

**Acceptance Criteria**:

- [ ] Сервер запускается на указанном порту
- [ ] Graceful shutdown работает (ждёт активные запросы)
- [ ] Логирует startup/shutdown события

---

#### 3.1.2 — Static File Serving

**Test**: `TestServer_ServesStatic`

```go
func TestServer_ServesStatic(t *testing.T) {
    srv := setupTestServer(t)

    resp, err := http.Get(srv.BaseURL() + "/static/css/style.css")
    require.NoError(t, err)
    defer resp.Body.Close()

    assert.Equal(t, http.StatusOK, resp.StatusCode)
    assert.Contains(t, resp.Header.Get("Content-Type"), "text/css")
}
```

**Implementation** (`internal/web/router.go`):

```go
func (s *Server) setupRoutes() {
    // Static files with cache headers
    fileServer := http.FileServer(http.Dir("./static"))
    s.router.Handle("/static/*", http.StripPrefix("/static/", fileServer))
}
```

**Acceptance Criteria**:

- [ ] `/static/css/style.css` возвращает CSS
- [ ] `/static/js/htmx.min.js` возвращает JS
- [ ] Cache-Control headers установлены
- [ ] 404 для несуществующих файлов

---

#### 3.1.3 — Template Engine

**Test**: `TestServer_RendersTemplate`

```go
func TestServer_RendersTemplate(t *testing.T) {
    srv := setupTestServer(t)

    resp, err := http.Get(srv.BaseURL() + "/")
    require.NoError(t, err)
    defer resp.Body.Close()

    body, _ := io.ReadAll(resp.Body)
    assert.Equal(t, http.StatusOK, resp.StatusCode)
    assert.Contains(t, string(body), "<!DOCTYPE html>")
    assert.Contains(t, string(body), "Dashboard")
}
```

**Implementation** (`internal/web/templates.go`):

```go
type TemplateEngine struct {
    templates *template.Template
    reload    bool // dev mode: reload on each request
}

func NewTemplateEngine(templatesDir string, reload bool) *TemplateEngine {
    // Parse all templates with glob
    // Support partials: {{ template "partials/job_row" . }}
}

func (te *TemplateEngine) Render(w io.Writer, name string, data interface{}) error
```

**Template Structure**:

```
internal/web/templates/
├── layout.html         # {{ define "layout" }}...{{ end }}
├── sidebar.html        # {{ define "sidebar" }}...{{ end }}
└── pages/
    └── dashboard.html  # {{ template "layout" . }}
```

**Acceptance Criteria**:

- [ ] Templates парсятся при старте
- [ ] Layout inheritance работает
- [ ] Partials можно включать
- [ ] Dev mode: hot reload templates
- [ ] Ошибки рендеринга логируются

---

#### 3.1.4 — Health Endpoint

**Test**: `TestServer_HealthEndpoint`

```go
func TestServer_HealthEndpoint(t *testing.T) {
    srv := setupTestServer(t)

    resp, err := http.Get(srv.BaseURL() + "/health")
    require.NoError(t, err)
    defer resp.Body.Close()

    assert.Equal(t, http.StatusOK, resp.StatusCode)

    var health struct {
        Status   string `json:"status"`
        Database string `json:"database"`
    }
    json.NewDecoder(resp.Body).Decode(&health)
    assert.Equal(t, "ok", health.Status)
}
```

**Implementation**:

```go
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
    health := map[string]string{
        "status":   "ok",
        "database": h.checkDB(),
        "version":  version.Version,
    }
    json.NewEncoder(w).Encode(health)
}
```

**Acceptance Criteria**:

- [ ] `/health` возвращает JSON
- [ ] Проверяет подключение к БД
- [ ] Возвращает версию приложения

---

#### 3.1.5 — Chi Middleware Stack

**Implementation** (`internal/web/router.go`):

```go
func (s *Server) setupMiddleware() {
    s.router.Use(middleware.RequestID)
    s.router.Use(middleware.RealIP)
    s.router.Use(middleware.Logger)      // request logging
    s.router.Use(middleware.Recoverer)   // panic recovery
    s.router.Use(middleware.Timeout(30 * time.Second))
    s.router.Use(middleware.Compress(5)) // gzip
}
```

**Acceptance Criteria**:

- [ ] Request ID в каждом запросе
- [ ] Логирование всех запросов
- [ ] Panic не крашит сервер
- [ ] Gzip для text/html, application/json

### Этап 2: Layout & Navigation (3.2) [COMPLETED]

**Цель**: Создать SPA-like навигацию с HTMX, sidebar и тёмной темой.

**Файлы для создания**:

```
internal/web/templates/
├── layout.html          # Base layout с sidebar
├── sidebar.html         # Navigation partial
└── pages/
    ├── dashboard.html   # Dashboard content
    ├── jobs.html        # Jobs list content
    └── settings.html    # Settings content
internal/web/handlers/
└── pages.go             # Page handlers
```

---

#### 3.2.1 — Base Layout Template

**Test**: `TestLayout_ContainsSidebar`

```go
func TestLayout_ContainsSidebar(t *testing.T) {
    srv := setupTestServer(t)

    resp, err := http.Get(srv.BaseURL() + "/")
    require.NoError(t, err)

    body, _ := io.ReadAll(resp.Body)
    html := string(body)

    assert.Contains(t, html, `id="sidebar"`)
    assert.Contains(t, html, `id="main-content"`)
    assert.Contains(t, html, "Dashboard")
    assert.Contains(t, html, "Jobs")
    assert.Contains(t, html, "Settings")
}
```

**Implementation** (`templates/layout.html`):

```html
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <title>{{ .Title }} | Job Hunter OS</title>
    <link rel="stylesheet" href="/static/css/style.css" />
    <script src="/static/js/htmx.min.js"></script>
    <script src="/static/js/htmx-ws.js"></script>
  </head>
  <body class="bg-bg-primary text-text-primary" hx-ext="ws" ws-connect="/ws">
    <div class="flex h-screen">
      {{ template "sidebar" . }}
      <main id="main-content" class="flex-1 overflow-auto p-6">
        {{ template "content" . }}
      </main>
    </div>
  </body>
</html>
```

**Acceptance Criteria**:

- [ ] Layout рендерится с sidebar и main content
- [ ] Title устанавливается динамически
- [ ] HTMX подключён и инициализирован
- [ ] WebSocket подключается при загрузке

---

#### 3.2.2 — Page Navigation

**Test**: `TestNavigation_AllPagesLoad`

```go
func TestNavigation_AllPagesLoad(t *testing.T) {
    srv := setupTestServer(t)

    pages := []struct {
        path     string
        contains string
    }{
        {"/", "Dashboard"},
        {"/jobs", "Jobs"},
        {"/settings", "Settings"},
    }

    for _, p := range pages {
        t.Run(p.path, func(t *testing.T) {
            resp, err := http.Get(srv.BaseURL() + p.path)
            require.NoError(t, err)
            assert.Equal(t, http.StatusOK, resp.StatusCode)

            body, _ := io.ReadAll(resp.Body)
            assert.Contains(t, string(body), p.contains)
        })
    }
}
```

**Implementation** (`internal/web/handlers/pages.go`):

```go
type PagesHandler struct {
    templates *TemplateEngine
    repo      *repository.JobsRepository
}

func (h *PagesHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
    data := PageData{
        Title:      "Dashboard",
        ActivePage: "dashboard",
    }
    h.templates.Render(w, "pages/dashboard", data)
}

func (h *PagesHandler) Jobs(w http.ResponseWriter, r *http.Request) {
    data := PageData{
        Title:      "Jobs",
        ActivePage: "jobs",
    }
    h.templates.Render(w, "pages/jobs", data)
}

func (h *PagesHandler) Settings(w http.ResponseWriter, r *http.Request) {
    data := PageData{
        Title:      "Settings",
        ActivePage: "settings",
    }
    h.templates.Render(w, "pages/settings", data)
}
```

**Acceptance Criteria**:

- [ ] `/` → Dashboard page
- [ ] `/jobs` → Jobs page
- [ ] `/settings` → Settings page
- [ ] Active page highlighted in sidebar

---

#### 3.2.3 — HTMX Partial Responses

**Test**: `TestNavigation_HTMXPartialResponse`

```go
func TestNavigation_HTMXPartialResponse(t *testing.T) {
    srv := setupTestServer(t)

    // Request with HX-Request header (HTMX request)
    req, _ := http.NewRequest("GET", srv.BaseURL()+"/jobs", nil)
    req.Header.Set("HX-Request", "true")

    resp, err := http.DefaultClient.Do(req)
    require.NoError(t, err)

    body, _ := io.ReadAll(resp.Body)
    html := string(body)

    // Should NOT contain full layout
    assert.NotContains(t, html, "<!DOCTYPE html>")
    assert.NotContains(t, html, "<head>")

    // Should contain only content
    assert.Contains(t, html, "Jobs")
}
```

**Implementation**:

```go
func (h *PagesHandler) Jobs(w http.ResponseWriter, r *http.Request) {
    data := PageData{Title: "Jobs", ActivePage: "jobs"}

    // Check if HTMX request
    if r.Header.Get("HX-Request") == "true" {
        // Return only content partial
        h.templates.Render(w, "pages/jobs_content", data)
        return
    }

    // Return full page with layout
    h.templates.Render(w, "pages/jobs", data)
}
```

**Sidebar HTMX navigation** (`templates/sidebar.html`):

```html
<nav id="sidebar" class="w-64 bg-bg-secondary border-r border-border">
    <div class="p-4">
        <h1 class="text-xl font-bold text-accent">Job Hunter OS</h1>
    </div>
    <ul class="space-y-2 p-4">
        <li>
            <a href="/"
               hx-get="/"
               hx-target="#main-content"
               hx-push-url="true"
               class="{{ if eq .ActivePage "dashboard" }}bg-bg-tertiary{{ end }}
                      block px-4 py-2 rounded hover:bg-bg-tertiary">
                🏠 Dashboard
            </a>
        </li>
        <li>
            <a href="/jobs"
               hx-get="/jobs"
               hx-target="#main-content"
               hx-push-url="true"
               class="{{ if eq .ActivePage "jobs" }}bg-bg-tertiary{{ end }}
                      block px-4 py-2 rounded hover:bg-bg-tertiary">
                💼 Jobs
            </a>
        </li>
        <li>
            <a href="/settings"
               hx-get="/settings"
               hx-target="#main-content"
               hx-push-url="true"
               class="{{ if eq .ActivePage "settings" }}bg-bg-tertiary{{ end }}
                      block px-4 py-2 rounded hover:bg-bg-tertiary">
                ⚙️ Settings
            </a>
        </li>
    </ul>
</nav>
```

**Acceptance Criteria**:

- [ ] HX-Request получает partial без layout
- [ ] Обычный GET получает полную страницу
- [ ] `hx-push-url` обновляет URL в браузере
- [ ] Back/Forward в браузере работает

---

#### 3.2.4 — Dark Theme CSS

**Test**: `TestLayout_DarkThemeApplied`

```go
func TestLayout_DarkThemeApplied(t *testing.T) {
    srv := setupTestServer(t)

    resp, err := http.Get(srv.BaseURL() + "/")
    require.NoError(t, err)

    body, _ := io.ReadAll(resp.Body)
    html := string(body)

    // Check dark theme classes
    assert.Contains(t, html, "bg-bg-primary")
    assert.Contains(t, html, "text-text-primary")
}
```

**CSS Variables** (`static/css/style.css`):

```css
:root {
  --bg-primary: #0d1117;
  --bg-secondary: #161b22;
  --bg-tertiary: #21262d;
  --border: #30363d;
  --text-primary: #c9d1d9;
  --text-secondary: #8b949e;
  --accent: #58a6ff;
  --success: #3fb950;
  --danger: #f85149;
  --warning: #d29922;
}

body {
  background-color: var(--bg-primary);
  color: var(--text-primary);
}

/* Tailwind custom classes compiled from input.css */
```

**Acceptance Criteria**:

- [ ] Все страницы в тёмной теме
- [ ] Цветовая схема соответствует GitHub Dark
- [ ] Hover эффекты видимы
- [ ] Текст читаем на всех фонах

### Этап 3: Jobs Page (3.3)

**Цель**: Реализовать API для списка вакансий с фильтрами, пагинацией и HTML-таблицей.

**Зависимости**:

- Repository: `ListJobs(ctx, filter) ([]Job, int, error)`
- Этап 2 (Layout) завершён

**Файлы для создания**:

```
internal/web/handlers/
├── jobs.go              # API handlers
└── jobs_test.go         # Handler tests
internal/web/templates/
├── pages/jobs.html      # Full jobs page
└── partials/
    ├── jobs_table.html  # Table partial
    ├── job_row.html     # Single row
    └── filter_bar.html  # Filters
```

---

#### 3.3.1 — Jobs List API

**Test**: `TestJobsAPI_ListReturnsJobs`

```go
func TestJobsAPI_ListReturnsJobs(t *testing.T) {
    // Setup
    mockRepo := &mocks.JobsRepository{}
    mockRepo.On("List", mock.Anything, mock.Anything).Return(
        []models.Job{
            {ID: "1", RawContent: "Go Developer"},
            {ID: "2", RawContent: "Python Dev"},
        },
        2, // total count
        nil,
    )

    handler := NewJobsHandler(mockRepo, nil)
    req := httptest.NewRequest("GET", "/api/v1/jobs", nil)
    rec := httptest.NewRecorder()

    // Act
    handler.List(rec, req)

    // Assert
    assert.Equal(t, http.StatusOK, rec.Code)
    assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")

    var resp JobsListResponse
    json.Unmarshal(rec.Body.Bytes(), &resp)
    assert.Len(t, resp.Jobs, 2)
    assert.Equal(t, 2, resp.Total)
}
```

**Implementation** (`internal/web/handlers/jobs.go`):

```go
type JobsHandler struct {
    repo      JobsRepository
    templates *TemplateEngine
}

type JobsListResponse struct {
    Jobs  []JobDTO `json:"jobs"`
    Total int      `json:"total"`
    Page  int      `json:"page"`
    Limit int      `json:"limit"`
}

func (h *JobsHandler) List(w http.ResponseWriter, r *http.Request) {
    filter := parseJobFilter(r)

    jobs, total, err := h.repo.List(r.Context(), filter)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    resp := JobsListResponse{
        Jobs:  toJobDTOs(jobs),
        Total: total,
        Page:  filter.Page,
        Limit: filter.Limit,
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(resp)
}
```

**Acceptance Criteria**:

- [ ] GET /api/v1/jobs возвращает JSON массив
- [ ] Response содержит `jobs`, `total`, `page`, `limit`
- [ ] Content-Type: application/json

---

#### 3.3.2 — Filter by Status

**Test**: `TestJobsAPI_FilterByStatus`

```go
func TestJobsAPI_FilterByStatus(t *testing.T) {
    tests := []struct {
        name           string
        status         string
        expectedFilter repository.JobFilter
    }{
        {"ANALYZED", "ANALYZED", repository.JobFilter{Status: "ANALYZED"}},
        {"RAW", "RAW", repository.JobFilter{Status: "RAW"}},
        {"INTERESTED", "INTERESTED", repository.JobFilter{Status: "INTERESTED"}},
        {"REJECTED", "REJECTED", repository.JobFilter{Status: "REJECTED"}},
        {"empty", "", repository.JobFilter{Status: ""}},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockRepo := &mocks.JobsRepository{}
            mockRepo.On("List", mock.Anything, mock.MatchedBy(func(f repository.JobFilter) bool {
                return f.Status == tt.expectedFilter.Status
            })).Return([]models.Job{}, 0, nil)

            handler := NewJobsHandler(mockRepo, nil)
            req := httptest.NewRequest("GET", "/api/v1/jobs?status="+tt.status, nil)
            rec := httptest.NewRecorder()

            handler.List(rec, req)

            mockRepo.AssertExpectations(t)
        })
    }
}
```

**Implementation**:

```go
func parseJobFilter(r *http.Request) repository.JobFilter {
    q := r.URL.Query()

    return repository.JobFilter{
        Status:       q.Get("status"),
        Technologies: strings.Split(q.Get("tech"), ","),
        SalaryMin:    parseInt(q.Get("salary_min"), 0),
        SalaryMax:    parseInt(q.Get("salary_max"), 0),
        Query:        q.Get("q"),
        Sort:         q.Get("sort"),
        Order:        q.Get("order"),
        Page:         parseInt(q.Get("page"), 1),
        Limit:        parseInt(q.Get("limit"), 20),
    }
}
```

**Acceptance Criteria**:

- [ ] `?status=ANALYZED` фильтрует по статусу
- [ ] Несуществующий статус не ломает запрос
- [ ] Пустой status возвращает все

---

#### 3.3.3 — Filter by Technologies (JSONB)

**Test**: `TestJobsAPI_FilterByTech`

```go
func TestJobsAPI_FilterByTech(t *testing.T) {
    mockRepo := &mocks.JobsRepository{}
    mockRepo.On("List", mock.Anything, mock.MatchedBy(func(f repository.JobFilter) bool {
        return reflect.DeepEqual(f.Technologies, []string{"go", "postgresql"})
    })).Return([]models.Job{}, 0, nil)

    handler := NewJobsHandler(mockRepo, nil)
    req := httptest.NewRequest("GET", "/api/v1/jobs?tech=go,postgresql", nil)
    rec := httptest.NewRecorder()

    handler.List(rec, req)

    mockRepo.AssertExpectations(t)
}
```

**Repository Implementation** (`internal/repository/jobs.go`):

```go
// JSONB array contains any of the specified technologies
if len(filter.Technologies) > 0 && filter.Technologies[0] != "" {
    conditions = append(conditions, fmt.Sprintf(
        "structured_data->'technologies' ?| $%d::text[]", argNum))
    args = append(args, pq.Array(filter.Technologies))
    argNum++
}
```

**Acceptance Criteria**:

- [ ] `?tech=go` находит вакансии с Go
- [ ] `?tech=go,python` находит с Go ИЛИ Python
- [ ] Пустой tech возвращает все

---

#### 3.3.4 — Pagination

**Test**: `TestJobsAPI_Pagination`

```go
func TestJobsAPI_Pagination(t *testing.T) {
    mockRepo := &mocks.JobsRepository{}

    // Expect offset = (page-1) * limit = 10
    mockRepo.On("List", mock.Anything, mock.MatchedBy(func(f repository.JobFilter) bool {
        return f.Page == 2 && f.Limit == 10
    })).Return([]models.Job{}, 25, nil) // total = 25

    handler := NewJobsHandler(mockRepo, nil)
    req := httptest.NewRequest("GET", "/api/v1/jobs?page=2&limit=10", nil)
    rec := httptest.NewRecorder()

    handler.List(rec, req)

    var resp JobsListResponse
    json.Unmarshal(rec.Body.Bytes(), &resp)

    assert.Equal(t, 2, resp.Page)
    assert.Equal(t, 10, resp.Limit)
    assert.Equal(t, 25, resp.Total)
}
```

**Repository OFFSET/LIMIT**:

```go
offset := (filter.Page - 1) * filter.Limit
query := fmt.Sprintf("%s LIMIT $%d OFFSET $%d", baseQuery, argNum, argNum+1)
args = append(args, filter.Limit, offset)
```

**Acceptance Criteria**:

- [ ] Default: page=1, limit=20
- [ ] `?page=2&limit=10` → OFFSET 10
- [ ] Response содержит total для UI пагинатора
- [ ] limit > 100 → limit = 100 (cap)

---

#### 3.3.5 — Sorting

**Test**: `TestJobsAPI_Sorting`

```go
func TestJobsAPI_Sorting(t *testing.T) {
    tests := []struct {
        sort     string
        order    string
        expected string
    }{
        {"created_at", "desc", "ORDER BY created_at DESC"},
        {"created_at", "asc", "ORDER BY created_at ASC"},
        {"", "", "ORDER BY created_at DESC"}, // default
    }

    for _, tt := range tests {
        // Verify SQL contains expected ORDER BY clause
    }
}
```

**Implementation**:

```go
var allowedSortFields = map[string]bool{
    "created_at":  true,
    "analyzed_at": true,
    "title":       true,
}

func buildOrderClause(sort, order string) string {
    if !allowedSortFields[sort] {
        sort = "created_at"
    }
    if order != "asc" {
        order = "desc"
    }
    return fmt.Sprintf("ORDER BY %s %s", sort, strings.ToUpper(order))
}
```

**Acceptance Criteria**:

- [ ] Default sort: created_at DESC
- [ ] Whitelist для sort полей (SQL injection prevention)
- [ ] order только asc/desc

---

#### 3.3.6 — Jobs Table HTML Partial

**Test**: `TestJobsPartial_RendersTable`

```go
func TestJobsPartial_RendersTable(t *testing.T) {
    srv := setupTestServer(t)

    // HTMX request for partial
    req, _ := http.NewRequest("GET", srv.BaseURL()+"/partials/jobs-table", nil)
    req.Header.Set("HX-Request", "true")

    resp, _ := http.DefaultClient.Do(req)
    body, _ := io.ReadAll(resp.Body)
    html := string(body)

    assert.Contains(t, html, `<table`)
    assert.Contains(t, html, `<tbody id="jobs-tbody"`)
    assert.Contains(t, html, `hx-trigger="revealed"`) // infinite scroll
}
```

**Template** (`partials/jobs_table.html`):

```html
<table class="w-full">
  <thead class="bg-bg-tertiary">
    <tr>
      <th class="px-4 py-2 text-left">Title</th>
      <th class="px-4 py-2 text-left">Company</th>
      <th class="px-4 py-2 text-left">Salary</th>
      <th class="px-4 py-2 text-left">Status</th>
      <th class="px-4 py-2 text-center">Actions</th>
    </tr>
  </thead>
  <tbody id="jobs-tbody">
    {{ range .Jobs }} {{ template "partials/job_row" . }} {{ end }}
  </tbody>
</table>

<!-- Pagination controls -->
<div class="flex justify-center mt-4 space-x-2">
  {{ if gt .Page 1 }}
  <button
    hx-get="/partials/jobs-table?page={{ sub .Page 1 }}"
    hx-target="#jobs-container"
    class="btn-secondary"
  >
    ← Prev
  </button>
  {{ end }}
  <span class="px-4 py-2">Page {{ .Page }} of {{ .TotalPages }}</span>
  {{ if lt .Page .TotalPages }}
  <button
    hx-get="/partials/jobs-table?page={{ add .Page 1 }}"
    hx-target="#jobs-container"
    class="btn-secondary"
  >
    Next →
  </button>
  {{ end }}
</div>
```

**Acceptance Criteria**:

- [ ] Таблица рендерится с данными
- [ ] Пагинация через HTMX (без полной перезагрузки)
- [ ] Клик по строке выбирает job
- [ ] Фильтры применяются через HTMX

### Этап 3: Job Detail & Actions (3.3, 3.4) [COMPLETED]

**Цель**: Реализовать side panel с деталями вакансии и кнопками действий (Interested/Reject).

**Файлы для создания**:

```
internal/web/handlers/
└── jobs.go              # + Get, UpdateStatus methods
internal/web/templates/partials/
├── job_detail.html      # Side panel content
└── job_row.html         # + status badges, action buttons
```

---

#### 3.4.1 — Get Job by ID

**Test**: `TestJobsAPI_GetByID`

```go
func TestJobsAPI_GetByID(t *testing.T) {
    mockRepo := &mocks.JobsRepository{}
    mockRepo.On("GetByID", mock.Anything, "job-123").Return(
        &models.Job{
            ID:         "job-123",
            RawContent: "Go Developer needed...",
            Status:     "ANALYZED",
            StructuredData: map[string]interface{}{
                "title":        "Go Developer",
                "company":      "TechCorp",
                "technologies": []string{"go", "postgresql"},
            },
        },
        nil,
    )

    handler := NewJobsHandler(mockRepo, nil)

    // Chi URL params
    rctx := chi.NewRouteContext()
    rctx.URLParams.Add("id", "job-123")

    req := httptest.NewRequest("GET", "/api/v1/jobs/job-123", nil)
    req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
    rec := httptest.NewRecorder()

    handler.Get(rec, req)

    assert.Equal(t, http.StatusOK, rec.Code)

    var job JobDetailResponse
    json.Unmarshal(rec.Body.Bytes(), &job)
    assert.Equal(t, "job-123", job.ID)
    assert.Equal(t, "Go Developer", job.Title)
}

func TestJobsAPI_GetByID_NotFound(t *testing.T) {
    mockRepo := &mocks.JobsRepository{}
    mockRepo.On("GetByID", mock.Anything, "unknown").Return(nil, repository.ErrNotFound)

    handler := NewJobsHandler(mockRepo, nil)
    // ...setup request...

    handler.Get(rec, req)

    assert.Equal(t, http.StatusNotFound, rec.Code)
}
```

**Implementation**:

```go
func (h *JobsHandler) Get(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")

    job, err := h.repo.GetByID(r.Context(), id)
    if err != nil {
        if errors.Is(err, repository.ErrNotFound) {
            http.Error(w, "Job not found", http.StatusNotFound)
            return
        }
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(toJobDetailResponse(job))
}
```

**Acceptance Criteria**:

- [ ] GET /api/v1/jobs/:id возвращает полные данные
- [ ] 404 для несуществующего ID
- [ ] structured_data парсится корректно

---

#### 3.4.2 — Update Job Status

**Test**: `TestJobsAPI_UpdateStatus`

```go
func TestJobsAPI_UpdateStatus(t *testing.T) {
    tests := []struct {
        name      string
        newStatus string
        wantCode  int
    }{
        {"to INTERESTED", "INTERESTED", http.StatusOK},
        {"to REJECTED", "REJECTED", http.StatusOK},
        {"to ANALYZED", "ANALYZED", http.StatusOK},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockRepo := &mocks.JobsRepository{}
            mockRepo.On("UpdateStatus", mock.Anything, "job-123", tt.newStatus).Return(nil)

            handler := NewJobsHandler(mockRepo, nil)

            body := strings.NewReader(fmt.Sprintf(`{"status":"%s"}`, tt.newStatus))
            req := httptest.NewRequest("PATCH", "/api/v1/jobs/job-123/status", body)
            req.Header.Set("Content-Type", "application/json")
            // add chi context...

            rec := httptest.NewRecorder()
            handler.UpdateStatus(rec, req)

            assert.Equal(t, tt.wantCode, rec.Code)
            mockRepo.AssertExpectations(t)
        })
    }
}
```

**Implementation**:

```go
type UpdateStatusRequest struct {
    Status string `json:"status"`
}

var validStatuses = map[string]bool{
    "RAW":        true,
    "ANALYZED":   true,
    "INTERESTED": true,
    "REJECTED":   true,
}

func (h *JobsHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")

    var req UpdateStatusRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }

    if !validStatuses[req.Status] {
        http.Error(w, "Invalid status", http.StatusBadRequest)
        return
    }

    if err := h.repo.UpdateStatus(r.Context(), id, req.Status); err != nil {
        if errors.Is(err, repository.ErrNotFound) {
            http.Error(w, "Job not found", http.StatusNotFound)
            return
        }
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // Notify WebSocket clients
    h.hub.Broadcast(ws.JobUpdatedEvent(id, req.Status))

    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"status": req.Status})
}
```

**Acceptance Criteria**:

- [ ] PATCH /api/v1/jobs/:id/status обновляет статус
- [ ] WebSocket broadcast при изменении
- [ ] 404 для несуществующего job

---

#### 3.4.3 — Status Validation

**Test**: `TestJobsAPI_UpdateStatus_ValidationError`

```go
func TestJobsAPI_UpdateStatus_ValidationError(t *testing.T) {
    tests := []struct {
        name    string
        body    string
        wantErr string
    }{
        {"invalid status", `{"status":"UNKNOWN"}`, "Invalid status"},
        {"empty body", `{}`, "Invalid status"},
        {"invalid JSON", `{invalid}`, "Invalid JSON"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            handler := NewJobsHandler(nil, nil) // repo not called

            req := httptest.NewRequest("PATCH", "/api/v1/jobs/123/status",
                strings.NewReader(tt.body))
            rec := httptest.NewRecorder()

            handler.UpdateStatus(rec, req)

            assert.Equal(t, http.StatusBadRequest, rec.Code)
            assert.Contains(t, rec.Body.String(), tt.wantErr)
        })
    }
}
```

**Acceptance Criteria**:

- [ ] Invalid status → 400 Bad Request
- [ ] Malformed JSON → 400 Bad Request
- [ ] Empty status → 400 Bad Request

---

#### 3.4.4 — Side Panel HTML

**Test**: `TestJobDetail_RendersPanel`

```go
func TestJobDetail_RendersPanel(t *testing.T) {
    srv := setupTestServer(t)

    req, _ := http.NewRequest("GET", srv.BaseURL()+"/partials/job-detail/job-123", nil)
    req.Header.Set("HX-Request", "true")

    resp, _ := http.DefaultClient.Do(req)
    body, _ := io.ReadAll(resp.Body)
    html := string(body)

    // Panel structure
    assert.Contains(t, html, `id="job-detail-panel"`)
    assert.Contains(t, html, `class="side-panel"`)

    // Action buttons
    assert.Contains(t, html, `hx-patch="/api/v1/jobs/job-123/status"`)
    assert.Contains(t, html, "INTERESTED")
    assert.Contains(t, html, "REJECTED")

    // Content sections
    assert.Contains(t, html, "Technologies")
    assert.Contains(t, html, "Raw Content")
}
```

**Template** (`partials/job_detail.html`):

```html
<aside id="job-detail-panel" class="side-panel w-96 bg-bg-secondary border-l border-border p-4 overflow-y-auto">
    <header class="border-b border-border pb-4 mb-4">
        <h2 class="text-xl font-bold text-text-primary">{{ .Title }}</h2>
        <p class="text-text-secondary">{{ .Company }}</p>

        <!-- Status badge -->
        <span class="inline-block mt-2 px-2 py-1 rounded text-sm
            {{ if eq .Status "INTERESTED" }}bg-success/20 text-success{{ end }}
            {{ if eq .Status "REJECTED" }}bg-danger/20 text-danger{{ end }}
            {{ if eq .Status "ANALYZED" }}bg-accent/20 text-accent{{ end }}
            {{ if eq .Status "RAW" }}bg-warning/20 text-warning{{ end }}">
            {{ .Status }}
        </span>
    </header>

    <!-- Details -->
    <section class="space-y-4">
        {{ if .Salary }}
        <div>
            <h3 class="text-sm font-semibold text-text-secondary">Salary</h3>
            <p class="text-text-primary">{{ .Salary }}</p>
        </div>
        {{ end }}

        {{ if .Location }}
        <div>
            <h3 class="text-sm font-semibold text-text-secondary">Location</h3>
            <p class="text-text-primary">{{ .Location }}{{ if .IsRemote }} (Remote){{ end }}</p>
        </div>
        {{ end }}

        {{ if .Technologies }}
        <div>
            <h3 class="text-sm font-semibold text-text-secondary">Technologies</h3>
            <div class="flex flex-wrap gap-2 mt-1">
                {{ range .Technologies }}
                <span class="px-2 py-1 bg-bg-tertiary rounded text-sm">{{ . }}</span>
                {{ end }}
            </div>
        </div>
        {{ end }}

        <div>
            <h3 class="text-sm font-semibold text-text-secondary">Raw Content</h3>
            <pre class="mt-1 p-3 bg-bg-tertiary rounded text-sm whitespace-pre-wrap max-h-64 overflow-y-auto">{{ .RawContent }}</pre>
        </div>

        {{ if .Contacts }}
        <div>
            <h3 class="text-sm font-semibold text-text-secondary">Contacts</h3>
            <ul class="mt-1">
                {{ range .Contacts }}
                <li><a href="{{ . }}" class="text-accent hover:underline">{{ . }}</a></li>
                {{ end }}
            </ul>
        </div>
        {{ end }}
    </section>

    <!-- Actions -->
    <footer class="mt-6 pt-4 border-t border-border flex gap-2">
        <button hx-patch="/api/v1/jobs/{{ .ID }}/status"
                hx-vals='{"status":"INTERESTED"}'
                hx-swap="none"
                hx-on::after-request="htmx.trigger('#job-{{ .ID }}', 'refresh')"
                class="btn-success flex-1">
            ✓ Interested
        </button>
        <button hx-patch="/api/v1/jobs/{{ .ID }}/status"
                hx-vals='{"status":"REJECTED"}'
                hx-swap="none"
                hx-on::after-request="htmx.trigger('#job-{{ .ID }}', 'refresh')"
                class="btn-danger flex-1">
            ✗ Reject
        </button>
    </footer>
</aside>
```

**Row Selection** (`partials/job_row.html`):

```html
<tr
  id="job-{{ .ID }}"
  class="table-row {{ if .Selected }}selected{{ end }}"
  hx-get="/partials/job-detail/{{ .ID }}"
  hx-target="#detail-container"
  hx-swap="innerHTML"
  hx-trigger="click"
>
  <td class="px-4 py-3">{{ .Title }}</td>
  <td class="px-4 py-3">{{ .Company }}</td>
  <td class="px-4 py-3">{{ .Salary }}</td>
  <td class="px-4 py-3">
    <span class="status-badge status-{{ lower .Status }}">{{ .Status }}</span>
  </td>
  <td class="px-4 py-3 text-center">
    <button
      hx-patch="/api/v1/jobs/{{ .ID }}/status"
      hx-vals='{"status":"INTERESTED"}'
      hx-swap="none"
      class="text-success hover:bg-success/20 p-1 rounded"
      title="Interested"
    >
      ✓
    </button>
    <button
      hx-patch="/api/v1/jobs/{{ .ID }}/status"
      hx-vals='{"status":"REJECTED"}'
      hx-swap="none"
      class="text-danger hover:bg-danger/20 p-1 rounded"
      title="Reject"
    >
      ✗
    </button>
  </td>
</tr>
```

**Acceptance Criteria**:

- [ ] Клик по строке открывает side panel
- [ ] Panel показывает все данные job
- [ ] Кнопки HTMX обновляют статус без перезагрузки
- [ ] Статус-badge обновляется после действия

---

### Этап 5: WebSocket (3.6) [COMPLETED]

**Цель**: Реализовать real-time обновления через WebSocket с HTMX интеграцией.

**Зависимости**:

- `github.com/gorilla/websocket`
- HTMX ws extension

**Файлы для создания**:

```
internal/web/ws/
├── hub.go           # Connection manager
├── hub_test.go      # Hub unit tests
├── client.go        # WebSocket client
├── events.go        # Event types and payloads
└── handler.go       # HTTP → WS upgrade
```

---

#### 3.5.1 — Hub: Register Client

**Test**: `TestWSHub_RegisterClient`

```go
func TestWSHub_RegisterClient(t *testing.T) {
    hub := NewHub()
    go hub.Run()
    defer hub.Stop()

    client := &Client{
        hub:  hub,
        send: make(chan []byte, 256),
    }

    hub.Register(client)

    // Wait for registration to process
    require.Eventually(t, func() bool {
        return hub.ClientCount() == 1
    }, 100*time.Millisecond, 10*time.Millisecond)

    assert.True(t, hub.HasClient(client))
}
```

**Implementation** (`internal/web/ws/hub.go`):

```go
package ws

import (
    "context"
    "sync"
)

type Hub struct {
    clients    map[*Client]bool
    broadcast  chan []byte
    register   chan *Client
    unregister chan *Client

    mu     sync.RWMutex
    ctx    context.Context
    cancel context.CancelFunc
}

func NewHub() *Hub {
    ctx, cancel := context.WithCancel(context.Background())
    return &Hub{
        clients:    make(map[*Client]bool),
        broadcast:  make(chan []byte, 256),
        register:   make(chan *Client),
        unregister: make(chan *Client),
        ctx:        ctx,
        cancel:     cancel,
    }
}

func (h *Hub) Run() {
    for {
        select {
        case <-h.ctx.Done():
            return
        case client := <-h.register:
            h.mu.Lock()
            h.clients[client] = true
            h.mu.Unlock()
        case client := <-h.unregister:
            h.mu.Lock()
            if _, ok := h.clients[client]; ok {
                delete(h.clients, client)
                close(client.send)
            }
            h.mu.Unlock()
        case message := <-h.broadcast:
            h.broadcastToAll(message)
        }
    }
}

func (h *Hub) Register(client *Client) {
    h.register <- client
}

func (h *Hub) ClientCount() int {
    h.mu.RLock()
    defer h.mu.RUnlock()
    return len(h.clients)
}
```

**Acceptance Criteria**:

- [ ] Client регистрируется в Hub
- [ ] ClientCount() возвращает правильное число
- [ ] Thread-safe операции

---

#### 3.5.2 — Hub: Unregister Client

**Test**: `TestWSHub_UnregisterClient`

```go
func TestWSHub_UnregisterClient(t *testing.T) {
    hub := NewHub()
    go hub.Run()
    defer hub.Stop()

    client := &Client{
        hub:  hub,
        send: make(chan []byte, 256),
    }

    hub.Register(client)
    time.Sleep(10 * time.Millisecond)

    hub.Unregister(client)

    require.Eventually(t, func() bool {
        return hub.ClientCount() == 0
    }, 100*time.Millisecond, 10*time.Millisecond)

    // Channel should be closed
    _, ok := <-client.send
    assert.False(t, ok, "send channel should be closed")
}
```

**Implementation**:

```go
func (h *Hub) Unregister(client *Client) {
    h.unregister <- client
}

func (h *Hub) Stop() {
    h.cancel()

    h.mu.Lock()
    defer h.mu.Unlock()

    for client := range h.clients {
        close(client.send)
        delete(h.clients, client)
    }
}
```

**Acceptance Criteria**:

- [ ] Client удаляется из Hub
- [ ] send channel закрывается
- [ ] Graceful shutdown

---

#### 3.5.3 — Hub: Broadcast

**Test**: `TestWSHub_Broadcast`

```go
func TestWSHub_Broadcast(t *testing.T) {
    hub := NewHub()
    go hub.Run()
    defer hub.Stop()

    // Register 3 clients
    clients := make([]*Client, 3)
    for i := range clients {
        clients[i] = &Client{
            hub:  hub,
            send: make(chan []byte, 256),
        }
        hub.Register(clients[i])
    }
    time.Sleep(20 * time.Millisecond)

    // Broadcast message
    message := []byte(`<div id="notification">New job!</div>`)
    hub.Broadcast(message)

    // All clients should receive
    for i, client := range clients {
        select {
        case msg := <-client.send:
            assert.Equal(t, message, msg)
        case <-time.After(100 * time.Millisecond):
            t.Errorf("client %d did not receive message", i)
        }
    }
}
```

**Implementation**:

```go
func (h *Hub) Broadcast(message []byte) {
    h.broadcast <- message
}

func (h *Hub) broadcastToAll(message []byte) {
    h.mu.RLock()
    defer h.mu.RUnlock()

    for client := range h.clients {
        select {
        case client.send <- message:
        default:
            // Client buffer full, disconnect
            go func(c *Client) {
                h.Unregister(c)
            }(client)
        }
    }
}
```

**Acceptance Criteria**:

- [ ] Message доставляется всем клиентам
- [ ] Non-blocking send (drop slow clients)
- [ ] Concurrent safe

---

#### 3.5.4 — WebSocket Handler

**Test**: `TestWS_ConnectionUpgrade`

```go
func TestWS_ConnectionUpgrade(t *testing.T) {
    hub := NewHub()
    go hub.Run()
    defer hub.Stop()

    handler := NewWSHandler(hub)
    server := httptest.NewServer(http.HandlerFunc(handler.ServeWS))
    defer server.Close()

    // Convert http:// to ws://
    wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

    conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
    require.NoError(t, err)
    defer conn.Close()

    assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)

    // Client should be registered
    require.Eventually(t, func() bool {
        return hub.ClientCount() == 1
    }, 100*time.Millisecond, 10*time.Millisecond)
}
```

**Implementation** (`internal/web/ws/handler.go`):

```go
var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(r *http.Request) bool {
        return true // Allow all origins in dev
    },
}

type WSHandler struct {
    hub *Hub
}

func NewWSHandler(hub *Hub) *WSHandler {
    return &WSHandler{hub: hub}
}

func (h *WSHandler) ServeWS(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Printf("ws upgrade error: %v", err)
        return
    }

    client := &Client{
        hub:  h.hub,
        conn: conn,
        send: make(chan []byte, 256),
    }

    h.hub.Register(client)

    // Start read/write pumps
    go client.writePump()
    go client.readPump()
}
```

**Client pumps** (`internal/web/ws/client.go`):

```go
type Client struct {
    hub  *Hub
    conn *websocket.Conn
    send chan []byte
}

func (c *Client) writePump() {
    ticker := time.NewTicker(54 * time.Second)
    defer func() {
        ticker.Stop()
        c.conn.Close()
    }()

    for {
        select {
        case message, ok := <-c.send:
            if !ok {
                c.conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }

            c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
            if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
                return
            }

        case <-ticker.C:
            c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
            if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                return
            }
        }
    }
}

func (c *Client) readPump() {
    defer func() {
        c.hub.Unregister(c)
        c.conn.Close()
    }()

    c.conn.SetReadLimit(512)
    c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
    c.conn.SetPongHandler(func(string) error {
        c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
        return nil
    })

    for {
        _, _, err := c.conn.ReadMessage()
        if err != nil {
            break
        }
    }
}
```

**Acceptance Criteria**:

- [ ] HTTP → WebSocket upgrade работает
- [ ] Ping/Pong для keep-alive
- [ ] Client cleanup при disconnect

---

#### 3.5.5 — HTMX OOB Updates

**Test**: `TestWS_ReceivesJobUpdate`

```go
func TestWS_ReceivesJobUpdate(t *testing.T) {
    hub := NewHub()
    go hub.Run()
    defer hub.Stop()

    // Setup test server with WS and API
    srv := setupTestServerWithHub(t, hub)

    // Connect WebSocket client
    wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
    conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
    require.NoError(t, err)
    defer conn.Close()

    // Wait for connection
    time.Sleep(50 * time.Millisecond)

    // Update job status via API
    body := strings.NewReader(`{"status":"INTERESTED"}`)
    req, _ := http.NewRequest("PATCH", srv.URL+"/api/v1/jobs/job-123/status", body)
    req.Header.Set("Content-Type", "application/json")
    http.DefaultClient.Do(req)

    // Read WebSocket message
    conn.SetReadDeadline(time.Now().Add(time.Second))
    _, message, err := conn.ReadMessage()
    require.NoError(t, err)

    // Should be HTMX OOB swap
    assert.Contains(t, string(message), "hx-swap-oob")
    assert.Contains(t, string(message), "job-123")
    assert.Contains(t, string(message), "INTERESTED")
}
```

**HTMX Integration**:

```go
// При обновлении статуса отправляем OOB swap
func (h *Hub) NotifyJobUpdated(jobID, status string) {
    html := fmt.Sprintf(`
        <span id="job-%s-status" hx-swap-oob="true"
              class="status-badge status-%s">%s</span>
    `, jobID, strings.ToLower(status), status)

    h.Broadcast([]byte(html))
}

// При новой вакансии — prepend в таблицу
func (h *Hub) NotifyNewJob(job *models.Job) {
    html := fmt.Sprintf(`
        <tr id="job-%s" hx-swap-oob="afterbegin:#jobs-tbody">
            <td>%s</td>
            <td>%s</td>
            <td>%s</td>
            <td><span class="status-badge status-raw">RAW</span></td>
        </tr>
    `, job.ID, job.Title(), job.Company(), job.Salary())

    h.Broadcast([]byte(html))
}
```

**Acceptance Criteria**:

- [ ] Job update → WebSocket message всем
- [ ] HTMX OOB swap обновляет DOM
- [ ] New job появляется в таблице real-time

### Этап 6: Settings Page (3.6) [COMPLETED]

**Цель**: Реализовать CRUD для scraping targets с валидацией и HTMX формами. Plus Telegram QR auth с авто-стартом.

**Файлы для создания**:

```
internal/web/handlers/
├── targets.go           # CRUD handlers
├── auth.go              # Telegram QR auth handler
└── auth_test.go         # Auth tests
internal/web/templates/
├── pages/settings.html  # Settings page with QR auth
└── partials/
    ├── target_form.html # Add/Edit form
    └── target_row.html  # Target list item
```

---

#### 3.6.1 — Telegram QR Auth

**Цель**: Реализовать QR код авторизацию Telegram с авто-стартом при отсутствии коннекта.

**Test**: `TestAuth_AutoStartsWhenDisconnected` (TDD: RED→GREEN→REFACTOR)

```go
func TestAuth_AutoStartsWhenDisconnected(t *testing.T) {
    // When Telegram is not connected (UNAUTHORIZED)
    mockClient := new(MockTelegramClient)
    mockClient.On("GetStatus").Return(telegram.StatusUnauthorized)

    handler := NewAuthHandler(mockClient, mockHub)
    req := httptest.NewRequest("POST", "/api/v1/auth/qr", nil)
    rec := httptest.NewRecorder()

    // Act
    handler.StartQR(rec, req)

    // Assert QR code is broadcast via WebSocket
    select {
    case msg := <-mockHub.broadcast:
        var msg map[string]string
        json.Unmarshal(msg, &msg)
        assert.Equal(t, "tg_qr", msg["type"])
        assert.NotEmpty(t, msg["url"])
    case <-time.After(100 * time.Millisecond):
        t.Fatal("no QR broadcast")
    }
}
```

**Implementation** (`handlers/auth.go`):

```go
func (h *AuthHandler) StartQR(w http.ResponseWriter, r *http.Request) {
    // Check if already connected
    if h.client.GetStatus() == telegram.StatusReady {
        http.Error(w, "already logged in", http.StatusBadRequest)
        return
    }

    // Start QR flow in background
    go func() {
        ctx := context.Background()
        err := h.client.StartQR(ctx, func(url string) {
            if h.hub != nil {
                h.hub.Broadcast(map[string]string{
                    "type": "tg_qr",
                    "url":  url,
                })
            }
        })

        if err != nil {
            if h.hub != nil {
                h.hub.Broadcast(map[string]string{
                    "type":    "error",
                    "message": err.Error(),
                })
            }
            return
        }

        // Broadcast success when auth completes
        if h.hub != nil {
            h.hub.Broadcast(map[string]string{
                "type": "tg_auth_success",
            })
        }
    }()

    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"status": "started"})
}
```

**Template** (`pages/settings.html`):

```html
{{ define "content" }}
<div class="container" style="max-width: 800px; margin: 0 auto;">
  <h1>Settings</h1>

  <!-- Telegram Auth -->
  <article id="telegram-auth-card" aria-label="Telegram Connection">
    <header>
      <h2>Telegram Connection</h2>
    </header>

    <div id="auth-status" role="status" aria-live="polite">
      <small>Status:</small>
      <span id="connection-status">Wait...</span>
    </div>

    <figure id="qr-container" hidden style="text-align: center; padding: 2rem; background: #fff; border-radius: 8px;">
      <p style="margin-bottom: 1rem; color: #000;">Scan with Telegram App</p>
      <div id="qr-code" style="padding: 1rem; display: inline-block; background: #fff; border: 4px solid #000;"></div>
      <p id="qr-timer" style="margin-top: 1rem; color: #000;">
        <small>Expires in <span id="qr-timeout">60</span>s</small>
      </p>
    </figure>

    <div id="connect-btn-container">
      <button
        id="connect-btn"
        hx-post="/api/v1/auth/qr"
        hx-swap="none"
        hx-trigger="load"
      >
        Connect Telegram
      </button>
    </div>
  </article>

  <script src="https://unpkg.com/qrcodejs/qrcode.min.js"></script>
  <script>
    document.addEventListener("DOMContentLoaded", function () {
      const socket = new WebSocket(
        (location.protocol === "https:" ? "wss:" : "ws://") +
          location.host + "/ws"
      );
      const qrContainer = document.getElementById("qr-container");
      const qrCodeDiv = document.getElementById("qr-code");
      const statusSpan = document.getElementById("connection-status");
      const connectBtn = document.getElementById("connect-btn");

      socket.onmessage = function (event) {
        const msg = JSON.parse(event.data);
        if (msg.type === "tg_qr") {
          qrContainer.hidden = false;
          qrCodeDiv.innerHTML = "";
          new QRCode(qrCodeDiv, msg.url);
          statusSpan.textContent = "Scan QR Code...";
          statusSpan.style.color = "var(--warning)";

          // 60 second timer
          let timeLeft = 60;
          const timerSpan = document.getElementById("qr-timeout");
          const timerInterval = setInterval(function () {
            timeLeft--;
            if (timerSpan) timerSpan.textContent = timeLeft;
            if (timeLeft <= 0) {
              clearInterval(timerInterval);
              qrContainer.hidden = true;
              qrCodeDiv.innerHTML = "";
              statusSpan.textContent = "QR expired. Try again.";
              statusSpan.style.color = "var(--error)";
            }
          }, 1000);

          qrContainer.dataset.timerInterval = timerInterval;
        } else if (msg.type === "tg_auth_success") {
          const timerInterval = qrContainer.dataset.timerInterval;
          if (timerInterval) {
            clearInterval(parseInt(timerInterval));
            delete qrContainer.dataset.timerInterval;
          }

          qrContainer.hidden = true;
          qrCodeDiv.innerHTML = "";
          statusSpan.textContent = "Connected";
          statusSpan.style.color = "var(--success)";
          connectBtn.hidden = true;
        } else if (msg.type === "error") {
          const timerInterval = qrContainer.dataset.timerInterval;
          if (timerInterval) {
            clearInterval(parseInt(timerInterval));
            delete qr-container.dataset.timerInterval;
          }

          alert("Error: " + msg.message);
          qrContainer.hidden = true;
          statusSpan.textContent = "Disconnected";
          statusSpan.style.color = "var(--error)";
        }
      };
    });
  </script>

  <!-- Add Target Form -->
  <article aria-label="Add New Target">
    <header>
      <h2>Add New Target</h2>
    </header>

    <form hx-post="/api/v1/targets" hx-target="#targets-list" hx-swap="beforeend">
      <div class="grid">
        <label for="target-name">Name</label>
        <input type="text" id="target-name" name="name" required>

        <label for="target-type">Type</label>
        <select id="target-type" name="type">
          <option value="TG_CHANNEL">Telegram Channel</option>
          <option value="TG_FORUM">Telegram Forum</option>
          <option value="HH_SEARCH">HH Search</option>
        </select>

        <label for="target-url">URL / ID / Username</label>
        <input type="text" id="target-url" name="url" required placeholder="@username or https://...">
      </div>

      <button type="submit">Add Target</button>
    </form>
  </article>

  <!-- Targets List -->
  <section aria-label="Scraping Targets">
    <h2>Scraping Targets</h2>
    <div
      id="targets-list"
      hx-get="/api/v1/targets"
      hx-trigger="load"
    >
      <small>Loading targets...</small>
    </div>
  </section>
</div>
{{ end }}
```

**Acceptance Criteria**:
- [ ] GET /settings → QR код появляется автоматически если не подключен
- [ ] QR код имеет белый фон и чёрную рамку для контраста
- [ ] Таймер 60 секунд с авто-скрытием
- [ ] При успешном логине QR скрывается, статус меняется на "Connected"
- [ ] Pico.css semantic HTML (`<article>`, `<figure>`, `<label>`, `<mark>`)
- [ ] Стили через `style` атрибуты для dark theme compatibility

---

#### 3.6.2 — Pico.css Semantic HTML Refactor

**Цель**: Унифицировать все шаблоны используя Pico.css semantic HTML вместо Tailwind классов.

| Компонент                | Текущий (Tailwind)              | Target (Pico.css)                    |
| ------------------------- | -------------------------------- | ---------------------------------------- |
| **Settings page**          | `class="flex flex-col"`           | `<div class="container">`                   |
| **Forms**                   | `class="w-full bg-bg-primary"`     | Bare `<input>`, `<select>` с Pico.css        |
| **Buttons**                 | `class="btn-primary"`              | `<button>` + `[aria-label]`            |
| **Status badges**             | `class="text-success text-danger"`   | `<mark>` или `<small>`                   |
| **Side panel**               | `class="w-96 bg-bg-secondary"`    | `<aside>` или `<details>`                 |
| **Tables**                   | `class="w-full"`                  | `<table role="grid">`                       |
| **Articles/Cards**             | `class="card p-6"`               | `<article>`                                  |

**Pico.css Patterns**:

```html
<!-- Form с semantic HTML -->
<div class="grid">
  <label for="name">Name</label>
  <input id="name" name="name" required placeholder="Your name">
</div>

<!-- Кнопки с aria-label -->
<button hx-post="/api/v1/jobs/{{ .ID }}/status"
        aria-label="Mark as interested">
  ✓ Interested
</button>

<!-- Статусы через <mark> -->
<mark>{{ .Status }}</mark>

<!-- Side panel с collapsible -->
<details open>
  <summary>Job Details</summary>
  <article>
    <!-- content -->
  </article>
</details>
```

**Acceptance Criteria**:
- [ ] Tailwind классы заменены на semantic HTML
- [ ] Pico.css стили (без кастомных классов)
- [] Dark theme через CSS переменные
- [ ] Таблицы используют `<table role="grid">`
- [] Формы используют `<div class="grid">` для раскладки

---

**Test**: `TestTargetsAPI_List`

```go
func TestTargetsAPI_List(t *testing.T) {
    mockRepo := &mocks.TargetsRepository{}
    mockRepo.On("List", mock.Anything).Return(
        []models.ScrapingTarget{
            {ID: "1", Name: "Go Jobs", Type: "TG_CHANNEL", URL: "@golang_jobs", Active: true},
            {ID: "2", Name: "Rust Jobs", Type: "TG_FORUM", URL: "@rust_jobs", TopicIDs: []int{15, 28}},
        },
        nil,
    )

    handler := NewTargetsHandler(mockRepo)
    req := httptest.NewRequest("GET", "/api/v1/targets", nil)
    rec := httptest.NewRecorder()

    handler.List(rec, req)

    assert.Equal(t, http.StatusOK, rec.Code)

    var resp TargetsListResponse
    json.Unmarshal(rec.Body.Bytes(), &resp)
    assert.Len(t, resp.Targets, 2)
    assert.Equal(t, "Go Jobs", resp.Targets[0].Name)
}
```

**Implementation**:

```go
type TargetsHandler struct {
    repo      TargetsRepository
    templates *TemplateEngine
}

type TargetsListResponse struct {
    Targets []TargetDTO `json:"targets"`
}

func (h *TargetsHandler) List(w http.ResponseWriter, r *http.Request) {
    targets, err := h.repo.List(r.Context())
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(TargetsListResponse{
        Targets: toTargetDTOs(targets),
    })
}
```

**Acceptance Criteria**:

- [ ] GET /api/v1/targets возвращает список
- [ ] Включает все поля: id, name, type, url, topic_ids, active

---

#### 3.6.2 — Create Target

**Test**: `TestTargetsAPI_Create`

```go
func TestTargetsAPI_Create(t *testing.T) {
    mockRepo := &mocks.TargetsRepository{}
    mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(t *models.ScrapingTarget) bool {
        return t.Name == "New Channel" && t.Type == "TG_CHANNEL"
    })).Return("new-id", nil)

    handler := NewTargetsHandler(mockRepo)

    body := `{"name":"New Channel","type":"TG_CHANNEL","url":"@new_channel","active":true}`
    req := httptest.NewRequest("POST", "/api/v1/targets", strings.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    rec := httptest.NewRecorder()

    handler.Create(rec, req)

    assert.Equal(t, http.StatusCreated, rec.Code)

    var resp CreateTargetResponse
    json.Unmarshal(rec.Body.Bytes(), &resp)
    assert.Equal(t, "new-id", resp.ID)
}
```

**Implementation**:

```go
type CreateTargetRequest struct {
    Name     string `json:"name"`
    Type     string `json:"type"`
    URL      string `json:"url"`
    TopicIDs []int  `json:"topic_ids,omitempty"`
    Active   bool   `json:"active"`
}

func (h *TargetsHandler) Create(w http.ResponseWriter, r *http.Request) {
    var req CreateTargetRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }

    if err := validateTarget(req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    target := &models.ScrapingTarget{
        Name:     req.Name,
        Type:     req.Type,
        URL:      req.URL,
        TopicIDs: req.TopicIDs,
        Active:   req.Active,
    }

    id, err := h.repo.Create(r.Context(), target)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(map[string]string{"id": id})
}
```

**Acceptance Criteria**:

- [ ] POST /api/v1/targets создаёт target
- [ ] Возвращает 201 Created с ID
- [ ] TopicIDs опционально для TG_FORUM

---

#### 3.6.3 — Update Target

**Test**: `TestTargetsAPI_Update`

```go
func TestTargetsAPI_Update(t *testing.T) {
    mockRepo := &mocks.TargetsRepository{}
    mockRepo.On("Update", mock.Anything, "target-1", mock.Anything).Return(nil)

    handler := NewTargetsHandler(mockRepo)

    body := `{"name":"Updated Name","active":false}`
    req := httptest.NewRequest("PUT", "/api/v1/targets/target-1", strings.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    // add chi context

    rec := httptest.NewRecorder()
    handler.Update(rec, req)

    assert.Equal(t, http.StatusOK, rec.Code)
}
```

**Implementation**:

```go
func (h *TargetsHandler) Update(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")

    var req UpdateTargetRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }

    if err := h.repo.Update(r.Context(), id, req.toModel()); err != nil {
        if errors.Is(err, repository.ErrNotFound) {
            http.Error(w, "Target not found", http.StatusNotFound)
            return
        }
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusOK)
}
```

**Acceptance Criteria**:

- [ ] PUT /api/v1/targets/:id обновляет target
- [ ] 404 для несуществующего ID
- [ ] Partial updates поддерживаются

---

#### 3.6.4 — Delete Target

**Test**: `TestTargetsAPI_Delete`

```go
func TestTargetsAPI_Delete(t *testing.T) {
    mockRepo := &mocks.TargetsRepository{}
    mockRepo.On("Delete", mock.Anything, "target-1").Return(nil)

    handler := NewTargetsHandler(mockRepo)

    req := httptest.NewRequest("DELETE", "/api/v1/targets/target-1", nil)
    // add chi context

    rec := httptest.NewRecorder()
    handler.Delete(rec, req)

    assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestTargetsAPI_Delete_NotFound(t *testing.T) {
    mockRepo := &mocks.TargetsRepository{}
    mockRepo.On("Delete", mock.Anything, "unknown").Return(repository.ErrNotFound)

    // ... test returns 404
}
```

**Implementation**:

```go
func (h *TargetsHandler) Delete(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")

    if err := h.repo.Delete(r.Context(), id); err != nil {
        if errors.Is(err, repository.ErrNotFound) {
            http.Error(w, "Target not found", http.StatusNotFound)
            return
        }
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusNoContent)
}
```

**Acceptance Criteria**:

- [ ] DELETE /api/v1/targets/:id удаляет target
- [ ] 204 No Content при успехе
- [ ] 404 для несуществующего ID

---

#### 3.6.5 — Validation

**Test**: `TestTargetsAPI_ValidationError`

```go
func TestTargetsAPI_ValidationError(t *testing.T) {
    tests := []struct {
        name    string
        body    string
        wantErr string
    }{
        {"missing name", `{"type":"TG_CHANNEL","url":"@test"}`, "name is required"},
        {"missing type", `{"name":"Test","url":"@test"}`, "type is required"},
        {"invalid type", `{"name":"Test","type":"INVALID","url":"@test"}`, "invalid type"},
        {"missing url", `{"name":"Test","type":"TG_CHANNEL"}`, "url is required"},
        {"forum without topics", `{"name":"Test","type":"TG_FORUM","url":"@test"}`, "topic_ids required for TG_FORUM"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            handler := NewTargetsHandler(nil)

            req := httptest.NewRequest("POST", "/api/v1/targets", strings.NewReader(tt.body))
            req.Header.Set("Content-Type", "application/json")
            rec := httptest.NewRecorder()

            handler.Create(rec, req)

            assert.Equal(t, http.StatusBadRequest, rec.Code)
            assert.Contains(t, rec.Body.String(), tt.wantErr)
        })
    }
}
```

**Implementation**:

```go
var validTargetTypes = map[string]bool{
    "TG_CHANNEL": true,
    "TG_FORUM":   true,
}

func validateTarget(req CreateTargetRequest) error {
    if req.Name == "" {
        return errors.New("name is required")
    }
    if req.Type == "" {
        return errors.New("type is required")
    }
    if !validTargetTypes[req.Type] {
        return errors.New("invalid type")
    }
    if req.URL == "" {
        return errors.New("url is required")
    }
    if req.Type == "TG_FORUM" && len(req.TopicIDs) == 0 {
        return errors.New("topic_ids required for TG_FORUM")
    }
    return nil
}
```

**Acceptance Criteria**:

- [ ] Missing fields → 400 + message
- [ ] Invalid type → 400
- [ ] TG_FORUM требует topic_ids

---

#### 3.6.6 — Settings Page Template

**Test**: `TestSettingsPage_RendersForms`

```go
func TestSettingsPage_RendersForms(t *testing.T) {
    srv := setupTestServer(t)

    resp, _ := http.Get(srv.BaseURL() + "/settings")
    body, _ := io.ReadAll(resp.Body)
    html := string(body)

    assert.Equal(t, http.StatusOK, resp.Code)
    assert.Contains(t, html, "Scraping Targets")
    assert.Contains(t, html, `<form`)
    assert.Contains(t, html, `hx-post="/api/v1/targets"`)
    assert.Contains(t, html, "Add New Target")
}
```

**Template** (`pages/settings.html`):

```html
{{ define "content" }}
<div class="space-y-6">
  <h1 class="text-2xl font-bold">Settings</h1>

  <section class="card p-6">
    <h2 class="text-xl font-semibold mb-4">Scraping Targets</h2>

    <!-- Targets List -->
    <div id="targets-list" class="space-y-4 mb-6">
      {{ range .Targets }} {{ template "partials/target_row" . }} {{ end }}
    </div>

    <!-- Add New Form -->
    <details class="group">
      <summary class="cursor-pointer text-accent hover:underline">
        + Add New Target
      </summary>
      <form
        hx-post="/api/v1/targets"
        hx-target="#targets-list"
        hx-swap="beforeend"
        hx-on::after-request="this.reset()"
        class="mt-4 space-y-4 p-4 bg-bg-tertiary rounded"
      >
        <div>
          <label class="block text-sm text-text-secondary mb-1">Name</label>
          <input
            type="text"
            name="name"
            required
            class="w-full bg-bg-primary border border-border rounded px-3 py-2"
          />
        </div>

        <div>
          <label class="block text-sm text-text-secondary mb-1">Type</label>
          <select
            name="type"
            required
            class="w-full bg-bg-primary border border-border rounded px-3 py-2"
            hx-on:change="document.getElementById('topic-ids-field').style.display = 
                                          this.value === 'TG_FORUM' ? 'block' : 'none'"
          >
            <option value="TG_CHANNEL">Telegram Channel</option>
            <option value="TG_FORUM">Telegram Forum</option>
          </select>
        </div>

        <div>
          <label class="block text-sm text-text-secondary mb-1"
            >URL / Username</label
          >
          <input
            type="text"
            name="url"
            required
            placeholder="@channel_name"
            class="w-full bg-bg-primary border border-border rounded px-3 py-2"
          />
        </div>

        <div id="topic-ids-field" style="display:none">
          <label class="block text-sm text-text-secondary mb-1"
            >Topic IDs (comma-separated)</label
          >
          <input
            type="text"
            name="topic_ids"
            placeholder="15, 28, 42"
            class="w-full bg-bg-primary border border-border rounded px-3 py-2"
          />
        </div>

        <div class="flex items-center gap-2">
          <input type="checkbox" name="active" id="active" checked />
          <label for="active" class="text-sm">Active</label>
        </div>

        <button type="submit" class="btn-primary">Add Target</button>
      </form>
    </details>
  </section>
</div>
{{ end }}
```

**Target Row** (`partials/target_row.html`):

```html
<div
  id="target-{{ .ID }}"
  class="flex items-center justify-between p-4 bg-bg-tertiary rounded"
>
  <div>
    <span class="font-medium">{{ .Name }}</span>
    <span class="text-text-secondary ml-2">({{ .Type }})</span>
    <span class="text-accent ml-2">{{ .URL }}</span>
    {{ if not .Active }}
    <span class="ml-2 px-2 py-0.5 bg-warning/20 text-warning text-xs rounded"
      >Paused</span
    >
    {{ end }}
  </div>
  <div class="flex gap-2">
    <button
      hx-delete="/api/v1/targets/{{ .ID }}"
      hx-target="#target-{{ .ID }}"
      hx-swap="outerHTML"
      hx-confirm="Delete target '{{ .Name }}'?"
      class="text-danger hover:bg-danger/20 p-2 rounded"
    >
      🗑️
    </button>
  </div>
</div>
```

**Acceptance Criteria**:

- [ ] Settings page показывает список targets
- [ ] Add form создаёт target через HTMX
- [ ] Delete удаляет с подтверждением
- [ ] Topic IDs показывается только для TG_FORUM

### Этап 7: Dashboard (3.7) [COMPLETED]

**Цель**: Реализовать Dashboard со статистикой и карточками.

**Файлы для создания**:

```
internal/web/handlers/
├── stats.go             # Stats API
└── stats_test.go        # Stats tests
internal/repository/
└── stats.go             # Aggregation queries
internal/web/templates/
└── pages/dashboard.html # Dashboard page
```

---

#### 3.7.1 — Stats API

**Test**: `TestDashboardAPI_Stats`

```go
func TestDashboardAPI_Stats(t *testing.T) {
    mockRepo := &mocks.StatsRepository{}
    mockRepo.On("GetStats", mock.Anything).Return(
        &models.DashboardStats{
            TotalJobs:      150,
            AnalyzedJobs:   120,
            InterestedJobs: 25,
            RejectedJobs:   45,
            TodayJobs:      12,
            ActiveTargets:  3,
        },
        nil,
    )

    handler := NewStatsHandler(mockRepo)
    req := httptest.NewRequest("GET", "/api/v1/stats", nil)
    rec := httptest.NewRecorder()

    handler.GetStats(rec, req)

    assert.Equal(t, http.StatusOK, rec.Code)

    var stats models.DashboardStats
    json.Unmarshal(rec.Body.Bytes(), &stats)
    assert.Equal(t, 150, stats.TotalJobs)
    assert.Equal(t, 25, stats.InterestedJobs)
}
```

**Implementation**:

```go
type DashboardStats struct {
    TotalJobs      int `json:"total_jobs"`
    AnalyzedJobs   int `json:"analyzed_jobs"`
    InterestedJobs int `json:"interested_jobs"`
    RejectedJobs   int `json:"rejected_jobs"`
    TodayJobs      int `json:"today_jobs"`
    ActiveTargets  int `json:"active_targets"`
}

func (h *StatsHandler) GetStats(w http.ResponseWriter, r *http.Request) {
    stats, err := h.repo.GetStats(r.Context())
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(stats)
}
```

**Repository**:

```go
func (r *StatsRepository) GetStats(ctx context.Context) (*DashboardStats, error) {
    stats := &DashboardStats{}

    // Total and by status
    err := r.db.QueryRowContext(ctx, `
        SELECT
            COUNT(*) as total,
            COUNT(*) FILTER (WHERE status = 'ANALYZED') as analyzed,
            COUNT(*) FILTER (WHERE status = 'INTERESTED') as interested,
            COUNT(*) FILTER (WHERE status = 'REJECTED') as rejected,
            COUNT(*) FILTER (WHERE created_at >= CURRENT_DATE) as today
        FROM jobs
    `).Scan(&stats.TotalJobs, &stats.AnalyzedJobs, &stats.InterestedJobs,
            &stats.RejectedJobs, &stats.TodayJobs)
    if err != nil {
        return nil, err
    }

    // Active targets
    err = r.db.QueryRowContext(ctx, `
        SELECT COUNT(*) FROM scraping_targets WHERE active = true
    `).Scan(&stats.ActiveTargets)

    return stats, err
}
```

**Acceptance Criteria**:

- [ ] GET /api/v1/stats возвращает статистику
- [ ] Подсчёт по статусам корректен
- [ ] TodayJobs считает за текущий день

---

#### 3.7.2 — Dashboard Page

**Test**: `TestDashboard_Renders`

```go
func TestDashboard_Renders(t *testing.T) {
    srv := setupTestServer(t)

    resp, _ := http.Get(srv.BaseURL() + "/")
    body, _ := io.ReadAll(resp.Body)
    html := string(body)

    assert.Equal(t, http.StatusOK, resp.StatusCode)
    assert.Contains(t, html, "Dashboard")
    assert.Contains(t, html, `hx-get="/api/v1/stats"`)
}
```

**Template** (`pages/dashboard.html`):

```html
{{ define "content" }}
<div class="space-y-6">
  <h1 class="text-2xl font-bold">Dashboard</h1>

  <!-- Stats Cards (load via HTMX) -->
  <div
    id="stats-cards"
    hx-get="/partials/stats-cards"
    hx-trigger="load"
    hx-swap="innerHTML"
    class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4"
  >
    <!-- Loading skeleton -->
    <div class="card p-4 animate-pulse">
      <div class="h-4 bg-bg-tertiary rounded w-1/2 mb-2"></div>
      <div class="h-8 bg-bg-tertiary rounded w-1/4"></div>
    </div>
  </div>

  <!-- Recent Activity -->
  <section class="card p-6">
    <h2 class="text-xl font-semibold mb-4">Recent Jobs</h2>
    <div
      id="recent-jobs"
      hx-get="/partials/recent-jobs"
      hx-trigger="load"
      hx-swap="innerHTML"
    >
      <div class="animate-pulse">Loading...</div>
    </div>
  </section>
</div>
{{ end }}
```

**Acceptance Criteria**:

- [ ] Dashboard рендерится на /
- [ ] Stats загружаются через HTMX
- [ ] Loading skeleton показывается

---

#### 3.7.3 — Stats Cards Partial

**Test**: `TestDashboard_ShowsStats`

```go
func TestDashboard_ShowsStats(t *testing.T) {
    srv := setupTestServer(t)

    req, _ := http.NewRequest("GET", srv.BaseURL()+"/partials/stats-cards", nil)
    req.Header.Set("HX-Request", "true")

    resp, _ := http.DefaultClient.Do(req)
    body, _ := io.ReadAll(resp.Body)
    html := string(body)

    // Should show stat cards
    assert.Contains(t, html, "Total Jobs")
    assert.Contains(t, html, "Interested")
    assert.Contains(t, html, "Today")
}
```

**Template** (`partials/stats_cards.html`):

```html
<div class="card p-4 bg-bg-secondary border-l-4 border-accent">
  <div class="text-sm text-text-secondary">Total Jobs</div>
  <div class="text-3xl font-bold">{{ .TotalJobs }}</div>
</div>

<div class="card p-4 bg-bg-secondary border-l-4 border-success">
  <div class="text-sm text-text-secondary">Interested</div>
  <div class="text-3xl font-bold text-success">{{ .InterestedJobs }}</div>
</div>

<div class="card p-4 bg-bg-secondary border-l-4 border-danger">
  <div class="text-sm text-text-secondary">Rejected</div>
  <div class="text-3xl font-bold text-danger">{{ .RejectedJobs }}</div>
</div>

<div class="card p-4 bg-bg-secondary border-l-4 border-warning">
  <div class="text-sm text-text-secondary">Today</div>
  <div class="text-3xl font-bold text-warning">+{{ .TodayJobs }}</div>
</div>
```

**Acceptance Criteria**:

- [ ] 4 карточки со статистикой
- [ ] Цветовая индикация (success, danger, warning)
- [ ] Обновляются при HTMX trigger

---

### Этап 8: Integration (3.8)

**Цель**: E2E тесты, проверяющие полный workflow.

**Файлы для создания**:

```
tests/e2e/
├── workflow_test.go     # Full workflow tests
├── websocket_test.go    # WebSocket integration
└── setup_test.go        # Test helpers
```

---

#### 3.8.1 — Full Workflow Test

**Test**: `TestE2E_FullWorkflow`

```go
func TestE2E_FullWorkflow(t *testing.T) {
    // Setup: real DB, real server
    db := setupTestDB(t)
    srv := setupRealServer(t, db)
    defer srv.Close()

    // 1. Create target
    targetBody := `{"name":"Test Channel","type":"TG_CHANNEL","url":"@test","active":true}`
    resp, _ := http.Post(srv.URL+"/api/v1/targets", "application/json",
        strings.NewReader(targetBody))
    assert.Equal(t, http.StatusCreated, resp.StatusCode)

    // 2. Simulate job creation (as if scraped)
    jobID := insertTestJob(t, db, "Go Developer at TechCorp")

    // 3. List jobs
    resp, _ = http.Get(srv.URL + "/api/v1/jobs")
    var jobsResp JobsListResponse
    json.NewDecoder(resp.Body).Decode(&jobsResp)
    assert.Equal(t, 1, jobsResp.Total)
    assert.Equal(t, "RAW", jobsResp.Jobs[0].Status)

    // 4. Update status to INTERESTED
    statusBody := `{"status":"INTERESTED"}`
    req, _ := http.NewRequest("PATCH", srv.URL+"/api/v1/jobs/"+jobID+"/status",
        strings.NewReader(statusBody))
    req.Header.Set("Content-Type", "application/json")
    resp, _ = http.DefaultClient.Do(req)
    assert.Equal(t, http.StatusOK, resp.StatusCode)

    // 5. Verify status changed
    resp, _ = http.Get(srv.URL + "/api/v1/jobs/" + jobID)
    var jobResp JobDetailResponse
    json.NewDecoder(resp.Body).Decode(&jobResp)
    assert.Equal(t, "INTERESTED", jobResp.Status)

    // 6. Check stats updated
    resp, _ = http.Get(srv.URL + "/api/v1/stats")
    var stats DashboardStats
    json.NewDecoder(resp.Body).Decode(&stats)
    assert.Equal(t, 1, stats.InterestedJobs)
}
```

**Acceptance Criteria**:

- [ ] Full flow: Target → Job → Status update
- [ ] Data correctly persisted in DB
- [ ] Stats reflect changes

---

#### 3.8.2 — WebSocket Real-time Updates

**Test**: `TestE2E_WebSocketUpdates`

```go
func TestE2E_WebSocketUpdates(t *testing.T) {
    db := setupTestDB(t)
    srv := setupRealServer(t, db)
    defer srv.Close()

    // Connect WebSocket
    wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
    conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
    require.NoError(t, err)
    defer conn.Close()

    // Wait for connection
    time.Sleep(100 * time.Millisecond)

    // Insert job
    jobID := insertTestJob(t, db, "Test Job")

    // Update job status
    go func() {
        time.Sleep(50 * time.Millisecond)
        statusBody := `{"status":"REJECTED"}`
        req, _ := http.NewRequest("PATCH", srv.URL+"/api/v1/jobs/"+jobID+"/status",
            strings.NewReader(statusBody))
        req.Header.Set("Content-Type", "application/json")
        http.DefaultClient.Do(req)
    }()

    // Read WebSocket message
    conn.SetReadDeadline(time.Now().Add(2 * time.Second))
    _, message, err := conn.ReadMessage()
    require.NoError(t, err)

    // Should contain OOB swap HTML
    assert.Contains(t, string(message), "REJECTED")
    assert.Contains(t, string(message), "hx-swap-oob")
}
```

**Acceptance Criteria**:

- [ ] WebSocket receives updates on job status change
- [ ] HTMX OOB swap format correct
- [ ] Multiple clients receive same update

---

#### 3.8.3 — Settings CRUD E2E

**Test**: `TestE2E_SettingsCRUD`

```go
func TestE2E_SettingsCRUD(t *testing.T) {
    db := setupTestDB(t)
    srv := setupRealServer(t, db)
    defer srv.Close()

    // CREATE
    body := `{"name":"Test","type":"TG_CHANNEL","url":"@test","active":true}`
    resp, _ := http.Post(srv.URL+"/api/v1/targets", "application/json",
        strings.NewReader(body))
    require.Equal(t, http.StatusCreated, resp.StatusCode)

    var created struct{ ID string }
    json.NewDecoder(resp.Body).Decode(&created)
    targetID := created.ID

    // READ
    resp, _ = http.Get(srv.URL + "/api/v1/targets")
    var list TargetsListResponse
    json.NewDecoder(resp.Body).Decode(&list)
    assert.Len(t, list.Targets, 1)
    assert.Equal(t, "Test", list.Targets[0].Name)

    // UPDATE
    updateBody := `{"name":"Updated","active":false}`
    req, _ := http.NewRequest("PUT", srv.URL+"/api/v1/targets/"+targetID,
        strings.NewReader(updateBody))
    req.Header.Set("Content-Type", "application/json")
    resp, _ = http.DefaultClient.Do(req)
    assert.Equal(t, http.StatusOK, resp.StatusCode)

    // Verify update
    resp, _ = http.Get(srv.URL + "/api/v1/targets")
    json.NewDecoder(resp.Body).Decode(&list)
    assert.Equal(t, "Updated", list.Targets[0].Name)
    assert.False(t, list.Targets[0].Active)

    // DELETE
    req, _ = http.NewRequest("DELETE", srv.URL+"/api/v1/targets/"+targetID, nil)
    resp, _ = http.DefaultClient.Do(req)
    assert.Equal(t, http.StatusNoContent, resp.StatusCode)

    // Verify deletion
    resp, _ = http.Get(srv.URL + "/api/v1/targets")
    json.NewDecoder(resp.Body).Decode(&list)
    assert.Len(t, list.Targets, 0)
}
```

**Acceptance Criteria**:

- [ ] Create → Read → Update → Delete workflow
- [ ] Data correctly persisted
- [ ] No orphaned data after delete

---

### Test Structure

```go
// internal/web/handlers/jobs_test.go
package handlers

import (
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestJobsAPI_ListReturnsJobs(t *testing.T) {
    // Arrange
    handler := NewJobsHandler(mockRepo)
    req := httptest.NewRequest("GET", "/api/v1/jobs", nil)
    rec := httptest.NewRecorder()

    // Act
    handler.List(rec, req)

    // Assert
    assert.Equal(t, http.StatusOK, rec.Code)
    assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")

    var response JobsListResponse
    err := json.Unmarshal(rec.Body.Bytes(), &response)
    require.NoError(t, err)
    assert.NotNil(t, response.Jobs)
}

func TestJobsAPI_FilterByStatus(t *testing.T) {
    tests := []struct {
        name           string
        status         string
        expectedCount  int
    }{
        {"filter ANALYZED", "ANALYZED", 3},
        {"filter RAW", "RAW", 5},
        {"filter REJECTED", "REJECTED", 1},
        {"no filter", "", 9}, // all
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            req := httptest.NewRequest("GET", "/api/v1/jobs?status="+tt.status, nil)
            rec := httptest.NewRecorder()

            handler.List(rec, req)

            var response JobsListResponse
            json.Unmarshal(rec.Body.Bytes(), &response)
            assert.Len(t, response.Jobs, tt.expectedCount)
        })
    }
}
```

```go
// internal/web/ws/hub_test.go
package ws

func TestWSHub_RegisterClient(t *testing.T) {
    hub := NewHub()
    go hub.Run()

    client := &Client{
        hub:  hub,
        send: make(chan []byte, 256),
    }

    hub.Register(client)

    // wait for registration
    time.Sleep(10 * time.Millisecond)

    assert.True(t, hub.HasClient(client))
}

func TestWSHub_Broadcast(t *testing.T) {
    hub := NewHub()
    go hub.Run()

    // register 3 clients
    clients := make([]*Client, 3)
    for i := range clients {
        clients[i] = &Client{
            hub:  hub,
            send: make(chan []byte, 256),
        }
        hub.Register(clients[i])
    }

    time.Sleep(10 * time.Millisecond)

    // broadcast
    message := []byte(`{"type":"job.new"}`)
    hub.Broadcast(message)

    // each client should receive
    for i, client := range clients {
        select {
        case msg := <-client.send:
            assert.Equal(t, message, msg)
        case <-time.After(100 * time.Millisecond):
            t.Errorf("client %d did not receive message", i)
        }
    }
}
```

---

## 🧪 Testing

### E2E Test Scenario

```bash
#!/bin/bash
# scripts/test-webui.sh

BASE_URL="http://localhost:3100"

echo "1. Check dashboard loads..."
curl -s "$BASE_URL/" | grep -q "Dashboard" && echo "✓ Dashboard" || echo "✗ Dashboard"

echo "2. Check jobs page..."
curl -s "$BASE_URL/jobs" | grep -q "Jobs" && echo "✓ Jobs" || echo "✗ Jobs"

echo "3. Check settings page..."
curl -s "$BASE_URL/settings" | grep -q "Settings" && echo "✓ Settings" || echo "✗ Settings"

echo "4. Check jobs API..."
curl -s "$BASE_URL/api/v1/jobs" | jq -e '.jobs' && echo "✓ Jobs API" || echo "✗ Jobs API"

echo "5. Check WebSocket..."
# requires wscat or similar
```

---

## 📦 Dependencies

```bash
# WebSocket
go get github.com/gorilla/websocket

# Tailwind (npm, build time only)
npm install -D tailwindcss
npx tailwindcss -i ./static/css/input.css -o ./static/css/style.css --minify
```

---

## 🔮 Следующий шаг

После Web UI переходим к **Фазе 4: Brain** — генерация персонализированных резюме.
