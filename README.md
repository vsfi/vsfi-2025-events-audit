# Events Audit Server with NATS JetStream

Высокопроизводительный сервер для прослушивания и логирования событий из NATS JetStream с поддержкой структурированного логирования через logrus, персистентности сообщений и гарантированной доставки.

## 🚀 Возможности

- **NATS JetStream Integration**: Подключение к NATS JetStream с персистентностью и гарантированной доставкой
- **Durable Consumers**: Долговременные подписчики с возможностью восстановления после сбоев
- **Message Acknowledgments**: Подтверждение обработки сообщений с повторными попытками
- **Stream Management**: Автоматическое создание и управление JetStream потоками
- **Structured Logging**: Логирование через logrus в текстовом или JSON формате
- **Event Parsing**: Автоматическое распознавание JSON событий и обработка raw сообщений
- **Graceful Shutdown**: Корректная обработка сигналов завершения (SIGTERM/SIGINT)
- **Flexible Configuration**: Настройка через CLI параметры или переменные окружения
- **High Performance**: Оптимизирован для обработки высоких нагрузок с pull-based подписками
- **Production Ready**: Готов к использованию в production среде с HA поддержкой

## 📦 Установка

### Сборка из исходников

```bash
git clone <repository-url>
cd vsfi-2025-events-audit
make build
```

### Или с использованием Go

```bash
go build -o events-audit .
```

### Сборка с примерами

```bash
make build build-sender
```

## 🔧 Быстрый старт

### 1. Запуск NATS JetStream сервера (Docker)

```bash
make docker-up
# или
docker-compose up -d
```

### 2. Запуск сервера аудита

```bash
# Базовый запуск с JetStream
./events-audit --audit=nats --audit-nats-addr=nats://localhost:4222 --audit-topic=events.>

# Или с использованием Makefile
make run
```

### 3. Отправка тестовых событий в JetStream

```bash
# В отдельном терминале
make sender
```

### 4. Полная демонстрация JetStream

```bash
make demo
```

## 🎯 Использование

### Запуск с различными конфигурациями JetStream

```bash
# Базовые параметры
./events-audit --audit=nats --audit-nats-addr=nats://localhost:4222 --audit-topic=events.>

# JSON логирование с пользовательским потоком
./events-audit --audit=nats --audit-nats-addr=nats://localhost:4222 \
  --audit-topic=audit.events.> \
  --audit-stream-name=AUDIT_EVENTS \
  --log-format=json

# Production конфигурация
./events-audit --audit=nats --audit-nats-addr=nats://cluster.prod.com:4222 \
  --audit-topic=prod.events.> \
  --audit-stream-name=PROD_EVENTS \
  --audit-consumer-name=prod-audit-consumer \
  --audit-durable-name=prod-audit-durable \
  --audit-max-deliver=5 \
  --audit-ack-wait=60s \
  --audit-pull-max-messages=50 \
  --audit-stream-replicas=3 \
  --log-level=info \
  --log-format=json

# Высокопроизводительная конфигурация
./events-audit --audit=nats --audit-nats-addr=nats://localhost:4222 \
  --audit-topic=events.> \
  --audit-pull-max-messages=100 \
  --audit-pull-timeout=1s \
  --log-level=warn
```

### Использование переменных окружения

```bash
# Создайте .env файл на основе .env.example
cp .env.example .env

# Отредактируйте переменные для JetStream
export AUDIT_LISTNER_AUDIT=nats
export AUDIT_LISTNER_AUDIT_NATS_ADDR=nats://localhost:4222
export AUDIT_LISTNER_AUDIT_TOPIC=events.>
export AUDIT_LISTNER_AUDIT_STREAM_NAME=EVENTS
export AUDIT_LISTNER_AUDIT_CONSUMER_NAME=events-audit-consumer
export AUDIT_LISTNER_AUDIT_DURABLE_NAME=events-audit-durable
export AUDIT_LISTNER_LOG_LEVEL=info
export AUDIT_LISTNER_LOG_FORMAT=json

# Запуск
./events-audit
```

