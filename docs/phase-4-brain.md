# Фаза 4: Brain + WebSocket Events

## Обзор

Brain — сервис для генерации персонализированных резюме и cover letters под конкретную вакансию.
WebSocket Events — единая система real-time событий для всего UI.

**На выходе:**

- Адаптация базового резюме под требования вакансии
- Конвертация в PDF
- Генерация сопроводительного письма
- Real-time обновления UI через WebSocket

---

## Архитектура

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                              Browser (UI)                                    │
│  ┌────────────────────────────────────────────────────────────────────────┐  │
│  │  WebSocket Client (HTMX ws extension)                                  │  │
│  │  ws://localhost:3100/ws                                                │  │
│  └────────────────────────────────────────────────────────────────────────┘  │
└───────────────────────────────────────┬──────────────────────────────────────┘
                                        │
                                        ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│                            Unified Service (Go)                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │  Collector  │  │  Analyzer   │  │    Brain    │  │   WebSocket Hub     │  │
│  │  (scraping) │  │  (LLM)      │  │  (tailoring)│  │   (events broker)   │  │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └──────────┬──────────┘  │
│         │                │                │                    │             │
│         └────────────────┴────────────────┴────────────────────┘             │
│                                   │                                          │
│                                   ▼                                          │
│  ┌────────────────────────────────────────────────────────────────────────┐  │
│  │  PostgreSQL  │  NATS  │  LLM API  │  File Storage                     │  │
│  └────────────────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────────────────┘
```

---

## Компоненты

| #   | Компонент        | Описание                              |
| --- | ---------------- | ------------------------------------- |
| 4.1 | Resume Storage   | Базовое резюме в Markdown             |
| 4.2 | LLM Client       | OpenAI-compatible API + NATS queue    |
| 4.3 | Resume Tailorer  | Адаптация под вакансию                |
| 4.4 | Cover Letter Gen | Генерация сопроводительного           |
| 4.5 | PDF Renderer     | HTML → PDF (chromedp)                 |
| 4.6 | API Endpoints    | REST + WebSocket для живых обновлений |
| 4.7 | API Docs         | Scalar — современный API docs UI      |
| 4.8 | WebSocket Events | Real-time события для всего UI        |

---

## Требования

### Общие требования ко всем компонентам:

- **Логирование**: Все операции логируются (start, progress, success, error)
- **NATS Queue**: Задачи на генерацию складываются в NATS, обрабатываются асинхронно
- **Rate Limiting**: LLM вызовы ограничены 1 req/sec (hardcoded)
- **Structured Logging**: JSON формат, уровни: debug, info, warn, error

---

## 4.1 Resume Storage

```
storage/
├── resume.md           # базовое резюме (source of truth)
└── outputs/
    └── {job_id}/
        ├── resume_tailored.md
        └── resume_tailored.pdf  # только resume PDF (для attachment)
# cover_letter остаётся как TEXT для email/сообщения, не PDF
```

**Псевдо-код:**

```
func LoadBaseResume() -> string:
    log.Info("loading base resume")
    return ReadFile("storage/resume.md")

func SaveTailoredResume(job_id, content):
    log.Info("saving tailored resume", job_id=job_id)
    WriteFile("storage/outputs/{job_id}/resume_tailored.md", content)
```

---

## 4.2 LLM Integration + NATS Queue

**Config (.env):**

```env
# Brain LLM — OpenAI-compatible API
BRAIN_LLM_URL=https://api.openai.com/v1
BRAIN_LLM_MODEL=gpt-4o
BRAIN_LLM_KEY=sk-...
BRAIN_LLM_MAX_TOKENS=4096
BRAIN_LLM_TEMPERATURE=0.7
```

**NATS Queue:**

Вместо прямого вызова LLM, складываем задачи в очередь:

```
Subject: brain.jobs.prepare
Payload: { "job_id": "uuid-123" }  # только ID, остальное из БД
```

**Псевдо-код:**

```
type BrainLLM:
    client: OpenAIClient
    rateLimiter: 1 req/sec  # hardcoded limit
    natsConn: NATSClient

    func tailor_resume(base_resume, job_data) -> string:
        rateLimiter.Wait()
        log.Info("calling LLM for resume tailoring")
        prompt = load_prompt("resume-tailoring.xml")
        prompt.inject(resume=base_resume, job=job_data)
        result = client.complete(prompt)
        log.Info("resume tailoring complete", tokens=result.usage)
        return result

    func generate_cover(job_data, tailored_resume, template) -> string:
        rateLimiter.Wait()
        log.Info("calling LLM for cover letter", template=template)
        prompt = load_prompt("cover-letter.xml")
        prompt.inject(job=job_data, resume=tailored_resume, template=template)
        return client.complete(prompt)
