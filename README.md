# Retail POS & Inventory Backend

A high-performance, modular monolithic backend for a Retail Point of Sale (POS) and Inventory Management system built in Go 1.22+ adhering strictly to Clean Architecture, Domain-Driven Design (DDD), and RFC 7807 problem details standards.

---

## 🌟 Key Features

1. **Authentication & RBAC**:
   - Secure JWT token pair issuance (Access Token 15m, Refresh Token 7d with single-use rotation and SHA-256 token hashing).
   - Role-Based Access Control distinguishing **Owner** (Store Director) and **Seller** (Cashier).
   - Automated bootstrap seeder creating default owner account on startup if the database is empty.

2. **Inventory Management & Security**:
   - Product catalog with SKU, QR code, stock quantity tracking, and low-stock threshold alerts.
   - **Cost Price Masking**: `cost_price` is strictly masked to `0.00` for sellers and only visible to the owner.
   - Dynamic QR code generation in PNG binary format for physical label printing.

3. **Transactional POS Checkout & Returns**:
   - **Atomic Stock Decrement**: Uses row-level conditional updates (`stock_quantity >= $1`) preventing race conditions and overselling.
   - **Commission Snapshots**: Sellers receive their contract commission rate snapshotted at checkout time; owner checkouts are strictly locked to `0.00%`.
   - **Refunds with Stock Restoration**: Owner-only return process that atomically restores product stock back into inventory and marks the receipt as `refunded`.

4. **Operational Expenses Tracking**:
   - Store overhead recording (rent, utilities, salaries, cleaning, supplies) with date filtering and physical deletion for erroneous records. Strictly owner-only.

5. **Financial P&L & Real-time Analytics**:
   - Real-time Profit & Loss statement based on standard financial formulas:
     $$\text{Gross Profit } (GP) = \text{Revenue } (R) - \text{Cost of Goods Sold } (C)$$
     $$\text{Net Profit } (NP) = GP - \text{Total Expenses } (E) - \text{Total Commissions } (Comm)$$
   - Seller performance rankings with safe relative revenue share calculation.
   - Top-selling products ranked by revenue.
   - Cashier personal earnings summary (`/api/v1/sellers/my-earnings`).

---

## 🏗️ Architecture & Technology Stack

