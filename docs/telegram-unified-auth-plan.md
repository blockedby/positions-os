# План внедрения единой системы авторизации Telegram (v4)

## 🎯 Цели

1.  **Никаких секретов в .env**: Полный отказ от `TG_SESSION_STRING`.
2.  **Единственный источник**: Все данные сессии хранятся в Postgres через GORM.
3.  **Web-First Auth**: Авторизация через QR-код прямо в браузере.
4.  **Реактивность**: Статус и QR-код передаются через WebSocket.

---

## 🏗 Список Атомарных Задач (TDD + Pseudo-code)

### 📁 Задача 01: Очистка конфигурации

**Что**: Удалить `TG_SESSION_STRING` из кода.
**Псевдокод**:

```go
// internal/config/config.go
type Config struct {
    TGApiID int
    TGApiHash string
    // TGSessionStr - удаляем
}

// cmd/collector/main.go
if cfg.TGApiID == 0 { log.Fatal("API_ID required") }
// логика проверки TG_SESSION_STRING - удаляем
```

### 📁 Задача 02: Интеграция БД и GORM

**Что**: Совместить `pgxpool` с GORM.
**Псевдокод**:

```go
// internal/database/database.go
type DB struct {
    Pool *pgxpool.Pool
    GORM *gorm.DB
}

func New(url string) (*DB, error) {
    p, _ := pgxpool.New(ctx, url)
    g, _ := gorm.Open(postgres.Open(url))
    return &DB{Pool: p, GORM: g}, nil
}
```

### 📁 Задача 03: TDD — Тест на персистентность

**Что**: Тест, доказывающий, что сессия берется из БД.
**Псевдокод**:

```go
// internal/telegram/persistence_test.go
func TestDBSession(t *testing.T) {
    db := setupTestDB()
    m := NewManager(db)

    // Эмуляция пустой базы
    assert.Equal(t, StatusUnauthorized, m.Init())

    // Эмуляция записи сессии в таблицу 'sessions'
    db.Create(&SessionRecord{Data: "valid_json_bytes"})
    assert.Equal(t, StatusReady, m.Init())
}
```

### 📁 Задача 04: Менеджер и статус

**Что**: Логика «Тихого запуска» (не падать без ТГ).
**Псевдокод**:

```go
// internal/telegram/manager.go
func (m *Manager) Init() {
    client, err := gotgproto.NewClient(..., sessionMaker.SqlSession(m.db))
    if err != nil {
        m.status = StatusUnauthorized
        return
    }
    m.status = StatusReady
}
```

### 📁 Задача 05: Web QR Login (Backend + Frontend)

**Что**: Генерация QR и отправка в сокет.
**Псевдокод**:

```go
// BACKEND: manager.go
func (m *Manager) StartQR(hub *web.Hub) {
    m.rawClient.QR().Auth(ctx, ..., func(t Token) {
        hub.Broadcast(JSON{"type": "tg_qr", "url": t.URL()})
    })
}

// FRONTEND: settings.html
socket.onmessage = (msg) => {
    if (msg.type == 'tg_qr') {
        document.getElementById('qr-container').src = `/api/qr?url=${msg.url}`;
    }
}
```

### 📁 Задача 06: Docker Compose & Environment

**Что**: Очистка переменных и зависимостей.
**Псевдокод (YAML)**:

```yaml
# docker-compose.yml
collector:
  environment:
    - TG_API_ID=${TG_API_ID}
    - TG_API_HASH=${TG_API_HASH}
    # TG_SESSION_STRING - УДАЛИТЬ
  depends_on:
    postgres:
      condition: service_healthy # Обязательно, так как сессия в БД
```

---

## ✅ Критерии приемки (Acceptance Criteria)

1.  **Чистый запуск**: Приложение стартует без переменной сессии.
2.  **База как источник**: Если в `Postgres` есть сессия, бот оживает без лишних действий.
3.  **UI Login**: Вкладка настроек показывает QR-код при нажатии "Connect".
4.  **Re-keying**: При обновлении ключа в БД меняется поле `data` в таблице `sessions`.

---

**Если псевдокод для каждой задачи теперь ок — подтверди, и я начну с Этапа 0.**
