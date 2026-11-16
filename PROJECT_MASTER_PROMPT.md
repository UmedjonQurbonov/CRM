# MASTER SPECIFICATION & PROMPT: Retail POS & Inventory Backend (Go Modular Monolith)

> **Назначение документа:** Этот файл является полным мастер-промптом (Technical Specification & Master Prompt). Его можно передать любой AI-модели или агенту разработки (Claude, ChatGPT, Cursor, Windsurf, Antigravity) для воспроизведения, расширения или генерации данного бэкенда с нуля без потери контекста и архитектурных решений.

---

## 1. СИСТЕМНЫЙ ПРОМПТ (System Directive)

```text
Ты — Senior Go Backend Engineer и Solution Architect. 
Твоя задача — реализовать production-ready бэкенд для системы мобильной кассы (POS), складского учета и P&L аналитики розничного магазина на языке Go в парадигме Clean Architecture (модульный монолит).

Требования к коду:
1. Запрещены любые заглушки, моки или TODO в бизнес-логике. Все методы должны быть полностью функциональными и строго типизированными.
2. Не используй тяжелые веб-фреймворки (Gin, Fiber, Beego). Используй chi router (github.com/go-chi/chi/v5) и стандартную библиотеку net/http.
3. Доступ к PostgreSQL — только через нативный драйвер github.com/jackc/pgx/v5 (pgxpool) с ручным написанием эффективных SQL-запросов (без ORM).
4. Все методы сервисов и репозиториев обязаны принимать context.Context первым аргументом. Ошибки оборачиваются через fmt.Errorf("...: %w", err).
5. Все ресурсы базы данных (rows) закрываются через defer rows.Close().
6. Все эндпоинты снабжаются подробными OpenAPI/Swagger аннотациями для swaggo/swag.
```

---

## 2. ТЕХНОЛОГИЧЕСКИЙ СТЕК

| Компонент | Технология | Назначение / Обоснование |
| :--- | :--- | :--- |
| **Язык** | Go 1.22+ | Статическая типизация, предсказуемая производительность |
| **Архитектурный стиль** | Modular Monolith (Clean Architecture) | Изоляция доменов, отсутствие циклов импортов |
| **HTTP Маршрутизатор** | `github.com/go-chi/chi/v5` | Совместим со `standard net/http`, легковесный |
| **База данных** | PostgreSQL 16 Alpine | `github.com/jackc/pgx/v5` (`pgxpool`), пул 25 соединений |
| **Миграции БД** | `github.com/golang-migrate/migrate/v4` | Автоматический накат SQL-миграций при старте приложения до открытия сокета |
| **Кэш & Черный список** | Redis 7 Alpine | `github.com/redis/go-redis/v9` (Blacklist JWT, кэш каталога и P&L) |
| **Аутентификация** | JWT HS256 (`github.com/golang-jwt/jwt/v5`) | Access-токен (15м), Refresh-токен (30д, SHA-256 в БД) |
| **Хэширование паролей** | `golang.org/x/crypto/bcrypt` | Стоимость `cost = 12` |
| **Генерация QR-кодов** | `github.com/skip2/go-qrcode` | Потоковая генерация PNG на лету без сохранения на диск |
| **Документация API** | Swagger 2.0 (`swaggo/swag`) | Интерактивный UI по адресу `/swagger/index.html` |
| **Контейнеризация** | Docker & Docker Compose | Multi-stage сборка на базе `alpine:3.20` |

---

## 3. СТРУКТУРА ПАПОК (Directory Tree)