```

---

## 4.3 Resume Tailoring

**Стратегия:**

1. Взять базовое резюме (skills, experience, achievements)
2. Сопоставить с требованиями вакансии
3. Переупорядочить и акцентировать релевантный опыт
4. НЕ врать, только restructure

**Промпт: `docs/prompts/resume-tailoring.xml`**

```xml
<prompt>
  <system>
    Ты HR-консультант. Адаптируй резюме под вакансию:
    - Выдели релевантные навыки первыми
    - Акцентируй подходящий опыт
    - НЕ добавляй то, чего нет
    - Сохрани факты, измени акценты
    - НЕ меняй структуру резюме (секции, порядок)
    - Язык резюме: если вакансия на английском → резюме на английском, иначе на русском
    Ответ: только Markdown резюме, без комментариев.
  </system>
  <user>
    ## Вакансия:
    {{JOB_DATA}}

    ## Базовое резюме:
    {{BASE_RESUME}}
  </user>
</prompt>
```

**Псевдо-код:**

```
func TailorResume(job_id) -> TailoredResult:
    log.Info("starting tailoring pipeline", job_id=job_id)

    job = repo.GetJob(job_id)
    if job.status != INTERESTED:
        log.Warn("invalid job status", status=job.status)
        return Error("Job must be INTERESTED")

    base_resume = LoadBaseResume()
    job_data = FormatJobForLLM(job.structured_data)

    ws.Send(job_id, { step: "tailoring", progress: 25 })
    tailored_md = llm.tailor_resume(base_resume, job_data)

    # Определяем язык и выбираем шаблон cover letter
    # template = SelectCoverTemplate(job.structured_data.language)

    ws.Send(job_id, { step: "cover_letter", progress: 50 })
    cover_md = llm.generate_cover(job_data, tailored_md, template)

    SaveTailoredResume(job_id, tailored_md)
    SaveCoverLetter(job_id, cover_md)

    ws.Send(job_id, { step: "pdf_rendering", progress: 75 })
    resume_pdf = RenderPDF(tailored_md, "resume")

    repo.UpdateJobOutputs(job_id, {
        tailored_resume_path: resume_pdf,
        cover_letter_text: cover_md,  # TEXT для email/сообщения
        status: "PREPARED"
    })

    ws.Send(job_id, { step: "complete", progress: 100 })
    log.Info("tailoring complete", job_id=job_id)

    return { resume_pdf, cover_text }
```

---

## 4.4 Cover Letter Generation

**Промпт: `docs/prompts/cover-letter.xml`**

Содержит 3 базовых шаблона, которые LLM адаптирует под конкретную вакансию:

```xml
<prompt>
  <system>
    Напиши сопроводительное письмо на основе шаблона.
    Адаптируй под конкретную вакансию, сохраняя структуру.
    Тон: профессиональный, но не формальный.
    Язык: соответствует шаблону.
  </system>

  <templates>
    <!-- Шаблон 1: Формальный (RU) -->
    <template id="formal_ru">
      Уважаемый(-ая) {{CONTACT_NAME}},

      Меня заинтересовала позиция {{POSITION}} в {{COMPANY}}.

      {{RELEVANT_EXPERIENCE}}

      {{WHY_COMPANY}}

      Буду рад обсудить возможное сотрудничество.

      С уважением,
      {{MY_NAME}}
    </template>

    <!-- Шаблон 2: Современный (RU) -->
    <template id="modern_ru">
      Привет!

      Увидел вакансию {{POSITION}} и понял — это то, что ищу.

      {{RELEVANT_EXPERIENCE}}

      {{WHY_COMPANY}}

      Давайте созвонимся?

      {{MY_NAME}}
    </template>

    <!-- Шаблон 3: Professional (EN) -->
    <template id="professional_en">
      Dear Hiring Manager,

      I am writing to express my interest in the {{POSITION}} role at {{COMPANY}}.

      {{RELEVANT_EXPERIENCE}}

      {{WHY_COMPANY}}

      I look forward to discussing this opportunity.

      Best regards,
      {{MY_NAME}}
    </template>
  </templates>

  <user>
    ## Вакансия:
    {{JOB_DATA}}

    ## Моё резюме (адаптированное):
    {{TAILORED_RESUME}}

    ## Используй шаблон:
    {{TEMPLATE_ID}}
  </user>
</prompt>
```

---

## 4.5 PDF Renderer

**Подход:** HTML template + chromedp → PDF

### Почему НЕ используем Markdown парсер (goldmark):

- Резюме — структурированные данные (имя, опыт, навыки)
- Проще использовать Go HTML template с точным контролем вёрстки
- Zero dependencies (кроме chromedp)

### Docker setup для chromedp:

**Рекомендуемый подход — готовый образ `chromedp/headless-shell`:**

```dockerfile
# Dockerfile.brain
FROM golang:1.21-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o brain cmd/brain/main.go

# Production image с headless Chrome
FROM chromedp/headless-shell:latest
COPY --from=builder /build/brain /app/brain
COPY storage/ /app/storage/
COPY static/pdf-templates/ /app/static/pdf-templates/

