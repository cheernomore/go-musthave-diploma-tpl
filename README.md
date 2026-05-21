# Гофермарт

Накопительная система лояльности «Гофермарт» — HTTP-сервис на Go, который принимает заказы зарегистрированных пользователей, опрашивает внешнюю систему расчёта баллов и ведёт балансы лояльности.

## Запуск

```
go run ./cmd/gophermart \
  -a :8080 \
  -d "postgres://gophermart:gophermart@localhost:5432/gophermart?sslmode=disable" \
  -r "http://localhost:8081"
```

Любой флаг можно заменить переменной окружения: `RUN_ADDRESS`, `DATABASE_URI`, `ACCRUAL_SYSTEM_ADDRESS`. Env имеет приоритет над флагом.

Дополнительные переменные окружения:

| Переменная | По умолчанию | Назначение |
|---|---|---|
| `JWT_SECRET` | `gophermart-dev-secret` | Секрет для подписи JWT-токенов |
| `LOG_LEVEL` | `info` | Уровень логирования (`debug`, `info`, `warn`, `error`) |

При старте сервис автоматически применяет SQL-миграции из `internal/storage/postgres/migrations` (на основе `golang-migrate`).

## API

| Метод и путь | Назначение |
|---|---|
| `POST /api/user/register` | Регистрация пользователя |
| `POST /api/user/login` | Аутентификация |
| `POST /api/user/orders` | Загрузка номера заказа (text/plain) |
| `GET /api/user/orders` | Список заказов пользователя |
| `GET /api/user/balance` | Текущий баланс |
| `POST /api/user/balance/withdraw` | Списание баллов |
| `GET /api/user/withdrawals` | История списаний |
| `GET /healthz` | Проверка готовности |

Токен возвращается в заголовке `Authorization: Bearer <token>` и cookie `Authorization` после успешной регистрации/аутентификации. Защищённые маршруты принимают токен в любом из этих способов.

## Архитектура

```
cmd/gophermart            — entrypoint, graceful shutdown
internal/app              — сборка зависимостей и errgroup
internal/config           — конфиг (флаги + env)
internal/logger           — log/slog JSON handler
internal/domain           — модели и доменные ошибки
internal/luhn             — валидатор номеров заказов
internal/service/auth     — регистрация, логин, JWT
internal/service/order    — загрузка и список заказов
internal/service/balance  — баланс и списания
internal/storage/postgres — pgxpool + миграции + репозитории
internal/httpapi          — HTTP-обработчики и middleware
internal/accrual          — клиент к системе расчёта баллов
internal/worker           — пул фоновых воркеров опроса accrual
```

Транзакционные операции (создание пользователя + создание баланса, списание, начисление по результату accrual) выполняются в одной транзакции pgx. Конкурентный пул воркеров использует `SELECT ... FOR UPDATE SKIP LOCKED` для непересекающегося разбора очереди.

## Тесты

```
go test ./...
go test -coverprofile=cover.out ./... && go tool cover -func=cover.out
```

Покрытие — выше 60 % по проекту.
