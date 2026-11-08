# ApiStatusCheck

A distributed service availability and uptime monitoring platform engineered with a microservice architecture in Go, PostgreSQL, TimescaleDB, and containerized deployment.

---

## Architecture Overview

The system is partitioned into four decoupled services:

1. **UserManager** *(Implemented)*: Handles user registration, authentication, JWT issuance, and session management using Refresh Token Rotation.
2. **Catalog** *(Implemented)*: Central state store for monitored endpoints, subscriptions, live state cache, incident lifecycles (flapping protection), and historical availability metrics using TimescaleDB continuous aggregates.
3. **APIChecker** *(Planned)*: High-throughput polling worker pool utilizing Go goroutines to inspect endpoints according to scheduled intervals and stream batched check results to Catalog.
4. **NotifierService** *(Planned)*: Event-driven dispatch engine for user alerts (Telegram Bot, Email) with notification throttling to prevent alert fatigue.

```
[Client]
    │
    ├── POST /auth/* ──────────────> [ UserManager ] (PostgreSQL :8001)
    │                                       │
    │ (Bearer JWT: user_id)                 │
    ▼                                       ▼
[ Catalog ] <──── Batch Checks ───── [ APIChecker ]
(TimescaleDB :8002)                    (Stateless, Concurrent Worker Pool)
    │
    └── Status Changed ────────────> [ NotifierService ] ──> Telegram / Email
```

---

## 1. UserManager Service

The `UserManager` service provides identity and access management for the platform.

### Technical Stack
* **Language:** Go 1.22+
* **Port:** `:8001`
* **Database:** PostgreSQL 15+
* **Driver:** `jackc/pgx/v5` via connection pooling (`pgxpool.Pool`)
* **Security & Cryptography:**
  * Password hashing: `golang.org/x/crypto/bcrypt`
  * Access tokens: `golang-jwt/jwt/v5` (HMAC-SHA256, short-lived)
  * Refresh tokens: High-entropy CSPRNG tokens (32-byte `crypto/rand`, stored as SHA-256 hashes)
  * Token Rotation: Refresh tokens are invalidated upon renewal to protect against replay attacks
* **Design Pattern:** Clean Layered Architecture (`handlers` -> `service` -> `repository` -> `postgres`) with explicit dependency injection.

### API Specifications

Base URL: `http://localhost:8001`

#### 1.1 User Registration
* **Endpoint:** `POST /auth/register`
* **Content-Type:** `application/json`
* **Request Payload:**
  ```json
  {
    "login": "username",
    "password": "secret_password"
  }
  ```
* **Responses:**
  * `201 Created` - Account created successfully.
  * `409 Conflict` - Login is already taken.
  * `400 Bad Request` - Malformed JSON body.

#### 1.2 User Authentication
* **Endpoint:** `POST /auth/login`
* **Content-Type:** `application/json`
* **Request Payload:**
  ```json
  {
    "login": "username",
    "password": "secret_password"
  }
  ```
* **Responses:**
  * `200 OK`
    ```json
    {
      "access_token": "eyJhbGciOiJIUzI1NiIs...",
      "refresh_token": "7a8b9c0d1e2f3a4b..."
    }
    ```
  * `401 Unauthorized` - Invalid credentials.

#### 1.3 Token Renewal
* **Endpoint:** `POST /auth/refresh`
* **Content-Type:** `application/json`
* **Request Payload:**
  ```json
  {
    "refresh_token": "7a8b9c0d1e2f3a4b..."
  }
  ```
* **Responses:**
  * `200 OK`
    ```json
    {
      "access_token": "eyJhbGciOiJIUzI1NiIs...",
      "refresh_token": "9f8e7d6c5b4a3b2a..."
    }
    ```
  * `401 Unauthorized` - Token is expired, invalid, or revoked.

---

## 2. Catalog Service

The `Catalog` service acts as the central state store, access manager, metric aggregator, and incident detection engine.

### Technical Stack
* **Language:** Go 1.22+
* **Port:** `:8002`
* **Database:** PostgreSQL 15+ with **TimescaleDB** extension
* **Driver:** `jackc/pgx/v5` with `pgx.Batch` high-performance batch ingestion
* **Features:**
  * **Resource Deduplication:** Automatically links users to existing URLs to eliminate redundant HTTP probes.
  * **Flapping Mitigation:** Threshold-based incident state machine (opens incident after $N$ failures; resolves after $N$ successes).
  * **Continuous Aggregates:** Precomputed hourly (`checks_hourly`) and daily (`checks_daily`) rollups with auto-refresh policies.
  * **Decoupled Notification Dispatch:** Non-blocking async dispatch to `NotifierService` with bounded HTTP client timeouts.
  * **Dual Authentication:** Strict JWT validation for private management, optional JWT parsing for shared metric endpoints.

### API Specifications

Base URL: `http://localhost:8002`

#### 2.1 Create Monitored Resource
* **Endpoint:** `POST /api/v1/resources`
* **Headers:** `Authorization: Bearer <access_token>`
* **Request Payload:**
  ```json
  {
    "url": "https://google.com",
    "name": "Google Search",
    "check_interval_sec": 60,
    "is_public": false
  }
  ```
