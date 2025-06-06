# PubSub gRPC Service

Реализация gRPC сервиса для публикации и подписки на события. Сервис позволяет клиентам подписываться на темы и получать сообщения в реальном времени.

## 🚀 Особенности

- Подписка на события по ключу
- Публикация сообщений для подписчиков
- Потоковая передача данных (server-side streaming)
- Поддержка graceful shutdown
- Логирование операций в файл в production

## 📦 Установка

1. Клонируйте репозиторий:
```bash
git clone https://github.com/ukawop/vk_pubsub.git
cd vk_pubsub
```
2. Установите зависимости:
```bash
go mod download
```
3. Запустите приложение:
```bash
make run
```
4. Подпишитесь на уведомления:
```bash
grpcurl -plaintext -proto proto/subpub/subpub.proto \
  -d '{"key": "news"}' localhost:9000 subpubv1.PubSub/Subscribe
```
5. Получайте публикации:
```bash
grpcurl -plaintext -proto proto/subpub/subpub.proto \
  -d '{"key": "news", "data": "New update available!"}' \
  localhost:9000 subpubv1.PubSub/Publish
```