WORKDIR /app
CMD ["./brain"]
```

**Размер:** ~300MB (оптимизированный Chrome headless)

**Acceptance Criteria для Docker:**

- [x] Dockerfile.brain создан и протестирован
- [x] Образ собирается без ошибок
- [x] Chrome доступен в контейнере
- [x] PDF генерация работает внутри контейнера

### Псевдо-код:

```
func RenderPDF(resumeData ResumeData, template_name) -> filepath:
    log.Info("rendering PDF", template=template_name)

    // Используем Go HTML template
    tmpl = template.ParseFiles("static/pdf-templates/" + template_name + ".html")
    var htmlBuf bytes.Buffer
    tmpl.Execute(&htmlBuf, resumeData)

    // chromedp → PDF
    ctx, cancel = chromedp.NewContext(context.Background())
    defer cancel()

    var pdfBuf []byte
    chromedp.Run(ctx,
        chromedp.Navigate("data:text/html," + base64(htmlBuf.String())),
        chromedp.ActionFunc(func(ctx) {
            pdfBuf = page.PrintToPDF().WithPrintBackground(true).Do(ctx)
        })
    )

    path = SaveFile(pdfBuf, "{job_id}/{template_name}.pdf")
    log.Info("PDF saved", path=path)
    return path
```

**Зависимости Go:**

```bash
go get github.com/chromedp/chromedp  # headless Chrome control
```

---

## 4.6 API Endpoints

См. подробную документацию WebSocket: **[phase-4-websocket.md](phase-4-websocket.md)**

### REST Endpoints:

```
POST /api/v1/jobs/{id}/prepare
  → triggers tailoring pipeline
  → returns { status: "processing", ws_channel: "brain.{job_id}" }

GET /api/v1/jobs/{id}/documents
  → returns { resume_pdf_url, cover_letter_text, status }

GET /api/v1/jobs/{id}/documents/resume.pdf
  → file download (resume PDF для attachment)
```

### WebSocket Events:

```
ws://localhost:3100/ws

# Subscribe to job processing
→ { "subscribe": "brain.{job_id}" }

# Progress updates
← { "type": "brain.progress", "job_id": "...", "step": "tailoring", "progress": 25 }
← { "type": "brain.progress", "job_id": "...", "step": "cover_letter", "progress": 50 }
← { "type": "brain.progress", "job_id": "...", "step": "pdf_rendering", "progress": 75 }
← { "type": "brain.complete", "job_id": "...", "resume_url": "...", "cover_url": "..." }
```

**Псевдо-код handler:**

```
func PrepareJobHandler(w, r):
    job_id = chi.URLParam(r, "id")
    log.Info("prepare request received", job_id=job_id)

    // async processing
    go func():
        result = TailorResume(job_id)
        if result.error:
            log.Error("tailoring failed", error=result.error)
            ws.Send(job_id, { type: "brain.error", error: result.error })
        else:
            nats.Publish("jobs.prepared", { job_id, result })

    respond(w, { status: "processing", ws_channel: "brain." + job_id })
```

---

## 4.7 API Documentation — Scalar

**Выбор:** [Scalar](https://github.com/scalar/scalar) — современный API docs UI (2024)

**Почему Scalar:**

- 🎨 Красивый, современный дизайн
- 🌙 Dark mode из коробки
- ⚡ Интерактивный Try-it-out
- 📦 Простая интеграция (один HTML файл)
- 🔥 Рекомендован Microsoft для .NET 9

**Альтернативы:**

- **Redoc** — чистый, но менее интерактивный
- **Swagger UI** — устаревший дизайн
- **Stoplight** — тяжёлый, для enterprise

### Интеграция:

**1. Создать OpenAPI spec вручную:**

```yaml
# static/docs/openapi.yaml
openapi: 3.1.0
info:
  title: Positions OS API
  version: 1.0.0
paths:
  /api/v1/jobs/{id}/prepare:
    post:
      summary: Prepare resume and cover letter
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      responses:
        "200":
          description: Processing started
```

**2. Добавить Scalar UI:**

```html
<!-- static/docs/index.html -->
<!DOCTYPE html>
<html>
  <head>
    <title>Positions OS API</title>
    <meta charset="utf-8" />
  </head>
  <body>
    <script id="api-reference" data-url="/docs/openapi.yaml"></script>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
  </body>
</html>
```

**3. Serve на `/docs`:**

```go
// internal/web/router.go
r.Get("/docs", func(w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w, r, "static/docs/index.html")
})
r.Get("/docs/*", http.FileServer(http.Dir("static/docs")).ServeHTTP)
```

---

## 4.8 WebSocket Events

Real-time события для всего UI. Все компоненты публикуют события через единый Hub.

### Event Types

#### Scraping Events

```json
// Scraping started
{ "type": "scrape.started", "target": "@golang_jobs", "limit": 100 }

