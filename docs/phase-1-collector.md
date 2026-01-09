# Фаза 1: Collector — План Реализации

## Обзор

Collector — это API-сервис для парсинга Telegram каналов и групп. На выходе:

- REST API для запуска/остановки парсинга
- Сохранение вакансий в БД
- Публикация событий в NATS

---

## Архитектура

```
                                    ┌─────────────────┐
                                    │   Telegram API  │
                                    │    (MTProto)    │
                                    └────────┬────────┘
                                             │
┌──────────────┐    HTTP     ┌───────────────▼───────────────┐
│   Web UI     │◄───────────►│       Collector Service       │
│  (будущее)   │             │                               │
└──────────────┘             │  ┌─────────────────────────┐  │
                             │  │     REST API Server     │  │
                             │  │  POST /scrape/telegram  │  │
                             │  │  DELETE /scrape/current │  │
                             │  └───────────┬─────────────┘  │
                             │              │                │
                             │  ┌───────────▼─────────────┐  │
                             │  │    Scraper Engine       │  │
                             │  │  ┌───────────────────┐  │  │
                             │  │  │ Telegram Strategy │  │  │
                             │  │  │   (gotgproto)     │  │  │
                             │  │  └───────────────────┘  │  │
                             │  └───────────┬─────────────┘  │
                             │              │                │
                             └──────────────┼────────────────┘
                                            │
                    ┌───────────────────────┼───────────────────────┐
                    │                       │                       │
                    ▼                       ▼                       ▼
            ┌───────────────┐       ┌───────────────┐       ┌───────────────┐
            │   PostgreSQL  │       │     NATS      │       │   Log Files   │
            │    (jobs)     │       │  (jobs.new)   │       │ (collector)   │
            └───────────────┘       └───────────────┘       └───────────────┘
```

---

## 📚 Telegram API: Базовые Концепции

### Что такое MTProto?

MTProto — это протокол Telegram для взаимодействия клиентов с серверами. В отличие от Bot API, MTProto даёт полный доступ ко всем функциям Telegram как обычному пользователю.

### Основные объекты Telegram

#### 1. Channel / Supergroup

Канал или супергруппа идентифицируется двумя полями:

| Поле          | Тип    | Описание                                            |
| ------------- | ------ | --------------------------------------------------- |
| `channel_id`  | int64  | Уникальный ID канала (числовой)                     |
| `access_hash` | int64  | Токен доступа (нужен для запросов)                  |
| `username`    | string | Username канала (например `go_jobs` для `@go_jobs`) |

**Пример:**

```go
// Резолвим username в InputChannel
resolved, _ := client.API().ContactsResolveUsername(ctx, "go_jobs")
channel := resolved.Chats[0].(*tg.Channel)

fmt.Println(channel.ID)         // 1234567890
fmt.Println(channel.AccessHash) // 8765432109876543210
fmt.Println(channel.Username)   // "go_jobs"
```

#### 2. Message

Сообщение в канале/группе:

| Поле       | Тип                | Описание                            |
| ---------- | ------------------ | ----------------------------------- |
| `id`       | int                | ID сообщения (уникален внутри чата) |
| `message`  | string             | Текст сообщения                     |
| `date`     | int                | Unix timestamp создания             |
| `from_id`  | PeerClass          | Кто отправил                        |
| `reply_to` | MessageReplyHeader | Если это ответ — на что             |
| `views`    | int                | Количество просмотров               |
| `forwards` | int                | Количество пересылок                |

**Важно**: `message.id` — это **порядковый номер** внутри чата. Новые сообщения получают бо́льший ID. Это позволяет использовать ID для инкрементального парсинга.

#### 3. ForumTopic

Топик (подчат) внутри супергруппы-форума:

| Поле            | Тип    | Описание                          |
| --------------- | ------ | --------------------------------- |
| `id`            | int    | ID топика (= `message_thread_id`) |
| `title`         | string | Название топика                   |
| `icon_color`    | int    | Цвет иконки (RGB)                 |
| `icon_emoji_id` | int64  | ID кастомного эмодзи              |
| `top_message`   | int    | ID последнего сообщения           |
| `closed`        | bool   | Закрыт ли топик                   |
| `pinned`        | bool   | Закреплён ли топик                |

**Особенности**:

- Топик "General" всегда имеет `id = 1`
- ID топика равен ID сервисного сообщения о его создании
- Сообщения в топике имеют `reply_to.forum_topic = true`

### Ключевые методы API

| Метод                      | Описание                | Возвращает   |
| -------------------------- | ----------------------- | ------------ |
| `contacts.resolveUsername` | Username → Channel/User | ResolvedPeer |
| `channels.getFullChannel`  | Полная инфа о канале    | ChannelFull  |
| `messages.getHistory`      | История сообщений       | Messages     |
| `channels.getForumTopics`  | Список топиков форума   | ForumTopics  |
| `messages.getReplies`      | Сообщения в топике      | Messages     |

---

## Библиотека gotgproto

### Почему gotgproto?

`gotgproto` — это высокоуровневая обёртка над `gotd/td` (низкоуровневый MTProto клиент).

**Преимущества**:

- Автоматическое управление сессиями (session string)
- Встроенный PeerStorage (кэширует access_hash)
- Обработка FloodWait из коробки
- Dispatcher для обработки событий
- Упрощённый API

### Инициализация клиента

```go
package main

import (
    "context"
    "os"

    "github.com/celestix/gotgproto"
    "github.com/celestix/gotgproto/sessionMaker"
)

func main() {
    ctx := context.Background()

    // создаём клиент
    client, err := gotgproto.NewClient(
        os.Getenv("TG_API_ID"),      // API ID (int)
        os.Getenv("TG_API_HASH"),    // API Hash (string)
        gotgproto.ClientTypePhone(""), // пустой = используем session string
        &gotgproto.ClientOpts{
            Session: sessionMaker.NewSession(
                os.Getenv("TG_SESSION_STRING"),
                sessionMaker.StringSession,
            ),
        },
    )
    if err != nil {
        panic(err)
    }
    defer client.Stop()

    // получаем API клиент для вызова методов
    api := client.API()

    // теперь можно делать запросы
    // ...
}
```

### Получение истории канала

