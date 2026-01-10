# Telegram QR Authentication Guide

## Руководство по аутентификации через QR-код в Go

Это руководство описывает правильный подход к аутентификации через QR-код с использованием библиотек `gotd/td` и `gotgproto`, а также получение session string для использования в gotgproto.

---

## Оглавление

1. [Обзор библиотек](#обзор-библиотек)
2. [Архитектура сессий](#архитектура-сессий)
3. [QR Login Flow](#qr-login-flow)
4. [Получение Session String для gotgproto](#получение-session-string-для-gotgproto)
5. [Типичные ошибки](#типичные-ошибки)
6. [Рабочий пример](#рабочий-пример)

---

## Обзор библиотек

### gotd/td

Базовая библиотека для работы с Telegram MTProto API:

- `github.com/gotd/td/telegram` — высокоуровневый клиент
- `github.com/gotd/td/telegram/auth/qrlogin` — QR-авторизация
- `github.com/gotd/td/session` — хранение сессий
- `github.com/gotd/td/tg` — типы Telegram API

### gotgproto

Обёртка над gotd с упрощённым API:

- `github.com/celestix/gotgproto` — основной клиент
- `github.com/celestix/gotgproto/storage` — хранение сессий и peers
- `github.com/celestix/gotgproto/functions` — вспомогательные функции
- `github.com/celestix/gotgproto/sessionMaker` — создание сессий

---

## Архитектура сессий

### Формат gotd session.Data

Структура `session.Data` в gotd содержит:

```go
// github.com/gotd/td/session
type Data struct {
    Config    Config     // Конфигурация DC
    DC        int        // Номер Data Center
    Addr      string     // Адрес сервера
    AuthKey   []byte     // 256 байт ключа авторизации
    AuthKeyID []byte     // 8 байт ID ключа
    Salt      int64      // Server salt
}
```

При сохранении через `session.Loader`, данные оборачиваются в:

```go
type jsonData struct {
    Version int  // = 1
    Data    Data
}
```

### Формат gotgproto storage.Session

Структура `storage.Session` в gotgproto:

```go
// github.com/celestix/gotgproto/storage
type Session struct {
    Version int    `gorm:"primary_key"`
    Data    []byte // Сериализованные данные session.Data в формате JSON
}
```

> **ВАЖНО**: Поле `Data` содержит **сырые байты** JSON-сериализованной структуры `session.Data`, **а не** структуру `jsonData` из gotd!

### Session String

Session string в gotgproto — это base64-закодированная JSON-сериализация `storage.Session`:

```
base64(json.Marshal(storage.Session{Version: 1, Data: jsonBytes}))
```

---

## QR Login Flow

### Официальный Flow (по документации Telegram)

1. **Export Login Token**: Клиент вызывает `auth.exportLoginToken`, получает `loginToken` (время жизни ~30 сек)
2. **Display QR**: Token кодируется в URL `tg://login?token=<base64url>` и отображается как QR
3. **Scan & Accept**: Мобильный клиент сканирует QR и вызывает `auth.acceptLoginToken`
4. **Receive Update**: Desktop клиент получает `updateLoginToken`
5. **Confirm**: Повторный вызов `auth.exportLoginToken` возвращает `auth.loginTokenSuccess`

### Использование gotd qrlogin

```go
import (
    "github.com/gotd/td/telegram"
    "github.com/gotd/td/telegram/auth/qrlogin"
    "github.com/gotd/td/tg"
    "github.com/gotd/td/session"
)

func qrAuth() error {
    // Хранилище для захвата сессии
    memStorage := &session.StorageMemory{}

    // Диспетчер для получения updateLoginToken
    dispatcher := tg.NewUpdateDispatcher()

    client := telegram.NewClient(apiID, apiHash, telegram.Options{
        SessionStorage: memStorage,
        UpdateHandler:  dispatcher,
    })

    return client.Run(ctx, func(ctx context.Context) error {
        // Использовать встроенный client.QR() с Migrate
        qr := client.QR()

        // Регистрация обработчика updateLoginToken
        loggedIn := qrlogin.OnLoginToken(dispatcher)

        // Auth блокируется до подтверждения
        _, err := qr.Auth(ctx, loggedIn, func(ctx context.Context, token qrlogin.Token) error {
            // Отображение QR
            fmt.Println("URL:", token.URL())
            return nil
        })

        return err
    })
}
```

### Критически важные моменты

1. **UpdateHandler обязателен**: Без диспетчера `updateLoginToken` не будет получен
2. **client.QR()** vs **qrlogin.NewQR()**:
   - `client.QR()` — автоматически обрабатывает DC migration
   - `qrlogin.NewQR()` — требует ручной обработки `MigrationNeededError`
3. **OnLoginToken**: Должен быть вызван **до** `qr.Auth()`

---

## Получение Session String для gotgproto

### Правильный способ (рекомендуемый)

```go
import (
    "context"
    "encoding/json"

    "github.com/gotd/td/session"
    "github.com/celestix/gotgproto/storage"
    "github.com/celestix/gotgproto/functions"
)

func exportToGotgprotoString(memStorage *session.StorageMemory, ctx context.Context) (string, error) {
    // 1. Загружаем сырые байты сессии gotd
    rawBytes, err := memStorage.LoadSession(ctx)
    if err != nil {
        return "", fmt.Errorf("load session: %w", err)
    }

    // 2. Парсим в session.Data для валидации (опционально)
    loader := session.Loader{Storage: memStorage}
    data, err := loader.Load(ctx)
    if err != nil {
        return "", fmt.Errorf("parse session: %w", err)
    }

    // 3. Сериализуем session.Data в JSON (без обёртки jsonData!)
    sessionDataJSON, err := json.Marshal(data)
    if err != nil {
        return "", fmt.Errorf("marshal session data: %w", err)
    }

    // 4. Создаём gotgproto storage.Session
    gotgSession := storage.Session{
        Version: storage.LatestVersion, // = 1
        Data:    sessionDataJSON,
    }

    // 5. Кодируем в session string
    return functions.EncodeSessionToString(&gotgSession)
}
```

### Альтернативный способ (ручное кодирование)

```go
import (
    "bytes"
    "encoding/base64"
    "encoding/json"
)

func manualEncode(gotgSession *storage.Session) (string, error) {
    var buf bytes.Buffer
    encoder := base64.NewEncoder(base64.StdEncoding, &buf)
    err := json.NewEncoder(encoder).Encode(gotgSession)
    if err != nil {
        return "", err
    }
    _ = encoder.Close()
    return buf.String(), nil
}
```

---

## Типичные ошибки

### ❌ Ошибка 1: Неправильное содержимое Data

```go
// НЕПРАВИЛЬНО: LoadSession возвращает []byte с jsonData, а не session.Data
rawBytes, _ := memStorage.LoadSession(ctx)
gotgSession := storage.Session{
    Version: 1,
    Data:    rawBytes,  // ❌ Это уже сериализованная jsonData!
}
```

**Проблема**: `LoadSession()` возвращает JSON вида:

```json
{"Version":1,"Data":{"Config":{...},"DC":2,"Addr":"...",...}}
```

А gotgproto ожидает **только** содержимое `Data`:

```json
{"Config":{...},"DC":2,"Addr":"...","AuthKey":"..."}
```

### ✅ Правильно:

```go
// Загружаем и парсим через Loader
loader := session.Loader{Storage: memStorage}
data, _ := loader.Load(ctx)

// Сериализуем ТОЛЬКО session.Data
dataBytes, _ := json.Marshal(data)

gotgSession := storage.Session{
    Version: 1,
    Data:    dataBytes, // ✅ Чистый session.Data
}
```

### ❌ Ошибка 2: Отсутствие UpdateHandler

```go
// НЕПРАВИЛЬНО: QR login никогда не завершится
client := telegram.NewClient(apiID, apiHash, telegram.Options{
    SessionStorage: memStorage,
    // ❌ UpdateHandler не указан!
})
```

### ✅ Правильно:

```go
dispatcher := tg.NewUpdateDispatcher()

client := telegram.NewClient(apiID, apiHash, telegram.Options{
    SessionStorage: memStorage,
    UpdateHandler:  dispatcher, // ✅
})

// В Run:
loggedIn := qrlogin.OnLoginToken(dispatcher) // ✅ Регистрируем обработчик
qr.Auth(ctx, loggedIn, showFunc)
```

### ❌ Ошибка 3: Использование client.Run внутри без корректного возврата

```go
// НЕПРАВИЛЬНО: вызов os.Exit внутри client.Run
client.Run(ctx, func(ctx context.Context) error {
    // ...
    os.Exit(0)  // ❌ Соединение не закроется корректно
    return nil
})
```

### ✅ Правильно:

```go
var sessionString string

err := client.Run(ctx, func(ctx context.Context) error {
    // ... авторизация ...
    sessionString = exportedString
    return nil  // ✅ Корректный выход
})

if err != nil {
    log.Fatal(err)
}
fmt.Println(sessionString)
```

---

## Рабочий пример

Полный пример QR-авторизации с экспортом в gotgproto session string:

```go
package main

import (
    "bytes"
    "context"
    "encoding/base64"
    "encoding/json"
    "fmt"
    "os"

    "github.com/celestix/gotgproto/storage"
    "github.com/gotd/td/session"
    "github.com/gotd/td/telegram"
    "github.com/gotd/td/telegram/auth/qrlogin"
    "github.com/gotd/td/tg"
    "github.com/mdp/qrterminal/v3"
)

func main() {
    apiID := 12345
    apiHash := "your_api_hash"

    ctx := context.Background()
    memStorage := &session.StorageMemory{}
    dispatcher := tg.NewUpdateDispatcher()

    client := telegram.NewClient(apiID, apiHash, telegram.Options{
        SessionStorage: memStorage,
        UpdateHandler:  dispatcher,
    })

    var sessionString string

    err := client.Run(ctx, func(ctx context.Context) error {
        qr := client.QR()
        loggedIn := qrlogin.OnLoginToken(dispatcher)

        auth, err := qr.Auth(ctx, loggedIn, func(ctx context.Context, token qrlogin.Token) error {
            fmt.Println("\nScan this QR code with Telegram app:")
            qrterminal.GenerateHalfBlock(token.URL(), qrterminal.L, os.Stdout)
            fmt.Printf("URL: %s\n", token.URL())
            return nil
        })

        if err != nil {
            return fmt.Errorf("qr auth failed: %w", err)
        }

        // Получаем пользователя
        user, err := client.Self(ctx)
        if err != nil {
            return fmt.Errorf("get self: %w", err)
        }
        fmt.Printf("Logged in as: @%s\n", user.Username)

        // Экспортируем сессию для gotgproto
        sessionString, err = exportSession(memStorage, ctx)
        if err != nil {
            return fmt.Errorf("export session: %w", err)
        }

        _ = auth
        return nil
    })

    if err != nil {
        fmt.Printf("Error: %v\n", err)
        os.Exit(1)
    }

    fmt.Println("\n=== Session String ===")
    fmt.Println(sessionString)
    fmt.Println("======================")
    fmt.Println("Add to .env as TG_SESSION_STRING")
}

func exportSession(memStorage *session.StorageMemory, ctx context.Context) (string, error) {
    // Загружаем через Loader для правильного парсинга
    loader := session.Loader{Storage: memStorage}
    data, err := loader.Load(ctx)
    if err != nil {
        return "", fmt.Errorf("load session data: %w", err)
    }

    // Сериализуем session.Data в JSON
    dataJSON, err := json.Marshal(data)
    if err != nil {
        return "", fmt.Errorf("marshal session data: %w", err)
    }

    // Создаём gotgproto-совместимую структуру
    gotgSession := storage.Session{
        Version: storage.LatestVersion,
        Data:    dataJSON,
    }

    // Кодируем в base64
    var buf bytes.Buffer
    encoder := base64.NewEncoder(base64.StdEncoding, &buf)
    if err := json.NewEncoder(encoder).Encode(&gotgSession); err != nil {
        return "", fmt.Errorf("encode session: %w", err)
    }
    _ = encoder.Close()

    return buf.String(), nil
}
```

---

## Ссылки

- [gotd/td Documentation](https://gotd.dev/)
- [gotd/td session package](https://pkg.go.dev/github.com/gotd/td/session)
- [gotd/td qrlogin package](https://pkg.go.dev/github.com/gotd/td/telegram/auth/qrlogin)
- [gotgproto Documentation](https://pkg.go.dev/github.com/celestix/gotgproto)
- [Telegram QR Login API](https://core.telegram.org/api/qr-login)
- [gotgproto functions.EncodeSessionToString](https://pkg.go.dev/github.com/celestix/gotgproto/functions#EncodeSessionToString)