// Progress update
{ "type": "scrape.progress", "target": "@golang_jobs", "processed": 45, "total": 100, "new_jobs": 12 }

// Completed
{ "type": "scrape.completed", "target": "@golang_jobs", "total_processed": 100, "new_jobs": 23 }

// Error
{ "type": "scrape.error", "target": "@golang_jobs", "error": "FLOOD_WAIT: retry in 30s" }
```

#### Job Events

```json
// New job added
{ "type": "job.new", "job_id": "uuid-123", "title": "Go Developer", "company": "Yandex", "status": "RAW" }

// Job analyzed
{ "type": "job.analyzed", "job_id": "uuid-123", "technologies": ["go", "postgresql"], "salary_min": 250000 }

// Job status updated
{ "type": "job.updated", "job_id": "uuid-123", "status": "INTERESTED" }
```

#### Brain Events

```json
// Processing started
{ "type": "brain.started", "job_id": "uuid-123" }

// Progress updates (25% → 50% → 75% → 100%)
{ "type": "brain.progress", "job_id": "uuid-123", "step": "tailoring", "progress": 25, "message": "Адаптирую резюме..." }
{ "type": "brain.progress", "job_id": "uuid-123", "step": "cover_letter", "progress": 50, "message": "Генерирую сопроводительное..." }
{ "type": "brain.progress", "job_id": "uuid-123", "step": "pdf_rendering", "progress": 75, "message": "Создаю PDF..." }

// Completed
{ "type": "brain.completed", "job_id": "uuid-123", "resume_url": "/api/v1/jobs/uuid-123/documents/resume.pdf", "cover_letter_text": "..." }

// Error
{ "type": "brain.error", "job_id": "uuid-123", "step": "tailoring", "error": "LLM timeout" }
```

#### System Events

```json
// Notification
{ "type": "notification", "level": "success", "message": "Новая вакансия добавлена", "duration": 5000 }

// Stats update
{ "type": "stats.updated", "total_jobs": 1234, "new_today": 45, "interested": 23 }
```

### Channel Subscriptions

```json
// Subscribe to specific events
{ "action": "subscribe", "channel": "job.*" }           // все job events
{ "action": "subscribe", "channel": "brain.uuid-123" }  // конкретный job
{ "action": "subscribe", "channel": "scrape.*" }        // все scraping events

// Unsubscribe
{ "action": "unsubscribe", "channel": "job.*" }
```

### Backend Implementation

```go
// internal/web/ws/hub.go

type Hub struct {
    clients    map[*Client]bool
    register   chan *Client
    unregister chan *Client
    mu         sync.RWMutex
}

type Client struct {
    hub           *Hub
    conn          *websocket.Conn
    send          chan []byte
    subscriptions map[string]bool
}

// BroadcastToChannel отправляет событие всем подписчикам
func (h *Hub) BroadcastToChannel(channel string, event WSEvent) {
    h.mu.RLock()
    defer h.mu.RUnlock()

    for client := range h.clients {
        if client.IsSubscribed(channel) {
            client.send <- event.ToJSON()
        }
    }
}

// Event helpers
func (h *Hub) ScrapeProgress(target string, processed, total, newJobs int) {
    h.BroadcastToChannel("scrape.*", WSEvent{
        Type: "scrape.progress",
        Payload: map[string]interface{}{
            "target": target, "processed": processed, "total": total, "new_jobs": newJobs,
        },
    })
}

func (h *Hub) BrainProgress(jobID, step string, progress int, message string) {
    h.BroadcastToChannel("brain."+jobID, WSEvent{
        Type: "brain.progress",
        Payload: map[string]interface{}{
            "job_id": jobID, "step": step, "progress": progress, "message": message,
        },
    })
}

func (h *Hub) Notify(level, message string) {
    h.BroadcastToChannel("notification", WSEvent{
        Type: "notification",
        Payload: map[string]interface{}{"level": level, "message": message, "duration": 5000},
    })
}
```

### Frontend Integration (HTMX)

```html
<body hx-ext="ws" ws-connect="/ws">
  <!-- Auto-subscribe to all events -->

  <!-- Jobs table updates via OOB swap -->
  <table id="jobs-table">
    <tbody id="jobs-tbody"></tbody>
  </table>

  <!-- Brain progress bar -->
  <div id="brain-progress" class="hidden">
    <div id="progress-bar" class="h-2 bg-accent" style="width: 0%"></div>
    <span id="progress-text"></span>
  </div>

  <!-- Toast notifications -->
  <div id="notifications"></div>
</body>

