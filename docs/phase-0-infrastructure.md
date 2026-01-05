# Фаза 0: Инфраструктура — План Реализации

## Обзор

Фаза 0 создаёт фундамент для всех последующих сервисов. На выходе получаем:

- Работающий `docker compose up`
- Настроенную базу данных с миграциями
- Общую Go библиотеку для переиспользования кода

---

## 0.1 Docker Compose

**Статус**: ✅ Готово

Файл `docker-compose.yml` уже создан и включает:

- PostgreSQL 16
- NATS с JetStream

---

## 0.2 Database Schema

### Задачи

| #     | Задача                        | Описание                       |
| ----- | ----------------------------- | ------------------------------ |
| 0.2.1 | Выбор инструмента миграций    | golang-migrate или goose       |
| 0.2.2 | Создание структуры директорий | `migrations/`                  |
| 0.2.3 | Миграция: scraping_targets    | Таблица источников             |
| 0.2.4 | Миграция: jobs                | Центральная таблица вакансий   |
| 0.2.5 | Миграция: job_applications    | Результаты tailoring           |
| 0.2.6 | Индексы и constraints         | Уникальные ключи, FK           |
| 0.2.7 | Docker entrypoint             | Автоматический запуск миграций |

### 0.2.1 Выбор инструмента миграций

**Решение**: `golang-migrate/migrate`

Причины:

- Standalone CLI (можно запускать из Docker)
- Поддержка PostgreSQL
- Простой формат: `NNNN_name.up.sql` / `NNNN_name.down.sql`
- Широко используется в Go сообществе

Установка:

```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

### 0.2.2 Структура директорий

```
positions-os/
├── migrations/
│   ├── 0001_create_scraping_targets.up.sql
│   ├── 0001_create_scraping_targets.down.sql
│   ├── 0002_create_jobs.up.sql
│   ├── 0002_create_jobs.down.sql
│   ├── 0003_create_job_applications.up.sql
│   └── 0003_create_job_applications.down.sql
├── docker-compose.yml
└── ...
```

### 0.2.3 Миграция: scraping_targets

```sql
-- 0001_create_scraping_targets.up.sql

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TYPE scraping_target_type AS ENUM (
    'TG_CHANNEL',
    'TG_GROUP',
    'TG_FORUM',
    'HH_SEARCH',
    'LINKEDIN_SEARCH'
);

