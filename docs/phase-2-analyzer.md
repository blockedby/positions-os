# Фаза 2: Analyzer — План Реализации

## Обзор

Analyzer — это фоновый сервис для извлечения структурированных данных из сырых текстов вакансий с помощью LLM. На выходе:

- Автоматическая обработка вакансий со статусом `RAW`
- Извлечение структурированных данных (title, salary, technologies, contacts)
- Обновление статуса до `ANALYZED`
- Публикация событий в NATS

---

## Архитектура

```
┌─────────────────┐       jobs.new (job_id only)      ┌──────────────────────┐
│    Collector    │──────────────────────────────────►│   Analyzer Service   │
└─────────────────┘                                   │                      │
                                                      │  ┌────────────────┐  │
                                                      │  │ NATS Consumer  │  │
                                                      │  └───────┬────────┘  │
                                                      │          │           │
                                                      │  ┌───────▼────────┐  │
                                                      │  │ Job Processor  │  │
                                                      │  │  (fetch from   │  │
                                                      │  │   database)    │  │
                                                      │  └───────┬────────┘  │
                                                      │          │           │
                      ┌──────────────────────────────┼──┤ LLM Client    │  │
                      │                               │  └───────┬────────┘  │
                      │                               │          │           │
                      ▼                               │  ┌───────▼────────┐  │
          ┌───────────────────────┐                   │  │   Validator    │  │
          │   Local LLM Server    │                   │  └───────┬────────┘  │
          │  (LM Studio/Ollama)   │                   │          │           │
          │  or OpenAI API        │                   └──────────┼───────────┘
          └───────────────────────┘                              │
                                                     ┌───────────▼───────────┐
                                                     │      PostgreSQL       │
                                                     │  UPDATE structured_   │
                                                     │       data + status   │
                                                     └───────────────────────┘
```

### Почему только `job_id` в NATS?

| Аспект                 | job_id only            | Full data                 |
| ---------------------- | ---------------------- | ------------------------- |
| Трафик                 | ✅ Минимальный         | ❌ Килобайты на сообщение |
| Single source of truth | ✅ БД                  | ❌ Дублирование           |
| При retry              | ✅ Свежие данные из БД | ❌ Устаревшие данные      |
| Зависимость от БД      | ❌ Нужна               | ✅ Не нужна               |
| Сложность              | ✅ Проще               | ❌ Синхронизация          |

**Решение**: Передаём только `job_id`, Analyzer сам достаёт данные из БД.

---

## 🤖 LLM Integration

### Конфигурация (OpenAI-compatible)

```env
# LLM settings
LLM_BASE_URL=http://localhost:1234/v1  # LM Studio, Ollama, OpenAI
LLM_MODEL=gpt-4o-mini                  # модель
LLM_API_KEY=                           # пусто для локального
LLM_MAX_TOKENS=2048
LLM_TEMPERATURE=0.1
LLM_TIMEOUT_SECONDS=60
```

### Сравнение Go LLM библиотек

| Библиотека                 | Stars | Официальная  | Streaming | Structured Output | Azure Support |
| -------------------------- | ----- | ------------ | --------- | ----------------- | ------------- |
| **openai/openai-go**       | 1k+   | ✅ Да        | ✅        | ✅                | ✅            |
| **sashabaranov/go-openai** | 9k+   | ❌ Community | ✅        | ✅                | ✅            |
| **langchaingo**            | 5k+   | ❌ Framework | ✅        | Partial           | ✅            |

**Выбор**: `sashabaranov/go-openai` — самая популярная, feature-complete, активно поддерживается.

```bash
go get github.com/sashabaranov/go-openai
```

### Использование библиотеки

```go
// internal/llm/client.go
package llm

import (
    "context"
    "time"

    openai "github.com/sashabaranov/go-openai"
)

// Client обёртка над go-openai с нашими настройками.
type Client struct {
    client      *openai.Client
    model       string
    maxTokens   int
    temperature float32
    timeout     time.Duration
}

// Config настройки LLM клиента.
type Config struct {
    BaseURL     string
    Model       string
    APIKey      string
    MaxTokens   int
    Temperature float32
    Timeout     time.Duration
}

// NewClient создаёт LLM клиент.
func NewClient(cfg Config) *Client {
    config := openai.DefaultConfig(cfg.APIKey)
    config.BaseURL = cfg.BaseURL

    return &Client{
        client:      openai.NewClientWithConfig(config),
        model:       cfg.Model,
        maxTokens:   cfg.MaxTokens,
        temperature: cfg.Temperature,
        timeout:     cfg.Timeout,
    }
}

// ExtractJobData извлекает структурированные данные из текста вакансии.
// Использует промпт из файла docs/prompts/job-extraction.xml
func (c *Client) ExtractJobData(ctx context.Context, rawContent string, systemPrompt, userPrompt string) (string, error) {
    ctx, cancel := context.WithTimeout(ctx, c.timeout)
    defer cancel()

    resp, err := c.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
        Model: c.model,
        Messages: []openai.ChatCompletionMessage{
            {Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
            {Role: openai.ChatMessageRoleUser, Content: userPrompt},
        },
        MaxTokens:   c.maxTokens,
        Temperature: c.temperature,
    })
    if err != nil {
        return "", fmt.Errorf("llm completion: %w", err)
    }

    if len(resp.Choices) == 0 {
        return "", fmt.Errorf("no choices in response")
    }

    return resp.Choices[0].Message.Content, nil
}
```