<script>
  const ws = new WebSocket("ws://localhost:3100/ws");

  ws.onmessage = (event) => {
    const data = JSON.parse(event.data);

    switch (data.type) {
      case "job.new":
        showToast("success", `New: ${data.title} @ ${data.company}`);
        htmx.ajax("GET", `/partials/job-row/${data.job_id}`, {
          target: "#jobs-tbody",
          swap: "afterbegin",
        });
        break;

      case "brain.progress":
        document.getElementById("progress-bar").style.width =
          data.progress + "%";
        document.getElementById("progress-text").textContent = data.message;
        break;

      case "brain.completed":
        showToast("success", "Documents ready!");
        showDownloadButtons(data.resume_url, data.cover_letter_text);
        break;

      case "notification":
        showToast(data.level, data.message);
        break;
    }
  };
</script>
```

---

## 📁 Структура файлов

```
positions-os/
├── cmd/
│   └── brain/
│       └── main.go                  # Entry point (может быть частью collector)
├── internal/
│   ├── brain/
│   │   ├── service.go               # TailorResume, GenerateCover
│   │   ├── llm.go                   # LLM client with rate limiting
│   │   ├── pdf.go                   # PDF rendering (chromedp)
│   │   └── prompts.go               # Load XML prompt templates
│   └── web/
│       └── ws/
│           ├── hub.go               # WebSocket connection manager
│           ├── client.go            # WebSocket client handler
│           └── events.go            # Event types and helpers
├── storage/
│   ├── resume.md                    # Базовое резюме
│   └── outputs/                     # Сгенерированные файлы
├── docs/
│   ├── prompts/
│   │   ├── resume-tailoring.xml
│   │   └── cover-letter.xml         # Содержит 3 шаблона
│   └── phase-4-brain.md             # Этот файл
└── static/
    ├── docs/
    │   ├── index.html               # Scalar API docs
    │   └── openapi.yaml             # OpenAPI spec
    └── pdf-templates/
        ├── resume.html              # HTML template для PDF резюме
        └── cover.html               # HTML template для cover letter