CREATE TABLE scraping_targets (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name            VARCHAR(255) NOT NULL,
    type            scraping_target_type NOT NULL,
    url             TEXT NOT NULL,

    -- telegram specific
    tg_access_hash  BIGINT,
    tg_channel_id   BIGINT,

    -- parsing config
    metadata        JSONB DEFAULT '{}',

    -- state
    last_scraped_at TIMESTAMPTZ,
    last_message_id BIGINT,
    is_active       BOOLEAN DEFAULT true,

    -- timestamps
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

-- индекс для быстрого поиска активных источников
CREATE INDEX idx_scraping_targets_active ON scraping_targets (is_active) WHERE is_active = true;

-- индекс для поиска по типу
CREATE INDEX idx_scraping_targets_type ON scraping_targets (type);

COMMENT ON TABLE scraping_targets IS 'Источники для парсинга вакансий';
COMMENT ON COLUMN scraping_targets.metadata IS 'JSON с настройками: keywords, limit, include_topics и т.д.';
COMMENT ON COLUMN scraping_targets.last_message_id IS 'ID последнего обработанного сообщения (для инкрементального парсинга)';
```

```sql
-- 0001_create_scraping_targets.down.sql

DROP TABLE IF EXISTS scraping_targets;
DROP TYPE IF EXISTS scraping_target_type;
```

### 0.2.4 Миграция: jobs

```sql
-- 0002_create_jobs.up.sql

CREATE TYPE job_status AS ENUM (
    'RAW',          -- только что спарсено
    'ANALYZED',     -- обработано LLM
    'REJECTED',     -- отклонено пользователем
    'INTERESTED',   -- пользователь заинтересован
    'TAILORED',     -- резюме адаптировано
    'SENT',         -- отклик отправлен
    'RESPONDED'     -- получен ответ
);

CREATE TABLE jobs (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    target_id       UUID NOT NULL REFERENCES scraping_targets(id) ON DELETE CASCADE,

    -- идентификация
    external_id     VARCHAR(255) NOT NULL,  -- id сообщения/вакансии на источнике
    content_hash    VARCHAR(64),             -- sha256 для дедупликации

    -- контент
    raw_content     TEXT NOT NULL,

    -- структурированные данные (заполняется Analyzer)
    structured_data JSONB DEFAULT '{}',
    -- пример структуры:
    -- {
    --   "title": "Go Developer",
    --   "description": "...",
    --   "salary_min": 3000,
    --   "salary_max": 5000,
    --   "currency": "USD",
    --   "location": "Remote",
    --   "is_remote": true,
    --   "language": "EN",
    --   "technologies": ["go", "postgresql", "docker"],
    --   "experience_years": 3,
    --   "company": "TechCorp",
    --   "contacts": ["@recruiter", "hr@company.com"]
    -- }

    -- метаданные источника
    source_url      TEXT,
    source_date     TIMESTAMPTZ,

    -- для telegram
    tg_message_id   BIGINT,
    tg_topic_id     BIGINT,  -- если из forum topic

    -- статус
    status          job_status DEFAULT 'RAW',

    -- timestamps
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW(),
    analyzed_at     TIMESTAMPTZ,

    -- уникальность: один external_id на источник
    CONSTRAINT uq_jobs_target_external UNIQUE (target_id, external_id)
);

-- индекс для фильтрации по статусу
CREATE INDEX idx_jobs_status ON jobs (status);

-- индекс для поиска RAW вакансий (для Analyzer)
CREATE INDEX idx_jobs_raw ON jobs (created_at) WHERE status = 'RAW';

-- индекс для поиска по технологиям (GIN для JSONB)
CREATE INDEX idx_jobs_technologies ON jobs USING GIN ((structured_data -> 'technologies'));

-- индекс для полнотекстового поиска
CREATE INDEX idx_jobs_content_search ON jobs USING GIN (to_tsvector('russian', raw_content));

COMMENT ON TABLE jobs IS 'Центральная таблица вакансий';
COMMENT ON COLUMN jobs.external_id IS 'ID вакансии на источнике (message_id для TG, vacancy_id для HH)';
COMMENT ON COLUMN jobs.content_hash IS 'SHA256 от raw_content для обнаружения дубликатов';
```

```sql
-- 0002_create_jobs.down.sql

DROP TABLE IF EXISTS jobs;
DROP TYPE IF EXISTS job_status;
```

### 0.2.5 Миграция: job_applications

```sql
-- 0003_create_job_applications.up.sql

CREATE TYPE delivery_channel AS ENUM (
    'TG_DM',
    'EMAIL',
    'HH_RESPONSE'
);

CREATE TYPE delivery_status AS ENUM (
    'PENDING',
    'SENT',
    'DELIVERED',
    'READ',
    'FAILED'
);

CREATE TABLE job_applications (
    id                      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_id                  UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,

    -- сгенерированный контент
    tailored_resume_md      TEXT,           -- markdown резюме
    cover_letter_md         TEXT,           -- markdown cover letter

    -- сгенерированные файлы
    resume_pdf_path         VARCHAR(512),   -- путь к PDF в volume
    cover_letter_pdf_path   VARCHAR(512),

    -- отправка
    delivery_channel        delivery_channel,
    delivery_status         delivery_status DEFAULT 'PENDING',
    recipient               VARCHAR(255),   -- @username или email

    -- tracking
    sent_at                 TIMESTAMPTZ,
    delivered_at            TIMESTAMPTZ,
    read_at                 TIMESTAMPTZ,
    response_received_at    TIMESTAMPTZ,

    -- ответ рекрутера (если есть)
    recruiter_response      TEXT,

    -- timestamps
    created_at              TIMESTAMPTZ DEFAULT NOW(),
    updated_at              TIMESTAMPTZ DEFAULT NOW(),

    -- можно создать несколько версий для одной вакансии
    version                 INT DEFAULT 1
);

-- индекс для поиска по вакансии
CREATE INDEX idx_job_applications_job ON job_applications (job_id);

-- индекс для pending отправок
CREATE INDEX idx_job_applications_pending ON job_applications (created_at)
    WHERE delivery_status = 'PENDING';

COMMENT ON TABLE job_applications IS 'Результаты tailoring и отправки откликов';
COMMENT ON COLUMN job_applications.version IS 'Версия отклика (можно создать несколько итераций)';
```

```sql
-- 0003_create_job_applications.down.sql

DROP TABLE IF EXISTS job_applications;
DROP TYPE IF EXISTS delivery_status;
DROP TYPE IF EXISTS delivery_channel;
```

### 0.2.6 Дополнительные индексы и функции

```sql
-- 0004_add_helpers.up.sql

-- функция для автоматического обновления updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- триггеры
CREATE TRIGGER update_scraping_targets_updated_at
    BEFORE UPDATE ON scraping_targets
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_jobs_updated_at
    BEFORE UPDATE ON jobs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_job_applications_updated_at
    BEFORE UPDATE ON job_applications
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

```sql
-- 0004_add_helpers.down.sql

DROP TRIGGER IF EXISTS update_job_applications_updated_at ON job_applications;
DROP TRIGGER IF EXISTS update_jobs_updated_at ON jobs;
DROP TRIGGER IF EXISTS update_scraping_targets_updated_at ON scraping_targets;
DROP FUNCTION IF EXISTS update_updated_at_column();
```

### 0.2.7 Docker entrypoint для миграций

Добавить в `docker-compose.yml`:

```yaml
services:
  migrate:
    image: migrate/migrate:latest
    container_name: jhos-migrate
    volumes:
      - ./migrations:/migrations
    command:
      [
        "-path=/migrations",
        "-database=postgres://${POSTGRES_USER:-jhos}:${POSTGRES_PASSWORD:-jhos_secret}@postgres:5432/${POSTGRES_DB:-jhos}?sslmode=disable",
        "up",
      ]
    depends_on:
      postgres:
        condition: service_healthy
    profiles:
      - migrate # запускать только явно: docker compose --profile migrate up
```

Или через Makefile:

```makefile
.PHONY: migrate-up migrate-down migrate-create

DB_URL=postgres://jhos:jhos_secret@localhost:5432/jhos?sslmode=disable

migrate-up:
	migrate -path migrations -database "$(DB_URL)" up

migrate-down:
	migrate -path migrations -database "$(DB_URL)" down 1

migrate-create:
	migrate create -ext sql -dir migrations -seq $(name)
```

---

## 0.3 Shared Library

### Задачи

| #     | Задача                  | Описание                      |
| ----- | ----------------------- | ----------------------------- |
| 0.3.1 | Инициализация Go модуля | `go.mod`                      |
| 0.3.2 | Config package          | Загрузка конфигурации из env  |
| 0.3.3 | Database package        | Подключение к PostgreSQL      |
| 0.3.4 | NATS package            | Клиент для pub/sub            |
| 0.3.5 | Logger package          | Структурированное логирование |
| 0.3.6 | Models package          | Общие типы данных             |

### 0.3.1 Структура Go модуля

```
positions-os/
├── go.mod
├── go.sum
├── cmd/
│   └── (будущие сервисы)
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── database/
│   │   ├── database.go
│   │   └── queries.go
│   ├── nats/
│   │   └── client.go
│   ├── logger/
│   │   └── logger.go
│   └── models/
│       ├── job.go
│       ├── target.go
│       └── application.go
├── pkg/
│   └── (публичные пакеты, если понадобятся)
├── migrations/
├── docker-compose.yml
└── ...
```

### 0.3.2 Config package

```go
// internal/config/config.go
package config

import (
    "os"
    "strconv"
)

type Config struct {
    // database
    DatabaseURL string

    // nats
    NatsURL string

    // llm
    LLMBaseURL string
    LLMModel   string

    // telegram
    TGApiID       int
    TGApiHash     string
    TGSessionStr  string

    // server
    HTTPPort int

    // logging
    LogLevel string
    LogFile  string
}

func Load() (*Config, error) {
    cfg := &Config{
        DatabaseURL: getEnv("DATABASE_URL", "postgres://jhos:jhos_secret@localhost:5432/jhos?sslmode=disable"),
        NatsURL:     getEnv("NATS_URL", "nats://localhost:4222"),
        LLMBaseURL:  getEnv("LLM_BASE_URL", "http://localhost:1234/v1"),
        LLMModel:    getEnv("LLM_MODEL", "local-model"),
        TGApiHash:   getEnv("TG_API_HASH", ""),
        TGSessionStr: getEnv("TG_SESSION_STRING", ""),
        LogLevel:    getEnv("LOG_LEVEL", "info"),
        LogFile:     getEnv("LOG_FILE", "./logs/app.log"),
        HTTPPort:    getEnvInt("HTTP_PORT", 3100),
    }

    cfg.TGApiID = getEnvInt("TG_API_ID", 0)

    return cfg, nil
}

func getEnv(key, defaultVal string) string {
    if val := os.Getenv(key); val != "" {
        return val
    }
    return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
    if val := os.Getenv(key); val != "" {
        if i, err := strconv.Atoi(val); err == nil {
            return i
        }
    }
    return defaultVal
}
```

### 0.3.3 Database package

```go
// internal/database/database.go
package database

import (
    "context"
    "fmt"

    "github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
    Pool *pgxpool.Pool
}

func New(ctx context.Context, databaseURL string) (*DB, error) {
    pool, err := pgxpool.New(ctx, databaseURL)
    if err != nil {
        return nil, fmt.Errorf("unable to create connection pool: %w", err)
    }

    // проверяем соединение
    if err := pool.Ping(ctx); err != nil {
        return nil, fmt.Errorf("unable to ping database: %w", err)
    }

    return &DB{Pool: pool}, nil
}

func (db *DB) Close() {
    db.Pool.Close()
}
```

### 0.3.4 NATS package

```go
// internal/nats/client.go
package nats

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/nats-io/nats.go"
    "github.com/nats-io/nats.go/jetstream"
)

type Client struct {
    conn *nats.Conn
    js   jetstream.JetStream
}

func New(ctx context.Context, natsURL string) (*Client, error) {
    conn, err := nats.Connect(natsURL)
    if err != nil {
        return nil, fmt.Errorf("connect to nats: %w", err)
    }

    js, err := jetstream.New(conn)
    if err != nil {
        return nil, fmt.Errorf("create jetstream context: %w", err)
    }

    return &Client{conn: conn, js: js}, nil
}

func (c *Client) Publish(ctx context.Context, subject string, data any) error {
    payload, err := json.Marshal(data)
    if err != nil {
        return fmt.Errorf("marshal payload: %w", err)
    }

    _, err = c.js.Publish(ctx, subject, payload)
    return err
}

func (c *Client) Subscribe(ctx context.Context, subject string, handler func([]byte) error) error {
    consumer, err := c.js.CreateOrUpdateConsumer(ctx, "JOBS", jetstream.ConsumerConfig{
        Durable:       subject,
        FilterSubject: subject,
    })
    if err != nil {
        return fmt.Errorf("create consumer: %w", err)
    }

    _, err = consumer.Consume(func(msg jetstream.Msg) {
        if err := handler(msg.Data()); err != nil {
            msg.Nak()
            return
        }
        msg.Ack()
    })

    return err
}

func (c *Client) Close() {
    c.conn.Close()
}
```

### 0.3.5 Logger package

```go
// internal/logger/logger.go
package logger

import (
    "io"
    "os"
    "path/filepath"

    "github.com/rs/zerolog"
)

type Logger struct {
    zerolog.Logger
}

func New(level string, logFile string) (*Logger, error) {
    // парсим уровень
    lvl, err := zerolog.ParseLevel(level)
    if err != nil {
        lvl = zerolog.InfoLevel
    }

    // создаём writers
    writers := []io.Writer{
        zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "15:04:05"},
    }

    // добавляем файловый writer если указан
    if logFile != "" {
        // создаём директорию если не существует
        if err := os.MkdirAll(filepath.Dir(logFile), 0755); err != nil {
            return nil, err
        }

        file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
        if err != nil {
            return nil, err
        }
        writers = append(writers, file)
    }

    multi := zerolog.MultiLevelWriter(writers...)

    logger := zerolog.New(multi).
        Level(lvl).
        With().
        Timestamp().
        Caller().
        Logger()

    return &Logger{logger}, nil
}

// глобальный логгер для удобства
var Global *Logger

func Init(level string, logFile string) error {
    l, err := New(level, logFile)
    if err != nil {
        return err
    }
    Global = l
    return nil
}
```

### 0.3.6 Models package

```go
// internal/models/target.go
package models

import (
    "time"

    "github.com/google/uuid"
)

type ScrapingTargetType string

const (
    TargetTypeTGChannel  ScrapingTargetType = "TG_CHANNEL"
    TargetTypeTGGroup    ScrapingTargetType = "TG_GROUP"
    TargetTypeTGForum    ScrapingTargetType = "TG_FORUM"
    TargetTypeHHSearch   ScrapingTargetType = "HH_SEARCH"
)

type ScrapingTarget struct {
    ID            uuid.UUID          `json:"id" db:"id"`
    Name          string             `json:"name" db:"name"`
    Type          ScrapingTargetType `json:"type" db:"type"`
    URL           string             `json:"url" db:"url"`

    TGAccessHash  *int64             `json:"tg_access_hash,omitempty" db:"tg_access_hash"`
    TGChannelID   *int64             `json:"tg_channel_id,omitempty" db:"tg_channel_id"`

    Metadata      map[string]any     `json:"metadata" db:"metadata"`

    LastScrapedAt *time.Time         `json:"last_scraped_at,omitempty" db:"last_scraped_at"`
    LastMessageID *int64             `json:"last_message_id,omitempty" db:"last_message_id"`
    IsActive      bool               `json:"is_active" db:"is_active"`

    CreatedAt     time.Time          `json:"created_at" db:"created_at"`
    UpdatedAt     time.Time          `json:"updated_at" db:"updated_at"`
}
```

```go
// internal/models/job.go
package models

import (
    "time"

    "github.com/google/uuid"
)

type JobStatus string

const (
    JobStatusRaw        JobStatus = "RAW"
    JobStatusAnalyzed   JobStatus = "ANALYZED"
    JobStatusRejected   JobStatus = "REJECTED"
    JobStatusInterested JobStatus = "INTERESTED"
    JobStatusTailored   JobStatus = "TAILORED"
    JobStatusSent       JobStatus = "SENT"
    JobStatusResponded  JobStatus = "RESPONDED"
)

type Job struct {
    ID             uuid.UUID      `json:"id" db:"id"`
    TargetID       uuid.UUID      `json:"target_id" db:"target_id"`

    ExternalID     string         `json:"external_id" db:"external_id"`
    ContentHash    string         `json:"content_hash" db:"content_hash"`

    RawContent     string         `json:"raw_content" db:"raw_content"`
    StructuredData *JobData       `json:"structured_data" db:"structured_data"`

    SourceURL      *string        `json:"source_url,omitempty" db:"source_url"`
    SourceDate     *time.Time     `json:"source_date,omitempty" db:"source_date"`

    TGMessageID    *int64         `json:"tg_message_id,omitempty" db:"tg_message_id"`
    TGTopicID      *int64         `json:"tg_topic_id,omitempty" db:"tg_topic_id"`

    Status         JobStatus      `json:"status" db:"status"`

    CreatedAt      time.Time      `json:"created_at" db:"created_at"`
    UpdatedAt      time.Time      `json:"updated_at" db:"updated_at"`
    AnalyzedAt     *time.Time     `json:"analyzed_at,omitempty" db:"analyzed_at"`
}

// структура данных после анализа LLM
type JobData struct {
    Title           *string   `json:"title,omitempty"`
    Description     *string   `json:"description,omitempty"`
    SalaryMin       *int      `json:"salary_min,omitempty"`
    SalaryMax       *int      `json:"salary_max,omitempty"`
    Currency        *string   `json:"currency,omitempty"`
    Location        *string   `json:"location,omitempty"`
    IsRemote        bool      `json:"is_remote"`
    Language        string    `json:"language"`
    Technologies    []string  `json:"technologies"`
    ExperienceYears *int      `json:"experience_years,omitempty"`
    Company         *string   `json:"company,omitempty"`
    Contacts        []string  `json:"contacts"`
}
```

```go
// internal/models/application.go
package models

import (
    "time"

    "github.com/google/uuid"
)

type DeliveryChannel string

const (
    DeliveryChannelTGDM  DeliveryChannel = "TG_DM"
    DeliveryChannelEmail DeliveryChannel = "EMAIL"
    DeliveryChannelHH    DeliveryChannel = "HH_RESPONSE"
)

type DeliveryStatus string

const (
    DeliveryStatusPending   DeliveryStatus = "PENDING"
    DeliveryStatusSent      DeliveryStatus = "SENT"
    DeliveryStatusDelivered DeliveryStatus = "DELIVERED"
    DeliveryStatusRead      DeliveryStatus = "READ"
    DeliveryStatusFailed    DeliveryStatus = "FAILED"
)

type JobApplication struct {
    ID                   uuid.UUID        `json:"id" db:"id"`
    JobID                uuid.UUID        `json:"job_id" db:"job_id"`

    TailoredResumeMD     *string          `json:"tailored_resume_md,omitempty" db:"tailored_resume_md"`
    CoverLetterMD        *string          `json:"cover_letter_md,omitempty" db:"cover_letter_md"`

    ResumePDFPath        *string          `json:"resume_pdf_path,omitempty" db:"resume_pdf_path"`
    CoverLetterPDFPath   *string          `json:"cover_letter_pdf_path,omitempty" db:"cover_letter_pdf_path"`

    DeliveryChannel      *DeliveryChannel `json:"delivery_channel,omitempty" db:"delivery_channel"`
    DeliveryStatus       DeliveryStatus   `json:"delivery_status" db:"delivery_status"`
    Recipient            *string          `json:"recipient,omitempty" db:"recipient"`

    SentAt               *time.Time       `json:"sent_at,omitempty" db:"sent_at"`
    DeliveredAt          *time.Time       `json:"delivered_at,omitempty" db:"delivered_at"`
    ReadAt               *time.Time       `json:"read_at,omitempty" db:"read_at"`
    ResponseReceivedAt   *time.Time       `json:"response_received_at,omitempty" db:"response_received_at"`

    RecruiterResponse    *string          `json:"recruiter_response,omitempty" db:"recruiter_response"`

    CreatedAt            time.Time        `json:"created_at" db:"created_at"`
    UpdatedAt            time.Time        `json:"updated_at" db:"updated_at"`
    Version              int              `json:"version" db:"version"`
}
```

---

## Зависимости Go модуля

```
go get github.com/jackc/pgx/v5
go get github.com/nats-io/nats.go
go get github.com/nats-io/nats.go/jetstream
go get github.com/rs/zerolog
go get github.com/google/uuid
```

---

## Чеклист завершения Фазы 0

- [x] 0.2.1 — Установлен golang-migrate (в docker-compose)
- [x] 0.2.2 — Создана папка `migrations/`
- [x] 0.2.3 — Миграция scraping_targets
- [x] 0.2.4 — Миграция jobs
- [x] 0.2.5 — Миграция job_applications
- [x] 0.2.6 — Миграция helpers (triggers)
- [x] 0.2.7 — Makefile с командами migrate
- [x] 0.3.1 — Инициализирован go.mod
- [x] 0.3.2 — Пакет config
- [x] 0.3.3 — Пакет database
- [x] 0.3.4 — Пакет nats
- [x] 0.3.5 — Пакет logger
- [x] 0.3.6 — Пакет models
- [x] Тест: код компилируется (`go build ./...`)
- [x] Тест: `docker compose up` работает ✅
- [x] Тест: миграции применяются ✅
- [x] Тест: таблицы созданы (scraping_targets, jobs, job_applications) ✅

---

## Следующий шаг

После завершения Фазы 0 переходим к **Фазе 1: Collector** — реализация TG парсера.