## ⚙️ Конфигурация JetStream

### Основные параметры

| Параметр | Переменная окружения | Описание | По умолчанию |
|----------|---------------------|----------|--------------|
| `--audit` | `AUDIT_LISTNER_AUDIT` | Тип аудита (`nats`, `nsq`, `nope`) | `nope` |
| `--audit-nats-addr` | `AUDIT_LISTNER_AUDIT_NATS_ADDR` | Адрес NATS сервера | - |
| `--audit-topic` | `AUDIT_LISTNER_AUDIT_TOPIC` | Subject pattern для подписки | `accountats` |

### JetStream конфигурация

| Параметр | Переменная окружения | Описание | По умолчанию |
|----------|---------------------|----------|--------------|
| `--audit-stream-name` | `AUDIT_LISTNER_AUDIT_STREAM_NAME` | Имя JetStream потока | `EVENTS` |
| `--audit-consumer-name` | `AUDIT_LISTNER_AUDIT_CONSUMER_NAME` | Имя consumer | `events-audit-consumer` |
| `--audit-durable-name` | `AUDIT_LISTNER_AUDIT_DURABLE_NAME` | Имя durable consumer | `events-audit-durable` |
| `--audit-create-stream` | `AUDIT_LISTNER_AUDIT_CREATE_STREAM` | Создавать поток автоматически | `true` |

### Обработка сообщений

| Параметр | Переменная окружения | Описание | По умолчанию |
|----------|---------------------|----------|--------------|
| `--audit-max-deliver` | `AUDIT_LISTNER_AUDIT_MAX_DELIVER` | Максимум попыток доставки | `3` |
| `--audit-ack-wait` | `AUDIT_LISTNER_AUDIT_ACK_WAIT` | Время ожидания ACK | `30s` |
| `--audit-pull-max-messages` | `AUDIT_LISTNER_AUDIT_PULL_MAX_MESSAGES` | Максимум сообщений за раз | `10` |
| `--audit-pull-timeout` | `AUDIT_LISTNER_AUDIT_PULL_TIMEOUT` | Timeout для pull запросов | `5s` |

### Хранение потока

| Параметр | Переменная окружения | Описание | По умолчанию |
|----------|---------------------|----------|--------------|
| `--audit-stream-max-age` | `AUDIT_LISTNER_AUDIT_STREAM_MAX_AGE` | Максимальный возраст сообщений | `24h` |
| `--audit-stream-max-bytes` | `AUDIT_LISTNER_AUDIT_STREAM_MAX_BYTES` | Максимум байт в потоке | `1GB` |
| `--audit-stream-max-msgs` | `AUDIT_LISTNER_AUDIT_STREAM_MAX_MSGS` | Максимум сообщений в потоке | `1000000` |
| `--audit-stream-replicas` | `AUDIT_LISTNER_AUDIT_STREAM_REPLICAS` | Количество реплик потока | `1` |

### Логирование

| Параметр | Переменная окружения | Описание | По умолчанию |
|----------|---------------------|----------|--------------|
| `--log-level` | `AUDIT_LISTNER_LOG_LEVEL` | Уровень логирования | `debug` |
| `--log-format` | `AUDIT_LISTNER_LOG_FORMAT` | Формат логов (`text`/`json`) | `text` |

### Примеры NATS URL

```bash
# Локальный сервер
nats://localhost:4222

# С аутентификацией
nats://user:password@localhost:4222

# Кластер (автоматический failover)
nats://nats1.example.com:4222,nats2.example.com:4222,nats3.example.com:4222

# TLS
tls://localhost:4443

# С токеном
nats://token@localhost:4222
```

## 📋 JetStream Features

### Персистентность сообщений

JetStream сохраняет сообщения на диск, что гарантирует их доступность даже после перезапуска сервера:

```bash
# Сообщения сохраняются в потоке до истечения max-age или достижения лимитов
--audit-stream-max-age=7d          # Хранить 7 дней
--audit-stream-max-bytes=10GB      # Максимум 10GB
--audit-stream-max-msgs=5000000    # Максимум 5M сообщений
```

### Durable Consumers

Durable consumers позволяют продолжить обработку с места остановки:

```bash
# Consumer запоминает последнее обработанное сообщение
--audit-durable-name=my-audit-consumer

# При перезапуске обработка продолжится с необработанных сообщений
```

### Гарантированная доставка

```bash
# Настройка повторных попыток
--audit-max-deliver=5              # До 5 попыток доставки
--audit-ack-wait=60s               # 60 секунд на обработку

# Сообщения требуют явного подтверждения (ACK)
# Неподтвержденные сообщения будут переданы повторно
```

### High Availability

```bash
# Кластерная конфигурация с репликацией
--audit-nats-addr=nats://nats1.prod:4222,nats2.prod:4222,nats3.prod:4222
--audit-stream-replicas=3          # 3 реплики для отказоустойчивости
```

## 📊 Примеры логов

### Текстовый формат (development)

```
INFO[11.01.2025 10:30:00.123] Starting JetStream events audit server                
INFO[11.01.2025 10:30:00.124] Connected to NATS                            url=nats://localhost:4222
INFO[11.01.2025 10:30:00.125] Created JetStream stream                     stream=EVENTS subjects=[events.>]
INFO[11.01.2025 10:30:00.126] Created JetStream consumer                   consumer=events-audit-consumer durable=events-audit-durable stream=EVENTS
INFO[11.01.2025 10:30:00.127] Starting to listen for JetStream events     subject=events.> stream=EVENTS consumer=events-audit-consumer
INFO[11.01.2025 10:30:05.456] Received structured event                   event_id=event-12345 event_type=user.action source=user-service stream=EVENTS sequence=1 delivered=1 event_data=map[action:login user_id:67890]
INFO[11.01.2025 10:30:06.789] Received raw event                          raw_data="Simple text message" subject=events.test stream=EVENTS sequence=2 delivered=1
```

### JSON формат (production)

```json
{"level":"info","msg":"Starting JetStream events audit server","time":"11.01.2025 10:30:00.123"}
{"level":"info","msg":"Connected to NATS","time":"11.01.2025 10:30:00.124","url":"nats://localhost:4222"}
{"level":"info","msg":"Created JetStream stream","stream":"EVENTS","subjects":["events.>"],"time":"11.01.2025 10:30:00.125"}
{"level":"info","msg":"Created JetStream consumer","consumer":"events-audit-consumer","durable":"events-audit-durable","stream":"EVENTS","time":"11.01.2025 10:30:00.126"}
{"level":"info","msg":"Received structured event","event_id":"event-12345","event_type":"user.action","source":"user-service","stream":"EVENTS","sequence":1,"delivered":1,"event_data":{"action":"login","user_id":"67890"},"time":"11.01.2025 10:30:05.456"}
{"level":"info","msg":"Received raw event","raw_data":"Simple text message","subject":"events.test","stream":"EVENTS","sequence":2,"delivered":1,"time":"11.01.2025 10:30:06.789"}
```

## 🔄 JetStream Workflow

### 1. Инициализация
1. Подключение к NATS серверу
2. Получение JetStream context
3. Создание/обновление потока (если включено)
4. Создание durable consumer
5. Запуск pull-based подписки

### 2. Обработка сообщений
1. Pull сообщений из потока (batch)
2. Обработка каждого сообщения
3. Логирование события
4. Отправка ACK при успехе
5. NAK при ошибке (для повторной попытки)

### 3. Обработка ошибок
- **Временные ошибки**: NAK → повторная доставка
- **Постоянные ошибки**: TERM после max-deliver попыток
- **Таймауты**: автоматическая повторная доставка