```

---

## 🎯 Порядок реализации

### Этап 1: Storage & Base Resume

- [ ] 4.1.1 — Создать `storage/resume.md` со своим резюме
- [ ] 4.1.2 — `internal/brain/storage.go` — Load/Save functions
- [ ] 4.1.3 — Тест: загрузка резюме
- [ ] 4.1.4 — Логирование всех операций

**Acceptance Criteria:**

- [ ] `storage/resume.md` существует и содержит валидный Markdown
- [ ] `LoadBaseResume()` возвращает содержимое файла
- [ ] `SaveTailoredResume()` создаёт директорию и сохраняет файл
- [ ] Все операции логируются (info level)
- [ ] Тесты покрывают happy path и error cases

### Этап 2: LLM Integration + NATS

- [ ] 4.2.1 — Добавить конфиг BRAIN*LLM*\* в .env
- [ ] 4.2.2 — `internal/brain/llm.go` — client wrapper
- [ ] 4.2.3 — Rate limiter 1 req/sec (hardcoded)
- [ ] 4.2.4 — NATS consumer для `brain.jobs.prepare`
- [ ] 4.2.5 — Тест: вызов LLM с test job
- [ ] 4.2.6 — Логирование вызовов и usage

**Acceptance Criteria:**

- [ ] LLM client подключается к OpenAI-compatible API
- [ ] Rate limiter ограничивает вызовы до 1/sec
- [ ] NATS consumer читает job_id из очереди
- [ ] Данные вакансии загружаются из БД по job_id
- [ ] LLM вызовы логируются с token usage
- [ ] Тест проверяет rate limiting

### Этап 3: Prompts

- [ ] 4.3.1 — `docs/prompts/resume-tailoring.xml`
- [ ] 4.3.2 — `docs/prompts/cover-letter.xml` с 3 шаблонами
- [ ] 4.3.3 — Интеграция с LLM client

**Acceptance Criteria:**

- [ ] Промпты в XML формате с `<system>` и `<user>` секциями
- [ ] Resume prompt содержит требование не менять структуру
- [ ] Resume prompt содержит правило выбора языка (EN/RU)
- [ ] Cover letter prompt содержит 3 шаблона (formal_ru, modern_ru, professional_en)
- [ ] Промпты загружаются и парсятся без ошибок
- [ ] Плейсхолдеры {{JOB_DATA}}, {{BASE_RESUME}} корректно заменяются

### Этап 4: PDF Rendering

- [ ] 4.4.1 — Dockerfile.brain с chromedp/headless-shell
- [ ] 4.4.2 — `internal/brain/pdf.go` (chromedp + Go templates)
- [ ] 4.4.3 — HTML шаблоны для PDF (простой минималистичный стиль)
- [ ] 4.4.4 — Тест: HTML → PDF
- [ ] 4.4.5 — Логирование рендеринга

**Acceptance Criteria:**

- [ ] Dockerfile.brain собирается без ошибок
- [ ] Chrome доступен в контейнере
- [ ] HTML template рендерится с данными резюме
- [ ] PDF генерируется с корректным форматированием
- [ ] CSS стили применяются (margins, fonts, colors)
- [ ] PDF сохраняется в `storage/outputs/{job_id}/`
- [ ] Рендеринг логируется (start, success, error)

### Этап 5: Service Layer

- [ ] 4.5.1 — `internal/brain/service.go` — TailorResume pipeline
- [ ] 4.5.2 — Интеграция с repository (UpdateJobOutputs)
- [ ] 4.5.3 — WebSocket progress events
- [ ] 4.5.4 — Unit tests
- [ ] 4.5.5 — Логирование pipeline

**Acceptance Criteria:**

- [ ] `TailorResume()` выполняет полный pipeline (tailor → cover TEXT → resume PDF)
- [ ] WebSocket отправляет события на каждом этапе (25%, 50%, 75%, 100%)
- [ ] Job status обновляется на `PREPARED`
- [ ] Resume PDF путь сохраняется в БД (`tailored_resume_path`)
- [ ] Cover letter TEXT сохраняется в БД (`cover_letter_text`)
- [ ] Ошибки обрабатываются и логируются
- [ ] Тесты покрывают весь pipeline

### Этап 6: API & Integration

- [ ] 4.6.1 — `POST /api/v1/jobs/{id}/prepare` → публикует в NATS
- [ ] 4.6.2 — `GET /api/v1/jobs/{id}/documents`
- [ ] 4.6.3 — Resume PDF download endpoint
- [ ] 4.6.4 — NATS event `jobs.prepared`
- [ ] 4.6.5 — Логирование всех requests

**Acceptance Criteria:**

- [ ] POST /prepare публикует job_id в NATS `brain.jobs.prepare`
- [ ] POST /prepare возвращает ws_channel для подписки
- [ ] GET /documents возвращает resume_pdf_url, cover_letter_text, статус
- [ ] Download endpoint отдаёт resume PDF с правильными headers
- [ ] NATS event `jobs.prepared` публикуется после завершения
- [ ] Все endpoints логируются (request, response, errors)

### Этап 7: API Documentation

- [ ] 4.7.1 — Создать `static/docs/openapi.yaml`
- [ ] 4.7.2 — Создать `static/docs/index.html` с Scalar
- [ ] 4.7.3 — Добавить routes для `/docs`

**Acceptance Criteria:**

- [ ] OpenAPI spec описывает все Brain endpoints
- [ ] Scalar UI доступен на `/docs`
- [ ] Try-it-out работает для всех endpoints
- [ ] Dark mode включен
- [ ] Документация актуальна

### Этап 8: UI Integration

- [ ] 4.8.1 — Кнопка "Prepare Response" в job detail
- [ ] 4.8.2 — Progress bar через WebSocket
- [ ] 4.8.3 — Показ сгенерированных документов
- [ ] 4.8.4 — Download links

**Acceptance Criteria:**

- [ ] Кнопка "Prepare" появляется только для INTERESTED jobs
- [ ] Progress bar обновляется в real-time
- [ ] После завершения показываются preview и download кнопки
- [ ] Resume PDF download открывает в новой вкладке
- [ ] Cover letter copy-to-clipboard работает
- [ ] Ошибки показываются в toast notifications
- [ ] UI логирует WebSocket события

### Этап 9: WebSocket Events System

- [ ] 4.9.1 — `internal/web/ws/hub.go` — Connection manager с channel subscriptions
- [ ] 4.9.2 — `internal/web/ws/client.go` — Client handler
- [ ] 4.9.3 — `internal/web/ws/events.go` — Event types и helpers
- [ ] 4.9.4 — Интеграция с Collector (scrape.\* events)
- [ ] 4.9.5 — Интеграция с Analyzer (job.analyzed events)
- [ ] 4.9.6 — Интеграция с Brain (brain.\* events)
- [ ] 4.9.7 — System events (notifications, stats)
- [ ] 4.9.8 — Frontend JS handler для всех событий

**Acceptance Criteria:**

- [ ] Hub поддерживает channel subscriptions (job._, scrape._, brain.{id})
- [ ] Wildcard subscriptions работают (e.g., `job.*` ловит `job.new`, `job.analyzed`)
- [ ] Все сервисы публикуют события через Hub
- [ ] События доставляются только подписчикам
- [ ] Frontend обрабатывает все типы событий
- [ ] Reconnection работает автоматически
- [ ] События логируются (debug level)
- [ ] Тесты покрывают подписки и broadcast

---

## Database Schema Updates

```sql
ALTER TABLE jobs ADD COLUMN tailored_resume_path TEXT;
ALTER TABLE jobs ADD COLUMN cover_letter_text TEXT;  -- TEXT для email, не путь к файлу
ALTER TABLE jobs ADD COLUMN prepared_at TIMESTAMPTZ;
```

**Job Status Flow:**

```
RAW → ANALYZED → INTERESTED → PREPARED → SENT
                     ↓
                  REJECTED
