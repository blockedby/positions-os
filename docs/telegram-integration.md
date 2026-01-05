# Интеграция с Telegram: Техническая Спецификация

## Оглавление

1. [Обзор](#1-обзор)
2. [Архитектура](#2-архитектура)
3. [Получение API Credentials](#3-получение-api-credentials)
4. [Библиотеки и Инструменты](#4-библиотеки-и-инструменты)
5. [Session Management](#5-session-management)
6. [Rate Limits и Best Practices](#6-rate-limits-и-best-practices)
7. [Сценарии Использования](#7-сценарии-использования)
8. [Структуры Данных](#8-структуры-данных)
9. [Flow Диаграммы](#9-flow-диаграммы)
10. [Безопасность](#10-безопасность)

---

## 1. Обзор

Telegram интеграция в Job-Hunter OS выполняет две ключевые функции:

1. **Сбор вакансий (Collector)** — парсинг каналов и групп с вакансиями
2. **Отправка откликов (Dispatcher)** — автоматическая отправка резюме рекрутерам

Мы используем **MTProto API** (не Bot API), что даёт полный доступ к функционалу Telegram как обычного пользователя.

### Почему MTProto, а не Bot API?

| Критерий          | Bot API                       | MTProto (Userbot)      |
| ----------------- | ----------------------------- | ---------------------- |
| Чтение каналов    | Только если бот — админ       | Любые публичные каналы |
| Чтение групп      | Только если бот добавлен      | Любые публичные группы |
| Отправка ЛС       | Только если юзер начал диалог | Любому пользователю\*  |
| История сообщений | Ограничена                    | Полная                 |
| Rate Limits       | Строгие                       | Более гибкие           |

> \*С ограничениями — нельзя спамить, Telegram заблокирует аккаунт.

---

## 2. Архитектура

```
┌─────────────────────────────────────────────────────────────┐
│                      Job-Hunter OS                          │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────┐     NATS      ┌─────────────────────────┐ │
│  │  Web UI     │◄────────────► │   Collector Service     │ │
│  └─────────────┘               │                         │ │
│                                │  ┌───────────────────┐  │ │
│                                │  │ TG Strategy       │  │ │
│                                │  │ ┌───────────────┐ │  │ │
│                                │  │ │ gotgproto     │ │──┼─┼──► Telegram MTProto
│                                │  │ │ (MTProto)     │ │  │ │
│                                │  │ └───────────────┘ │  │ │
│                                │  └───────────────────┘  │ │
│                                └─────────────────────────┘ │
│                                                             │
│  ┌─────────────────────────────────────────────────────────┐│
│  │                   Dispatcher Service                    ││
│  │  ┌─────────────────────────────────────────────────┐   ││
│  │  │  gotgproto (MTProto) - отправка сообщений       │───┼┼──► Telegram MTProto
│  │  └─────────────────────────────────────────────────┘   ││
│  └─────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────┘
```

---

## 3. Получение API Credentials

### Шаг 1: Регистрация приложения

1. Перейдите на [my.telegram.org](https://my.telegram.org)
2. Войдите по номеру телефона
3. Выберите **"API development tools"**
4. Заполните форму:
   - **App title**: `Job Hunter OS` (или любое название)
   - **Short name**: `jhos` (только буквы, 5-32 символа)
   - **Platform**: `Desktop`
   - **Description**: `Personal job hunting automation`

### Шаг 2: Сохранение credentials

После создания вы получите:

- `api_id` — числовой идентификатор (например: `12345678`)
- `api_hash` — hex-строка (например: `a1b2c3d4e5f6...`)

**⚠️ ВАЖНО**: Никогда не публикуйте эти данные! Храните в `.env` файле.

```env
TG_API_ID=12345678
TG_API_HASH=a1b2c3d4e5f6g7h8i9j0
```

---

## 4. Библиотеки и Инструменты

### Рекомендуемый стек для Go

| Библиотека           | Назначение                       | Ссылка                                                                 |
| -------------------- | -------------------------------- | ---------------------------------------------------------------------- |
| `gotd/td`            | Низкоуровневый MTProto клиент    | [github.com/gotd/td](https://github.com/gotd/td)                       |
| `celestix/gotgproto` | Высокоуровневая обёртка над gotd | [github.com/celestix/gotgproto](https://github.com/celestix/gotgproto) |

### Почему gotgproto?

- Упрощает работу с MTProto
- Встроенное управление сессиями (session string)
- Автоматическая обработка FloodWait
- Peer storage (кэширование ID пользователей/каналов)
- Активное развитие (2024+)

### Установка

```bash
go get github.com/celestix/gotgproto@latest
go get github.com/gotd/td@latest
```

---

## 5. Session Management

### Что такое Session String?

Session String — это зашифрованное представление вашей авторизованной сессии. Позволяет переиспользовать логин без повторной аутентификации.

### Генерация Session String

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/celestix/gotgproto"
    "github.com/celestix/gotgproto/sessionMaker"
    "github.com/gotd/td/telegram"
)

func main() {
    // получаем credentials из окружения
    apiID := os.Getenv("TG_API_ID")
    apiHash := os.Getenv("TG_API_HASH")

    client, err := gotgproto.NewClient(
        apiID,
        apiHash,
        gotgproto.ClientTypePhone("YOUR_PHONE_NUMBER"),
        &gotgproto.ClientOpts{
            Session: sessionMaker.NewSession(
                "jhos_session",
                sessionMaker.Session, // creates .session file
            ),
        },
    )
    if err != nil {
        panic(err)
    }
    defer client.Stop()

    // после успешной авторизации извлекаем session string
    sessionString, err := client.ExportStringSession()
    if err != nil {
        panic(err)
    }

    fmt.Println("session string (save to .env):")
    fmt.Println(sessionString)
}
```

### Использование Session String

```go
client, err := gotgproto.NewClient(
    apiID,
    apiHash,
    gotgproto.ClientTypePhone(""),
    &gotgproto.ClientOpts{
        Session: sessionMaker.NewSession(
            os.Getenv("TG_SESSION_STRING"),
            sessionMaker.StringSession,
        ),
    },
)
```

### Правила безопасности

1. **Никогда не коммитьте session string** — только в `.env`
2. **Один аккаунт = один session** — не запускайте параллельно с разных устройств
3. **Периодическая ротация** — если подозреваете утечку, терминируйте сессию в настройках Telegram

---

## 6. Rate Limits и Best Practices

### Официальные лимиты Telegram

| Действие                | Лимит    | Период    |
| ----------------------- | -------- | --------- |
| Сообщения (разные чаты) | ~30      | в секунду |
| Сообщения (один чат)    | ~1       | в секунду |
| Сообщения (группа)      | ~20      | в минуту  |
| GetMessages (история)   | ~300-500 | в запрос  |
| ResolveUsername         | ~50      | в минуту  |

### FloodWait Error

Telegram возвращает `FLOOD_WAIT_X` где X — секунды ожидания.

```go
import "github.com/gotd/td/tgerr"

// проверка на FloodWait
if flood, ok := tgerr.AsFloodWait(err); ok {
    log.Printf("flood wait: sleeping for %d seconds", flood.Argument)
    time.Sleep(time.Duration(flood.Argument) * time.Second)
    // retry the request
}
```

### Best Practices

1. **Добавляйте задержки между запросами**

   ```go
   // между сообщениями
   time.Sleep(1 * time.Second)

   // между итерациями по каналам
   time.Sleep(2 * time.Second)
   ```

2. **Кэшируйте Peer ID**

   ```go
   // плохо: каждый раз резолвим username
   peer, _ := client.API().ContactsResolveUsername(ctx, "channel_name")

   // хорошо: резолвим один раз, сохраняем в БД
   // gotgproto делает это автоматически через PeerStorage
   ```

3. **Используйте exponential backoff**

   ```go
   for attempt := 0; attempt < maxRetries; attempt++ {
       err := doRequest()
       if err == nil {
           break
       }

       backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
       time.Sleep(backoff)
   }
   ```

4. **Лимитируйте параллельные запросы**

   ```go
   sem := make(chan struct{}, 3) // max 3 concurrent

   for _, channel := range channels {
       sem <- struct{}{}
       go func(ch Channel) {
           defer func() { <-sem }()
           scrapeChannel(ch)
       }(channel)
   }
   ```

5. **Не парсите слишком часто**
   - Один канал: не чаще 1 раза в 5 минут
   - Весь цикл: не чаще 1 раза в час

---

## 7. Сценарии Использования

### 7.1. Парсинг публичного канала

```go
func scrapeChannel(ctx context.Context, client *gotgproto.Client, username string) ([]Message, error) {
    // резолвим канал по username
    peer := client.PeerStorage.GetPeerByUsername(username)
    if peer == nil {
        resolved, err := client.API().ContactsResolveUsername(ctx, username)
        if err != nil {
            return nil, fmt.Errorf("resolve username: %w", err)
        }
        peer = resolved.Peer
    }

    // получаем последние сообщения
    inputPeer := &tg.InputPeerChannel{
        ChannelID:  peer.ChannelID,
        AccessHash: peer.AccessHash,
    }

    history, err := client.API().MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
        Peer:  inputPeer,
        Limit: 100, // последние 100 сообщений
    })
    if err != nil {
        return nil, fmt.Errorf("get history: %w", err)
    }

    // парсим сообщения
    var messages []Message
    switch h := history.(type) {
    case *tg.MessagesChannelMessages:
        for _, msg := range h.Messages {
            if m, ok := msg.(*tg.Message); ok {
                messages = append(messages, Message{
                    ID:   m.ID,
                    Text: m.Message,
                    Date: time.Unix(int64(m.Date), 0),
                })
            }
        }
    }

    return messages, nil
}
```

### 7.2. Парсинг публичной группы

```go
func scrapeGroup(ctx context.Context, client *gotgproto.Client, username string) ([]Message, error) {
    // для групп используем InputPeerChat или InputPeerChannel (supergroup)
    peer := client.PeerStorage.GetPeerByUsername(username)

    // supergroup — технически канал
    inputPeer := &tg.InputPeerChannel{
        ChannelID:  peer.ChannelID,
        AccessHash: peer.AccessHash,
    }

    // остальной код аналогичен каналам
    // ...
}
```

### 7.3. Мониторинг новых сообщений (Real-time)

```go
func startMonitoring(client *gotgproto.Client) {
    dispatcher := client.Dispatcher

    // регистрируем обработчик новых сообщений
    dispatcher.AddHandler(handlers.NewMessage(
        filters.Message.Channel, // только сообщения из каналов
        func(ctx *ext.Context, update *ext.Update) error {
            msg := update.EffectiveMessage

            log.Printf("new message in %s: %s",
                update.EffectiveChat().Username,
                msg.Text,
            )

            // отправляем в NATS для обработки
            natsClient.Publish("jobs.new", JobMessage{
                ChannelID:  update.EffectiveChat().ID,
                MessageID:  msg.ID,
                Text:       msg.Text,
                Date:       msg.Date,
            })

            return nil
        },
    ))
}
```

### 7.4. Отправка сообщения рекрутеру

```go
func sendToRecruiter(ctx context.Context, client *gotgproto.Client, username string, text string, pdfPath string) error {
    // резолвим пользователя
    peer := client.PeerStorage.GetPeerByUsername(username)
    if peer == nil {
        resolved, err := client.API().ContactsResolveUsername(ctx, username)
        if err != nil {
            return fmt.Errorf("resolve recruiter: %w", err)
        }
        peer = resolved.Peer
    }

    inputPeer := &tg.InputPeerUser{
        UserID:     peer.UserID,
        AccessHash: peer.AccessHash,
    }

    // загружаем PDF как документ
    file, err := os.Open(pdfPath)
    if err != nil {
        return fmt.Errorf("open pdf: %w", err)
    }
    defer file.Close()

    // используем uploader для загрузки файла
    uploader := uploader.NewUploader(client.API())
    uploaded, err := uploader.FromFile(ctx, file)
    if err != nil {
        return fmt.Errorf("upload pdf: %w", err)
    }

    // отправляем сообщение с документом
    _, err = client.API().MessagesSendMedia(ctx, &tg.MessagesSendMediaRequest{
        Peer: inputPeer,
        Media: &tg.InputMediaUploadedDocument{
            File:     uploaded,
            MimeType: "application/pdf",
            Attributes: []tg.DocumentAttributeClass{
                &tg.DocumentAttributeFilename{
                    FileName: "Resume.pdf",
                },
            },
        },
        Message: text, // сопроводительное письмо
    })

    return err
}
```

### 7.5. Парсинг Discussion Group (комментарии к постам)

Telegram каналы могут иметь привязанную группу для обсуждений (Linked Chat). Комментарии к постам канала — это сообщения в этой группе.

**Как это работает:**

1. Канал имеет `linked_chat_id` — ID группы обсуждений
2. Когда пользователь комментирует пост — сообщение идёт в Linked Chat
3. Комментарий содержит `reply_to_msg_id` — ссылку на ID оригинального поста

```go
// получение linked chat id
func getDiscussionGroupID(ctx context.Context, client *gotgproto.Client, channelUsername string) (int64, error) {
    // резолвим канал
    resolved, err := client.API().ContactsResolveUsername(ctx, channelUsername)
    if err != nil {
        return 0, fmt.Errorf("resolve channel: %w", err)
    }

    // получаем полную информацию о канале
    inputChannel := &tg.InputChannel{
        ChannelID:  resolved.Chats[0].(*tg.Channel).ID,
        AccessHash: resolved.Chats[0].(*tg.Channel).AccessHash,
    }

    fullChannel, err := client.API().ChannelsGetFullChannel(ctx, inputChannel)
    if err != nil {
        return 0, fmt.Errorf("get full channel: %w", err)
    }

    // извлекаем linked chat id
    channelFull := fullChannel.FullChat.(*tg.ChannelFull)
    if channelFull.LinkedChatID == 0 {
        return 0, fmt.Errorf("channel has no discussion group")
    }

    return channelFull.LinkedChatID, nil
}

// парсинг комментариев к конкретному посту
func scrapePostComments(ctx context.Context, client *gotgproto.Client, channelID int64, postMsgID int) ([]Message, error) {
    // комментарии хранятся в discussion group и ссылаются на пост через reply_to_msg_id
    // используем messages.getReplies для получения комментариев к посту

    inputPeer := &tg.InputPeerChannel{
        ChannelID:  channelID,
        // access_hash нужно получить из PeerStorage или резолвить
    }

    replies, err := client.API().MessagesGetReplies(ctx, &tg.MessagesGetRepliesRequest{
        Peer:  inputPeer,
        MsgID: postMsgID,
        Limit: 100,
    })
    if err != nil {
        return nil, fmt.Errorf("get replies: %w", err)
    }

    var comments []Message
    switch r := replies.(type) {
    case *tg.MessagesChannelMessages:
        for _, msg := range r.Messages {
            if m, ok := msg.(*tg.Message); ok {
                comments = append(comments, Message{
                    ID:           m.ID,
                    Text:         m.Message,
                    Date:         time.Unix(int64(m.Date), 0),
                    ReplyToMsgID: &postMsgID,
                    IsComment:    true,
                })
            }
        }
    }

    return comments, nil
}

// парсинг всей discussion group
func scrapeDiscussionGroup(ctx context.Context, client *gotgproto.Client, discussionChatID int64, limit int) ([]Message, error) {
    // для linked chat используем тот же подход что и для канала
    // но нужно отдельно получить access_hash

    // если группа публичная — можно резолвить по username
    // если приватная — access_hash должен быть в PeerStorage после getFullChannel

    inputPeer := &tg.InputPeerChannel{
        ChannelID: discussionChatID,
        // access_hash из fullChannel.Chats
    }

    history, err := client.API().MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
        Peer:  inputPeer,
        Limit: limit,
    })
    if err != nil {
        return nil, fmt.Errorf("get discussion history: %w", err)
    }

    var messages []Message
    switch h := history.(type) {
    case *tg.MessagesChannelMessages:
        for _, msg := range h.Messages {
            if m, ok := msg.(*tg.Message); ok {
                var replyTo *int
                if m.ReplyTo != nil {
                    if reply, ok := m.ReplyTo.(*tg.MessageReplyHeader); ok {
                        replyTo = &reply.ReplyToMsgID
                    }
                }

                messages = append(messages, Message{
                    ID:           m.ID,
                    Text:         m.Message,
                    Date:         time.Unix(int64(m.Date), 0),
                    ReplyToMsgID: replyTo,
                    IsComment:    replyTo != nil, // если есть reply — это комментарий
                })
            }
        }
    }

    return messages, nil
}
```

**Важные моменты:**

1. **Access Hash**: Для linked chat нужен отдельный access_hash. После вызова `getFullChannel` он обычно появляется в `fullChannel.Chats`
2. **messages.getReplies**: Самый удобный метод для получения комментариев к конкретному посту
3. **Пагинация**: Для больших discussion групп используйте `offset_id` для пагинации
4. **Права доступа**: Discussion group может быть приватной — тогда нужно быть участником

---

## 8. Структуры Данных

### Telegram Message (упрощённо)

```go
type TelegramMessage struct {
    ID            int       `json:"id"`
    ChannelID     int64     `json:"channel_id"`
    ChannelName   string    `json:"channel_name"`
    Text          string    `json:"text"`
    Date          time.Time `json:"date"`
    Views         int       `json:"views"`
    Forwards      int       `json:"forwards"`
    ReplyToMsgID  *int      `json:"reply_to_msg_id,omitempty"`
    MediaType     string    `json:"media_type,omitempty"` // photo, document, etc.
    HasContact    bool      `json:"has_contact"`          // есть ли контакт рекрутера
}
```

### Scraping Target для TG

```go
type TelegramTarget struct {
    ID          uuid.UUID `json:"id"`
    Name        string    `json:"name"`        // "Go Jobs TG"
    Type        string    `json:"type"`        // TG_CHANNEL, TG_GROUP
    Username    string    `json:"username"`    // @go_jobs или invite link
    AccessHash  int64     `json:"access_hash"` // кэшированный access hash
    LastScraped time.Time `json:"last_scraped"`
    LastMsgID   int       `json:"last_msg_id"` // ID последнего обработанного
    IsActive    bool      `json:"is_active"`
    Metadata    struct {
        Keywords      []string `json:"keywords"`       // фильтр по ключевым словам
        MinViews      int      `json:"min_views"`      // минимум просмотров
        ScrapeReplies bool     `json:"scrape_replies"` // парсить ли ответы
    } `json:"metadata"`
}
```

---

## 9. Flow Диаграммы

### 9.1. Flow: Первичная авторизация

```
┌──────────────┐     ┌───────────────┐     ┌──────────────┐
│   Operator   │     │  Auth Script  │     │   Telegram   │
└──────┬───────┘     └───────┬───────┘     └──────┬───────┘
       │                     │                    │
       │  Set API_ID/HASH    │                    │
       │────────────────────►│                    │
       │                     │                    │
       │                     │  Connect MTProto   │
       │                     │───────────────────►│
       │                     │                    │
       │                     │◄───────────────────│
       │                     │   Request code     │
       │                     │                    │
       │  Enter phone        │                    │
       │────────────────────►│                    │
       │                     │  Send auth code    │
       │                     │───────────────────►│
       │                     │                    │
       │  SMS/TG code        │                    │
       │◄────────────────────┼────────────────────│
       │                     │                    │
       │  Enter code         │                    │
       │────────────────────►│                    │
       │                     │  Verify code       │
       │                     │───────────────────►│
       │                     │                    │
       │                     │◄───────────────────│
       │                     │   Session created  │
       │                     │                    │
       │  Session String     │                    │
       │◄────────────────────│                    │
       │  (save to .env)     │                    │
       │                     │                    │
```

### 9.2. Flow: Парсинг канала

```
┌──────────┐  ┌───────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐
│ Scheduler│  │ Collector │  │ Telegram │  │ Postgres │  │  NATS    │
└────┬─────┘  └─────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘
     │              │             │             │             │
     │ scrape.start │             │             │             │
     │─────────────►│             │             │             │
     │              │             │             │             │
     │              │ GetHistory  │             │             │
     │              │────────────►│             │             │
     │              │             │             │             │
     │              │ Messages[]  │             │             │
     │              │◄────────────│             │             │
     │              │             │             │             │
     │              │ Check existing external_ids            │
     │              │────────────────────────►│             │
     │              │             │             │             │
     │              │ Known IDs[] │             │             │
     │              │◄────────────────────────│             │
     │              │             │             │             │
     │              │ Filter new messages     │             │
     │              │─────────────────────────┼─────────────┼─►
     │              │             │             │             │
     │              │ INSERT new jobs (RAW)   │             │
     │              │────────────────────────►│             │
     │              │             │             │             │
     │              │             │             │ jobs.new   │
     │              │─────────────┼─────────────┼────────────►│
     │              │             │             │             │
```

### 9.3. Flow: Отправка отклика

```
┌──────────┐  ┌──────────┐  ┌────────────┐  ┌──────────┐  ┌──────────┐
│  Web UI  │  │   NATS   │  │ Dispatcher │  │ Telegram │  │ Postgres │
└────┬─────┘  └────┬─────┘  └─────┬──────┘  └────┬─────┘  └────┬─────┘
     │             │              │              │             │
     │ "Send"      │              │              │             │
     │────────────►│              │              │             │
     │             │ dispatch.send│              │             │
     │             │─────────────►│              │             │
     │             │              │              │             │
     │             │              │ Get job_app  │             │
     │             │              │─────────────────────────►│
     │             │              │              │             │
     │             │              │◄─────────────────────────│
     │             │              │ pdf_path, text            │
     │             │              │              │             │
     │             │              │ Upload PDF   │             │
     │             │              │─────────────►│             │
     │             │              │              │             │
     │             │              │ Send message │             │
     │             │              │─────────────►│             │
     │             │              │              │             │
     │             │              │ ✓ Delivered  │             │
     │             │              │◄─────────────│             │
     │             │              │              │             │
     │             │              │ UPDATE status=SENT        │
     │             │              │─────────────────────────►│
     │             │              │              │             │
     │ ✓ Success   │              │              │             │
     │◄────────────┼──────────────│              │             │
     │             │              │              │             │
```

---

## 10. Безопасность

### Риски

| Риск                   | Последствие              | Митигация                                         |
| ---------------------- | ------------------------ | ------------------------------------------------- |
| Утечка session string  | Полный доступ к аккаунту | Хранить только в secrets, не коммитить            |
| Спам                   | Бан аккаунта             | Rate limiting, задержки, не отправлять незнакомым |
| FloodWait накопление   | Временный бан            | Exponential backoff, respecting wait time         |
| Telegram TOS нарушение | Перманентный бан         | Не автоматизировать массовую рассылку             |

### Рекомендации

1. **Используйте отдельный аккаунт** — не основной
2. **Прогрейте аккаунт** — перед автоматизацией попользуйтесь вручную 1-2 недели
3. **Не отправляйте одинаковые сообщения** — Telegram детектит спам
4. **Лимитируйте отправку** — максимум 10-20 сообщений в день незнакомым
5. **Добавьте человеческие паузы** — случайные задержки 1-5 секунд

### Мониторинг здоровья аккаунта

```go
func checkAccountHealth(ctx context.Context, client *gotgproto.Client) error {
    // пробуем получить информацию о себе
    self, err := client.API().UsersGetFullUser(ctx, &tg.InputUserSelf{})
    if err != nil {
        return fmt.Errorf("account may be restricted: %w", err)
    }

    log.Printf("account healthy: @%s", self.Users[0].(*tg.User).Username)
    return nil
}
```

---

## Следующие шаги

1. [ ] Создать auth-сервис для генерации session string
2. [ ] Реализовать TG Strategy в Collector
3. [ ] Добавить TG каналы в `scraping_targets`
4. [ ] Тестирование на реальных каналах (readonly)
5. [ ] Реализовать Dispatcher для отправки
