# Code Review: cmd/tg-auth/main.go

## 🔴 Критическая ошибка в функции `authWithQR`

### Проблема: Неправильный формат session.Data

**Строки 305-316:**

```go
// Extract session data from memory storage
data, err := memStorage.LoadSession(ctx)  // ← возвращает []byte
if err != nil {
    return fmt.Errorf("failed to load session from memory: %w", err)
}

// Create gotgproto storage.Session format
// storage.Session expects JSON-encoded session.Data in Data field
sessionDataJSON, err := json.Marshal(data)  // ← ОШИБКА: двойная сериализация!
```

#### Что происходит:

1. `memStorage.LoadSession(ctx)` возвращает `[]byte` — это **уже JSON-сериализованные** данные в формате:

   ```json
   {"Version":1,"Data":{"Config":{...},"DC":2,"Addr":"...","AuthKey":"...","AuthKeyID":"...","Salt":123}}
   ```

2. Затем код делает `json.Marshal(data)` — это **двойная сериализация**, результат:

   ```json
   "eyJWZXJzaW9uIjoxLCJEYXRhIjp7Li4ufX0=" // Строка в кавычках, не объект!
   ```

3. Эти байты кладутся в `storage.Session.Data`, что даёт **некорректный формат**.

#### Правильное решение:

```go
// Вариант 1: Использовать session.Loader для парсинга
loader := session.Loader{Storage: memStorage}
sessionData, err := loader.Load(ctx)
if err != nil {
    return fmt.Errorf("failed to load session: %w", err)
}

// Сериализуем ТОЛЬКО session.Data (без обёртки jsonData)
sessionDataJSON, err := json.Marshal(sessionData)
if err != nil {
    return fmt.Errorf("failed to marshal session data: %w", err)
}

gotgSession := storage.Session{
    Version: storage.LatestVersion,
    Data:    sessionDataJSON,
}
```

---

## 🟡 Предупреждение: Использование `os.Exit(0)` внутри `client.Run`

**Строка 348:**

```go
printSuccess(username, sessionString)

// Exit successfully
os.Exit(0)  // ← Проблема: преждевременный выход
return nil
```

#### Проблема:

Вызов `os.Exit(0)` внутри callback'а `client.Run`:

- Не позволяет корректно закрыть соединение с Telegram
- Пропускает очистку ресурсов
- Может оставить сессию в некорректном состоянии на сервере

#### Рекомендация:

```go
// Сохраняем результат в переменные внешнего scope
var sessionString string
var username string

err := client.Run(ctx, func(ctx context.Context) error {
    // ... авторизация ...

    sessionString = exportedString
    username = user.Username

    return nil  // Нормальный выход
})

if err != nil {
    fmt.Printf("error: %v\n", err)
    os.Exit(1)
}

printSuccess(username, sessionString)
```

---

## 🟡 Предупреждение: Возможный race condition в переменной `client`

**Строки 255-262:**

```go
// Create a reference to the client for the Migrate function
var client *telegram.Client

// Initialize gotd client directly with the dispatcher
client = telegram.NewClient(apiID, apiHash, telegram.Options{
    SessionStorage: memStorage,
    UpdateHandler:  dispatcher,
})
```

#### Замечание:

Комментарий "for the Migrate function" устарел — в текущем коде `client.QR()` используется вместо `qrlogin.NewQR()`, и Migrate обрабатывается автоматически. Можно упростить:

```go
client := telegram.NewClient(apiID, apiHash, telegram.Options{
    SessionStorage: memStorage,
    UpdateHandler:  dispatcher,
})
```

---

## 🟢 Возможные улучшения

