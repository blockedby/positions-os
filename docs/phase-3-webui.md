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

### Этап 1: Server Setup

| #     | Test First (Red)                                     | Implementation (Green)  |
| ----- | ---------------------------------------------------- | ----------------------- |
| 3.1.1 | `TestServer_Starts` — сервер запускается на порту    | Chi router с middleware |
| 3.1.2 | `TestServer_ServesStatic` — /static/\* отдаёт файлы  | Static file serving     |
| 3.1.3 | `TestServer_RendersTemplate` — шаблон рендерится     | Template engine setup   |
| 3.1.4 | `TestServer_HealthEndpoint` — /health возвращает 200 | Health endpoint         |

### Этап 2: Layout & Navigation

| #     | Test First (Red)                                          | Implementation (Green) |
| ----- | --------------------------------------------------------- | ---------------------- |
| 3.2.1 | `TestLayout_ContainsSidebar` — layout содержит sidebar    | Base layout template   |
| 3.2.2 | `TestNavigation_AllPagesLoad` — /, /jobs, /settings 200   | Sidebar navigation     |
| 3.2.3 | `TestNavigation_HTMXPartialResponse` — partial без layout | HTMX page switching    |
| 3.2.4 | `TestLayout_DarkThemeApplied` — CSS классы dark theme     | Dark theme styles      |

### Этап 3: Jobs Page

| #     | Test First (Red)                                                 | Implementation (Green) |
| ----- | ---------------------------------------------------------------- | ---------------------- |
| 3.3.1 | `TestJobsAPI_ListReturnsJobs` — GET /api/v1/jobs возвращает JSON | Jobs list endpoint     |
| 3.3.2 | `TestJobsAPI_FilterByStatus` — ?status=ANALYZED работает         | Status filter          |
| 3.3.3 | `TestJobsAPI_FilterByTech` — ?tech=go,python работает            | Tech filter            |
| 3.3.4 | `TestJobsAPI_Pagination` — ?page=2&limit=10 корректно            | Pagination logic       |
| 3.3.5 | `TestJobsAPI_Sorting` — ?sort=created_at&order=desc              | Sorting logic          |
| 3.3.6 | `TestJobsPartial_RendersTable` — HTML таблица рендерится         | Jobs table template    |

### Этап 4: Job Detail Panel

| #     | Test First (Red)                                                  | Implementation (Green) |
| ----- | ----------------------------------------------------------------- | ---------------------- |
| 3.4.1 | `TestJobsAPI_GetByID` — GET /api/v1/jobs/:id возвращает job       | Get job endpoint       |
| 3.4.2 | `TestJobsAPI_UpdateStatus` — PATCH /api/v1/jobs/:id/status        | Status update endpoint |
| 3.4.3 | `TestJobsAPI_UpdateStatus_ValidationError` — invalid status → 400 | Validation             |
| 3.4.4 | `TestJobDetail_RendersPanel` — partial рендерит детали            | Side panel template    |

### Этап 5: WebSocket

| #     | Test First (Red)                                              | Implementation (Green) |
| ----- | ------------------------------------------------------------- | ---------------------- |
| 3.5.1 | `TestWSHub_RegisterClient` — клиент регистрируется            | Hub implementation     |
| 3.5.2 | `TestWSHub_UnregisterClient` — клиент отключается             | Client cleanup         |
| 3.5.3 | `TestWSHub_Broadcast` — сообщение всем клиентам               | Broadcast logic        |
| 3.5.4 | `TestWS_ConnectionUpgrade` — HTTP → WebSocket                 | WS handler             |
| 3.5.5 | `TestWS_ReceivesJobUpdate` — при изменении job приходит event | Event integration      |

### Этап 6: Settings Page

| #     | Test First (Red)                                              | Implementation (Green) |
| ----- | ------------------------------------------------------------- | ---------------------- |
| 3.6.1 | `TestTargetsAPI_List` — GET /api/v1/targets возвращает список | List targets           |
| 3.6.2 | `TestTargetsAPI_Create` — POST создаёт target                 | Create target          |
| 3.6.3 | `TestTargetsAPI_Update` — PUT обновляет target                | Update target          |
| 3.6.4 | `TestTargetsAPI_Delete` — DELETE удаляет target               | Delete target          |
| 3.6.5 | `TestTargetsAPI_ValidationError` — invalid data → 400         | Validation             |
| 3.6.6 | `TestSettingsPage_RendersForms` — форма рендерится            | Settings template      |

### Этап 7: Dashboard

| #     | Test First (Red)                                               | Implementation (Green) |
| ----- | -------------------------------------------------------------- | ---------------------- |
| 3.7.1 | `TestDashboardAPI_Stats` — /api/v1/stats возвращает статистику | Stats endpoint         |
| 3.7.2 | `TestDashboard_Renders` — dashboard страница рендерится        | Dashboard template     |
| 3.7.3 | `TestDashboard_ShowsStats` — отображает карточки со статами    | Stats cards            |

### Этап 8: Integration

| #     | Test                       | Description                   |
| ----- | -------------------------- | ----------------------------- |
| 3.8.1 | `TestE2E_FullWorkflow`     | Scrape → View → Status update |
| 3.8.2 | `TestE2E_WebSocketUpdates` | Real-time обновления работают |
| 3.8.3 | `TestE2E_SettingsCRUD`     | Полный CRUD для targets       |

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
