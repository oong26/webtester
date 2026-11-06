# WebTester — Engineering Documentation

WebTester is an automated web security scanner and load testing service written in Go. It provides security scanning (passive baseline, OpenAPI fuzzing, full active scan via OWASP ZAP, BOLA probing), configurable HTTP load testing, Redis session caching, JWT authentication, and a built-in single-page web UI.

Product specifications live in [specification.md](specification.md). Engineering standards and architecture live in this directory.

---

## Start Here

| Goal | Document |
| --- | --- |
| Understand product features & specs | [specification.md](specification.md) |
| Setup and run locally | [development.md](development.md) |
| Architecture & layer boundaries | [architecture.md](architecture.md) |
| Codebase conventions & rules | [standards/standard.md](standards/standard.md) |
| Error handling & HTTP status | [error_handling.md](error_handling.md), [standards/http_status.md](standards/http_status.md) |
| Commit & branch workflow | [git_workflow.md](git_workflow.md) |
| Documentation rules | [standards/documentation.md](standards/documentation.md) |

---

## Tech Stack

| Domain | Technology |
| --- | --- |
| Language | Go ≥ 1.22 |
| Router | `go-chi/chi` v5 |
| Database | SQLite (`modernc.org/sqlite` - CGO free) |
| Cache & Sessions | Redis (`github.com/redis/go-redis/v9`) |
| Authentication | JWT (`github.com/golang-jwt/jwt/v5`) + `bcrypt` |
| Security Scanner | OWASP ZAP (Docker integration) + built-in probes |
| Validation | `go-playground/validator/v10` |
| UI | Embedded SPA HTML/JS (`web/embed.go`) |

---

## Directory Layout Overview

```
webtester/
├── cmd/
│   └── app/                  # Application entrypoint (wires DB, Redis, services, router)
├── internal/
│   ├── auth/                 # JWT tokens, Redis token store, SQLite user store, middleware
│   ├── checks/               # Security scan rules (passive, BOLA probes)
│   ├── clients/zap/          # OWASP ZAP Docker runner client
│   ├── configs/              # Config loader from environment
│   ├── dtos/                 # Data transfer objects for API requests/responses
│   ├── handlers/             # HTTP handlers (Scan, LoadTest, Auth)
│   ├── loadtest/             # Concurrent load test runner engine
│   ├── routes/               # Chi router setup, route groups, auth protection
│   ├── services/             # Business logic (ScanService, LoadTestService, AuthService)
│   └── stores/               # Data access layer (SQLite scan store, loadtest store)
├── pkg/                      # Independent shared packages
│   ├── errors/               # AppError definition & constructors
│   ├── middleware/           # Access logging, panic recovery
│   └── response/             # Standard JSON response wrappers
├── web/                      # Single page app frontend embedded into binary
└── docs/                     # Engineering documentation
```