```go
import "github.com/gotd/td/tg"

// resolveChannel резолвит username канала в InputPeerChannel.
func resolveChannel(ctx context.Context, client *gotgproto.Client, username string) (*tg.InputPeerChannel, error) {
    resolved, err := client.API().ContactsResolveUsername(ctx, username)
    if err != nil {
        return nil, fmt.Errorf("resolve username %s: %w", username, err)
    }

    if len(resolved.Chats) == 0 {
        return nil, fmt.Errorf("channel %s not found", username)
    }

    channel, ok := resolved.Chats[0].(*tg.Channel)
    if !ok {
        return nil, fmt.Errorf("%s is not a channel", username)
    }

    return &tg.InputPeerChannel{
        ChannelID:  channel.ID,
        AccessHash: channel.AccessHash,
    }, nil
}

// getChannelMessages получает сообщения канала.
// offsetID = 0 означает "с самого нового"
// offsetID = N означает "сообщения старше чем N"
func getChannelMessages(
    ctx context.Context,
    client *gotgproto.Client,
    peer *tg.InputPeerChannel,
    offsetID int,
    limit int,
) ([]tg.Message, error) {
    history, err := client.API().MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
        Peer:     peer,
        OffsetID: offsetID,
        Limit:    limit,
    })
    if err != nil {
        return nil, fmt.Errorf("get history: %w", err)
    }

    var messages []tg.Message

    switch h := history.(type) {
    case *tg.MessagesChannelMessages:
        for _, msg := range h.Messages {
            if m, ok := msg.(*tg.Message); ok {
                messages = append(messages, *m)
            }
        }
    }

    return messages, nil
}
```

### Получение списка топиков

```go
// ForumTopic представляет топик форума с нужными полями.
type ForumTopic struct {
    ID         int    // уникальный ID топика
    Title      string // название топика
    TopMessage int    // ID последнего сообщения
    Closed     bool   // закрыт ли
    Pinned     bool   // закреплён ли
}

// getForumTopics получает список всех топиков в форуме.
func getForumTopics(
    ctx context.Context,
    client *gotgproto.Client,
    channelID int64,
    accessHash int64,
) ([]ForumTopic, error) {
    inputChannel := &tg.InputChannel{
        ChannelID:  channelID,
        AccessHash: accessHash,
    }

    result, err := client.API().ChannelsGetForumTopics(ctx, &tg.ChannelsGetForumTopicsRequest{
        Channel: inputChannel,
        Limit:   100, // максимум топиков
    })
    if err != nil {
        return nil, fmt.Errorf("get forum topics: %w", err)
    }

    var topics []ForumTopic

    switch t := result.(type) {
    case *tg.MessagesForumTopics:
        for _, topic := range t.Topics {
            if ft, ok := topic.(*tg.ForumTopic); ok {
                topics = append(topics, ForumTopic{
                    ID:         ft.ID,
                    Title:      ft.Title,
                    TopMessage: ft.TopMessage,
                    Closed:     ft.Closed,
                    Pinned:     ft.Pinned,
                })
            }
        }
    }

    return topics, nil
}
```

---

## Компоненты

### 1.1 TG Auth Tool

**Цель**: Утилита CLI для генерации Telegram session string.

| Файл                  | Описание                    |
| --------------------- | --------------------------- |
| `cmd/tg-auth/main.go` | CLI утилита для авторизации |

**Флоу**:

1. Пользователь запускает утилиту
2. Вводит номер телефона
3. Получает код в Telegram
4. Вводит код
5. Получает session string для `.env`

---

### 1.2 TG Topics Lister

**Цель**: Утилита CLI для просмотра топиков в форуме.

Нужна для того, чтобы пользователь мог:

1. Увидеть список топиков с их ID и названиями
2. Выбрать какие топики парсить
3. Скопировать ID для конфигурации

| Файл                    | Описание                 |
| ----------------------- | ------------------------ |
| `cmd/tg-topics/main.go` | CLI для листинга топиков |

**Использование**:

```bash
$ go run cmd/tg-topics/main.go @my_forum_group

Forum: My Job Forum (@my_forum_group)
Total topics: 5

ID     | Title                | Messages | Status
-------|----------------------|----------|--------
1      | General              | 1547     | open
15     | Go Vacancies         | 342      | open
28     | Python Jobs          | 891      | open
45     | Remote Only          | 156      | open
67     | Archived             | 23       | closed
```

---

### 1.3 TG Channel Parser

**Цель**: Парсинг сообщений из публичных каналов.

| Файл                          | Описание                |
| ----------------------------- | ----------------------- |
| `internal/telegram/client.go` | Обёртка над gotgproto   |
| `internal/telegram/parser.go` | Логика парсинга каналов |
| `internal/telegram/types.go`  | Типы для TG сообщений   |

---

### 1.4 TG Forum Topics

**Цель**: Парсинг выбранных топиков (подчатов) в supergroup.

**Как это работает:**

1. **Получение списка топиков** — через `channels.getForumTopics`:

   - Возвращает массив `ForumTopic` объектов
   - У каждого есть `id` (число) и `title` (строка)
   - Топик "General" всегда id=1

2. **Связь ID и названия**:

   - ID топика — это целое число
   - Название — строка `title`
   - В конфиге указываем массив ID: `topic_ids: [15, 28, 45]`

3. **Парсинг сообщений топика** — через `messages.getReplies`:
   - Передаём `msg_id = topic_id`
   - Получаем все сообщения этого топика

**Конфигурация источника с топиками**:

```json
{
  "name": "Go Jobs Forum",
  "type": "TG_FORUM",
  "url": "@go_jobs_forum",
  "metadata": {
    "topic_ids": [15, 28], // парсить только эти топики
    "topic_names": {
      // для логов/отображения
      "15": "Go Vacancies",
      "28": "Remote Jobs"
    }
  }
}
```

---

### 1.5 Deduplication: Умная стратегия по Message ID

**Цель**: Эффективно определять какие сообщения уже спарсены.

**Ключевое наблюдение**: Message ID в Telegram — это **порядковый номер**. Новые сообщения всегда получают бо́льший ID. Если посты не удаляются, можно использовать диапазоны.

#### Концепция "Заполненных промежутков"

Храним в БД диапазоны уже спарсенных message_id:

```sql
-- таблица для хранения спарсенных диапазонов
CREATE TABLE parsed_ranges (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    target_id   UUID NOT NULL REFERENCES scraping_targets(id),
    min_msg_id  BIGINT NOT NULL,  -- начало диапазона
    max_msg_id  BIGINT NOT NULL,  -- конец диапазона
    created_at  TIMESTAMPTZ DEFAULT NOW()
);
```

#### Алгоритм парсинга

```
1. Получаем последние N сообщений канала
2. Извлекаем их message_id (например: [1050, 1049, 1048, 1045, 1044, 1040])
3. Получаем уже спарсенные диапазоны из БД (например: [1000-1042])
4. Находим новые сообщения: те ID что > max(спарсенных)
   → новые: [1050, 1049, 1048, 1045, 1044] (всё что > 1042)
5. Парсим только новые
6. Обновляем диапазон: [1000-1050]
```

#### Реализация

```go
// ParsedRange представляет диапазон спарсенных сообщений.
type ParsedRange struct {
    TargetID uuid.UUID
    MinMsgID int64
    MaxMsgID int64
}

// MessageIDFilter фильтрует уже спарсенные сообщения.
type MessageIDFilter struct {
    ranges []ParsedRange
}

// NewFilter возвращает новый фильтр, который знает какие
// сообщения уже были спарсены для данного target.
func (r *Repository) NewFilter(ctx context.Context, targetID uuid.UUID) (*MessageIDFilter, error) {
    ranges, err := r.GetParsedRanges(ctx, targetID)
    if err != nil {
        return nil, err
    }
    return &MessageIDFilter{ranges: ranges}, nil
}

// FilterNew возвращает только те message ID, которые ещё не спарсены.
func (f *MessageIDFilter) FilterNew(messageIDs []int64) []int64 {
    if len(f.ranges) == 0 {
        return messageIDs // всё новое
    }

    // находим максимальный спарсенный ID
    var maxParsed int64
    for _, r := range f.ranges {
        if r.MaxMsgID > maxParsed {
            maxParsed = r.MaxMsgID
        }
    }

    // оставляем только те что больше
    var newIDs []int64
    for _, id := range messageIDs {
        if id > maxParsed {
            newIDs = append(newIDs, id)
        }
    }

    return newIDs
}

// UpdateRange обновляет диапазон спарсенных сообщений.
func (r *Repository) UpdateRange(ctx context.Context, targetID uuid.UUID, minID, maxID int64) error {
    // upsert: создать или расширить существующий диапазон
    _, err := r.db.Pool.Exec(ctx, `
        INSERT INTO parsed_ranges (target_id, min_msg_id, max_msg_id)
        VALUES ($1, $2, $3)
        ON CONFLICT (target_id)
        DO UPDATE SET
            min_msg_id = LEAST(parsed_ranges.min_msg_id, $2),
            max_msg_id = GREATEST(parsed_ranges.max_msg_id, $3),
            created_at = NOW()
    `, targetID, minID, maxID)
    return err
}
```

**Преимущества**:

- Не нужно проверять каждое сообщение в БД
- Один запрос на получение диапазона
- Эффективно для больших каналов
- Работает корректно если посты не удаляются

---

### 1.6 NATS Integration

**Цель**: Публикация событий о новых вакансиях.

**Stream**: `JOBS`
**Subject**: `jobs.new`

```go
// JobNewEvent представляет событие о новой вакансии.
// Публикуется в NATS stream JOBS, subject jobs.new.
type JobNewEvent struct {
    JobID      uuid.UUID `json:"job_id"`      // ID вакансии в нашей БД
    TargetID   uuid.UUID `json:"target_id"`   // ID источника
    ExternalID string    `json:"external_id"` // message_id в Telegram
    RawContent string    `json:"raw_content"` // текст сообщения
    CreatedAt  time.Time `json:"created_at"`  // время создания
}
```

---

### 1.7 REST API для запуска

**Цель**: HTTP endpoint для триггера парсинга.

| Endpoint                     | Метод  | Описание                 |
| ---------------------------- | ------ | ------------------------ |
| `/api/v1/scrape/telegram`    | POST   | Запустить парсинг        |
| `/api/v1/scrape/current`     | DELETE | Остановить текущий       |
| `/api/v1/scrape/status`      | GET    | Статус парсинга          |
| `/api/v1/targets`            | GET    | Список источников        |
| `/api/v1/targets`            | POST   | Добавить источник        |
| `/api/v1/targets/:id/topics` | GET    | Список топиков источника |
| `/health`                    | GET    | Healthcheck              |

#### Request: POST /api/v1/scrape/telegram

```go
// ScrapeRequest представляет запрос на парсинг Telegram канала.
type ScrapeRequest struct {
    // TargetID — ID из таблицы scraping_targets.
    // Если указан, channel игнорируется.
    TargetID *uuid.UUID `json:"target_id,omitempty"`

    // Channel — username канала (например "@go_jobs").
    // Используется если TargetID не указан.
    Channel string `json:"channel,omitempty"`

    // Limit — максимальное количество сообщений для парсинга.
    // 0 означает без лимита (парсить все доступные).
    Limit int `json:"limit,omitempty"`

    // Until — дата до которой парсить (формат YYYY-MM-DD).
    // Сообщения старше этой даты игнорируются.
    Until string `json:"until,omitempty"`

    // TopicIDs — список ID топиков для парсинга.
    // Только для TG_FORUM источников.
    // Если пусто — парсятся все топики.
    TopicIDs []int `json:"topic_ids,omitempty"`
}
```

#### Валидация запроса

