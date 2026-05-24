# Aseek — семантический поиск с RAG на Aurora OS

Aseek — приложение для ОС Аврора с локальным RAG-пайплайном.  
Фронтенд на Qt/QML, бэкенд-оркестратор на Go, инференс ч��рез llama.cpp.

## Архитектура

```
┌─────────────────────────┐
│  Qt/QML (Aurora OS)     │
│  src/main.cpp, qml/     │
└──────────┬──────────────┘
           │ IPC (Unix socket, TLV)
┌──────────▼──────────────┐
│  Go Orchestrator         │
│  orchestrator/           │
│  ├── cmd/cli/            │  тестовый CLI-клиент
│  ├── internal/config/    │  runtime-конфигурация
│  ├── internal/ipc/       │  IPC-сервер + TLV
│  ├── internal/llama/     │  HTTP-клиент к llama.cpp
│  ├── internal/logging/   │  structured logging
│  ├── internal/pipeline/  │  RAG-пайплайн
│  ├── internal/profile/   │  профили поисковых серверов
│  ├── internal/prompt/    │  сборка промптов
│  ├── internal/request/   │  менеджер запросов
│  ├── internal/streaming/ │  стриминг токенов
│  └── internal/supervisor/│  управление llama-server
└──────────┬──────────────┘
           │ HTTP (OpenAI-compatible)
┌──────────▼──────────────┐
│  llama.cpp (submodule)   │
└─────────────────────────┘
```

Жизненный цикл:
1. Qt-приложение запускает оркестратор как дочерний процесс
2. Оркестратор стартует `llama-server` и ожидает его готовности
3. Qt подключается к оркестратору через Unix domain socket
4. Пользовательский запрос проходит RAG-пайплайн: embedding → top-k search → reranking → prompt build → генерация → стриминг
5. При завершении Qt оркестратор останавливает `llama-server` и завершается

## Структура проекта

```
.
├── CMakeLists.txt                 # Сборка Aurora-приложения
├── src/main.cpp                   # Точка входа Qt
├── qml/                           # QML-интерфейс
│   ├── Aseek.qml
│   ├── cover/DefaultCoverPage.qml
│   └── pages/
│       ├── MainPage.qml
│       └── AboutPage.qml
├── icons/                         # Иконки приложения
├── translations/                  # Переводы (ru)
├── rpm/                           # RPM-спек и changelog
├── orchestrator/                  # Go-бэкенд
│   ├── main.go                    # Точка входа оркестратора
│   ├── go.mod
│   ├── internal/                  # Внутренние пакеты
│   ├── cmd/cli/                   # CLI-клиент для отладки
│   ├── api/search_protocol.md     # Протокол поискового API
│   └── orchestrator.md            # Архитектурная спецификация
├── llama.cpp/                     # Сабмодуль llama.cpp
└── .gitignore
```

## Сборка

### Aurora OS приложение

```bash
mb2 -t AuroraOS_5.2.0.180_aarch64 build
```

### Оркестратор

```bash
cd orchestrator
go build -o aseek-orchestrator .
```

### CLI-клиент (отладка)

```bash
cd orchestrator
go run ./cmd/cli
```

## Конфигурация

Оркестратор читает JSON-конфиг из `~/.config/aurora-rag/orchestrator.json` (или путь из аргумента командной строки):

```json
{
  "llama": {
    "binary": "/opt/aurora/bin/llama-server",
    "model": "/opt/aurora/models/model.gguf",
    "port": 8081,
    "ctx_size": 4096,
    "threads": 4,
    "batch": 256
  },
  "streaming": {
    "flush_interval_ms": 40
  },
  "network": {
    "request_timeout_ms": 15000
  },
  "logging": {
    "level": "debug"
  }
}
```

## IPC-протокол

Оркестратор и фронтенд общаются через Unix domain socket (`$XDG_RUNTIME_DIR/aurora-rag.sock`) по бинарному TLV-протоколу.

Формат заголовка (16 байт, Big-Endian):

| Поле      | Размер  |
|-----------|---------|
| Magic     | 2 байта |
| Version   | 2 байта |
| Type      | 4 байта |
| Length    | 4 байта |
| RequestID | 4 байта |

Типы сообщений: `Query`, `Cancel`, `Ping`, `Token`, `Done`, `Busy`, `Error`, `ProfileList`.

Подробнее: [orchestrator/orchestrator.md](orchestrator/orchestrator.md)

## Поисковое API

Удалённые поисковые серверы реализуют единый endpoint:

```
POST /api/search
Content-Type: application/json

{"query": "...", "top_k": 5}
```

Подробнее: [orchestrator/api/search_protocol.md](orchestrator/api/search_protocol.md)

## Лицензия

BSD 3-Clause