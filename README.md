# 🌐 ApiStatusCheck

<p align="center">
  <strong>A high-throughput, distributed uptime and service availability monitoring platform built with Go, PostgreSQL, TimescaleDB, and a concurrent Worker Pool architecture.</strong>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go version" />
  <img src="https://img.shields.io/badge/PostgreSQL-16-316192?style=for-the-badge&logo=postgresql&logoColor=white" alt="PostgreSQL" />
  <img src="https://img.shields.io/badge/TimescaleDB-Continuous_Aggregates-FDB515?style=for-the-badge&logo=timescale&logoColor=white" alt="TimescaleDB" />
  <img src="https://img.shields.io/badge/Architecture-Microservices-blueviolet?style=for-the-badge" alt="Microservices" />
  <img src="https://img.shields.io/badge/License-MIT-green.svg?style=for-the-badge" alt="License" />
</p>

---

## 📑 Table of Contents

- [Overview](#-overview)
- [System Architecture](#-system-architecture)
- [Microservices Breakdown](#-microservices-breakdown)
  - [1. UserManager Service (:8001)](#1-usermanager-service-8001)
  - [2. Catalog Service (:8002)](#2-catalog-service-8002)
  - [3. APIChecker Engine (:8003)](#3-apichecker-engine-8003)
  - [4. NotifierService (:8004)](#4-notifierservice-8004-planned)
- [Database Architectures](#-database-architectures)
  - [UserManager Schema (`users_db`)](#usermanager-schema-users_db)
  - [Catalog & TimescaleDB Schema (`catalogService`)](#catalog--timescaledb-schema-catalogservice)
- [Quick Start & Local Setup](#-quick-start--local-setup)
- [Development Roadmap](#-development-roadmap)
- [License](#-license)

---

## 🔭 Overview

`ApiStatusCheck` is an open-source, resilient uptime monitoring platform modeled after enterprise status-monitoring systems (such as *UptimeRobot* and *BetterStack*). It is designed to track hundreds of HTTP/HTTPS endpoints concurrently, detect outages while suppressing false-positive flapping, calculate time-bucketed availability metrics (1h/1d rollups), and alert subscribers.

### Key Capabilities:
- **Zero-Polling Stateless Checker**: Polling workers operate as an autonomous concurrent pipeline (Fan-Out / Fan-In) that pulls task targets and flushes batch results.
- **TimescaleDB Continuous Aggregates**: Automated rolling rollups (`checks_hourly`, `checks_daily`) computed in the database engine without costly on-the-fly table scans.
- **Flapping Protection**: Threshold-based incident lifecycle management (opens only after $N$ sequential failures; auto-resolves after $N$ sequential successes).
- **Target Deduplication**: Identical target URLs are polled exactly once regardless of how many users subscribe to them.
- **CSPRNG Refresh Token Rotation**: Secure, replay-resistant session management with short-lived HMAC-SHA256 JWTs.

---

## 🏛 System Architecture

```
[ Client / Web UI ]
       │
       ├── POST /auth/* ───────────────► [ UserManager ] (:8001)
       │                                  (Auth, JWT Issuance, Token Rotation)
       │
       ├── GET,POST /api/v1/resources ─► [ CatalogService ] (:8002)
       │    (Bearer JWT)                  (Metadata, TimescaleDB Hypertables, Aggregates)
       │                                         ▲
       │                                         │
       │                         GET /internal/resources (Sync active targets)
       │                         POST /internal/checks   (Batch ingestion)
       │                                         │
       │                                  [ APIChecker ] (:8003)
       │                                  (Concurrent Goroutine Worker Pool)
       │                                         │
       │                                         ▼  HTTP Probes
       │                                   [ Target APIs ]
       │
       └───────────────────────────────► [ NotifierService ] (:8004 - Planned)
                                          (Alert Throttling, Telegram Bot, SMTP)
```

---

## 🧩 Microservices Breakdown

| Service | Port | Database | Primary Responsibility | Status |
|---|---|---|---|---|
| **UserManager** | `:8001` | PostgreSQL 16 | Identity, registration, bcrypt hashing, JWT issuance & rotation | **Done** |
| **Catalog** | `:8002` | TimescaleDB (PG16) | Target endpoints CRUD, subscriptions, live state, incident detection, metric rollups | **Done** |
| **APIChecker** | `:8003` | *Stateless* | In-memory target scheduler, Fan-Out worker pool, batch collector | **Done** |
| **NotifierService** | `:8004` | PostgreSQL 16 | Alert dispatching (Telegram, Email) with 10-minute throttling | *Planned* |

---

### 1. UserManager Service (`:8001`)

Responsible for authentication, authorization, and secure session lifecycle.

* **Stack:** Go 1.22+, `pgx/v5` connection pool, `golang.org/x/crypto/bcrypt`, `golang-jwt/jwt/v5`.
* **Security:** Cryptographic CSPRNG 32-byte tokens for refresh tokens (stored as SHA-256 hashes). Automatic revocation upon rotation.

#### API Endpoints:

| Method | Endpoint | Auth | Description |
|---|---|---|---|
| `POST` | `/auth/register` | None | Create new user account |
| `POST` | `/auth/login` | None | Authenticate with credentials, returns Access & Refresh tokens |
| `POST` | `/auth/refresh` | None | Exchange active refresh token for a fresh token pair |

<details>
<summary><strong>View Request / Response Payloads</strong></summary>

##### Registration & Login:
```http
POST /auth/login HTTP/1.1
Content-Type: application/json

{
  "login": "developer",
  "password": "SecurePassword123!"
}
```

##### Success Response (`200 OK`):
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "8f3b2a1c9e4d5f6a..."
}
```
</details>

---

### 2. Catalog Service (`:8002`)

The central state authority and analytical engine.

* **Stack:** Go 1.22+, TimescaleDB, `pgx/v5` with `pgx.Batch`.
* **Authentication:** Dual middleware:
  * Strict `AuthMiddleware`: Requires valid Bearer JWT.
  * `OptionalAuthMiddleware`: Allows anonymous access to public targets while extracting user identity for private targets.

#### API Endpoints:

| Method | Endpoint | Auth | Description |
|---|---|---|---|
| `POST` | `/api/v1/resources` | Bearer JWT | Create target endpoint (with URL deduplication) |
| `GET` | `/api/v1/resources` | Bearer JWT | List caller's subscribed resources with live `is_up` state |
| `GET` | `/api/v1/resources/public` | Public | List global public endpoints and live health status |
| `POST` | `/api/v1/resources/metrics` | Optional JWT | Query aggregated uptime percentage and min/max/avg latency |
| `GET` | `/internal/resources` | `X-Internal-Secret` | Polled by `APIChecker` to sync active target pool |
| `POST` | `/internal/checks` | `X-Internal-Secret` | Ingest batched check probe results from `APIChecker` |

<details>
<summary><strong>View Metrics Query Payload & Example Output</strong></summary>

```http
POST /api/v1/resources/metrics HTTP/1.1
Content-Type: application/json
Authorization: Bearer <access_token>

{
  "resource_id": "54309482-4dae-470f-b16e-e058518fc24c",
  "from_period": "2026-11-01T00:00:00Z",
  "to_period": "2026-11-15T00:00:00Z",
  "frequency": "hourly"
}
```

##### Output (`200 OK`):
```json
{
  "resource_id": "54309482-4dae-470f-b16e-e058518fc24c",
  "from_period": "2026-11-01T00:00:00Z",
  "to_period": "2026-11-15T00:00:00Z",
  "frequency": "hourly",
  "uptime_percentage": 99.85,
  "avg_latency_ms": 112.4,
  "min_latency_ms": 18,
  "max_latency_ms": 480
}
```
</details>

---

### 3. APIChecker Engine (`:8003`)

A high-performance, stateless daemon engineered for distributed probe execution.

```
                  ┌──────────────────────────────────────────────┐
                  │                 APIChecker                   │
                  │                                              │
                  │   [Scheduler] (In-memory targets cache)     │
                  │        │                                     │
                  │        ▼ jobsChan (buffered channel)         │
                  │   ┌────────────────────────────────┐         │
                  │   │  WorkerPool (Fan-Out: 20 goroutines)     │
                  │   │   Worker 1  ──► HTTP GET/HEAD  │         │
                  │   │   Worker 2  ──► HTTP GET/HEAD  │         │
                  │   │   ...                          │         │
                  │   │   Worker 20 ──► HTTP GET/HEAD  │         │
                  │   └────────────────────────────────┘         │
                  │        │                                     │
                  │        ▼ resultsChan (buffered channel)      │
                  │   [Batcher] (Buffer: 50 items OR 2s timeout) │
                  └────────│─────────────────────────────────────┘
                           ▼ POST /internal/checks
                    [CatalogService]
```

* **Non-Blocking Architecture:** Workers drain targets from a buffered `jobsChan`. Results flow through `resultsChan` into an asynchronous `Batcher`.
* **Zero Socket Leaks:** Reusable `http.Client` with `Transport` keep-alives, `io.Copy(io.Discard, resp.Body)`, and bounded redirect caps.
* **Network Fault Tolerance:** DNS resolution failures, connection refused, and timeouts are treated as valid failure events (`is_up: false`, `status_code: 0`, full error string).

---

## 🗄 Database Architectures

### UserManager Schema (`users_db`)
```sql
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    login VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_users_login ON users(login);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_hash ON refresh_tokens(token_hash);
```

### Catalog & TimescaleDB Schema (`catalogService`)
```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "timescaledb";

-- 1. Monitored Target Endpoints
CREATE TABLE IF NOT EXISTS resources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    url TEXT NOT NULL,
    name VARCHAR(255) NOT NULL,
    check_interval_sec INT NOT NULL DEFAULT 60,
    is_active BOOLEAN NOT NULL DEFAULT true,
    is_public BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_resources_url ON resources(url);

-- 2. User Subscriptions (Access Control mapping)
CREATE TABLE IF NOT EXISTS subscriptions (
    user_id UUID NOT NULL,
    resource_id UUID NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, resource_id)
);

-- 3. Live State Cache & Flapping Thresholds
CREATE TABLE IF NOT EXISTS resource_state (
    resource_id UUID PRIMARY KEY REFERENCES resources(id) ON DELETE CASCADE,
    is_up BOOLEAN NOT NULL DEFAULT true,
    consecutive_failures INT NOT NULL DEFAULT 0,
    consecutive_successes INT NOT NULL DEFAULT 0,
    current_incident_id UUID,
    last_checked_at TIMESTAMPTZ
);

-- 4. Incident Records
CREATE TABLE IF NOT EXISTS incidents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resource_id UUID NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at TIMESTAMPTZ,
    description TEXT
);

-- 5. Time-Series Raw Probes Hypertable
CREATE TABLE IF NOT EXISTS checks (
    resource_id UUID NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
    checked_at TIMESTAMPTZ NOT NULL,
    status_code INT,
    latency_ms INT,
    is_up BOOLEAN NOT NULL,
    error_msg TEXT
);

SELECT create_hypertable('checks', 'checked_at', if_not_exists => TRUE);
CREATE INDEX IF NOT EXISTS idx_checks_resource_time ON checks(resource_id, checked_at DESC);

-- 6. Continuous Aggregate: Hourly Rollup
CREATE MATERIALIZED VIEW IF NOT EXISTS checks_hourly
WITH (timescaledb.continuous) AS
SELECT
    resource_id,
    time_bucket('1 hour', checked_at) AS bucket,
    COUNT(*) AS total_checks,
    COUNT(*) FILTER (WHERE is_up) AS up_checks,
    AVG(latency_ms) AS avg_latency_ms,
    MIN(latency_ms) AS min_latency_ms,
    MAX(latency_ms) AS max_latency_ms
FROM checks
GROUP BY resource_id, bucket
WITH NO DATA;

SELECT add_continuous_aggregate_policy('checks_hourly',
    start_offset => INTERVAL '1 day',
    end_offset   => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour',
    if_not_exists => TRUE
);

-- 7. Continuous Aggregate: Daily Rollup
CREATE MATERIALIZED VIEW IF NOT EXISTS checks_daily
WITH (timescaledb.continuous) AS
SELECT
    resource_id,
    time_bucket('1 day', bucket) AS bucket,
    SUM(total_checks) AS total_checks,
    SUM(up_checks) AS up_checks,
    AVG(avg_latency_ms) AS avg_latency_ms,
    MIN(min_latency_ms) AS min_latency_ms,
    MAX(max_latency_ms) AS max_latency_ms
FROM checks_hourly
GROUP BY resource_id, time_bucket('1 day', bucket)
WITH NO DATA;

SELECT add_continuous_aggregate_policy('checks_daily',
    start_offset => INTERVAL '7 days',
    end_offset   => INTERVAL '1 day',
    schedule_interval => INTERVAL '1 day',
    if_not_exists => TRUE
);
```

---

## 🚀 Quick Start & Local Setup

### 1. Prerequisites
- **Go**: 1.22 or higher
- **Docker**: For running PostgreSQL and TimescaleDB containers

### 2. Launch Databases

Start TimescaleDB container for `catalogService`:
```bash
docker run -d --name catalog-timescale \
  -p 5434:5432 \
  -e POSTGRES_PASSWORD=1234 \
  -e POSTGRES_DB=catalogService \
  timescale/timescaledb:latest-pg16
```

### 3. Run Microservices

Open 3 terminal sessions:

```bash
# Terminal 1: UserManager
cd userManager
go run cmd/main.go

# Terminal 2: Catalog Service
cd catalogService
go run cmd/main.go

# Terminal 3: APIChecker Engine
cd apiChecker
go run cmd/main.go
```

The system is now fully autonomous: `apiChecker` will query `catalogService`, pick up active resources, probe them concurrently, and push metrics to TimescaleDB.

---

## 🗺 Development Roadmap

- [x] Global system architecture definition and database schema designs.
- [x] **UserManager Service**:
  - [x] User registration & bcrypt credential hashing.
  - [x] HMAC-SHA256 JWT generation with short TTL.
  - [x] CSPRNG Refresh Token Rotation.
- [x] **Catalog Service**:
  - [x] PostgreSQL + TimescaleDB hypertable partition setup.
  - [x] Continuous aggregates (`checks_hourly`, `checks_daily`) with automated refresh policies.
  - [x] Flapping-resistant incident state machine.
  - [x] Access control and target deduplication.
  - [x] Live state querying (`is_up`, `last_checked_at`).
- [x] **APIChecker Engine**:
  - [x] Non-blocking in-memory scheduler with periodic target synchronization.
  - [x] Concurrency control via Go worker pool (Fan-Out).
  - [x] Dynamic batching collector with flush timeouts (Fan-In).
  - [x] Graceful shutdown handling.
- [ ] **Web Dashboard (Frontend)**:
  - [ ] Public status showcase with real-time health badges.
  - [ ] User authentication and dashboard for managing target endpoints.
  - [ ] Interactive latency and uptime charts (24h/7d).
- [ ] **NotifierService**:
  - [ ] Telegram Bot notification transport.
  - [ ] SMTP email transport.
  - [ ] 10-minute alert throttling.

---

## 📄 License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.
