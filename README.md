# Backend — FQW

Бэкенд состоит из трёх Go-сервисов и одного Python-сервиса, оркестрируемых через Docker Compose.

## Сервисы

| Сервис | Язык | Порт | Назначение |
|---|---|---|---|
| `auth-service` | Go (Gin) | 8081 | Аутентификация, JWT, управление пользователями |
| `manage-service` | Go (Gin) | 8080 | Основная бизнес-логика: абитуриенты, документы, эксперты |
| `statistics-service` | Go (Gin) | 8083 | Агрегация статистики по программам |
| `data-extraction-service` | Python (FastAPI) | 8000 | AI-извлечение данных из документов (Qwen3-VL) |

## Инфраструктура

| Сервис | Назначение |
|---|---|
| PostgreSQL 15 | Основная БД (порт 5433 на хосте) |
| MinIO | S3-совместимое хранилище документов (порты 9000, 9001) |
| RabbitMQ 3.12 | Очередь задач для AI-обработки (порты 5672, 15672) |
| Redis 7 | Хранилище токенов / сессий |

## Быстрый старт

```bash
# Собрать и запустить все сервисы
docker-compose up --build -d

# Только пересобрать конкретный сервис
docker-compose up --build manage-service -d

# Логи в реальном времени
docker-compose logs -f manage-service

# Остановить всё
docker-compose down
```

## Структура manage-service

```
manage-service/
├── internal/
│   ├── controller/http/v1/   # Роутер, хэндлеры, middleware
│   ├── usecase/              # Бизнес-логика
│   ├── repository/           # Работа с БД (pgx)
│   ├── domain/entity/        # Доменные модели
│   └── websocket/            # WebSocket-хаб (live-статусы обработки)
├── migrations/               # SQL-миграции (применяются при старте)
└── docs/                     # Swagger (генерируется swag)
```

## Миграции

Миграции применяются автоматически при запуске `manage-service`. Файлы в `manage-service/migrations/` пронумерованы по возрастанию и применяются последовательно.

## Swagger UI

После запуска:
- manage-service: [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)
- auth-service: [http://localhost:8081/swagger/index.html](http://localhost:8081/swagger/index.html)

## Переменные окружения (manage-service)

| Переменная | Значение по умолчанию | Описание |
|---|---|---|
| `HTTP_PORT` | `8080` | Порт HTTP-сервера |
| `PG_URL` | — | PostgreSQL DSN |
| `MINIO_ENDPOINT` | `minio:9000` | Адрес MinIO |
| `MINIO_ACCESS_KEY` | `minioadmin` | Логин MinIO |
| `MINIO_SECRET_KEY` | `minioadmin` | Пароль MinIO |
| `MINIO_BUCKET` | `applicants` | Имя бакета |
| `RABBITMQ_URL` | `amqp://guest:guest@rabbitmq:5672/` | Подключение к RabbitMQ |
| `EXTRACTION_SERVICE_URL` | `http://extraction-service:8000` | URL Python AI-сервиса |
| `JWT_SIGN_KEY` | `secret-key` | Секрет подписи JWT |

## Жизненный цикл абитуриента

```
uploaded → processing → verifying → assessed → completed
```

- `uploaded` — абитуриент создан
- `processing` — документы обрабатываются AI
- `verifying` — оператор проверяет данные
- `assessed` — передан экспертам на оценивание
- `completed` — оценивание завершено
