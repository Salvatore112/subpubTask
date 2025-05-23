## Сервис PubSub
![Build Status](https://github.com/Salvatore112/subpubTask/actions/workflows/go.yml/badge.svg)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

### Основная функция
Это gRPC-сервис для обмена сообщениями по принципу **Publisher-Subscriber**.

- Клиенты могут подписаться на события по ключу (например, `news` или `weather`).
- Клиенты могут публиковать сообщения по ключу.
- Все подписчики ключа получают сообщения в реальном времени.

### Технологии
- **Go** (основной язык)
- **gRPC** (для коммуникации)
- **Protocol Buffers** (для определения API)

## Запуск 

1. **Подготовка**
   Убедитесь, что у вас установлены:
   - Docker
   - Docker Compose

2. **Запуск сервиса**
   ```bash
   docker-compose up --build
   ```
Сервис запустится на порту 50051.

## Примеры использования

### Подписка на сообщения
   ```bash
grpcurl -plaintext  -d '{"key": "news"}' localhost:50051 pubsub.PubSub/Subscribe
   ```

### Публикация сообщения 
```bash
grpcurl -plaintext -d '{"key": "news", "data": "Тестовое сообщение"}' localhost:50051 pubsub.PubSub/Publish
   ```