### 1. Добавить timeout для QR-сканирования

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()
```

### 2. Использовать `context.WithCancel` для graceful shutdown

```go
ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
defer cancel()
```

### 3. Обработка истечения QR токена

QR токен живёт ~30 секунд. Текущий код не информирует пользователя об истечении. Можно добавить:

```go
auth, err := qr.Auth(ctx, loggedIn, func(ctx context.Context, token qrlogin.Token) error {
    fmt.Printf("QR expires in: %v\n", time.Until(token.Expires()))
    // ... display QR ...
    return nil
})
```

### 4. Улучшить отображение QR-кода

Текущие настройки `qrterminal.Config` могут плохо отображаться в некоторых терминалах:

```go
// Более универсальный вариант:
qrterminal.GenerateHalfBlock(token.URL(), qrterminal.L, os.Stdout)
```

---

## 📋 Сводка изменений

| Приоритет      | Строки  | Описание                                    |
| -------------- | ------- | ------------------------------------------- |
| 🔴 Критическая | 305-316 | Двойная JSON-сериализация session data      |
| 🟡 Важная      | 348     | Некорректный выход через os.Exit внутри Run |
| 🟡 Косметика   | 255-256 | Устаревший комментарий                      |
| 🟢 Улучшение   | 264     | Добавить timeout/cancel context             |
| 🟢 Улучшение   | 276     | Показывать время истечения QR               |

---

## Исправленная версия функции `authWithQR`

```go
func authWithQR(apiID int, apiHash string) {
    fmt.Println("\ninitializing qr login... please wait")

    memStorage := &session.StorageMemory{}
    dispatcher := tg.NewUpdateDispatcher()

    client := telegram.NewClient(apiID, apiHash, telegram.Options{
        SessionStorage: memStorage,
        UpdateHandler:  dispatcher,
    })

    // Добавляем timeout и обработку Ctrl+C
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()

    var sessionString string
    var username string

    err := client.Run(ctx, func(ctx context.Context) error {
        qr := client.QR()
        loggedIn := qrlogin.OnLoginToken(dispatcher)

        auth, err := qr.Auth(ctx, loggedIn, func(ctx context.Context, token qrlogin.Token) error {
            fmt.Println("\n➡️  SCAN THIS QR CODE WITH YOUR TELEGRAM APP:")
            fmt.Println("    Settings -> Devices -> Link Desktop Device")
            fmt.Printf("    (expires in %v)\n\n", time.Until(token.Expires()).Round(time.Second))

            qrterminal.GenerateHalfBlock(token.URL(), qrterminal.L, os.Stdout)
            fmt.Printf("\nRaw Token URL: %s\n", token.URL())
            fmt.Println("\nwaiting for scan...")
            return nil
        })

        if err != nil {
            return fmt.Errorf("qr auth failed: %w", err)
        }
        _ = auth

        user, err := client.Self(ctx)
        if err != nil {
            return fmt.Errorf("failed to get self: %w", err)
        }

        // ИСПРАВЛЕНО: Используем Loader для правильного парсинга
        loader := session.Loader{Storage: memStorage}
        sessionData, err := loader.Load(ctx)
        if err != nil {
            return fmt.Errorf("failed to load session: %w", err)
        }

        // Сериализуем только session.Data
        sessionDataJSON, err := json.Marshal(sessionData)
        if err != nil {
            return fmt.Errorf("failed to marshal session data: %w", err)
        }

        gotgSession := storage.Session{
            Version: storage.LatestVersion,
            Data:    sessionDataJSON,
        }

        // Кодируем в base64
        var buf bytes.Buffer
        enc := base64.NewEncoder(base64.StdEncoding, &buf)
        if err := json.NewEncoder(enc).Encode(&gotgSession); err != nil {
            return fmt.Errorf("failed to encode session: %w", err)
        }
        _ = enc.Close()

        sessionString = buf.String()
        if user.Username != "" {
            username = user.Username
        } else {
            username = fmt.Sprintf("%d (%s)", user.ID, user.FirstName)
        }

        return nil // Корректный выход
    })

    if err != nil {
        fmt.Printf("error during qr login: %v\n", err)
        os.Exit(1)
    }

    printSuccess(username, sessionString)
}
```

---

## Дополнительно: требуется импорт

```go
import (
    "time"
    // ... остальные импорты
)
```