```go
// ErrValidation представляет ошибку валидации.
var (
    ErrChannelRequired   = errors.New("either target_id or channel is required")
    ErrChannelNotFound   = errors.New("channel not found")
    ErrNotAChannel       = errors.New("specified username is not a channel")
    ErrInvalidDate       = errors.New("until date must be in YYYY-MM-DD format")
    ErrFutureDate        = errors.New("until date cannot be in the future")
    ErrInvalidLimit      = errors.New("limit must be positive")
    ErrTopicsForForum    = errors.New("topic_ids can only be used with TG_FORUM targets")
    ErrTopicNotFound     = errors.New("one or more topic_ids not found in the forum")
)

// Validate проверяет корректность запроса.
func (r *ScrapeRequest) Validate(ctx context.Context, tgClient TelegramClient) error {
    // 1. Проверяем что указан источник
    if r.TargetID == nil && r.Channel == "" {
        return ErrChannelRequired
    }

    // 2. Проверяем что канал существует
    if r.Channel != "" {
        exists, err := tgClient.ChannelExists(ctx, r.Channel)
        if err != nil {
            return fmt.Errorf("check channel: %w", err)
        }
        if !exists {
            return ErrChannelNotFound
        }
    }

    // 3. Проверяем формат и значение даты
    if r.Until != "" {
        until, err := time.Parse("2006-01-02", r.Until)
        if err != nil {
            return ErrInvalidDate
        }
        if until.After(time.Now()) {
            return ErrFutureDate
        }
    }

    // 4. Проверяем limit
    if r.Limit < 0 {
        return ErrInvalidLimit
    }

    // 5. Проверяем топики (если указаны)
    if len(r.TopicIDs) > 0 {
        // топики можно указать только для форумов
        // проверка на существование топиков делается в сервисе
    }

    return nil
}
```

#### Response

```go
// ScrapeResponse представляет ответ на запрос парсинга.
type ScrapeResponse struct {
    ScrapeID  uuid.UUID     `json:"scrape_id"`  // уникальный ID задачи парсинга
    Status    string        `json:"status"`     // "running" | "completed" | "failed"
    Target    TargetInfo    `json:"target"`     // информация об источнике
    StartedAt time.Time     `json:"started_at"` // время начала
}

// TargetInfo содержит краткую информацию об источнике.
type TargetInfo struct {
    ID      uuid.UUID `json:"id"`
    Name    string    `json:"name"`
    Channel string    `json:"channel"`
}
```

---

### 1.8 REST API для остановки

**Цель**: Graceful shutdown текущего парсинга.

**Механизм**: Context cancellation

```go
// ScrapeManager управляет активными задачами парсинга.
// Гарантирует что одновременно выполняется только одна задача.
// Потокобезопасен.
type ScrapeManager struct {
    mu       sync.Mutex      // защита от race conditions
    current  *ScrapeJob      // текущая активная задача
    cancelFn context.CancelFunc // функция отмены
    logger   *logger.Logger  // логгер
}

// ScrapeJob представляет активную задачу парсинга.
type ScrapeJob struct {
    ID        uuid.UUID    // уникальный ID задачи
    TargetID  uuid.UUID    // ID источника
    StartedAt time.Time    // время начала
    Options   ScrapeOptions // параметры парсинга
}

// NewScrapeManager создаёт новый менеджер задач парсинга.
func NewScrapeManager(logger *logger.Logger) *ScrapeManager {
    return &ScrapeManager{
        logger: logger,
    }
}

// Start запускает новую задачу парсинга.
// Возвращает ErrAlreadyRunning если уже выполняется другая задача.
func (m *ScrapeManager) Start(ctx context.Context, opts ScrapeOptions) (*ScrapeJob, error) {
    m.mu.Lock()
    defer m.mu.Unlock()

    if m.current != nil {
        return nil, ErrAlreadyRunning
    }

    ctx, cancel := context.WithCancel(ctx)
    m.cancelFn = cancel

    job := &ScrapeJob{
        ID:        uuid.New(),
        TargetID:  opts.TargetID,
        StartedAt: time.Now(),
        Options:   opts,
    }
    m.current = job

    go m.run(ctx, job, opts)

    return job, nil
}

// Stop останавливает текущую задачу парсинга.
// Если нет активной задачи, ничего не делает.
func (m *ScrapeManager) Stop() {
    m.mu.Lock()
    defer m.mu.Unlock()

    if m.cancelFn != nil {
        m.logger.Info().Msg("stopping current scrape job")
        m.cancelFn()
        m.cancelFn = nil
        m.current = nil
    }
}

// Current возвращает информацию о текущей задаче.
// Возвращает nil если нет активной задачи.
func (m *ScrapeManager) Current() *ScrapeJob {
    m.mu.Lock()
    defer m.mu.Unlock()
    return m.current
}

// run выполняет задачу парсинга в отдельной горутине.
func (m *ScrapeManager) run(ctx context.Context, job *ScrapeJob, opts ScrapeOptions) {
    defer func() {
        m.mu.Lock()
        m.current = nil
        m.cancelFn = nil
        m.mu.Unlock()
    }()

    m.logger.Info().
        Str("job_id", job.ID.String()).
        Str("target_id", job.TargetID.String()).
        Msg("starting scrape job")

    // здесь вызываем service.Scrape(ctx, opts)
    // ...

    m.logger.Info().
        Str("job_id", job.ID.String()).
        Msg("scrape job completed")
}
```

---

### 1.9 File Logging

**Цель**: Логирование в файл для отладки.

**Файл**: `./logs/collector.log`
**Формат**: JSON lines

```go
// использует уже созданный internal/logger
logger.Init("debug", "./logs/collector.log")
```

---

## Структура файлов

```
positions-os/
├── cmd/
│   ├── tg-auth/
│   │   └── main.go              # CLI для авторизации TG
│   ├── tg-topics/
│   │   └── main.go              # CLI для просмотра топиков
│   └── collector/
│       └── main.go              # Точка входа сервиса
├── internal/
│   ├── config/                  # ✅ уже есть
│   ├── database/                # ✅ уже есть
│   ├── logger/                  # ✅ уже есть
│   ├── models/                  # ✅ уже есть
│   ├── nats/                    # ✅ уже есть
│   ├── repository/
│   │   ├── jobs.go              # CRUD для jobs
│   │   ├── targets.go           # CRUD для targets
│   │   └── ranges.go            # работа с parsed_ranges
│   ├── telegram/
│   │   ├── client.go            # TG клиент
│   │   ├── parser.go            # Логика парсинга
│   │   └── types.go             # TG типы
│   └── collector/
│       ├── handler.go           # HTTP handlers
│       ├── router.go            # Chi router
│       ├── service.go           # Бизнес-логика
│       ├── manager.go           # Управление scrape jobs
│       └── validation.go        # Валидация запросов
└── ...
```

---

## Дополнительная миграция

Для хранения диапазонов спарсенных сообщений:

```sql
-- 0005_create_parsed_ranges.up.sql

CREATE TABLE parsed_ranges (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    target_id   UUID NOT NULL REFERENCES scraping_targets(id) ON DELETE CASCADE,
    min_msg_id  BIGINT NOT NULL,
    max_msg_id  BIGINT NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT uq_parsed_ranges_target UNIQUE (target_id)
);

CREATE INDEX idx_parsed_ranges_target ON parsed_ranges (target_id);

COMMENT ON TABLE parsed_ranges IS 'Диапазоны уже спарсенных message_id для каждого источника';
```

---

## Порядок реализации

### Этап 1: TG Auth (первым!)

1. [ ] Создать `cmd/tg-auth/main.go`
2. [ ] Тестовая авторизация
3. [ ] Получить session string
4. [ ] Сохранить в `.env`

### Этап 2: TG Topics Lister

1. [ ] Создать `cmd/tg-topics/main.go`
2. [ ] Вывод списка топиков с ID и названиями

### Этап 3: Repository Layer

1. [ ] `internal/repository/targets.go`
   - GetByID, GetActive, Create, Update
2. [ ] `internal/repository/jobs.go`
   - Create, Exists, GetByStatus
3. [ ] `internal/repository/ranges.go`
   - GetRange, UpdateRange
4. [ ] Миграция `0005_create_parsed_ranges`

### Этап 4: Telegram Client

1. [ ] `internal/telegram/client.go`
   - Инициализация gotgproto
   - Подключение через session string
   - ChannelExists для валидации
2. [ ] `internal/telegram/parser.go`
   - ParseChannel с условиями остановки
   - ParseForumTopics
3. [ ] `internal/telegram/types.go`
   - Message, Topic, Channel

### Этап 5: Collector Service

1. [ ] `internal/collector/service.go`
   - Orchestration парсинга
   - Сохранение в БД
   - Публикация в NATS
2. [ ] `internal/collector/manager.go`
   - Start/Stop scrape jobs
   - Concurrent safety
3. [ ] `internal/collector/validation.go`
   - Валидация с проверкой канала

### Этап 6: HTTP API

1. [ ] `internal/collector/router.go`
   - Chi router setup
2. [ ] `internal/collector/handler.go`
   - POST /scrape/telegram
   - DELETE /scrape/current
   - GET /health
3. [ ] `cmd/collector/main.go`
   - Wiring всех компонентов

### Этап 7: Тестирование

1. [ ] Парсинг тестового канала
2. [ ] Проверка дедупликации по диапазонам
3. [ ] Проверка NATS событий
4. [ ] Проверка остановки

---

## Зависимости

```bash
# telegram mtproto client
go get github.com/celestix/gotgproto

# http router
go get github.com/go-chi/chi/v5
```

---

## Чеклист

### Этап 1: TG Auth

- [ ] 1.1.1 — Установить gotgproto
- [ ] 1.1.2 — Создать cmd/tg-auth
- [ ] 1.1.3 — Получить session string
- [ ] 1.1.4 — Сохранить в .env

### Этап 2: TG Topics

- [ ] 1.2.1 — Создать cmd/tg-topics
- [ ] 1.2.2 — Вывод ID и названий топиков

### Этап 3: Repository

- [ ] 1.3.1 — targets repository
- [ ] 1.3.2 — jobs repository
- [ ] 1.3.3 — ranges repository
- [ ] 1.3.4 — миграция parsed_ranges

### Этап 4: Telegram

- [ ] 1.4.1 — TG client wrapper
- [ ] 1.4.2 — Channel parser
- [ ] 1.4.3 — Forum topics parser
- [ ] 1.4.4 — ChannelExists валидация

### Этап 5: Service

- [ ] 1.5.1 — Collector service
- [ ] 1.5.2 — Scrape manager
- [ ] 1.5.3 — Request validation
- [ ] 1.5.4 — NATS publishing

### Этап 6: API

- [ ] 1.6.1 — Chi router
- [ ] 1.6.2 — HTTP handlers
- [ ] 1.6.3 — main.go entry point

### Этап 7: Testing

- [ ] 1.7.1 — Parse test channel
- [ ] 1.7.2 — Verify range-based dedup
- [ ] 1.7.3 — Verify NATS events
- [ ] 1.7.4 — Verify stop functionality

---

## 🧪 Тестирование

### Unit Tests

Цель: покрыть основные сценарии и точки отказа. Не стремимся к 100% покрытию.

#### Repository Layer

**Файл**: `internal/repository/jobs_test.go`

```go
func TestJobsRepository_Create(t *testing.T) {
    // успешное создание
    t.Run("creates job successfully", func(t *testing.T) {
        // ...
    })

    // дубликат external_id
    t.Run("returns error on duplicate external_id", func(t *testing.T) {
        // ...
    })
}

func TestJobsRepository_Exists(t *testing.T) {
    t.Run("returns true for existing job", func(t *testing.T) {})
    t.Run("returns false for non-existing job", func(t *testing.T) {})
}
```

**Файл**: `internal/repository/ranges_test.go`

```go
func TestRangesRepository_GetRange(t *testing.T) {
    t.Run("returns empty for new target", func(t *testing.T) {})
    t.Run("returns existing range", func(t *testing.T) {})
}

func TestRangesRepository_UpdateRange(t *testing.T) {
    t.Run("creates new range", func(t *testing.T) {})
    t.Run("extends existing range upward", func(t *testing.T) {})
    t.Run("extends existing range downward", func(t *testing.T) {})
}
```

#### Message ID Filter

**Файл**: `internal/repository/filter_test.go`

| Тест                     | Сценарий                  | Ожидание                |
| ------------------------ | ------------------------- | ----------------------- |
| `TestFilter_EmptyRanges` | Нет спарсенных диапазонов | Все ID считаются новыми |
| `TestFilter_AllNew`      | Все ID > max_parsed       | Все возвращаются        |
| `TestFilter_AllOld`      | Все ID <= max_parsed      | Пустой результат        |
| `TestFilter_Mixed`       | Часть новых, часть старых | Только новые            |

