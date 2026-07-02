# Соцсети-ВСЁ! (SM_BCE)

Веб-чат на Go и WebSocket. Есть общий чат, личные диалоги, теги пользователей, история сообщений и тёмная тема.

## Что внутри

- Go-сервер без фронтенд-сборщика.
- WebSocket для сообщений в реальном времени.
- PostgreSQL для пользователей и истории.
- Docker Compose для локального запуска.
- Клиентское шифрование сообщений в браузере.
- Serveo-скрипт для временной публичной ссылки.

## Запуск через Docker

```powershell
docker compose up --build
```

По умолчанию приложение открывается на:

```text
http://localhost:8080
```

Если порт занят:

```powershell
$env:HOST_PORT=8081
docker compose up --build
```

Остановка:

```powershell
docker compose down
```

Остановка с удалением базы:

```powershell
docker compose down -v
```

## Запуск без Docker

```powershell
go run ./cmd/server
```

Без `DATABASE_URL` сервер использует локальные JSON-файлы. Они не попадают в git.

## Публичная ссылка через Serveo

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\start-public-chat-serveo.ps1
```

Скрипт запускает Docker Compose, проверяет локальный порт и открывает SSH-туннель через Serveo. По умолчанию используется `localhost:8081`.

Остановить:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\stop-public-chat-serveo.ps1
```

## Переменные окружения

```text
DATABASE_URL=postgres://sm_bce:sm_bce_password@postgres:5432/sm_bce?sslmode=disable
PORT=8080
HOST_PORT=8081
```

## Структура

```text
cmd/server/   сервер
web/          клиентская часть
scripts/      запуск туннеля
```
