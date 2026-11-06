# WebTester Architecture

## High-Level Layered Structure

```
┌─────────────────────────────────────────────────────────────────┐
│                         HTTP Request                            │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  pkg/middleware (Recover, AccessLog)                            │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  internal/routes (Chi router, route groups)                     │
│   ├─ /healthz (public)                                          │
│   ├─ /api/v1/auth/* (public: status, setup, login, refresh)    │
│   └─ /api/v1/* (protected by auth.Middleware.Handler)          │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  internal/handlers (HTTP → DTO validation → Service call)       │
│   ├─ AuthHandler    (Status, Setup, Login, Refresh, Logout, Me)│
│   ├─ ScanHandler    (List, Create, Get, Findings)              │
│   └─ LoadTestHandler(List, Create, Get, Cancel)                │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  internal/services (Business logic, orchestration)              │
│   ├─ AuthService   (user store + redis token store + JWT)      │
│   ├─ ScanService   (scan store + ZAP client + check engine)    │
│   └─ LoadTestService (loadtest store + concurrent runner)      │
└─────────────────────────────────────────────────────────────────┘
                              │
          ┌───────────────────┼───────────────────┐
          ▼                   ▼                   ▼
┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
│ internal/stores │ │ internal/clients│ │  internal/auth  │
│                 │ │                 │ │                 │
│ • ScanStore     │ │ • ZapClient     │ │ • UserStore     │
│ • LoadTestStore │ │                 │ │   (SQLite)      │
│                 │ │                 │ │ • RedisStore    │
│                 │ │                 │ │   (tokens)      │
│                 │ │                 │ │ • Middleware    │
└─────────────────┘ └─────────────────┘ └─────────────────┘
          │
          ▼
┌─────────────────────────────────────────────────────────────────┐
│  SQLite (webtester.db)  +  Redis (localhost:6379)              │
└─────────────────────────────────────────────────────────────────┘
```

---

## Package Import Rules

| Package | May Import |
| --- | --- |
| `cmd/app` | `internal/*`, `pkg/*`, `configs` |
| `internal/routes` | `internal/handlers`, `internal/services`, `internal/auth`, `pkg/middleware`, `web` |
| `internal/handlers` | `internal/services`, `internal/dtos`, `pkg/errors`, `pkg/response` |
| `internal/services` | `internal/stores`, `internal/clients`, `internal/auth`, `internal/dtos`, `pkg/errors` |
| `internal/auth` | `pkg/errors`, `pkg/response`, **NO** other internal packages |
| `internal/stores` | **NO** other internal packages (only `internal/dtos`, `modernc.org/sqlite`) |
| `internal/clients` | **NO** other internal packages |
| `internal/dtos` | **NO** other internal packages |
| `internal/checks` | **NO** other internal packages |
| `internal/loadtest` | **NO** other internal packages |
| `pkg/*` | **NO** internal packages (zero dependency on app) |

**Enforcement:** Go compiler prevents cycles. Lint with `go vet` and `golangci-lint`.

---

## Request Flow: Create Security Scan

```
POST /api/v1/scans
        │
        ▼
[AuthMiddleware] Validate JWT → ctx.Value("user") = *Claims
        │
        ▼
[ScanHandler.Create]
  - Decode JSON → dtos.CreateScanRequest
  - validator.Struct(req)
  - auth header validation (if provided)
  - svc.Create(ctx, req)
        │
        ▼
[ScanService.Create]
  - Build Scan domain + initial status "queued"
  - store.Create(ctx, params)
  - Return dtos.ScanResponse
  - Start background: scanSvc.Execute(scanID)
        │
        ▼
[ScanService.Execute] (async goroutine)
  - Update status "running"
  - Run passive checks (internal/checks/passive.go)
  - If type=api: parse OpenAPI, fuzz params
  - If type=full: launch ZAP docker container (clients/zap)
  - If BOLA URLs: run BOLA probe (checks/bola.go)
  - Persist findings via store.SaveFindings
  - Update status "completed" or "failed"
```

---

## Request Flow: Load Test

```
POST /api/v1/load-tests
        │
        ▼
[AuthMiddleware]
        │
        ▼
[LoadTestHandler.Create]
  - Validate dtos.CreateLoadTestRequest
  - Parse headers (Name: Value or JSON)
  - svc.Create(ctx, req)
        │
        ▼
[LoadTestService.Create]
  - Build LoadTest domain
  - store.Create(ctx)
  - Start goroutine: runner.Run(runCtx, store, config)
        │
        ▼
[internal/loadtest/runner.Run]
  - runCtx = context.WithTimeout(ctx, duration)
  - Producer: emits requests at target RPS into channel
  - Workers (concurrency N): read from channel, exec HTTP, collect results
  - Aggregator: updates stats, progress in store every second
  - On timeout/cancellation: cancel runCtx, drain in-flight, finalize
```

---

## Auth Flow: Login & Token Refresh