```go
func TestMessageIDFilter_FilterNew(t *testing.T) {
    tests := []struct {
        name       string
        maxParsed  int64
        inputIDs   []int64
        expectedIDs []int64
    }{
        {
            name:        "all new when no parsed",
            maxParsed:   0,
            inputIDs:    []int64{100, 101, 102},
            expectedIDs: []int64{100, 101, 102},
        },
        {
            name:        "filters out old messages",
            maxParsed:   100,
            inputIDs:    []int64{99, 100, 101, 102},
            expectedIDs: []int64{101, 102},
        },
        {
            name:        "returns empty when all old",
            maxParsed:   200,
            inputIDs:    []int64{99, 100, 101},
            expectedIDs: []int64{},
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            filter := &MessageIDFilter{
                ranges: []ParsedRange{{MaxMsgID: tt.maxParsed}},
            }
            result := filter.FilterNew(tt.inputIDs)
            assert.Equal(t, tt.expectedIDs, result)
        })
    }
}
```

#### Validation

**Файл**: `internal/collector/validation_test.go`

| Тест                         | Сценарий                | Ожидание             |
| ---------------------------- | ----------------------- | -------------------- |
| `TestValidate_NoSource`      | Ни target_id ни channel | `ErrChannelRequired` |
| `TestValidate_InvalidDate`   | until = "invalid"       | `ErrInvalidDate`     |
| `TestValidate_FutureDate`    | until = "2099-01-01"    | `ErrFutureDate`      |
| `TestValidate_NegativeLimit` | limit = -5              | `ErrInvalidLimit`    |
| `TestValidate_ValidRequest`  | Всё корректно           | `nil`                |

```go
func TestScrapeRequest_Validate(t *testing.T) {
    t.Run("requires channel or target_id", func(t *testing.T) {
        req := &ScrapeRequest{}
        err := req.Validate(context.Background(), nil)
        assert.ErrorIs(t, err, ErrChannelRequired)
    })

    t.Run("validates date format", func(t *testing.T) {
        req := &ScrapeRequest{
            Channel: "@test",
            Until:   "not-a-date",
        }
        err := req.Validate(context.Background(), mockClient)
        assert.ErrorIs(t, err, ErrInvalidDate)
    })

    t.Run("rejects future date", func(t *testing.T) {
        req := &ScrapeRequest{
            Channel: "@test",
            Until:   "2099-12-31",
        }
        err := req.Validate(context.Background(), mockClient)
        assert.ErrorIs(t, err, ErrFutureDate)
    })
}
```

#### Scrape Manager

**Файл**: `internal/collector/manager_test.go`

| Тест                             | Сценарий                   | Ожидание               |
| -------------------------------- | -------------------------- | ---------------------- |
| `TestManager_Start`              | Запуск первой задачи       | Успех, job создан      |
| `TestManager_StartWhenRunning`   | Запуск при активной задаче | `ErrAlreadyRunning`    |
| `TestManager_Stop`               | Остановка активной задачи  | Context отменён        |
| `TestManager_StopWhenNotRunning` | Остановка без задачи       | Ничего не падает       |
| `TestManager_Current`            | Получение текущей задачи   | Возвращает job или nil |
| `TestManager_ConcurrentAccess`   | Параллельный доступ        | Нет race conditions    |

```go
func TestScrapeManager_ConcurrentAccess(t *testing.T) {
    manager := NewScrapeManager(testLogger)

    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            manager.Start(context.Background(), ScrapeOptions{})
            manager.Current()
            manager.Stop()
        }()
    }
    wg.Wait()
    // если дошли сюда без panic — тест пройден
}
```

#### HTTP Handlers

**Файл**: `internal/collector/handler_test.go`

```go
func TestHandler_StartScrape(t *testing.T) {
    t.Run("returns 400 on invalid json", func(t *testing.T) {
        req := httptest.NewRequest("POST", "/api/v1/scrape/telegram", strings.NewReader("invalid"))
        rec := httptest.NewRecorder()
        handler.ServeHTTP(rec, req)
        assert.Equal(t, 400, rec.Code)
    })

    t.Run("returns 400 on validation error", func(t *testing.T) {
        body := `{"limit": -1}`
        req := httptest.NewRequest("POST", "/api/v1/scrape/telegram", strings.NewReader(body))
        rec := httptest.NewRecorder()
        handler.ServeHTTP(rec, req)
        assert.Equal(t, 400, rec.Code)
    })

    t.Run("returns 409 when already running", func(t *testing.T) {
        // start first job
        // try to start second
        assert.Equal(t, 409, rec.Code)
    })

    t.Run("returns 200 on success", func(t *testing.T) {
        body := `{"channel": "@test_channel"}`
        req := httptest.NewRequest("POST", "/api/v1/scrape/telegram", strings.NewReader(body))
        rec := httptest.NewRecorder()
        handler.ServeHTTP(rec, req)
        assert.Equal(t, 200, rec.Code)
    })
}

func TestHandler_StopScrape(t *testing.T) {
    t.Run("returns 200 even when not running", func(t *testing.T) {})
    t.Run("stops running job", func(t *testing.T) {})
}

func TestHandler_Health(t *testing.T) {
    t.Run("returns 200", func(t *testing.T) {
        req := httptest.NewRequest("GET", "/health", nil)
        rec := httptest.NewRecorder()
        handler.ServeHTTP(rec, req)
        assert.Equal(t, 200, rec.Code)
    })
}
```

---

### Manual Tests: Подробные сценарии

#### Сценарий M1: Первый запуск парсинга

**Предусловия**:

- Docker запущен (`docker compose up -d`)
- Миграции применены
- Collector запущен
- Есть валидный TG session string

**Шаги**:

1. Добавить тестовый источник:

   ```bash
   curl -X POST http://localhost:3100/api/v1/targets \
     -H "Content-Type: application/json" \
     -d '{
       "name": "Test Channel",
       "type": "TG_CHANNEL",
       "url": "@test_vacancies_channel"
     }'
   ```

   **Ожидание**: 201 Created, возвращается target с ID

2. Запустить парсинг:

   ```bash
   curl -X POST http://localhost:3100/api/v1/scrape/telegram \
     -H "Content-Type: application/json" \
     -d '{
       "channel": "@test_vacancies_channel",
       "limit": 10
     }'
   ```

   **Ожидание**: 200 OK, status = "running"

3. Проверить статус:

   ```bash
   curl http://localhost:3100/api/v1/scrape/status
   ```

   **Ожидание**: Информация о текущей задаче или "no active scrape"