* **Responses:**
  * `201 Created`
    ```json
    {
      "id": "54309482-4dae-470f-b16e-e058518fc24c",
      "url": "https://google.com",
      "name": "Google Search",
      "check_interval_sec": 60,
      "is_active": true,
      "is_public": false
    }
    ```
  * `400 Bad Request` - Malformed body or invalid URL scheme/host.
  * `401 Unauthorized` - Missing or invalid Bearer token.

#### 2.2 Get User Subscribed Resources
* **Endpoint:** `GET /api/v1/resources`
* **Headers:** `Authorization: Bearer <access_token>`
* **Responses:**
  * `200 OK`
    ```json
    [
      {
        "id": "54309482-4dae-470f-b16e-e058518fc24c",
        "url": "https://google.com",
        "name": "Google Search",
        "check_interval_sec": 60,
        "is_active": true,
        "is_up": true,
        "last_checked_at": "2026-11-08T23:55:00Z"
      }
    ]
    ```

#### 2.3 Get Public Resources Catalog
* **Endpoint:** `GET /api/v1/resources/public`
* **Headers:** None (Anonymous access)
* **Responses:**
  * `200 OK` (Array of active public resources with live status)

#### 2.4 Get Aggregated Availability Metrics
* **Endpoint:** `POST /api/v1/resources/metrics`
* **Headers:** `Authorization: Bearer <access_token>` *(Required for private targets, optional for public)*
* **Request Payload:**
  ```json
  {
    "resource_id": "54309482-4dae-470f-b16e-e058518fc24c",
    "from_period": "2026-11-01T00:00:00Z",
    "to_period": "2026-11-09T00:00:00Z",
    "frequency": "hourly"
  }
  ```
* **Responses:**
  * `200 OK`
    ```json
    {
      "resource_id": "54309482-4dae-470f-b16e-e058518fc24c",
      "from_period": "2026-11-01T00:00:00Z",
      "to_period": "2026-11-09T00:00:00Z",
      "frequency": "hourly",
      "uptime_percentage": 99.85,
      "avg_latency_ms": 142.4,
      "min_latency_ms": 89,
      "max_latency_ms": 1240
    }
    ```
  * `403 Forbidden` - Target is private and caller is not subscribed.
  * `404 Not Found` - Resource does not exist.

#### 2.5 Worker Polling Endpoint (Internal)
* **Endpoint:** `GET /internal/resources`
* **Headers:** `X-Internal-Secret: <secret>`
* **Responses:**
  * `200 OK` - Array of all active target endpoints and check intervals for `APIChecker`.

#### 2.6 Batch Checks Ingestion (Internal)
* **Endpoint:** `POST /internal/checks`
* **Headers:** `X-Internal-Secret: <secret>`
* **Request Payload:**
  ```json
  {
    "results": [
      {
        "resource_id": "54309482-4dae-470f-b16e-e058518fc24c",
        "checked_at": "2026-11-08T23:55:00Z",
        "status_code": 200,
        "latency_ms": 125,
        "is_up": true,
        "error_msg": null
      }
    ]
  }
  ```
* **Responses:**
  * `202 Accepted` - Check batch queued, state updated, metrics stored.

---

## Database Schemas

### UserManager Database (`users_db`)
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

### Catalog Database (`catalog_db`)
```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "timescaledb";

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

CREATE TABLE IF NOT EXISTS subscriptions (
    user_id UUID NOT NULL,
    resource_id UUID NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, resource_id)
);

CREATE TABLE IF NOT EXISTS resource_state (
    resource_id UUID PRIMARY KEY REFERENCES resources(id) ON DELETE CASCADE,
    is_up BOOLEAN NOT NULL DEFAULT true,
    consecutive_failures INT NOT NULL DEFAULT 0,
    consecutive_successes INT NOT NULL DEFAULT 0,
    current_incident_id UUID,
    last_checked_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS incidents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resource_id UUID NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at TIMESTAMPTZ,
    description TEXT
);

CREATE TABLE IF NOT EXISTS checks (
    resource_id UUID NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
    checked_at TIMESTAMPTZ NOT NULL,
    status_code INT,
    latency_ms INT,
    is_up BOOLEAN NOT NULL,
    error_msg TEXT
);

SELECT create_hypertable('checks', 'checked_at', if_not_exists => TRUE);

-- Continuous Aggregates
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

## Development Roadmap

- [x] Global system architecture definition and database schema designs (PlantUML ER).
- [x] Core `UserManager` service implementation (Registration, Login, Refresh Token Rotation).
- [x] `Catalog` service development:
  - [x] PostgreSQL + TimescaleDB hypertable integration.
  - [x] Continuous aggregates for historical uptime metrics (1h, 1d intervals).
  - [x] Incident detection engine with flapping protection (threshold counters).
  - [x] Endpoint and user subscription management with URL deduplication.
  - [x] Live state querying (`is_up`, `last_checked_at`) via left join.
  - [x] Decoupled async notification publisher.
- [ ] `APIChecker` service development:
  - In-memory cache with periodic synchronization via pull model.
  - Concurrency control via Go worker pool.
  - Batched ingestion pipeline to Catalog.
- [ ] `NotifierService` development:
  - Transport adapters (Telegram Bot API, SMTP).
  - 10-minute notification throttling mechanism.
- [ ] Production deployment orchestration via Docker Compose and Caddy reverse proxy with automated TLS.
