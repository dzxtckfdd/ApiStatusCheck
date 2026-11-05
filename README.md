# ApiStatusCheck

A distributed service availability and uptime monitoring platform engineered with a microservice architecture in Go, PostgreSQL, TimescaleDB, and containerized deployment.

---

## Architecture Overview

The system is partitioned into four decoupled services:

1. **UserManager** *(Implemented)*: Handles user registration, authentication, JWT issuance, and session management using Refresh Token Rotation.
2. **Catalog** *(In Development)*: Central state store for monitored endpoints, subscriptions, incident lifecycles, and historical availability metrics using TimescaleDB.
3. **APIChecker** *(Planned)*: High-throughput polling worker pool utilizing Go goroutines to inspect endpoints according to scheduled intervals and stream batched check results to Catalog.
4. **NotifierService** *(Planned)*: Event-driven dispatch engine for user alerts (Telegram Bot, Email) with notification throttling to prevent alert fatigue.

```
[Client]
    │
    ├── POST /auth/* ──────────────> [ UserManager ] (PostgreSQL)
    │                                       │
    │ (Bearer JWT: user_id)                 │
    ▼                                       ▼
[ Catalog ] <──── Batch Checks ───── [ APIChecker ]
(TimescaleDB)                          (Stateless, Concurrent Worker Pool)
    │
    └── Status Changed ────────────> [ NotifierService ] ──> Telegram / Email
```

---

## Current Status: UserManager

The `UserManager` service provides identity and access management for the platform.

### Technical Stack
* **Language:** Go 1.22+
* **Database:** PostgreSQL 15+
* **Driver:** `jackc/pgx/v5` via connection pooling (`pgxpool.Pool`)
* **Security & Cryptography:**
  * Password hashing: `golang.org/x/crypto/bcrypt`
  * Access tokens: `golang-jwt/jwt/v5` (HMAC-SHA256, short-lived)
  * Refresh tokens: High-entropy CSPRNG tokens (32-byte `crypto/rand`, stored as SHA-256 hashes)
  * Token Rotation: Refresh tokens are invalidated upon renewal to protect against replay attacks
* **Design Pattern:** Clean Layered Architecture (`handlers` -> `service` -> `repository` -> `postgres`) with explicit dependency injection.

---

## API Specifications

Base URL: `http://localhost:8001`

### 1. User Registration
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
  * `500 Internal Server Error` - Database or internal error.

---

### 2. User Authentication
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

---

### 3. Token Renewal
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

## Database Schema

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

---

## Development Roadmap

- [x] Global system architecture definition and database schema designs (PlantUML ER).
- [x] Core `UserManager` service implementation (Registration, Login, Refresh Token Rotation).
- [ ] `Catalog` service development:
  - PostgreSQL + TimescaleDB hypertable integration.
  - Continuous aggregates for historical uptime metrics (1h, 1d intervals).
  - Incident detection engine with flapping protection (threshold counters).
  - Endpoint and user subscription management.
- [ ] `APIChecker` service development:
  - In-memory cache with periodic synchronization via pull model.
  - Concurrency control via Go worker pool.
  - Batched ingestion pipeline to Catalog.
- [ ] `NotifierService` development:
  - Transport adapters (Telegram Bot API, SMTP).
  - 10-minute notification throttling mechanism.
- [ ] Production deployment orchestration via Docker Compose and Caddy reverse proxy with automated TLS.
