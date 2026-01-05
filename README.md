# Positions OS

Система управления поиском работы: парсинг вакансий, AI-анализ, автоматизация откликов.

## Быстрый старт

### Требования

- Docker & Docker Compose
- Go 1.21+
- Make (опционально)

### Запуск

```bash
# 1. Скопируй .env файл
cp .env.example .env

# 2. Запусти инфраструктуру
docker compose up -d postgres nats

# 3. Примени миграции
docker compose --profile tools run --rm migrate

# 4. Проверь что всё работает
docker compose ps
```

### Остановка

```bash
docker compose down
```

## Структура проекта

```
positions-os/
├── cmd/                    # точки входа сервисов
├── internal/               # внутренние пакеты
│   ├── config/             # конфигурация
│   ├── database/           # работа с postgresql
│   ├── logger/             # логирование
│   ├── models/             # типы данных
│   └── nats/               # pub/sub клиент
├── migrations/             # sql миграции
├── docs/                   # документация
└── docker-compose.yml      # инфраструктура
```

## Makefile команды

```bash
make deps           # установить зависимости
make migrate-up     # применить миграции
make migrate-down   # откатить последнюю миграцию
make docker-up      # запустить docker
make docker-down    # остановить docker
make build          # собрать бинарники
make test           # запустить тесты
make lint           # проверить код
```

## Сервисы

| Сервис       | Порт | Описание             |
| ------------ | ---- | -------------------- |
| PostgreSQL   | 5432 | База данных          |
| NATS         | 4222 | Message broker       |
| NATS Monitor | 8222 | NATS мониторинг      |
| Web UI       | 3100 | Веб-интерфейс (TODO) |

## Документация

- [План реализации](docs/implementation-order.md)
- [Telegram интеграция](docs/telegram-integration.md)
- [Фаза 0: Инфраструктура](docs/phase-0-infrastructure.md)

## Переменные окружения

```env
# database
DATABASE_URL=postgres://jhos:jhos_secret@localhost:5432/jhos?sslmode=disable

# nats
NATS_URL=nats://localhost:4222

# llm (lm studio compatible)
LLM_BASE_URL=http://localhost:1234/v1
LLM_MODEL=local-model

# telegram
TG_API_ID=
TG_API_HASH=
TG_SESSION_STRING=

# server
HTTP_PORT=3100
LOG_LEVEL=info
LOG_FILE=./logs/app.log
```

## Лицензия

Private