4. Проверить БД:

   ```bash
   docker exec jhos-postgres psql -U jhos -d jhos -c "SELECT COUNT(*) FROM jobs"
   ```

   **Ожидание**: >= 1 записи

5. Проверить логи:
   ```bash
   tail -f logs/collector.log
   ```
   **Ожидание**: Записи о парсинге без ошибок

---

#### Сценарий M2: Инкрементальный парсинг (дедупликация)

**Предусловия**:

- Сценарий M1 выполнен
- В БД есть записи от первого парсинга

**Шаги**:

1. Запомнить количество записей:

   ```bash
   docker exec jhos-postgres psql -U jhos -d jhos -c "SELECT COUNT(*) FROM jobs"
   ```

   Записать: `count_before = N`

2. Повторить парсинг того же канала:

   ```bash
   curl -X POST http://localhost:3100/api/v1/scrape/telegram \
     -H "Content-Type: application/json" \
     -d '{"channel": "@test_vacancies_channel", "limit": 10}'
   ```

3. Подождать завершения и проверить:

   ```bash
   docker exec jhos-postgres psql -U jhos -d jhos -c "SELECT COUNT(*) FROM jobs"
   ```

   **Ожидание**: `count_after = count_before` (не увеличилось, т.к. все посты уже были)

4. Проверить диапазон:
   ```bash
   docker exec jhos-postgres psql -U jhos -d jhos \
     -c "SELECT min_msg_id, max_msg_id FROM parsed_ranges LIMIT 1"
   ```
   **Ожидание**: Диапазон существует

---

#### Сценарий M3: Остановка парсинга

**Предусловия**:

- Collector запущен

**Шаги**:

1. Запустить большой парсинг (без limit):

   ```bash
   curl -X POST http://localhost:3100/api/v1/scrape/telegram \
     -H "Content-Type: application/json" \
     -d '{"channel": "@big_channel"}'
   ```

2. Через 3 секунды остановить:

   ```bash
   sleep 3
   curl -X DELETE http://localhost:3100/api/v1/scrape/current
   ```

   **Ожидание**: 200 OK

3. Проверить статус:

   ```bash
   curl http://localhost:3100/api/v1/scrape/status
   ```

   **Ожидание**: "no active scrape"

4. Проверить логи:
   ```bash
   grep "stopping" logs/collector.log
   ```
   **Ожидание**: Сообщение об остановке

---

#### Сценарий M4: Парсинг форума с выбранными топиками

**Предусловия**:

- Есть форум с несколькими топиками
- Известны ID топиков (через `tg-topics` утилиту)

**Шаги**:

1. Получить список топиков:

   ```bash
   go run cmd/tg-topics/main.go @forum_channel
   ```

   Записать ID нужных топиков: например `15, 28`

2. Запустить парсинг с фильтром:

   ```bash
   curl -X POST http://localhost:3100/api/v1/scrape/telegram \
     -H "Content-Type: application/json" \
     -d '{
       "channel": "@forum_channel",
       "topic_ids": [15, 28],
       "limit": 20
     }'
   ```

3. Проверить что спарсились только сообщения из нужных топиков:
   ```bash
   docker exec jhos-postgres psql -U jhos -d jhos \
     -c "SELECT DISTINCT tg_topic_id FROM jobs WHERE tg_topic_id IS NOT NULL"
   ```
   **Ожидание**: Только 15 и 28

---

#### Сценарий M5: Обработка ошибок

**Шаги**:

1. Несуществующий канал:

   ```bash
   curl -X POST http://localhost:3100/api/v1/scrape/telegram \
     -H "Content-Type: application/json" \
     -d '{"channel": "@definitely_not_exists_12345"}'
   ```

   **Ожидание**: 400 Bad Request, `"error": "channel not found"`

2. Невалидная дата:

   ```bash
   curl -X POST http://localhost:3100/api/v1/scrape/telegram \
     -H "Content-Type: application/json" \
     -d '{"channel": "@test", "until": "not-a-date"}'
   ```

   **Ожидание**: 400, `"error": "until date must be in YYYY-MM-DD format"`

3. Дата в будущем:

   ```bash
   curl -X POST http://localhost:3100/api/v1/scrape/telegram \
     -H "Content-Type: application/json" \
     -d '{"channel": "@test", "until": "2099-01-01"}'
   ```

   **Ожидание**: 400, `"error": "until date cannot be in the future"`

4. Повторный запуск:
   ```bash
   # первый запуск
   curl -X POST http://localhost:3100/api/v1/scrape/telegram \
     -d '{"channel": "@test"}' &
   # сразу второй
   curl -X POST http://localhost:3100/api/v1/scrape/telegram \
     -d '{"channel": "@test2"}'
   ```
   **Ожидание**: Второй вернёт 409 Conflict

---

### 🔧 Скрипты для интеграционного тестирования

#### Структура скриптов

```
scripts/
├── test-api.sh           # основной тест-раннер
├── test-scrape.sh         # тесты парсинга
├── test-validation.sh     # тесты валидации
├── seed-targets.sh        # заполнение тестовыми данными
└── check-db.sh            # проверка состояния БД
```

#### scripts/test-api.sh

