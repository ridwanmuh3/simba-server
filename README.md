# SIMBA API

> **Sistem Informasi Manajemen Anggaran dan Barang** — Backend API untuk pengelolaan stok, keuangan, dan invoice koperasi.

[![Go Version](https://img.shields.io/badge/Go-1.25.5-blue.svg)](https://golang.org)
[![Fiber](https://img.shields.io/badge/Fiber-v2.52-purple.svg)](https://gofiber.io)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-18-blue.svg)](https://postgresql.org)
[![Tests](https://img.shields.io/badge/Tests-Passing-green.svg)]()

---

## 📋 Overview

SIMBA (Sistem Informasi Manajemen Anggaran dan Barang) adalah backend REST API untuk mengelola:
- **Inventory Management** — Bahan baku, stok masuk/keluar, opname, MAC (Moving Average Cost)
- **Financial Tracking** — Pemasukan/pengeluaran harian, 10-harian, bulanan
- **Invoice Generation** — PDF invoice dengan template A4, nomor urut otomatis
- **Multi-Tenancy** — Dapur (kitchen) sebagai tenant, RBAC (Super Admin / Admin)
- **Real-time Notifications** — WebSocket untuk low-stock alerts, activity feed
- **Audit Trail** — Activity log permanen untuk compliance

---

## 🏗 Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                            SIMBA API                                     │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌──────────┐    ┌──────────────┐    ┌──────────────────┐              │
│  │  HTTP    │───▶│  Middleware  │───▶│    Handlers      │              │
│  │  Request │    │  (Auth, CSRF, │    │ (User, Item,     │              │
│  └──────────┘    │   RateLimit,  │    │  Stock, Invoice, │              │
│                  │   RequestID)   │    │  Finance, etc)   │              │
│                  └──────────────┘    └────────┬─────────┘              │
│                                               │                        │
│                                               ▼                        │
│                                    ┌──────────────────┐               │
│                                    │    Services      │               │
│                                    │ (Business Logic) │               │
│                                    │ User, Item,      │               │
│                                    │ Stock, Invoice,  │               │
│                                    │ Finance, Notif   │               │
│                                    └────────┬─────────┘               │
│                                             │                         │
│                    ┌────────────────────────┼────────────────────┐   │
│                    ▼                        ▼                    ▼   │
│            ┌──────────────┐        ┌──────────────┐      ┌────────────┐
│            │ Repository   │        │ Repository   │      │ Repository │
│            │ (Data Access)│        │ (Data Access)│      │ (Data Access)│
│            └──────┬───────┘        └──────┬───────┘      └──────┬─────┘   │
│                   │                       │                    │          │
│                   ▼                       ▼                    ▼          │
│            ┌──────────────────────────────────────────────────────────┐  │
│            │              PostgreSQL (GORM)                           │  │
│            │  Users │ Items │ StockTracks │ Invoices │ Finances ...  │  │
│            └──────────────────────────────────────────────────────────┘  │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘

Real-time Layer:
┌─────────────────────────────────────────────────────────────────────────┐
│  WebSocket Hub ◀── ActivityEmitter ──▶ LowStockChecker ──▶ Notification │
│       │                                           Service                │
│       ▼                                           ▼                      │
│  Connected Clients (per-dapur rooms)           Email (SMTP)             │
└─────────────────────────────────────────────────────────────────────────┘
```

### Clean Architecture Layers

| Layer | Responsibility | Key Files |
|-------|---------------|-----------|
| **Delivery** | HTTP handlers, middleware, routing | `internal/delivery/` |
| **Service** | Business logic, transactions | `internal/service/` |
| **Repository** | Data access, GORM queries | `internal/repository/` |
| **Entity** | Database models | `internal/entity/` |
| **Model** | DTOs, Request/Response | `internal/model/` |

### Key Design Decisions

- **Interface-based DI** — Services depend on repository interfaces for testability
- **Transaction-per-request** — Each service method manages its own DB transaction
- **Row-level locking** — `SELECT FOR UPDATE` on critical sections to prevent race conditions
- **MAC recalculation** — Moving Average Cost rebuilt on every stock mutation (IN/OUT/EDIT/DELETE)
- **Multi-tenancy** — All queries scoped to `dapur_id` (kitchen/tenant)
- **Async activity emission** — Activity logs + WebSocket broadcasts via buffered channel + worker pool

---

## 🚀 Quick Start

### Prerequisites

- Go 1.25+
- PostgreSQL 18
- Redis (optional, for future session store)

### Configuration

```bash
# Copy example env
cp .example.env .env

# Edit .env with your database credentials
```

Required environment variables:
```env
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=simba
DB_HOST=localhost
DB_PORT=5432
APP_PORT=3000
APP_MODE=dev
APP_RESET_SECRET=your-secret-key
APP_CORS_ALLOWED_ORIGINS=http://localhost:5173
```

### Run

```bash
# Install dependencies
go mod download

# Run database migrations (auto-migrate on startup)
go run cmd/app/main.go

# Server starts on http://localhost:3000
```

### Docker

```bash
# Build and run with docker-compose
docker-compose up -d
```

---

## 📚 API Documentation

### Authentication

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/auth/login` | Login (rate limited: 5 req/min) |
| POST | `/api/auth/refresh` | Refresh access token |
| DELETE | `/api/auth/logout` | Logout & invalidate tokens |
| GET | `/api/auth/_current` | Get current user profile |
| POST | `/api/auth/csrf` | Get CSRF token |

### Items & Stock

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/items` | Create item |
| GET | `/api/items` | List items (paginated, searchable) |
| PUT | `/api/items/:id` | Update item (recalculates MAC) |
| DELETE | `/api/items/:id` | Delete item & stock tracks |
| POST | `/api/items/:id/stocks` | Add stock IN/OUT |
| PATCH | `/api/items/:id/stocks/:stock_id` | Edit stock entry |
| DELETE | `/api/items/:id/stocks/:stock_id` | Delete stock entry |
| GET | `/api/items/stocks` | List stock tracks |
| GET | `/api/items/stocks/opname` | Stock opname summary |

### Invoices

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/items/invoices` | Generate invoice PDF |
| GET | `/api/items/invoices` | List invoices |
| GET | `/api/items/invoices/:id` | Invoice detail |
| GET | `/api/items/invoices/:id/pdf` | Download PDF |
| PATCH | `/api/items/invoices/:id` | Update invoice |
| DELETE | `/api/items/invoices/:id` | Delete invoice |

### Finance

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/finances` | Add financial record |
| GET | `/api/finances` | List finances |
| PUT | `/api/finances/:id` | Update finance |
| DELETE | `/api/finances/:id` | Delete finance |
| GET | `/api/finances/export` | Export all finances |

### Dashboard

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/dashboard` | Dashboard stats (items, stocks, budgets, activities) |

### Notifications (Real-time)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/notifications` | List notifications (paginated, filterable) |
| GET | `/api/notifications/unread-count` | Unread count for badge |
| PATCH | `/api/notifications/:id/read` | Mark as read |
| PATCH | `/api/notifications/read-all` | Mark all as read |
| DELETE | `/api/notifications/:id` | Delete notification |
| GET | `/ws` | WebSocket upgrade (real-time events) |

### WebSocket Events

```json
// Client → Server: Join room
{"action": "join", "dapur_id": 1}

// Server → Client: Activity event
{"type": "ACTIVITY", "message": "Stok bahan diperbarui", "data": {...}}

// Server → Client: Low stock alert
{"type": "LOW_STOCK", "message": "Stok Beras rendah", "data": {"item_id": "MBG-BHN-0001", "current_stock": 2.5, "threshold": 5}}
```

---

## 🧪 Testing

```bash
# Unit tests (no DB required)
go test ./internal/util/ ./internal/service/ ./internal/delivery/middleware/ ./internal/model/converter/ -v

# Integration tests (requires PostgreSQL)
export TEST_DB_HOST=localhost
export TEST_DB_USER=postgres
export TEST_DB_PASSWORD=postgres
export TEST_DB_NAME=simba_test
export TEST_DB_PORT=5432
go test ./test/integration/... -v
```

---

## 📦 Project Structure

```
simba-server/
├── cmd/app/main.go              # Entry point
├── internal/
│   ├── config/                  # Configuration (Viper, GORM, Fiber, Logger)
│   ├── delivery/
│   │   ├── handler/             # HTTP handlers (User, Item, Stock, Invoice, etc.)
│   │   ├── middleware/          # Auth, CSRF, RBAC, RequestID
│   │   └── route/               # Route registration
│   ├── entity/                  # GORM models (User, Item, StockTracking, Invoice, etc.)
│   ├── model/                   # DTOs, Request/Response, Converters
│   ├── repository/              # Data access layer (interfaces + implementations)
│   ├── service/                 # Business logic (User, Item, Stock, Invoice, etc.)
│   │   ├── login_limiter.go     # Login rate limiter with cleanup
│   │   ├── activity_emitter.go  # Async activity log + WS broadcast
│   │   ├── low_stock_checker.go # Low stock threshold monitoring
│   │   └── notification_service.go
│   ├── transport/websocket/     # WebSocket hub, client management
│   └── util/                    # Utilities (PDF, validation, formatting)
├── test/integration/            # Integration tests
└── .example.env                 # Example environment config
```

---

## 🛠 Tech Stack

| Category | Technology |
|----------|------------|
| Language | Go 1.25.5 |
| Web Framework | Fiber v2.52 |
| Database | PostgreSQL 18 + GORM |
| JSON | bytedance/sonic |
| Validation | go-playground/validator v10 |
| Logging | Uber Zap |
| Config | Spf13 Viper |
| PDF | codeberg.org/go-pdf/fpdf |
| WebSocket | gofiber/contrib/websocket |
| Password Hash | bcrypt (golang.org/x/crypto) |
| UUID | google/uuid |

---

## 🔐 Security

- **Authentication** — JWT-like tokens in HttpOnly cookies (access + refresh rotation)
- **CSRF Protection** — Double-submit cookie pattern
- **Rate Limiting** — Login endpoint: 5 attempts/minute per IP+username
- **SQL Injection** — GORM parameterized queries
- **XSS Prevention** — JSON responses, HttpOnly cookies, CSP headers
- **CORS** — Configurable allowed origins
- **Helmet** — Security headers

---

## 📈 Observability

- **Request ID** — `X-Request-ID` header propagated through all layers
- **Structured Logging** — Zap JSON logs with request context
- **Health Check** — `/health` returns DB connectivity + uptime
- **Graceful Shutdown** — SIGINT/SIGTERM handling with 10s drain timeout

---

## 🗄 Database Migration

Uses **Atlas HCL** for schema management:

```bash
# Generate migration from schema changes
atlas migrate diff --env local

# Apply migrations
atlas migrate apply --env local
```

Schema versioning tracked in `atlas.sum`.

---

## 📝 License

Internal project — SATU TEKNOLOGI INDONESIA