```
POST /api/v1/auth/login
        │
        ▼
[AuthHandler.Login]
  - Validate email/password
  - svc.Login(ctx, req)
        │
        ▼
[AuthService.Login]
  - userStore.GetByEmail(email)
  - bcrypt.CompareHashAndPassword
  - userStore.UpdateLastLogin
  - generateSession(user)
        │
        ▼
[generateSession]
  - GenerateAccessToken(userID, email) → JWT 15m
  - GenerateRefreshToken(userID) → JWT 7d
  - redisStore.SaveRefreshToken(userID, refreshToken, 7d TTL)
  - Return TokenResponse {access, refresh, user}
        │
        ▼
Client stores both tokens (localStorage)

────────────────────────────────────────────

POST /api/v1/auth/refresh
        │
        ▼
[AuthHandler.Refresh]
  - svc.Refresh(ctx, refreshToken)
        │
        ▼
[AuthService.Refresh]
  - ParseRefreshToken → userID
  - redisStore.GetRefreshToken(userID) == provided? else revoked
  - userStore.GetByID(userID)
  - generateSession(user) → NEW access + NEW refresh
  - Old refresh deleted, new saved (rotation)
```

---

## Auth Middleware Enforcement

```go
func (m *Middleware) Handler(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w, r) {
        token := extractToken(r)              // "Bearer <jwt>"
        blacklisted := m.store.IsTokenBlacklisted(ctx, token)
        claims, err := ParseAccessToken(token, m.tokenConfig)
        ctx = context.WithValue(r.Context(), UserContextKey, claims)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

Applied in `routes.New`:
```go
r.Group(func(r chi.Router) {
    r.Use(authMiddleware.Handler)  // ALL protected routes
    r.Post("/auth/logout", ah.Logout)
    r.Get("/auth/me", ah.Me)
    r.Get("/scans", sh.List)       // Protected
    ...
})
```

---

## Database Schema

### SQLite Tables

| Table | Purpose | Key Fields |
| --- | --- | --- |
| `scans` | Security scan records | id, target_url, scan_type, status, progress, created_at |
| `findings` | Vulnerability findings per scan | id, scan_id, rule_id, severity, url, evidence, cwe |
| `load_tests` | Load test runs | id, target_url, method, duration, rps, concurrency, status, stats |
| `users` | Admin accounts (auth) | id, email, password_hash, name, role, created_at |

Schema SQL embedded via `//go:embed schema.sql` in each store.

### Redis Keys

| Pattern | TTL | Value |
| --- | --- | --- |
| `refresh:<user_id>` | 7d | refresh token string |
| `blacklist:<access_jwt>` | access TTL (15m) | "1" |
| `session:<session_id>` | configurable | JSON session data |
| `ratelimit:<key>` | window | counter |

---

## Background Jobs

| Job | Trigger | Concurrency |
| --- | --- | --- |
| Scan execution | ScanService.Create → goroutine | 1 per scan (sequential) |
| Load test runner | LoadTestService.Create → goroutine | 1 per test (multi-worker inside) |
| ZAP container | Full scan start | Docker pulls/starts ZAP, waits for API |

---

## Web UI (SPA)

- Embedded via `web/embed.go` (`//go:embed index.html`)
- Served at `/` via `http.FileServer(http.FS(web.FS))`
- Auth state in `localStorage` (access_token, refresh_token)
- Auto token refresh on 401, fallback to login modal
- First-run setup modal shown when `GET /api/v1/auth/status` returns `setup_required: true`

---

## Configuration (Environment Variables)

| Variable | Default | Description |
| --- | --- | --- |
| `HTTP_ADDR` | `:9090` | HTTP listen address |
| `DB_PATH` | `webtester.db` | SQLite database file |
| `ZAP_IMAGE` | `ghcr.io/zaproxy/zaproxy:stable` | ZAP Docker image |
| `ZAP_TIMEOUT_SECONDS` | `1800` | ZAP scan timeout |
| `REDIS_ADDR` | `localhost:6379` | Redis connection address |
| `REDIS_PASSWORD` | (empty) | Redis password |
| `REDIS_DB` | `0` | Redis DB number |

Loaded in `configs.Load()` from `os.Getenv` with defaults.

---

## Startup Sequence (cmd/app/main.go)

```go
func main() {
    cfg := configs.Load()                          // 1. Config from env

    store, _ := stores.NewSQLiteStore(cfg.DB.Path) // 2. Scan store (migrate schema)
    loadStore, _ := stores.NewLoadTestStore(...)   // 3. Load test store

    userStore, _ := auth.NewUserStore(cfg.DB.Path) // 4. User store (migrate users table)

    redisStore, _ := auth.NewRedisStore(...)       // 5. Redis client + ping

    tokenConfig := auth.DefaultTokenConfig()       // 6. JWT secrets (change in prod!)

    authSvc := services.NewAuthService(userStore, redisStore, tokenConfig)
    authMiddleware := auth.NewMiddleware(redisStore, tokenConfig)

    zapClient := zap.NewDockerClient(...)          // 7. ZAP docker client

    scanSvc := services.NewScanService(store, zapClient)
    loadSvc := services.NewLoadTestService(loadStore)

    r := routes.New(cfg, scanSvc, loadSvc, authSvc, authMiddleware) // 8. Router

    http.ListenAndServe(cfg.HTTPAddr, r)           // 9. Serve
}
```

---

## Testing Strategy

| Layer | Tool | Focus |
| --- | --- | --- |
| `internal/auth` | `go test` | Token generation/parsing, bcrypt, SQLite user CRUD |
| `internal/services` | (none yet) | Mock stores → test orchestration logic |
| `internal/handlers` | (none yet) | HTTP table tests, JSON decode, validator |
| `internal/loadtest` | (none yet) | Runner timing, stats aggregation |
| Integration | (planned) | Real SQLite + Redis via testcontainers |

Run: `go test ./...`