```text
.
├── cmd/
│   └── api/
│       └── main.go                 # Точка входа: config, DB pool, migrations, redis, router, graceful shutdown
├── docs/                           # Swagger документация (docs.go, swagger.json, swagger.yaml)
├── internal/
│   ├── platform/                   # Инфраструктурный платформенный слой
│   │   ├── config/config.go        # Загрузка и валидация .env
│   │   ├── database/postgres.go    # Инициализация pgxpool.Pool (MaxConns: 25, MinConns: 5)
│   │   ├── database/migrator.go    # Идемпотентный запуск golang-migrate
│   │   ├── cache/redis.go          # Nil-safe обертка над Redis (Set, Get, Del, Exists, DeleteByPattern)
│   │   └── logger/logger.go        # log/slog (JSON для production, Text для dev)
│   ├── middleware/                 # HTTP Middleware
│   │   ├── auth.go                 # JWT Authentication (проверка подписи, claims, Redis Blacklist)
│   │   └── require_role.go         # RBAC авторизация по ролям (owner, seller)
│   ├── domain/
│   │   └── errors.go               # Sentinel domain errors (ErrNotFound, ErrForbidden, ErrConflict и др.)
│   ├── dto/
│   │   ├── error_response.go       # RFC 7807 ErrorResponse + WriteError/WriteJSON
│   │   └── paginated_response.go   # Generic PaginatedResponse[T] + парсер limit/offset
│   ├── auth/                       # Модуль авторизации (domain, dto, repo, service, handler)
│   ├── bootstrap/                  # Модуль безопасного создания первого owner
│   ├── products/                   # Модуль каталога, остатков и QR-кодов
│   ├── orders/                     # Модуль кассовых чеков и атомарных транзакций продаж
│   ├── sellers/                    # Модуль продавцов и комиссионных вознаграждений
│   ├── expenses/                   # Модуль учета операционных расходов
│   └── analytics/                  # Модуль P&L аналитики (GP, NP, рейтинг продавцов, топ продаж)
├── migrations/
│   ├── 000001_init_schema.up.sql
│   └── 000001_init_schema.down.sql
├── Dockerfile                      # Multi-stage production Docker build
├── docker-compose.yml              # postgres, redis, backend
├── Makefile                        # build, run, swag, docker-up, test
├── .env.example
└── .env
```

---

## 4. СХЕМА БАЗЫ ДАННЫХ (PostgreSQL DDL)

### `migrations/000001_init_schema.up.sql`

```sql
-- Таблица пользователей системы (Директора и Продавцы)
CREATE TABLE IF NOT EXISTS users (
    id               UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    name             VARCHAR(100)  NOT NULL,
    phone            VARCHAR(20)   UNIQUE NOT NULL,
    password_hash    VARCHAR(255)  NOT NULL,
    role             VARCHAR(20)   NOT NULL CHECK (role IN ('owner', 'seller')),
    commission_rate  NUMERIC(5, 2) NOT NULL DEFAULT 0.00
                         CHECK (commission_rate >= 0.00 AND commission_rate <= 100.00),
    is_active        BOOLEAN       NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

-- Таблица refresh-сессий для мобильных клиентов кассы
CREATE TABLE IF NOT EXISTS refresh_sessions (
    id          UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID          NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  VARCHAR(255)  NOT NULL,
    device_info VARCHAR(255),
    expires_at  TIMESTAMPTZ   NOT NULL,
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

-- Таблица номенклатуры и складских остатков
CREATE TABLE IF NOT EXISTS products (
    id               UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    name             VARCHAR(255)   NOT NULL,
    sku              VARCHAR(64)    UNIQUE,
    qr_code          VARCHAR(128)   UNIQUE NOT NULL,
    cost_price       NUMERIC(12, 2) NOT NULL DEFAULT 0.00 CHECK (cost_price >= 0.00),
    selling_price    NUMERIC(12, 2) NOT NULL DEFAULT 0.00 CHECK (selling_price >= 0.00),
    stock_quantity   INT            NOT NULL DEFAULT 0  CHECK (stock_quantity >= 0),
    min_stock_alert  INT            NOT NULL DEFAULT 5  CHECK (min_stock_alert >= 0),
    is_active        BOOLEAN        NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

-- Таблица кассовых чеков (Заказов)
CREATE TABLE IF NOT EXISTS orders (
    id                       UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    seller_id                UUID           NOT NULL REFERENCES users(id),
    total_amount             NUMERIC(12, 2) NOT NULL CHECK (total_amount >= 0.00),
    total_cost               NUMERIC(12, 2) NOT NULL CHECK (total_cost >= 0.00),
    commission_rate_snapshot NUMERIC(5, 2)  NOT NULL,
    commission_earned        NUMERIC(12, 2) NOT NULL DEFAULT 0.00 CHECK (commission_earned >= 0.00),
    payment_method           VARCHAR(20)    NOT NULL CHECK (payment_method IN ('cash', 'card', 'transfer')),
    status                   VARCHAR(20)    NOT NULL DEFAULT 'completed' CHECK (status IN ('completed', 'refunded')),
    created_at               TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

-- Таблица позиций кассового чека
CREATE TABLE IF NOT EXISTS order_items (
    id              UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id        UUID           NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id      UUID           NOT NULL REFERENCES products(id),
    quantity        INT            NOT NULL CHECK (quantity > 0),
    unit_price      NUMERIC(12, 2) NOT NULL CHECK (unit_price >= 0.00),
    unit_cost_price NUMERIC(12, 2) NOT NULL CHECK (unit_cost_price >= 0.00),
    subtotal        NUMERIC(12, 2) NOT NULL CHECK (subtotal >= 0.00)
);

-- Таблица операционных расходов магазина
CREATE TABLE IF NOT EXISTS expenses (
    id           UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    created_by   UUID           NOT NULL REFERENCES users(id),
    category     VARCHAR(100)   NOT NULL,
    amount       NUMERIC(12, 2) NOT NULL CHECK (amount > 0.00),
    comment      TEXT,
    expense_date DATE           NOT NULL DEFAULT CURRENT_DATE,
    created_at   TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

-- Производительные индексы
CREATE INDEX IF NOT EXISTS idx_products_qr           ON products(qr_code) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_products_sku          ON products(sku) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_orders_seller_created ON orders(seller_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_orders_status_created ON orders(status, created_at);
CREATE INDEX IF NOT EXISTS idx_order_items_order_id  ON order_items(order_id);
CREATE INDEX IF NOT EXISTS idx_expenses_date         ON expenses(expense_date DESC);
```