```

---

## ⚠️ Обработка ошибок

| Ошибка             | Action                     | Логирование          |
| ------------------ | -------------------------- | -------------------- |
| Resume not found   | Return 400, setup required | log.Error + ws.error |
| LLM timeout        | Retry 2x, then fail        | log.Warn на retry    |
| LLM rate limited   | Wait and retry             | log.Info             |
| PDF render fail    | Save MD, skip PDF          | log.Error            |
| Job not INTERESTED | Return 400, wrong status   | log.Warn             |

---

## Решённые вопросы

## Решённые вопросы

| Вопрос          | Решение                                                 |
| --------------- | ------------------------------------------------------- |
| LLM выбор       | OpenAI-compatible API через .env                        |
| PDF стиль       | Начинаем простой минималистичный, потом улучшаем        |
| PDF renderer    | chromedp (пригодится для HH parser в Phase 7)           |
| Markdown parser | НЕ используем goldmark, используем Go HTML templates    |
| Rate limiting   | 1 req/sec к LLM, hardcoded                              |
| NATS queue      | Задачи складываются в очередь, только job_id передаётся |
| API docs        | Scalar — современный UI с dark mode                     |
| Cover templates | 3 шаблона в XML (formal_ru, modern_ru, professional_en) |
| Версионирование | Пока не нужно, просто перезаписываем                    |

---

## ⚠️ TODO: UI Design

**ВАЖНО:** После реализации backend компонентов нужно спроектировать и реализовать UI!

### 1. Дизайн UI для Brain (в рамках Phase 3/4)

- **Кнопка "Prepare Response"** в job detail panel

  - Появляется только для jobs со статусом `INTERESTED`
  - Disabled во время обработки
  - Показывает loading state

- **Progress indicator** во время генерации

  - Progress bar с процентами (0% → 25% → 50% → 75% → 100%)
  - Текстовое описание текущего этапа
  - Анимированный spinner

- **Preview сгенерированных документов**

  - Markdown preview резюме и cover letter
  - Tabs для переключения между документами
  - Syntax highlighting для Markdown

- **Download section**
  - Resume PDF download кнопка (для attachment)
  - Cover letter copy-to-clipboard кнопка (для email/сообщения)
  - Timestamp генерации

### 2. WebSocket UI Integration (см. Phase 3.5)

Полная документация WebSocket событий: **[phase-3.5-websocket-events.md](phase-3.5-websocket-events.md)**

- **Real-time progress updates**

  - Подписка на канал `brain.{job_id}`
  - Обновление progress bar по событиям
  - Показ текущего этапа

- **Error handling**

  - Toast notifications для ошибок
  - Retry button при ошибке
  - Логирование ошибок в console

- **Notifications**
  - Success toast при завершении
  - Info toast при старте
  - Warning toast при timeout

### 3. Document Viewer

- **Markdown viewer**

  - Рендеринг Markdown в HTML
  - Стили для резюме (sections, lists, bold)
  - Copy to clipboard button

- **PDF viewer**
  - Embed PDF в iframe или
  - Open in new tab
  - Download button

### 4. Референсы для дизайна

- **Стиль:** Минималистичный + премиум (как в Phase 3)
- **Цвета:** Dark theme из `phase-3-webui.md`
- **Компоненты:** Переиспользовать из Phase 3 (buttons, cards, modals)
- **Анимации:** Smooth transitions, micro-interactions

---

## 🔮 Следующий шаг

После Brain переходим к **Фазе 5: Dispatcher** — автоматическая отправка откликов в Telegram/Email.

---

## 📝 Implementation Notes & Decisions

### Completed Implementation (Stages 1-7)

All core brain functionality has been implemented in the `phase-4-brain` worktree using TDD:

| Stage | Component | Tests | Status |
|-------|-----------|-------|--------|
| 1 | Storage (`internal/brain/storage.go`) | 3 | ✅ |
| 2 | LLM Integration with rate limiting (`internal/brain/llm.go`) | 3 | ✅ |
| 3 | Prompts with XML templates (`internal/brain/prompts.go`) | 2 | ✅ |
| 4 | PDF Rendering with chromedp (`internal/brain/pdf.go`) | 1 | ✅ |
| 5 | Service Layer pipeline (`internal/brain/service.go`) | 3 | ✅ |
| 6 | API Handlers (`internal/brain/api.go`) | 8 | ✅ |
| 7 | WebSocket Events (`internal/web/events.go`) | 4 | ✅ |

**Total: 24 tests passing**

### Key Design Decisions

1. **Cover letters are TEXT, not PDF** — Critical spec correction during implementation. Cover letters are generated as plain text for email/messaging, only resumes become PDF attachments.

2. **Rate limiting is hardcoded** — 1 req/sec via `time.Ticker` in the LLM client. This is intentional for simplicity; can be made configurable later if needed.

3. **WebSocket events at every step** — Pipeline emits progress at 0%, 25%, 50%, 75%, 100% with meaningful step names ("tailoring", "cover_letter", "pdf_rendering").

4. **Async processing pattern** — POST /prepare returns immediately (202 Accepted) with a `ws_channel` for clients to subscribe to updates.

5. **Interface-based design** — All components use interfaces (Storage, LLM, Renderer, Broadcaster) for testability.

### Files Created (Worktree: `../positions-os-phase4-brain`)

```
internal/brain/
├── storage.go           # File storage for resume/outputs
├── llm.go               # OpenAI-compatible client with rate limiting
├── prompts.go           # XML prompt loader via embed
├── pdf.go               # chromedp PDF renderer (resume only)
├── service.go           # Pipeline orchestrator with WS events
├── api.go               # HTTP handlers for /prepare, /documents, /download
├── integration.go       # Repository wrapper for service
├── *_test.go            # TDD tests (24 total)
└── *.go.md              # md-indexer documentation

