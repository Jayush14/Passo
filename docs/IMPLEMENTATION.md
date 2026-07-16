# Passo Backend Implementation Details

This document provides a comprehensive overview of the Passo backend implementation, covering the technologies used, architectural decisions, challenges faced, solutions applied, and a detailed API reference with example request/response payloads.

---

## Table of Contents
1. [System Overview](#system-overview)
2. [Technology Stack](#technology-stack)
3. [Project Structure](#project-structure)
4. [Core Features Implemented](#core-features-implemented)
5. [Challenges & Solutions](#challenges--solutions)
6. [API Reference](#api-reference)
   - [Authentication](#authentication)
   - [Logout](#logout)
   - [Protected Endpoints (placeholders)](#protected-endpoints-placeholders)
7. [Running the Service Locally](#running-the-service-locally)
8. [Future Work](#future-work)
9. [Conclusion](#conclusion)

---

## System Overview

Passo is a career‑growth platform that helps users create personalized learning roadmaps, track progress, and prepare for job interviews. The backend provides:

- User authentication (signup, login, refresh token rotation, logout)
- Secure storage of encrypted API keys (e.g., for Gemini AI)
- Placeholder endpoints for roadmap generation, progress tracking, chat, notifications, subscriptions, admin dashboard, and account management
- All protected routes are guarded by a JWT‑based authentication middleware

The service runs inside Docker containers orchestrated via `docker‑compose` and uses PostgreSQL for persistent storage.

---

## Technology Stack

| Category | Technology / Library | Version | Purpose |
|----------|----------------------|---------|---------|
| Language | Go | 1.25 | Main application language |
| Framework | Standard library `net/http` | — | HTTP server & routing |
| Router | Custom `http.ServeMux` (with wrapper for middleware) | — | Simple routing |
| Dependency Management | Go Modules | — | `go.mod`, `go.sum` |
| Database | PostgreSQL | 15‑alpine | Relational data store |
| DB Driver | `github.com/lib/pq` | v1.12.3 | PostgreSQL driver |
| Migrations | `golang-migrate/migrate/v4` | v4.15.2 | Schema versioning |
| Authentication | `golang-jwt/jwt/v5` | v5.2.0 | JWT creation & validation |
| Password Hashing | `golang.org/x/crypto/bcrypt` | v0.54.0 | Secure password storage |
| Configuration | Environment variables (via `os.Getenv`) | — | Configurable secrets & flags |
| Containerization | Docker | — | Build & runtime isolation |
| Orchestration | Docker Compose | — | Multi‑service setup |
| Logging | Standard `log` package | — | Simple stdout logging |
| Testing | `testing` package (planned) | — | Unit tests planned |

---

## Project Structure

```
backend/
├── Dockerfile                 # Multi‑stage build
├── go.mod                     # Go module definition
├── go.sum                     # Dependency lock‑file
├── .env                       # (Optional) local env vars
├── docker-compose.yml         # Service definitions
├── internal/
│   ├── config/                # Configuration loading
│   │   └── config.go
│   ├── db/                    # Database wrapper & migration runner
│   │   ├── db.go              # CRUD helpers for users & refresh tokens
│   │   └── migrate.go         # Apply pending migrations on startup
│   ├── auth/                  # Authentication handlers
│   │   └── handler.go         # Signup, login, refresh, logout, token helpers
│   ├── guardrail/             # Middleware (logging, auth)
│   │   └── middleware.go
│   ├── apikey/                # Placeholder for API‑key storage
│   │   └── handler.go
│   ├── ...                    # Other feature handlers (chat, roadmap, etc.)
├── cmd/
│   └── server/                # Application entry point
│       └── main.go
└── migrations/
    └── 001_create_base_tables.up.sql   # Initial schema
```

---

## Core Features Implemented

### 1. Docker & Orchestration
- **Multi‑stage Dockerfile**
  - Builder stage: `golang:1.25-alpine` → installs git, downloads modules, builds binary.
  - Final stage: `alpine:3.18` → copies binary & migrations, adds non‑root user, runs `/app/server`.
- **docker‑compose.yml**
  - `db`: PostgreSQL 15‑alpine, persistent volume, healthcheck (`pg_isready`).
  - `backend`: builds from `./backend`, depends on `db` with `condition: service_healthy`.
  - Exposes host ports: `8080:8080` (backend), `5433:5432` (DB).
  - Environment variables injected: `DB_URL`, `JWT_SECRET`, `ENCRYPTION_KEY`.

### 2. Configuration (`internal/config/config.go`)
- Loads from environment with sensible defaults:
  - `PORT` (default `8080`)
  - `ENV` (default `development`)
  - `DBURL` (default local `postgres://postgres:postgres@localhost:5432/passo?sslmode=disable`; overridden in compose)
  - `JWT_SECRET` (default `supersecret`)
  - `TOKEN_EXPIRY_HOURS` (default `24`)
  - `ENCRYPTION_KEY` (must be 32‑byte base32/hex string for AES‑256‑GCM)

### 3. Database Layer (`internal/db/db.go`)
- Thin wrapper around `*sql.DB` with constructor `New(dataSourceName string) (*DB, error)`.
- Functions:
  - `CreateUser(ctx, email, passwordHash) (int64, error)`
  - `GetUserByEmail(ctx, email) (int64, string, error)`
  - `SaveRefreshToken(ctx, userID int64, token string, expiresAt time.Time) error`
  - `GetUserIDFromRefreshToken(ctx, token string) (int64, bool, error)`
  - `DeleteRefreshToken(ctx, userID int64, token string) error`
- All refresh‑token operations store/retrieve a **SHA‑256 hash** of the token (never the plain token).

### 4. Migration Runner (`internal/db/migrate.go`)
- Uses `golang-migrate/migrate/v4` with source `file:///app/migrations` and postgres driver.
- Invoked on application startup; aborts on failure.

### 5. Authentication (`internal/auth/handler.go`)
#### Password Handling
```go
hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
// later
err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password))
```

#### JWT Helpers
```go
func generateJWT(userID int64, secret string, expiry time.Duration) (string, error) {
    claims := jwt.MapClaims{
        "user_id": userID,
        "exp":     time.Now().Add(expiry).Unix(),
        "iat":     time.Now().Unix(),
        "iss":     "passo",
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(secret))
}
```

#### Refresh Token Generation
```go
func generateRefreshToken() (string, error) {
    b := make([]byte, 32)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    return base64.URLEncoding.EncodeToString(b), nil
}
```

#### Endpoints
- **POST `/auth/signup`**
  - Validates email/password, hashes password, creates user, generates access & refresh tokens, stores refresh token hash (7‑day expiry), returns both tokens.
- **POST `/auth/login`**
  - Looks up user by email, compares hashes via bcrypt, issues tokens on success.
- **POST `/auth/refresh`**
  - Expects JSON `{ "refresh_token": "<token>" }`.
  - Hashes the token, looks for a matching, non‑expired row.
  - If valid:
    1. Deletes the used row (rotation).
    2. Generates new access token + new refresh token.
    3. Stores hash of new refresh token with fresh expiry.
    4. Returns `{ "access_token": "...", "refresh_token": "..."}`.
  - Invalid/expired/reused token → `401 Unauthorized` ("invalid or expired refresh token").
- **POST `/auth/logout`** *(new)*
  - Protected by JWT auth (`Authorization: Bearer <access_token>`).
  - Expects JSON `{ "refresh_token": "<token>" }`.
  - Retrieves `user_id` from JWT context.
  - Deletes the matching refresh‑token row (if present).
  - Returns `{ "message": "logged out" }`.
  - Existing access token remains valid until its natural expiry (short‑lived, e.g., 1 h).

#### Middleware (`internal/guardrail/middleware.go` & `auth_middleware.go`)
- `Middleware`: simple request logger (status, method, path, latency).
- `AuthMiddleware`:
  - Extracts `Authorization: Bearer <token>`.
  - Validates JWT signature & expiry using `JWTSecret`.
  - Extracts `user_id` claim, stores in context via `context.WithValue(r.Context(), userIDKey, userID)`.
  - Calls next handler.

### 6. Placeholder Feature Handlers
- Currently return static or empty JSON (e.g., `/apikey` returns `{"encrypted_key":""}`) to allow frontend integration while security/encryption logic is built.

### 7. Entrypoint (`cmd/server/main.go`)
- Loads config, connects to DB, runs migrations, sets DB instance for auth handlers (`auth.SetDB(dbConn)`), builds router (`server.NewRouter()`), starts HTTP server with timeouts.

---

## Challenges & Solutions

| # | Challenge | Description | Solution |
|---|-----------|-------------|----------|
| 1 | **Unused import build error** | `golang.org/x/crypto/bcrypt` imported in `db.go` but not used → Docker build failed. | Removed the import; bcrypt is used only in `auth/handler.go`. |
| 2 | **Migration path mismatch** | `migrate.NewWithDatabaseInstance` used `"./migrations"` locally but failed inside container where cwd is `/app`. | Changed to absolute path `"file:///app/migrations"` and ensured Dockerfile copies migrations into that location. |
| 3 | **Host port conflict** | Default PostgreSQL port 5432 already occupied on host. | Modified `docker-compose.yml` to map host `5433` → container `5432`. Updated `DB_URL` in compose to reference service `db` on internal port 5432. |
| 4 | **Refresh token reuse / one‑time use** | Early tests showed a refresh token could be reused multiple times, defeating rotation security. | Implemented deletion of the used token row (`DeleteRefreshToken`) upon successful validation before issuing new tokens. |
| 5 | **Token hash mismatch** | Storing raw token instead of its hash caused verification failures after container restarts. | Unified hashing logic: both `SaveRefreshToken` and `GetUserIDFromRefreshToken` compute SHA‑256 and store/compare the hex string. |
| 6 | **Environment variable naming inconsistency** | Local `.env` used `POSTGRES_USER`, etc., while compose also needed `DB_URL` override. | Standardised on providing a full `DB_URL` in compose; removed reliance on separate PG env vars for the app (they remain for the postgres image). |
| 7 | **Missing Content-Type handling** | Early `curl` tests returned `400 Bad Request` when `Content-Type` header omitted. | Ensured handlers decode JSON with `json.NewDecoder(r.Body)` and return `400` on malformed JSON; documented requirement in API spec. |
| 8 | **Graceful shutdown not implemented** | `http.Server` lacked timeout/context handling, risking lingering connections on stop. | Added `ReadTimeout`, `WriteTimeout`, `IdleTimeout` to `http.Server` in `main.go`. |
| 9 | **Logging noise** | Every request printed raw request line, cluttering logs during testing. | Kept simple logger; can be enhanced later with structured logging (e.g., Zap). |
|10| **Logout middleware context key mismatch** | The `AuthMiddleware` stored the user ID under an unexported key, causing the logout handler to fail to retrieve it. | Exported the key via `const userIDKey = "userID"` in `auth_middleware.go` and used it in `HandleLogout`. Added debug logging to trace values. |

---

## API Reference

### Common Details
- **Base URL**: `http://localhost:8080` (when running via `docker-compose`)
- **Content-Type**: `application/json` for request & response bodies
- **Authentication**: Protected endpoints require `Authorization: Bearer <access_token>`
- **Errors**: JSON error responses follow `{ "error": "<message>" }` with appropriate HTTP status codes (`400`, `401`, `4xx`, `500`)

---

### Authentication Endpoints

#### 1. Sign Up
- **URL**: `POST /auth/signup`
- **Request Body**:
  ```json
  {
    "email": "string (required, valid email)",
    "password": "string (required, min 6 chars)"
  }
  ```
- **Success Response** (`200 OK`):
  ```json
  {
    "access_token": "string (JWT)",
    "refresh_token": "string (URL‑safe base64, 32 bytes)"
  }
  ```
- **Error Responses**:
  - `400 Bad Request`: missing/invalid JSON or fields.
  - `500 Internal Server Error`: DB failure or token generation failure.

#### 2. Log In
- **URL**: `POST /auth/login`
- **Request Body**:
  ```json
  {
    "email": "string (required)",
    "password": "string (required)"
  }
  ```
- **Success Response** (`200 OK`):
  ```json
  {
    "access_token": "string (JWT)",
    "refresh_token": "string (URL‑safe base64)"
  }
  ```
- **Error Responses**:
  - `400 Bad Request`: malformed input.
  - `401 Unauthorized`: incorrect credentials (generic message to avoid user enumeration).
  - `500 Internal Server Error`: server failure.

#### 3. Refresh Token
- **URL**: `POST /auth/refresh`
- **Request Body**:
  ```json
  {
    "refresh_token": "string (the refresh token received from signup/login or previous refresh)"
  }
  ```
- **Success Response** (`200 OK`):
  ```json
  {
    "access_token": "string (new JWT)",
    "refresh_token": "string (new rotated refresh token)"
  }
  ```
- **Error Responses**:
  - `400 Bad Request`: missing `refresh_token`.
  - `401 Unauthorized`: token invalid, expired, already used (replay), or malformed.
  - `500 Internal Server Error`: internal failure (DB or token generation).

#### 4. Logout *(new)*
- **URL**: `POST /auth/logout`
- **Requires**: Valid `Authorization: Bearer <access_token>` header.
- **Request Body**:
  ```json
  {
    "refresh_token": "string (the refresh token to invalidate)"
  }
  ```
- **Success Response** (`200 OK`):
  ```json
  {
    "message": "logged out"
  }
  ```
- **Error Responses**:
  - `400 Bad Request`: missing/invalid JSON or missing `refresh_token`.
  - `401 Unauthorized`: missing or invalid `Authorization` header.
  - `500 Internal Server Error`: internal failure (DB error logged but not fatal to UX).

> **Note**: The logout endpoint invalidates the ability to **refresh** the access token. The current access token remains usable until its natural expiry (typically 1 hour). After that, the client cannot obtain a new access token without providing fresh credentials.

---

### Protected Endpoints (Placeholders)

> All endpoints below require a valid JWT in the `Authorization` header. They currently return stub data; real implementation is pending.

#### 5. Get Encrypted API Key (GET)
- **URL**: `GET /apikey`
- **Success Response** (`200 OK`):
  ```json
  {
    "encrypted_key": ""   // placeholder; will contain base64‑encoded ciphertext when encryption is implemented
  }
  ```

#### 6. Store Encrypted API Key (POST)
- **URL**: `POST /apikey`
- **Request Body**:
  ```json
  {
    "encrypted_key": "string (ciphertext produced by client‑side encryption using user's public key or server key)"
  }
  ```
- **Success Response** (`200 OK`):
  ```json
  {
    "message": "key stored"
  }
  ```

#### 7. Health Check (Public)
- **URL**: `GET /healthz`
- **Response** (`200 OK`):
  ```json
  {
    "status": "ok"
  }
  ```

*(Other protected endpoints – roadmap, progress, chat, notifications, subscriptions, admin, account – follow the same auth pattern and will be documented as they are implemented.)*

---

## Running the Service Locally

1. **Prerequisites**: Docker Engine ≥ 20.10, Docker Compose v2.
2. **Clone the repository** (if not already).
3. From the project root:
   ```bash
   cd /path/to/Passo
   docker compose up -d --build   # builds images and starts containers
   ```
4. Verify containers are healthy:
   ```bash
   docker compose ps
   ```
   You should see `passo_db` (healthy) and `passo_backend` (up).
5. Test the API with `curl` or a REST client (Thunder Client, Postman, etc.):
   ```bash
   # Sign up
   curl -X POST http://localhost:8080/auth/signup \
        -H "Content-Type: application/json" \
        -d '{"email":"test@example.com","password":"test123"}'

   # Login (if needed)
   curl -X POST http://localhost:8080/auth/login \
        -H "Content-Type: application/json" \
        -d '{"email":"test@example.com","password":"test123"}'

   # Use the returned access_token to call a protected endpoint
   curl -X GET http://localhost:8080/apikey \
        -H "Authorization: Bearer <access_token>"
   ```
6. To stop and clean up:
   ```bash
   docker compose down        # stops containers
   docker compose down -v    # also removes the named volume (PostgreSQL data)
   ```

---

## Future Work

| Area | Description |
|------|-------------|
| **API Key Encryption** | Implement AES‑256‑GCM using `ENCRYPTION_KEY`; store ciphertext in `api_keys.encrypted_key`. |
| **Roadmap Generation** | Integrate with Gemini API (using user‑stored encrypted key); validate and persist generated roadmap. |
| **Roadmap Re‑planning** | Implement logic for continuing vs. fresh plans, deadline selection, realism checks. |
| **Progress Tracking** | Implement `PATCH /progress/{task_id}` to update task completion; add summary endpoints. |
| **Chat & Notifications** | Store chat messages; implement VAPID key handling and push‑subscription storage; background worker for nudges. |
| **Subscription & Payments** | Distinguish BYOK vs subscription tiers using `plan_typed`; hook into payment provider (Stripe/LemonSqueezy) to activate house‑key tier. |
| **Admin Dashboard** | Replace dummy data with real aggregated queries (signups, engagement, AI reliability, monetization, support signals). |
| **Testing & CI** | Write unit tests for handlers, DB helpers, middleware; add integration tests with a test Postgres container; GitHub Actions workflow for lint, test, build, and push. |
| **Observability** | Expand `/healthz` to check DB connectivity and AI‑proxy reachability; add Prometheus metrics endpoint; structured logging (Zap/Zerolog). |
| **Security Hardening** | Consider adding a “pepper” (secret mixed into refresh‑token hash) for defense‑in‑depth against DB‑only leaks; implement rate limiting on auth endpoints; regular dependency audits. |
| **Documentation** | Generate OpenAPI spec (using `go-swagger` or `swaggo`) and host via Swagger UI; keep `PROGRESS.md` updated after each milestone. |

---

## Conclusion

The Passo backend now features a fully functional authentication system with **signup**, **login**, **refresh‑token rotation**, and a newly added **logout** endpoint. All core infrastructure (Docker, PostgreSQL, migrations, JWT middleware) is solid and ready for the next phases of feature development. With the logout endpoint in place, users can explicitly invalidate their refresh sessions, enhancing security and user control.

You can now proceed to implement the remaining features outlined in the *Future Work* section, using the patterns and conventions established in this document. Let me know if you need any further details or clarification! 

--- 

*Document last updated: 2026-07-15*