```bash
#!/bin/bash
# Интеграционные тесты API
# Использование: ./scripts/test-api.sh

set -e

BASE_URL="${BASE_URL:-http://localhost:3100}"
LOG_FILE="logs/test-api-$(date +%Y%m%d-%H%M%S).log"

# цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # no color

# функции
log() {
    echo "[$(date +%H:%M:%S)] $1" | tee -a "$LOG_FILE"
}

pass() {
    echo -e "${GREEN}✓ PASS${NC}: $1" | tee -a "$LOG_FILE"
}

fail() {
    echo -e "${RED}✗ FAIL${NC}: $1" | tee -a "$LOG_FILE"
    echo "Response: $2" >> "$LOG_FILE"
}

test_endpoint() {
    local method=$1
    local endpoint=$2
    local data=$3
    local expected_code=$4
    local test_name=$5

    log "Testing: $test_name"

    if [ -n "$data" ]; then
        response=$(curl -s -w "\n%{http_code}" -X "$method" "$BASE_URL$endpoint" \
            -H "Content-Type: application/json" \
            -d "$data")
    else
        response=$(curl -s -w "\n%{http_code}" -X "$method" "$BASE_URL$endpoint")
    fi

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')

    if [ "$http_code" -eq "$expected_code" ]; then
        pass "$test_name (HTTP $http_code)"
        echo "$body" | jq . 2>/dev/null || echo "$body" >> "$LOG_FILE"
    else
        fail "$test_name (expected $expected_code, got $http_code)" "$body"
    fi

    echo "" >> "$LOG_FILE"
}

# === TESTS ===

log "Starting API tests..."
log "Base URL: $BASE_URL"
echo ""

# Health check
test_endpoint "GET" "/health" "" 200 "Health check"

# Validation errors
test_endpoint "POST" "/api/v1/scrape/telegram" '{}' 400 "Empty request should fail"
test_endpoint "POST" "/api/v1/scrape/telegram" '{"limit": -1}' 400 "Negative limit should fail"
test_endpoint "POST" "/api/v1/scrape/telegram" '{"channel": "@test", "until": "bad"}' 400 "Invalid date should fail"

# Targets CRUD
test_endpoint "GET" "/api/v1/targets" "" 200 "List targets"
test_endpoint "POST" "/api/v1/targets" '{"name":"Test","type":"TG_CHANNEL","url":"@test"}' 201 "Create target"

# Status when not running
test_endpoint "GET" "/api/v1/scrape/status" "" 200 "Scrape status"

log ""
log "Tests completed. See $LOG_FILE for details."
```

#### scripts/test-scrape.sh

```bash
#!/bin/bash
# Тестирование парсинга
# Использование: ./scripts/test-scrape.sh @channel_name [limit]

CHANNEL="${1:-@test_channel}"
LIMIT="${2:-5}"
BASE_URL="${BASE_URL:-http://localhost:3100}"

echo "=== Scrape Test ==="
echo "Channel: $CHANNEL"
echo "Limit: $LIMIT"
echo ""

# 1. Запускаем парсинг
echo "1. Starting scrape..."
response=$(curl -s -X POST "$BASE_URL/api/v1/scrape/telegram" \
    -H "Content-Type: application/json" \
    -d "{\"channel\": \"$CHANNEL\", \"limit\": $LIMIT}")

echo "Response: $response" | jq .
echo ""

# 2. Ждём и проверяем статус
echo "2. Waiting for completion..."
for i in {1..30}; do
    status=$(curl -s "$BASE_URL/api/v1/scrape/status")
    is_running=$(echo "$status" | jq -r '.status // "none"')

    if [ "$is_running" != "running" ]; then
        echo "Completed after ${i}s"
        break
    fi
    sleep 1
    echo -n "."
done
echo ""

# 3. Проверяем БД
echo "3. Checking database..."
docker exec jhos-postgres psql -U jhos -d jhos -c "
    SELECT COUNT(*) as total_jobs FROM jobs;
"

# 4. Показываем последние записи
echo "4. Latest jobs:"
docker exec jhos-postgres psql -U jhos -d jhos -c "
    SELECT
        id,
        LEFT(raw_content, 50) as content_preview,
        status,
        created_at
    FROM jobs
    ORDER BY created_at DESC
    LIMIT 5;
"

echo ""
echo "=== Done ==="
```

#### scripts/seed-targets.sh

```bash
#!/bin/bash
# Заполнение тестовыми источниками
# Использование: ./scripts/seed-targets.sh

BASE_URL="${BASE_URL:-http://localhost:3100}"

echo "Seeding test targets..."

# Тестовый канал
curl -s -X POST "$BASE_URL/api/v1/targets" \
    -H "Content-Type: application/json" \
    -d '{
        "name": "Go Jobs",
        "type": "TG_CHANNEL",
        "url": "@golang_jobs"
    }' | jq .

# Тестовый форум
curl -s -X POST "$BASE_URL/api/v1/targets" \
    -H "Content-Type: application/json" \
    -d '{
        "name": "Remote Jobs Forum",
        "type": "TG_FORUM",
        "url": "@remote_jobs_forum",
        "metadata": {
            "topic_ids": [1, 15, 28]
        }
    }' | jq .

echo "Done."
```

#### scripts/check-db.sh

```bash
#!/bin/bash
# Проверка состояния БД
# Использование: ./scripts/check-db.sh

echo "=== Database Status ==="
echo ""

echo "--- Tables ---"
docker exec jhos-postgres psql -U jhos -d jhos -c "\dt"

echo "--- Scraping Targets ---"
docker exec jhos-postgres psql -U jhos -d jhos -c "
    SELECT id, name, type, url, is_active, last_scraped_at
    FROM scraping_targets;
"

echo "--- Jobs by Status ---"
docker exec jhos-postgres psql -U jhos -d jhos -c "
    SELECT status, COUNT(*)
    FROM jobs
    GROUP BY status
    ORDER BY COUNT(*) DESC;
"

echo "--- Parsed Ranges ---"
docker exec jhos-postgres psql -U jhos -d jhos -c "
    SELECT
        st.name as target,
        pr.min_msg_id,
        pr.max_msg_id,
        pr.max_msg_id - pr.min_msg_id as range_size
    FROM parsed_ranges pr
    JOIN scraping_targets st ON pr.target_id = st.id;
"

echo "--- Latest Jobs ---"
docker exec jhos-postgres psql -U jhos -d jhos -c "
    SELECT
        LEFT(raw_content, 40) as content,
        status,
        created_at
    FROM jobs
    ORDER BY created_at DESC
    LIMIT 5;
"
```

#### Makefile additions

```makefile
# добавить в Makefile

.PHONY: test test-unit test-integration

# запуск unit тестов
test-unit:
	go test -v -race ./internal/...

# запуск интеграционных тестов (требует запущенных сервисов)
test-integration:
	./scripts/test-api.sh

# запуск всех тестов
test: test-unit test-integration

# seed тестовых данных
seed:
	./scripts/seed-targets.sh

# проверка состояния БД
check-db:
	./scripts/check-db.sh

# тест парсинга конкретного канала
test-scrape:
	./scripts/test-scrape.sh $(channel) $(limit)
```

---

## Следующий шаг

После завершения Фазы 1 переходим к **Фазе 2: Analyzer** — обработка вакансий через LLM.