docs/prompts/
├── resume-tailoring.xml
└── cover-letter.xml

static/pdf-templates/
├── resume.html          # Used for PDF generation
└── cover.html           # Exists but unused (cover is TEXT)
```

### Files Modified (Main Repo)

```
internal/web/events.go           # Added brain event helpers
internal/web/events_brain_test.go # Tests for brain events
docs/phase-4-brain.md             # This file (spec corrections)
```

### Pending Integration Work

To fully integrate Brain into the main application:

1. **Database Migration** — Add columns to `jobs` table:
   ```sql
   ALTER TABLE jobs ADD COLUMN tailored_resume_path TEXT;
   ALTER TABLE jobs ADD COLUMN cover_letter_text TEXT;
   ALTER TABLE jobs ADD COLUMN prepared_at TIMESTAMPTZ;
   ```

2. **Repository Implementation** — Implement `BrainJobRepository` interface that wraps existing `JobsRepository` and adds:
   - `GetJobData(id uuid.UUID) (map[string]string, error)` — Returns structured_data for LLM
   - `UpdateBrainOutputs(id, resumePath, coverText)` — Saves PDF path and cover text

3. **NATS Consumer** — Create consumer for `brain.jobs.prepare` subject that calls the prepare service.

4. **Main Service Integration** — Wire up brain handlers in `cmd/collector/main.go`:
   ```go
   brainService := brain.NewService(storage, llm, pdf)
   brainService.SetBroadcaster(hub)
   brainHandler := brain.NewHandler(brainRepo, brainSvc)
   server.RegisterBrainHandler(brainHandler)
   ```

5. **RegisterRoutes helper** — Add `RegisterBrainHandler` to `internal/web/server.go` following the existing pattern.

### Git Strategy Recommendations

**Current Situation:**
- Main repo (`main` branch): Has brain event changes in `internal/web/events.go`
- Worktree (`phase-4-brain` branch): Has complete brain package implementation

**Recommended Approach:**

Option A — **Merge worktree into main first** (Recommended):
```bash
# 1. Commit worktree changes
cd ../positions-os-phase4-brain
git add .
git commit -m "feat: implement phase-4 brain (resume tailoring, PDF, events)

- Storage layer for base resume and outputs
- LLM integration with rate limiting (1 req/s)
- XML prompt templates for resume/cover generation
- PDF rendering via chromedp (resume only, cover is TEXT)
- Service layer with WebSocket progress events
- REST API: POST /prepare, GET /documents, download resume.pdf
- Brain WebSocket events: started, progress, completed, error

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"

# 2. Switch to main, merge worktree branch
cd ../positions-os
git merge phase-4-brain --no-ff -m "Merge phase-4-brain: Brain service implementation"

# 3. Commit main repo changes together
git add internal/web/events.go internal/web/events_brain_test.go
git commit -m "feat(web): add brain WebSocket events"
```

Option B — **Create stacked PRs**:
1. PR for main repo changes (events.go) — small, focused
2. PR for worktree (brain package) — larger, independent
3. Merge events PR first, then brain PR

**My Recommendation:** Option A. The brain events in `events.go` are tightly coupled with the brain package. Merge them together to avoid merge conflicts and ensure consistency.

### Testing Before PR

```bash
# Run all tests
go test ./...

# Run brain package specifically
go test ./internal/brain/... -v

# Test with Chrome (for PDF)
go test ./internal/brain/... -v -run TestPDF

# Short mode (no Chrome)
go test ./internal/brain/... -v -short
```

### Open Questions / TODO

- [ ] Should rate limit be configurable via env var?
- [ ] Should we store cover letters in DB or only as files?
- [ ] Should PDF generation be retryable on failure?
- [ ] Should we add a "regenerate" endpoint for re-tailoring?
- [ ] Should brain events support wildcard subscriptions (e.g., `brain.*`)?

