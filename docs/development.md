# WebTester Development Guide

## Prerequisites

- Go ≥ 1.22 (Go 1.26 toolchain recommended)
- Redis server (local or Docker)
- Docker (optional, required only for OWASP ZAP Full Scan)
- SQLite3 (optional CLI for inspecting `webtester.db`)
- `curl` or HTTP client (Postman, Insomnia)

---

## Quickstart

### 1. Start Redis

```bash
# Via Docker
docker run -d --name webtester-redis -p 6379:6379 redis:7-alpine

# Or native (macOS Homebrew)
brew services start redis
```

### 2. Run Application

```bash
# Run server directly
go run ./cmd/app
```

Output:
```
2026/11/07 02:00:00 webtester listening on :9090
```

### 3. Open UI in Browser

Open `http://localhost:9090` in browser.
- If first run: First-time admin setup modal appears.
- Enter Name, Email, Password (min 8 chars) to create initial administrator.

---

## Configuration Reference

Loaded in `internal/configs/config.go` from environment variables:

| Variable | Default | Purpose |
| --- | --- | --- |
| `HTTP_ADDR` | `:9090` | Server listen port |
| `DB_PATH` | `webtester.db` | SQLite database file location |
| `REDIS_ADDR` | `localhost:6379` | Redis host:port |
| `REDIS_PASSWORD` | `""` | Redis authentication password |
| `REDIS_DB` | `0` | Redis logical DB index |
| `ZAP_IMAGE` | `ghcr.io/zaproxy/zaproxy:stable` | ZAP container image |
| `ZAP_TIMEOUT_SECONDS` | `1800` | Max timeout for active ZAP scans |

Example `.env` / export:
```bash
export HTTP_ADDR=":8080"
export DB_PATH="/var/data/webtester.db"
export REDIS_ADDR="127.0.0.1:6379"
```

---

## Build & Test Commands

```bash
# Build binary
go build -o bin/webtester ./cmd/app

# Run all unit tests
go test -v ./...

# Run auth tests only
go test -v ./internal/auth

# Run with race detector
go test -race ./...

# Check linting
golangci-lint run
```

---

## Testing Scans & Load Tests

### Baseline Scan (Passive)

Does not require Docker/ZAP. Runs built-in checks:
- Security headers (CSP, HSTS, X-Content-Type-Options)
- Cookie security flags (Secure, HttpOnly, SameSite)
- Server/Tech disclosure

### API Scan (OpenAPI Fuzzing)

Requires accessible OpenAPI YAML/JSON schema:
- Tests all declared paths
- Fuzzes path, query, and body parameters

### Full Scan (Active OWASP ZAP)

Requires local Docker daemon running:
```bash
docker pull ghcr.io/zaproxy/zaproxy:stable
```
WebTester starts an ephemeral container, executes spider and active scan rules, pulls findings, then terminates the container.

### Load Test

- Target RPS: 1 to 5000 req/sec
- Concurrency: 1 to 500 simultaneous workers
- Duration: 1 to 600 seconds
- Supports HTTP GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS
- Supports custom headers & payload
- In-flight requests safely cancelled at duration deadline
- Error breakdown and latency percentiles (P50, P90, P95, P99) aggregated in real time

---

## Database Management

SQLite schema files are embedded into Go packages:
- `internal/stores/schema.sql` (scans & findings)
- `internal/stores/loadtest_schema.sql` (load test runs)
- `internal/auth/user_schema.sql` (users)

All schemas use `CREATE TABLE IF NOT EXISTS` and run automatically on application startup.

To inspect database via CLI:
```bash
sqlite3 webtester.db ".tables"
sqlite3 webtester.db "SELECT id, email, role, created_at FROM users;"
sqlite3 webtester.db "SELECT id, target_url, status, progress FROM scans;"
```

---

## Redis Inspection

```bash
redis-cli
> KEYS *
1) "refresh:96a023649f072a9ea546ac80"
2) "blacklist:eyJhbGciOi..."
> GET refresh:<user_id>
```
