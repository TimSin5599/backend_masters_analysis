# Backend — FQW

Бэкенд состоит из трёх Go-сервисов и одного Python-сервиса, оркестрируемых через Docker Compose.

## Сервисы

| Сервис | Язык | Порт | Назначение |
|---|---|---|---|
| `auth-service` | Go (Gin) | 8081 | Аутентификация, JWT, управление пользователями |
| `manage-service` | Go (Gin) | 8080 | Основная бизнес-логика: абитуриенты, документы, эксперты |
| `statistics-service` | Go (Gin) | 8083 | Агрегация статистики по программам |
| `data-extraction-service` | Python (FastAPI) | 8000 | AI-извлечение данных из документов (gemma4 через Ollama) |

## Инфраструктура

| Сервис | Назначение |
|---|---|
| PostgreSQL 15 | Основная БД |
| MinIO | S3-совместимое хранилище документов (порты 9000, 9001) |
| RabbitMQ 3.12 | Очередь задач для AI-обработки (порты 5672, 15672) |
| Redis 7 | Хранилище refresh-токенов |
| Nginx | API-шлюз + раздача фронтенда + SSL-терминация |
| Prometheus + Grafana | Мониторинг и дашборды метрик |
| Certbot | Автоматическое обновление SSL-сертификатов (Let's Encrypt) |

## Быстрый старт

### Разработка (docker-compose.dev.yml)

```bash
cd backend/deploy

# Запуск в режиме разработки
docker compose -f docker-compose.dev.yml up --build -d

# Логи конкретного сервиса
docker compose -f docker-compose.dev.yml logs -f manage-service

# Остановить всё
docker compose -f docker-compose.dev.yml down
```

### Продакшен (docker-compose.yml)

```bash
cd backend/deploy

# Скопировать и заполнить переменные окружения
cp .env.example .env

# Собрать и запустить
docker compose up --build -d

# Пересобрать конкретный сервис
docker compose up --build manage-service -d
```

### Локальная разработка Go-сервиса

```bash
cd manage-service
make run         # запуск
make test        # unit-тесты
make e2e         # e2e-тесты (требует запущенную БД)
make swag        # регенерация Swagger-документации
```

## Структура manage-service

```
manage-service/
├── cmd/                          # Точка входа (main)
├── config/                       # Конфигурация через env
├── internal/
│   ├── controller/http/v1/
│   │   ├── handlers/             # HTTP-хэндлеры (applicant, document, expert, program, annotation)
│   │   ├── router.go             # Регистрация всех маршрутов
│   │   └── middleware.go         # JWT, CORS, NoCacheMiddleware
│   ├── usecase/                  # Бизнес-логика
│   ├── repository/               # Работа с БД (pgx)
│   ├── domain/entity/            # Доменные модели и константы статусов
│   ├── infrastructure/
│   │   ├── extraction/           # HTTP-клиент к data-extraction-service (+ AI-скоринг)
│   │   └── s3/                   # MinIO-клиент
│   ├── rabbitmq/                 # Публикация задач в очередь
│   └── sse/                      # SSE-хаб (live-статусы обработки документов)
├── pkg/
│   ├── metrics/                  # Prometheus-метрики (HTTP, документы, SSE, RabbitMQ)
│   ├── httpserver/               # Обёртка над http.Server
│   ├── logger/                   # Логирование
│   └── postgres/                 # Пул соединений pgx
├── migrations/                   # SQL-миграции (38 шт., применяются при старте)
├── e2e/                          # E2E-тесты
└── docs/                         # Swagger (генерируется swag)
```

## API маршруты (manage-service)

### Программы
| Метод | Путь | Права | Описание |
|---|---|---|---|
| `GET` | `/v1/programs` | all | Список программ |
| `POST` | `/v1/programs` | admin | Создать программу |
| `GET` | `/v1/programs/:id` | all | Получить программу |
| `PUT` | `/v1/programs/:id` | admin, manager | Изменить статус программы |

### Абитуриенты
| Метод | Путь | Описание |
|---|---|---|
| `GET` | `/v1/applicants` | Список с фильтрацией |
| `POST` | `/v1/applicants` | Создать |
| `DELETE` | `/v1/applicants/:id` | Удалить |
| `GET` | `/v1/applicants/:id/data` | Все данные портфолио |
| `PATCH` | `/v1/applicants/:id/data` | Обновить данные |
| `DELETE` | `/v1/applicants/:id/data/:category/:dataId` | Удалить запись категории |
| `POST` | `/v1/applicants/:id/transfer-to-operator` | Перевести на проверку оператору |
| `POST` | `/v1/applicants/:id/transfer-to-experts` | Передать экспертам |
| `GET` | `/v1/applicants/:id/status/stream` | SSE: live-статус обработки |
| `GET` | `/v1/applicants/:id/annotation/stream` | SSE: генерация AI-аннотации |

### Документы
| Метод | Путь | Описание |
|---|---|---|
| `POST` | `/v1/applicants/:id/documents` | Загрузить документ |
| `GET` | `/v1/applicants/:id/documents` | Список документов |
| `GET` | `/v1/applicants/:id/documents/view` | Просмотр (последнего) документа |
| `DELETE` | `/v1/applicants/:id/documents/:docId` | Удалить документ |
| `POST` | `/v1/applicants/:id/documents/reprocess` | Повторная обработка последнего документа |
| `GET` | `/v1/applicants/:id/queue-status` | Статус очереди обработки |
| `GET` | `/v1/documents/:id/status` | Статус документа |
| `PATCH` | `/v1/documents/:id/status` | Ручное обновление статуса |
| `GET` | `/v1/documents/:id/view` | Просмотр по ID |
| `POST` | `/v1/documents/:id/reprocess` | Повторная обработка по ID |
| `PATCH` | `/v1/documents/:id/category` | Смена категории документа |

### Экспертная оценка
| Метод | Путь | Права | Описание |
|---|---|---|---|
| `GET` | `/v1/applicants/:id/evaluations` | all | Список оценок |
| `PUT` | `/v1/applicants/:id/evaluations` | all | Сохранить оценку |
| `GET` | `/v1/applicants/:id/criteria` | all | Критерии оценки |
| `GET` | `/v1/applicants/:id/scoring-scheme` | all | Схема скоринга |
| `PATCH` | `/v1/applicants/:id/scoring-scheme` | admin | Изменить схему скоринга |
| `GET` | `/v1/experts/slots` | all | Слоты экспертов |
| `POST` | `/v1/experts/slots` | all | Назначить эксперта |
| `GET` | `/v1/experts` | all | Список экспертов |
| `GET` | `/v1/criteria` | all | Все критерии |
| `POST` | `/v1/criteria` | admin | Создать критерий |
| `PUT` | `/v1/criteria/:code` | admin | Обновить критерий |
| `DELETE` | `/v1/criteria/:code` | admin | Удалить критерий |

## Конвейер обработки документов

Документ проходит 8 стадий:

```
pending → classifying → [classification_failed] → classified → extracting → completed
                                                                          ↘ extraction_failed
```

| Статус | Описание |
|---|---|
| `pending` | Документ в очереди, ожидает воркера |
| `classifying` | AI определяет тип документа |
| `classification_failed` | Не удалось определить тип — пользователь выбирает категорию вручную |
| `classified` | Тип определён, готов к извлечению данных |
| `extracting` | AI извлекает структурированные данные |
| `completed` | Данные успешно извлечены и сохранены |
| `extraction_failed` | Извлечение не удалось — оператор может ввести данные вручную |

## Жизненный цикл абитуриента

```
uploaded → processing → verifying → assessed → completed
```

| Статус | Описание |
|---|---|
| `uploaded` | Абитуриент создан, документы ещё не обрабатываются |
| `processing` | Документы обрабатываются AI |
| `verifying` | Оператор проверяет и корректирует данные |
| `assessed` | Передан экспертам на оценивание |
| `completed` | Экспертное оценивание завершено |

## Категории данных портфолио

| Категория | Сущность |
|---|---|
| `identification` | Паспортные данные |
| `education` | Диплом о высшем образовании |
| `transcript` | Транскрипт (GPA, кредиты) |
| `work_experience` | Опыт работы (с полем `competencies`) |
| `language` | Языковые сертификаты |
| `motivation` | Мотивационное письмо |
| `recommendation` | Рекомендательное письмо |
| `achievement` | Достижения (с полем `achievement_type`) |
| `resume` | Резюме |
| `video` | Видеопрезентация (URL) |

## SSE (Server-Sent Events)

Два SSE-эндпоинта в manage-service:

- **`/v1/applicants/:id/status/stream`** — live-обновления статуса обработки документов через `sse.Hub`
- **`/v1/applicants/:id/annotation/stream`** — потоковая генерация AI-аннотации портфолио; поддерживает `?regenerate=true` для сброса кэша

## Система критериев оценки

Критерии имеют два типа: `BASE` (балльные) и `BLOCKING` (блокирующие — без прохождения нельзя передать абитуриента). Поддерживаются две схемы скоринга: `default` и `ieee`. Критерии могут быть привязаны к конкретной программе (`program_id`) или применяться ко всем.

## Мониторинг

Prometheus-метрики доступны на `/metrics`. Grafana настроена через `deploy/grafana/provisioning/`. Метрики:

- `http_requests_total` — счётчик HTTP-запросов (method, path, status)
- `http_request_duration_seconds` — гистограмма задержек
- `manage_documents_uploaded_total` — загруженные документы по категориям
- `manage_documents_processed_total` — результаты AI-обработки (success/error)
- `manage_sse_active_clients` — активные SSE-соединения
- `manage_rabbitmq_published_total` / `manage_rabbitmq_consumed_total` — трафик очереди

## Swagger UI

После запуска:
- manage-service: [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)
- auth-service: [http://localhost:8081/swagger/index.html](http://localhost:8081/swagger/index.html)

## Миграции

38 SQL-миграций применяются автоматически при запуске `manage-service`. Файлы в `manage-service/migrations/` пронумерованы и выполняются последовательно.

## Переменные окружения

| Переменная | Сервисы | Описание |
|---|---|---|
| `APP_NAME` | все Go | Имя сервиса (для логов) |
| `APP_VERSION` | все Go | Версия приложения |
| `HTTP_PORT` | все | Порт HTTP-сервера |
| `LOG_LEVEL` | все Go | Уровень логирования |
| `PG_URL` | auth, manage, stats | PostgreSQL DSN |
| `PG_POOL_MAX` | auth, manage, stats | Максимальный размер пула соединений |
| `JWT_SIGN_KEY` | auth, manage | Секрет подписи JWT |
| `REDIS_URL` | auth | URL Redis (`redis://redis:6379/0`) |
| `CORS_ALLOW_ORIGIN` | все Go | Разрешённые origins (через запятую) |
| `SWAGGER_HOST` | все Go | Хост для Swagger UI |
| `MINIO_ENDPOINT` | manage | Адрес MinIO (`minio:9000`) |
| `MINIO_ACCESS_KEY` | manage | Логин MinIO |
| `MINIO_SECRET_KEY` | manage | Пароль MinIO |
| `MINIO_BUCKET` | manage | Имя бакета (`applicants`) |
| `RABBITMQ_URL` | manage | URL RabbitMQ (`amqp://...`) |
| `EXTRACTION_SERVICE_URL` | manage | URL data-extraction-service |
| `OLLAMA_HOST` | extraction | Адрес Ollama (`http://host.docker.internal:11434`) |
| `MODEL_NAME` | extraction | Имя модели (`gemma4:latest`) |