---

## 5. ФУНДАМЕНТАЛЬНЫЕ БИЗНЕС-ПРАВИЛА И ИНВАРИАНТЫ

### 1. Атомарное оформление чека (Транзакционность кассы)
- Продажа нескольких позиций оформляется в рамках единой транзакции `pgx.Tx` (`ReadCommitted`).
- Для каждого товара остаток уменьшается с блокировкой условия:
  ```sql
  UPDATE products
  SET stock_quantity = stock_quantity - $1, updated_at = NOW()
  WHERE id = $2 AND stock_quantity >= $1 AND is_active = TRUE
  RETURNING cost_price;
  ```
- Если хотя бы одного товара не хватает (затронуто 0 строк) — транзакция немедленно откатывается (`Rollback`), и клиенту возвращается `400 Bad Request` с кодом `INSUFFICIENT_STOCK` и указанием идентификатора товара.
- В чеке фиксируются **неизменяемые слепки** на момент продажи: `unit_price`, `unit_cost_price`, `subtotal`, `commission_rate_snapshot`. Они **никогда не пересчитываются задним числом**, даже если изменилась цена товара в каталоге или процент продавца.
- Если заказ проводит `owner`, процент комиссии фиксируется как `0.00` и `commission_earned = 0.00`.

### 2. Сокрытие себестоимости и RBAC
- В системе существуют роли: `owner` (директор) и `seller` (кассир/продавец).
- Для роли `seller` поле `cost_price` (себестоимость/закупочная цена) **принудительно обнуляется ($0.00)** во всех ответах API. Кассир ни при каких условиях не должен видеть себестоимость и маржинальность.
- Продавец в списке заказов видит **только свои** чеки (`seller_id == current_user_id`).
- К разделам расходов (`/expenses`), аналитики (`/analytics`), созданию и редактированию товаров (`POST/PUT /products`) продавцы доступа не имеют (`403 Forbidden`).

### 3. P&L Финансовая аналитика (Profit & Loss)
- Выручка ($R$): `SUM(total_amount)` по заказам со статусом `completed` за выбранный интервал дат.
- Себестоимость ($C$): `SUM(total_cost)` (сумма `quantity * unit_cost_price`).
- Валовая прибыль (Gross Profit):
  $$GP = R - C$$
- Операционные расходы ($E$): `SUM(expenses.amount)`.
- Комиссии продавцов ($Comm$): `SUM(commission_earned)`.
- Чистая прибыль (Net Profit):
  $$NP = GP - E - Comm$$

### 4. Генерация QR-кодов on-the-fly
- Эндпоинт `GET /api/v1/products/{id}/qr-image` генерирует изображение PNG размером `256×256` пикселей прямо в оперативной памяти через `skip2/go-qrcode` и стримит в ответ с заголовком `Content-Type: image/png`. Никакие картинки на диск не сохраняются.