## 🛑 Graceful Shutdown

```bash
# При получении SIGTERM или SIGINT:
1. Остановка приема новых сообщений
2. Drain текущих pull запросов
3. Завершение обработки активных сообщений
4. Закрытие подписки
5. Закрытие NATS соединения
```

## 🧪 Тестирование

### Unit тесты

```bash
# Все тесты
go test ./...

# С покрытием
go test -cover ./...

# Бенчмарки
go test -bench=. ./internal/nats
```

### Интеграционное тестирование с JetStream

```bash
# Полный тест с NATS JetStream
./test.sh all

# Отдельные тесты
./test.sh basic          # Базовая функциональность
./test.sh json           # JSON логирование
./test.sh volume         # Высокая нагрузка
./test.sh jetstream      # JetStream специфические функции
./test.sh env            # Переменные окружения
```

### Доступные команды Makefile

```bash
make help               # Показать все доступные команды
make build              # Собрать приложение
make test               # Запустить тесты
make docker-up          # Запустить NATS JetStream сервер
make docker-down        # Остановить NATS сервер
make sender             # Отправить тестовые события
make demo               # Полная демонстрация JetStream
make demo-performance   # Высокопроизводительная демонстрация
make nats-status        # Проверить состояние NATS
make nats-streams       # Показать JetStream потоки
make nats-consumers     # Показать consumers
make nats-monitor       # Мониторинг метрик
make clean              # Очистить артефакты сборки
```

## 🐳 Docker & NATS JetStream

### Запуск NATS JetStream сервера

```bash
# Запуск с JetStream и персистентностью
docker-compose up -d

# Проверка логов
docker-compose logs -f nats

# Остановка
docker-compose down
```

### Доступные порты

- **4222** - NATS client connections
- **8222** - HTTP management interface с JetStream метриками
- **6222** - NATS cluster connections

### Management Interface

После запуска NATS доступен web-интерфейс с JetStream метриками: http://localhost:8222

#### JetStream endpoints:
- http://localhost:8222/jsz - JetStream информация
- http://localhost:8222/jsz?streams=1 - Детали потоков
- http://localhost:8222/jsz?consumers=1 - Детали consumers

## 🔍 Мониторинг JetStream

### Команды мониторинга

```bash
# Общее состояние JetStream
make nats-status

# Информация о потоках
make nats-streams

# Информация о consumers
make nats-consumers

# Непрерывный мониторинг
make nats-monitor
```

### Ключевые метрики JetStream

```bash
# Состояние потоков
curl -s http://localhost:8222/jsz?streams=1 | jq '.streams[0].state'

# Состояние consumers
curl -s http://localhost:8222/jsz?consumers=1 | jq '.streams[0].consumer_detail[0]'

# Общая память JetStream
curl -s http://localhost:8222/jsz | jq '.memory, .storage'
```

## 📈 Производительность

### Бенчмарк результаты

```
BenchmarkEventLogger_HandleEvent-24           177988    7722 ns/op
BenchmarkEventLogger_HandleRawEvent-24        712218    1722 ns/op
```

### Рекомендации для production с JetStream

1. **Логирование**: Используйте `info` уровень и `json` формат
2. **Pull конфигурация**: 
   - `--audit-pull-max-messages=50-100` для высокой нагрузки
   - `--audit-pull-timeout=1s-5s` в зависимости от latency требований
3. **Stream лимиты**: Настройте по потребностям
   - `--audit-stream-max-age=7d-30d` для длительного хранения
   - `--audit-stream-max-bytes=10GB-100GB` в зависимости от объема
4. **HA конфигурация**: `--audit-stream-replicas=3` для кластера
5. **Ресурсы**: Рекомендуется: 512MB RAM, 2 CPU cores, SSD storage

## 🚨 Troubleshooting

### Частые проблемы JetStream