### Загрузка промптов из файлов

```go
// internal/llm/prompts.go
package llm

import (
    "encoding/xml"
    "fmt"
    "os"
    "strings"
)

// PromptConfig промпт загруженный из XML файла.
// Содержит только SystemPrompt и UserPrompt.
type PromptConfig struct {
    XMLName    xml.Name `xml:"prompt"`
    System     string   `xml:"system"`      // системный промпт
    User       string   `xml:"user"`        // шаблон user промпта (с {{RAW_CONTENT}})
}

// LoadPrompt загружает промпт из XML файла.
func LoadPrompt(filepath string) (*PromptConfig, error) {
    data, err := os.ReadFile(filepath)
    if err != nil {
        return nil, fmt.Errorf("read prompt file: %w", err)
    }

    var config PromptConfig
    if err := xml.Unmarshal(data, &config); err != nil {
        return nil, fmt.Errorf("parse prompt xml: %w", err)
    }

    return &config, nil
}

// BuildUserPrompt заменяет {{RAW_CONTENT}} на реальный текст вакансии.
func (p *PromptConfig) BuildUserPrompt(rawContent string) string {
    return strings.ReplaceAll(p.User, "{{RAW_CONTENT}}", rawContent)
}
```

---

## 🚦 Telegram Rate Limits

### Официальные лимиты

| Тип операции       | Лимит       | Действие при превышении |
| ------------------ | ----------- | ----------------------- |
| Сообщения в личку  | 1 msg/sec   | FLOOD_WAIT_X            |
| Сообщения в группу | 20 msg/min  | FLOOD_WAIT_X            |
| Bulk операции      | ~30 msg/sec | HTTP 429                |
| API запросы        | ~20 req/sec | FLOOD_WAIT_X            |

### FLOOD_WAIT обработка

```go
// internal/telegram/ratelimit.go
package telegram

import (
    "context"
    "sync"
    "time"

    "golang.org/x/time/rate"
)

// RateLimiter контролирует частоту запросов к Telegram API.
type RateLimiter struct {
    // основной лимитер: 20 запросов в секунду
    limiter *rate.Limiter

    // дополнительная задержка после FLOOD_WAIT
    floodWaitUntil time.Time
    mu             sync.Mutex
}

// NewRateLimiter создаёт rate limiter для Telegram.
// rps - requests per second (рекомендуется 15-20)
// burst - допустимый burst (рекомендуется 5)
func NewRateLimiter(rps float64, burst int) *RateLimiter {
    return &RateLimiter{
        limiter: rate.NewLimiter(rate.Limit(rps), burst),
    }
}

// DefaultRateLimiter возвращает limiter с консервативными настройками.
func DefaultRateLimiter() *RateLimiter {
    return NewRateLimiter(15, 5) // 15 req/sec, burst 5
}

// Wait ждёт разрешения на следующий запрос.
func (r *RateLimiter) Wait(ctx context.Context) error {
    r.mu.Lock()
    waitUntil := r.floodWaitUntil
    r.mu.Unlock()

    // если есть flood wait — ждём его
    if time.Now().Before(waitUntil) {
        select {
        case <-time.After(time.Until(waitUntil)):
        case <-ctx.Done():
            return ctx.Err()
        }
    }

    return r.limiter.Wait(ctx)
}

// SetFloodWait устанавливает паузу после FLOOD_WAIT ошибки.
func (r *RateLimiter) SetFloodWait(seconds int) {
    r.mu.Lock()
    defer r.mu.Unlock()

    r.floodWaitUntil = time.Now().Add(time.Duration(seconds) * time.Second)
}

// Config для настройки лимитов.
type RateLimitConfig struct {
    RequestsPerSecond float64 `env:"TG_RATE_LIMIT_RPS" default:"15"`
    BurstSize         int     `env:"TG_RATE_LIMIT_BURST" default:"5"`

    // задержки между разными операциями
    MessageDelay      time.Duration `env:"TG_MESSAGE_DELAY" default:"100ms"`
    HistoryDelay      time.Duration `env:"TG_HISTORY_DELAY" default:"500ms"`
}
```