- **Language**: Go 1.22+
- **HTTP Router**: [Chi v5](https://github.com/go-chi/chi) with standard middleware (RequestID, RealIP, Logger, Recoverer, CORS, Timeout).
- **Database**: PostgreSQL 16 with [pgx/v5](https://github.com/jackc/pgx) connection pool (`pgxpool.Pool`).
- **Migrations**: [golang-migrate/migrate/v4](https://github.com/golang-migrate/migrate) with embedded SQL files (`embed.FS`).
- **Caching & Health**: Redis 7 ([go-redis/v9](https://github.com/redis/go-redis)).
- **Precision Math**: [shopspring/decimal](https://github.com/shopspring/decimal) for all monetary and commission calculations.
- **Documentation**: OpenAPI 3.0 / Swagger UI generated via [swaggo/swag](https://github.com/swaggo/swag).
- **Containerization**: Multi-stage minimal Docker container based on `alpine:3.20`.

---

## 📡 API Endpoints Summary

| Module | Method | Endpoint | Allowed Role | Description |
| :--- | :---: | :--- | :---: | :--- |
| **System** | `GET` | `/health` | Public | System health check (PostgreSQL & Redis connection status) |
| **System** | `GET` | `/swagger/*` | Public | Interactive Swagger UI API documentation |
| **Auth** | `POST` | `/api/v1/auth/login` | Public | Authenticate via phone and password; returns token pair |
| **Auth** | `POST` | `/api/v1/auth/refresh` | Public | Rotate refresh token and issue new token pair |
| **Auth** | `POST` | `/api/v1/auth/logout` | Authenticated | Invalidate refresh token session |
| **Sellers** | `GET` | `/api/v1/sellers` | **Owner** | List all registered staff members |
| **Sellers** | `POST` | `/api/v1/sellers` | **Owner** | Register a new seller (with custom commission rate) |
| **Sellers** | `PATCH`| `/api/v1/sellers/{id}/commission` | **Owner** | Update a seller's commission percentage |
| **Sellers** | `GET` | `/api/v1/sellers/my-earnings` | **Seller / Owner** | View caller's personal sales, commission rate, and earned payout |
| **Products** | `GET` | `/api/v1/products` | **Seller / Owner** | Browse catalog (cost price masked to 0.00 for sellers) |
| **Products** | `GET` | `/api/v1/products/by-qr/{code}` | **Seller / Owner** | Fast lookup of product by scanned barcode/QR code |
| **Products** | `POST` | `/api/v1/products` | **Owner** | Add a new product to catalog |
| **Products** | `PUT` | `/api/v1/products/{id}` | **Owner** | Update product attributes, prices, and stock threshold |
| **Products** | `GET` | `/api/v1/products/{id}/qr-image` | **Owner** | Stream PNG image of generated QR code |
| **Orders** | `POST` | `/api/v1/orders` | **Seller / Owner** | Process checkout transaction with atomic inventory decrement |
| **Orders** | `GET` | `/api/v1/orders` | **Seller / Owner** | List receipts (sellers isolated to own receipts, owner sees all) |
| **Orders** | `GET` | `/api/v1/orders/{id}` | **Seller / Owner** | View receipt details with line items |
| **Orders** | `POST` | `/api/v1/orders/{id}/refund` | **Owner** | Process return: restores stock to catalog and marks receipt refunded |
| **Expenses** | `POST` | `/api/v1/expenses` | **Owner** | Record a store operational expense |
| **Expenses** | `GET` | `/api/v1/expenses` | **Owner** | List expenses with category and date range filters |
| **Expenses** | `DELETE`| `/api/v1/expenses/{id}`| **Owner** | Permanently delete an erroneous expense entry |
| **Analytics**| `GET` | `/api/v1/analytics/summary` | **Owner** | Store financial Profit & Loss statement ($R, C, GP, E, Comm, NP$) |
| **Analytics**| `GET` | `/api/v1/analytics/sellers-ranking` | **Owner** | Sellers performance rankings with revenue share percentage |
| **Analytics**| `GET` | `/api/v1/analytics/top-products` | **Owner** | Best-selling products ranked by total revenue |

---

## 🚀 Quick Start & Deployment

### Option A: Running with Docker Compose (Recommended)

Start the complete application stack (PostgreSQL, Redis, and Backend):

```bash
docker compose up -d --build
```

Verify running containers:

```bash
docker compose ps
```

Access services:
- **API Base URL**: `http://localhost:8080`
- **Interactive Swagger UI**: `http://localhost:8080/swagger/index.html`
- **Health Check**: `http://localhost:8080/health`

View application logs:

```bash
docker compose logs -f backend
```

---

### Option B: Local Development Setup

1. **Start Infrastructure Services**:
   ```bash
   docker compose up -d postgres redis
   ```

2. **Configure Environment (`.env`)**:
   ```env
   SERVER_PORT=8080
   SERVER_ENV=development
   DB_HOST=localhost
   DB_PORT=5433
   DB_USER=postgres
   DB_PASSWORD=postgrespassword
   DB_NAME=crm_db
   DB_SSLMODE=disable
   REDIS_ADDR=localhost:6380
   JWT_SECRET=super-secret-jwt-key-for-retail-pos-local
   ADMIN_PHONE=+992900000000
   ADMIN_PASSWORD=AdminPass123!
   ```

3. **Run Migrations & Server**:
   ```bash
   go run cmd/api/main.go
   ```

---

## 🔑 Default Credentials

Upon startup, the database is automatically seeded with an initial Owner account if no users exist:
- **Phone**: `+992900000000` (configurable via `ADMIN_PHONE`)
- **Password**: `AdminPass123!` (configurable via `ADMIN_PASSWORD`)
- **Role**: `owner`
- **Commission Rate**: `0.00%`

---

## 🧪 Running Tests & Quality Checks

Run all unit tests across modules:

```bash
go test -v ./...
```

Run code formatting and static analysis:

```bash
go fmt ./...
go vet ./...
```

Regenerate Swagger API documentation:

```bash
go run github.com/swaggo/swag/cmd/swag init -g cmd/api/main.go -o docs
```