**1. Не удается подключиться к NATS JetStream**
```bash
# Проверьте, что JetStream включен
curl http://localhost:8222/jsz

# Проверьте логи NATS
docker-compose logs nats
```

**2. Stream не создается**
```bash
# Проверьте права доступа
curl http://localhost:8222/jsz?streams=1

# Проверьте конфигурацию
./events-audit --audit-create-stream=true --log-level=debug
```

**3. Consumer не получает сообщения**
```bash
# Проверьте consumer состояние
make nats-consumers

# Проверьте subject pattern
./examples/sender --subject=events.test --stream=EVENTS
```

**4. Сообщения не подтверждаются**
```bash
# Проверьте ack_pending
curl -s http://localhost:8222/jsz?consumers=1 | jq '.streams[0].consumer_detail[0].num_ack_pending'

# Увеличьте timeout
--audit-ack-wait=60s
```

**5. Высокое потребление диска JetStream**
```bash
# Проверьте размер потоков
make nats-streams

# Настройте лимиты
--audit-stream-max-age=24h
--audit-stream-max-bytes=1GB
```

## 🔧 Разработка

### Структура проекта

```
vsfi-2025-events-audit/
├── main.go                         # Точка входа с JetStream параметрами
├── internal/
│   ├── nats/
│   │   ├── client.go              # NATS JetStream клиент
│   │   ├── handler.go             # Обработчики событий с JetStream метаданными
│   │   └── handler_test.go        # Тесты
│   └── server/
│       └── server.go              # Основная логика сервера
├── examples/
│   └── sender.go                  # Пример отправки в JetStream
├── docker-compose.yml             # NATS JetStream сервер
├── Makefile                       # Автоматизация с JetStream командами
├── test.sh                        # Тестовые скрипты для JetStream
├── .env.example                   # Примеры конфигурации JetStream
└── README.md                      # Документация
```

### Архитектура JetStream

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Events App    │───▶│  NATS JetStream  │───▶│  Audit Server   │
│                 │    │                  │    │                 │
│ Publish events  │    │ ┌──────────────┐ │    │ Pull & Process  │
│ to subjects     │    │ │    Stream    │ │    │ Log events      │
│                 │    │ │   (EVENTS)   │ │    │ Send ACK/NAK    │
└─────────────────┘    │ │              │ │    └─────────────────┘
                       │ │  Persistent  │ │
                       │ │   Storage    │ │
                       │ └──────────────┘ │
                       │                  │
                       │ ┌──────────────┐ │
                       │ │   Consumer   │ │
                       │ │  (Durable)   │ │
                       │ └──────────────┘ │
                       └──────────────────┘
```

## 📄 Требования

- **Go**: 1.23.10 или выше
- **NATS**: 2.10 или выше с JetStream
- **Storage**: SSD рекомендуется для JetStream (особенно в production)
- **Memory**: Минимум 512MB для JetStream workloads
- **Docker**: для запуска NATS JetStream сервера (опционально)

## 📚 Зависимости

- `github.com/nats-io/nats.go` - NATS клиент с JetStream поддержкой
- `github.com/sirupsen/logrus` - Структурированное логирование
- `github.com/urfave/cli/v3` - CLI интерфейс
- `github.com/juju/errors` - Обработка ошибок

## 🔗 Полезные ссылки

- [NATS JetStream Documentation](https://docs.nats.io/nats-concepts/jetstream)
- [NATS Go Client](https://github.com/nats-io/nats.go)
- [JetStream Best Practices](https://docs.nats.io/running-a-nats-service/nats_admin/jetstream_admin)

## 📄 Лицензия

[Укажите лицензию проекта]

## 🤝 Поддержка

Для вопросов и предложений создавайте Issues в репозитории проекта.

---

**Версия**: 2.0.0 (JetStream)  
**Автор**: [Ваше имя]  
**Последнее обновление**: 11.01.2025  
**JetStream**: ✅ Enabled