### Интеграция в парсер

```go
// internal/telegram/parser.go
func (p *Parser) ParseChannel(ctx context.Context, opts ParseOptions) ([]Message, error) {
    var allMessages []Message
    offsetID := 0

    for {
        // ждём разрешения от rate limiter
        if err := p.rateLimiter.Wait(ctx); err != nil {
            return allMessages, err
        }

        messages, err := p.getHistory(ctx, opts.Channel, offsetID, 100)
        if err != nil {
            // проверяем на FLOOD_WAIT
            if floodWait := extractFloodWait(err); floodWait > 0 {
                p.logger.Warn().
                    Int("seconds", floodWait).
                    Msg("received FLOOD_WAIT, backing off")
                p.rateLimiter.SetFloodWait(floodWait)
                continue // retry после паузы
            }
            return allMessages, err
        }

        if len(messages) == 0 {
            break
        }

        allMessages = append(allMessages, messages...)
        offsetID = messages[len(messages)-1].ID

        // дополнительная задержка между batch запросами
        time.Sleep(p.config.HistoryDelay)
    }

    return allMessages, nil
}

// extractFloodWait извлекает время ожидания из FLOOD_WAIT ошибки.
func extractFloodWait(err error) int {
    // gotgproto/gotd обычно возвращает ошибку с типом *tg.Error
    // проверяем на FLOOD_WAIT
    // ...
    return 0
}
```

---

## 📋 Промпты в XML

Все промпты хранятся в `docs/prompts/` в XML формате с кастомными тегами:

```
docs/prompts/
├── job-extraction.xml     # извлечение данных из вакансий
├── job-filtering.xml      # фильтрация нерелевантных (будущее)
└── resume-tailoring.xml   # адаптация резюме (будущее)
```

См. `docs/prompts/job-extraction.xml` для полного примера.

---

## 📁 Структура файлов

```
positions-os/
├── cmd/
│   └── analyzer/
│       └── main.go              # Entry point
├── internal/
│   ├── llm/
│   │   ├── client.go            # go-openai wrapper
│   │   └── prompts.go           # XML prompt loader
│   ├── analyzer/
│   │   ├── processor.go         # Job processing logic
│   │   ├── consumer.go          # NATS consumer
│   │   └── validator.go         # Data validation
│   ├── telegram/
│   │   └── ratelimit.go         # Rate limiting
│   └── repository/
│       └── jobs.go              # GetByID, UpdateStructuredData
├── docs/
│   └── prompts/
│       └── job-extraction.xml   # Extraction prompt
└── scripts/
    └── test-analyzer.sh         # Integration test
```

---

## 🎯 Порядок реализации

### Этап 1: LLM Client

- [x] 2.1.1 — `go get github.com/sashabaranov/go-openai`
- [x] 2.1.2 — `internal/llm/client.go` wrapper
- [x] 2.1.3 — `internal/llm/prompts.go` XML loader
- [x] 2.1.4 — Тест с LM Studio (`go test -tags=integration ./internal/llm/...`)

### Этап 2: Rate Limiter

- [x] 2.2.1 — `internal/telegram/ratelimit.go`
- [x] 2.2.2 — Интеграция в parser
- [x] 2.2.3 — FLOOD_WAIT handling

### Этап 3: Processor

- [x] 2.3.1 — `internal/analyzer/processor.go`
- [x] 2.3.2 — JSON parsing & cleanup
- [x] 2.3.3 — Validator
- [x] 2.3.4 — Unit tests

### Этап 4: Repository Updates

- [x] 2.4.1 — GetByID
- [x] 2.4.2 — UpdateStructuredData
- [x] 2.4.3 — UpdateStatus

### Этап 5: NATS Consumer

- [x] 2.5.1 — `internal/analyzer/consumer.go`
- [x] 2.5.2 — Subscribe to jobs.new
- [x] 2.5.3 — Error handling & retry

### Этап 6: Main Service

- [x] 2.6.1 — `cmd/analyzer/main.go`
- [x] 2.6.2 — Graceful shutdown
- [x] 2.6.3 — Dockerfile

### Этап 7: Testing

- [x] 2.7.1 — Integration test script
- [ ] 2.7.2 — Manual E2E test (deferred to Phase 3)

---

## ⚠️ Обработка ошибок

| Ошибка       | Action                 | Retry        |
| ------------ | ---------------------- | ------------ |
| LLM timeout  | Nak, retry             | 3x с backoff |
| FLOOD_WAIT   | Pause, retry           | После wait   |
| Invalid JSON | Save partial + warning | No retry     |
| DB error     | Nak, retry             | Infinite     |

---

## 🔮 Следующий шаг

После Analyzer переходим к **Фазе 3: Web UI** — интерфейс для просмотра и фильтрации вакансий.