### 5. Первичная инициализация (Bootstrap)
- Так как регистрация пользователей в системе закрыта, первый директор создается через эндпоинт `POST /api/v1/bootstrap/owner`.
- Запрос защищен заголовком `X-Bootstrap-Secret`, который сверяется с переменной окружения `BOOTSTRAP_SECRET`.
- Если в базе уже есть хотя бы один пользователь с ролью `owner`, эндпоинт возвращает `409 Conflict` (повторный вызов невозможен).

### 6. Nil-Safe Redis & Blacklist токенов
- При выходе из системы (`POST /auth/logout`) идентификатор токена `jti` заносится в Redis `blacklist:{jti}` с TTL, равным остаточному времени жизни access-токена.
- При валидации входящего токена в `middleware.JWTAuth` проверяется наличие `jti` в Redis blacklist.
- Все методы клиента Redis (`Get`, `Set`, `Del`, `Exists`, `DeleteByPattern`) реализованы **nil-safe**: если Redis недоступен или упал, сервер не падает в panic, а логирует предупреждение и продолжает обслуживать запросы.
- Кэш списков товаров и аналитики инвалидируется по маске (`DeleteByPattern("cache:products:*")`, `DeleteByPattern("cache:analytics:*")`) при создании чеков, товаров и расходов.

---

## 6. СПИСОК МАРШРУТОВ API (Таблица эндпоинтов)

Все маршруты API имеют префикс `/api/v1`.

| HTTP Метод | URL Путь | Роли доступа | Описание |
| :--- | :--- | :--- | :--- |
| **POST** | `/bootstrap/owner` | Публичный + `X-Bootstrap-Secret` | Создание первого директора системы |
| **POST** | `/auth/login` | Публичный | Вход по номеру телефона и паролю (выдача Access + Refresh) |
| **POST** | `/auth/refresh` | Публичный | Ротация пары токенов по действующему refresh-токену |
| **POST** | `/auth/logout` | Авторизован (`owner`, `seller`) | Выход: занесение токена в Redis blacklist и удаление сессии |
| **GET** | `/products` | `owner`, `seller` | Каталог товаров с пагинацией и поиском по SKU/названию |
| **GET** | `/products/by-qr/{code}` | `owner`, `seller` | Мгновенный поиск товара по значению со сканера QR/штрихкода |
| **POST** | `/products` | `owner` | Создание нового товара на складе |
| **PUT** | `/products/{id}` | `owner` | Редактирование товара, цен и остатков |
| **GET** | `/products/{id}/qr-image`| `owner` | Получение PNG-картинки QR-кода товара на лету |
| **POST** | `/orders` | `owner`, `seller` | Оформление кассового чека (атомарное списание остатков) |
| **GET** | `/orders` | `owner`, `seller` | История заказов (продавец видит только свои, владелец — все) |
| **GET** | `/orders/{id}` | `owner`, `seller` | Детализация кассового чека с позициями |
| **POST** | `/orders/{id}/refund` | `owner` | Возврат чека с автоматическим возвратом остатков на склад |
| **GET** | `/sellers` | `owner` | Список зарегистрированных продавцов |
| **POST** | `/sellers` | `owner` | Регистрация нового продавца с указанием % комиссии |
| **PATCH**| `/sellers/{id}/commission`| `owner` | Изменение ставки комиссии продавца |
| **GET** | `/sellers/my-earnings` | `seller` | Личный кабинет продавца: сумма продаж и заработанная комиссия |
| **GET** | `/expenses` | `owner` | Список операционных расходов за интервал дат |
| **POST** | `/expenses` | `owner` | Внесение операционного расхода |
| **DELETE**| `/expenses/{id}` | `owner` | Удаление ошибочного расхода |
| **GET** | `/analytics/summary` | `owner` | Сводный отчет P&L (выручка, себестоимость, GP, расходы, NP) |
| **GET** | `/analytics/sellers-ranking` | `owner` | Рейтинг продавцов по объему продаж и их процентная доля |
| **GET** | `/analytics/top-products`| `owner` | Топ-20 самых продаваемых позиций за период |
| **GET** | `/swagger/*` | Публичный | Интерактивная документация Swagger UI |

---

## 7. КОНФИГУРАЦИЯ ОКРУЖЕНИЯ (`.env.example`)

```dotenv
# HTTP Server
APP_ENV=development
SERVER_PORT=8080
SERVER_READ_TIMEOUT=15s
SERVER_WRITE_TIMEOUT=15s

# PostgreSQL
POSTGRES_USER=pos_user
POSTGRES_PASSWORD=pos_secure_password_123
POSTGRES_DB=pos_inventory_db
POSTGRES_HOST=postgres
POSTGRES_PORT=5432
POSTGRES_HOST_EXTERNAL_PORT=5433

DATABASE_URL=postgres://pos_user:pos_secure_password_123@postgres:5432/pos_inventory_db?sslmode=disable
MIGRATIONS_DIR=/app/migrations

# Redis
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=

# JWT Secrets
JWT_ACCESS_SECRET=super_secret_access_jwt_key_987654321
JWT_REFRESH_SECRET=super_secret_refresh_jwt_key_123456789
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=720h

# Bootstrap Secret (для POST /api/v1/bootstrap/owner)
BOOTSTRAP_SECRET=super_secret_bootstrap_token_12345
```

---

## 8. ДЕПЛОЙ И ОРКЕСТРАЦИЯ

### Dockerfile (Multi-Stage)
```dockerfile
# Stage 1: Build binary
FROM golang:1.22-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git ca-certificates tzdata
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go install github.com/swaggo/swag/cmd/swag@latest && swag init -g cmd/api/main.go -o docs
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /app/bin/api ./cmd/api

# Stage 2: Minimal runtime image
FROM alpine:3.20
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
COPY --from=builder /app/bin/api /app/api
COPY --from=builder /app/migrations /app/migrations
COPY --from=builder /app/docs /app/docs
RUN chown -R appuser:appgroup /app
USER appuser
EXPOSE 8080
ENTRYPOINT ["/app/api"]
```

### docker-compose.yml
```yaml
version: '3.8'

services:
  postgres:
    image: postgres:16-alpine
    container_name: pos_postgres
    restart: unless-stopped
    environment:
      POSTGRES_USER: ${POSTGRES_USER:-pos_user}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-pos_secure_password_123}
      POSTGRES_DB: ${POSTGRES_DB:-pos_inventory_db}
    ports:
      - "${POSTGRES_HOST_EXTERNAL_PORT:-5433}:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    networks:
      - pos_network
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER:-pos_user} -d ${POSTGRES_DB:-pos_inventory_db}"]
      interval: 5s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    container_name: pos_redis
    restart: unless-stopped
    command: redis-server --appendonly yes
    ports:
      - "6380:6379"
    volumes:
      - redis_data:/data
    networks:
      - pos_network
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 5s
      retries: 5

  backend:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: pos_backend
    restart: unless-stopped
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    env_file:
      - .env
    environment:
      - DATABASE_URL=postgres://${POSTGRES_USER:-pos_user}:${POSTGRES_PASSWORD:-pos_secure_password_123}@postgres:5432/${POSTGRES_DB:-pos_inventory_db}?sslmode=disable
      - MIGRATIONS_DIR=/app/migrations
      - REDIS_HOST=redis
      - REDIS_PORT=6379
    ports:
      - "8080:8080"
    networks:
      - pos_network

volumes:
  postgres_data:
    name: pos_postgres_data
  redis_data:
    name: pos_redis_data

networks:
  pos_network:
    name: pos_network
    driver: bridge
```

---

## 9. ПРОВЕРОЧНЫЙ СЦЕНАРИЙ (Testing Protocol)
1. Запуск: `docker compose up -d --build`.
2. Открытие документации: `http://localhost:8080/swagger/index.html`.
3. Создание первого владельца: `POST /api/v1/bootstrap/owner` с заголовком `X-Bootstrap-Secret`.
4. Вход в систему: `POST /api/v1/auth/login` -> получение Bearer токена.
5. Авторизация в Swagger UI через кнопку **Authorize** (`Bearer <access_token>`).
6. Регистрация продавца: `POST /api/v1/sellers` со ставкой комиссии $5\%$.
7. Добавление товара на склад: `POST /api/v1/products` (цена, себестоимость, остаток).
8. Оформление продажи: `POST /api/v1/orders` -> проверка списания остатка на складе.
9. Внесение расхода: `POST /api/v1/expenses`.
10. Проверка P&L аналитики: `GET /api/v1/analytics/summary` (проверка формулы $NP = (R - C) - E - Comm$).
