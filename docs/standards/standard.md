# WebTester Codebase Standards

This document is the index of all coding standards. Each topic has its own file in this directory.

---

## Standards Index

| Topic | Document | Purpose |
| --- | --- | --- |
| API Naming | [api_naming.md](api_naming.md) | Endpoint paths, pluralization, verbs, nesting |
| Method Naming | [method_naming.md](method_naming.md) | Service/store method names (List, Get, Create, Update, Delete) |
| Structs & DTOs | [struct.md](struct.md) | Request/Response/Params naming, shared vs separate |
| Service & Store | [service_repo.md](service_repo.md) | Layer responsibilities, interfaces, transactions |
| HTTP Status Codes | [http_status.md](http_status.md) | Which code to return when, 409 vs 422 vs 412 |
| List API | [list_api.md](list_api.md) | Pagination, filtering, sorting, meta envelope |
| Error Handling | [error_handling.md](../error_handling.md) | AppError type, wrapping, mapping to HTTP |
| Unit Testing | [unit_test.md](unit_test.md) | What to test per layer, file layout, fakes |
| Documentation | [documentation.md](documentation.md) | Where docs live, link vs copy rules |

---

## Ground Rules (Non-Negotiable)

1. **Layer boundaries are strict.** `pkg/` never imports `internal/`. `internal/stores` never imports `internal/services` or `internal/handlers`. Import cycles = build failure.

2. **One struct per concern.** DTOs in `internal/dtos/` cross layer boundaries. Domain models stay in the layer that owns them (store, service).

3. **Validation at the edge.** HTTP handlers validate incoming JSON with `go-playground/validator`. Services trust validated input.

4. **Errors are values.** Use `pkg/errors.AppError` with status, code, message. Never return raw `error` from services/handlers unless it's already an `AppError`.

5. **Context everywhere.** All store/service methods take `context.Context` as first parameter. Pass `r.Context()` from handlers.

6. **No global state.** All dependencies injected via struct fields (store, client, config). No `init()`, no package-level vars except `validator` and `embed.FS`.

7. **Time is UTC.** Store `time.Time` in UTC (RFC3339Nano format in SQLite). Parse with `time.RFC3339Nano`.

8. **IDs are strings.** 24-char hex from 12 random bytes (`crypto/rand` + `hex.EncodeToString`). No UUID library.

9. **Passwords are bcrypt.** Cost = `bcrypt.DefaultCost`. Never log or return hash.

10. **JWT secrets configurable.** Default secrets in code are placeholders. Production must override via config.

---

## File Naming Conventions

| Artifact | Pattern | Example |
| --- | --- | --- |
| Go source | `snake_case.go` | `scan_handler.go` |
| Test file | `*_test.go` | `token_test.go` |
| SQL schema | `*_schema.sql` | `user_schema.sql` |
| Embedded files | `//go:embed` same dir | `schema.sql` next to `scan_store.go` |

---

## Directory Responsibilities

| Directory | Responsibility |
| --- | --- |
| `cmd/app` | Wiring only. No business logic. |
| `internal/auth` | JWT, Redis token store, SQLite user store, auth middleware. |
| `internal/checks` | Pure security check logic (passive, BOLA). No HTTP, no DB. |
| `internal/clients` | Outbound I/O only (Docker, HTTP calls). |
| `internal/configs` | Typed config struct + `Load()` from env. |
| `internal/dtos` | Request/Response/Params structs with validation tags. |
| `internal/handlers` | HTTP decode → validate → service call → response. |
| `internal/loadtest` | Concurrent request runner, stats aggregation. |
| `internal/routes` | Chi router, middleware groups, dependency injection. |
| `internal/services` | Business rules, orchestration, cross-store transactions. |
| `internal/stores` | Data access (SQLite, embedded SQL). Interfaces first. |
| `pkg/errors` | AppError definition + constructors (Validation, NotFound, etc.) |
| `pkg/middleware` | Recover, AccessLog, (auth lives in internal/auth). |
| `pkg/response` | JSON envelope writers (`JSON`, `Fail`). |

---

## Go Style (Enforced by `golangci-lint`)

- `goimports` formatting
- No unused variables/imports
- Error handling: `if err != nil { return ... }` on same line
- No `panic` except in `pkg/middleware/Recover`
- Struct tags: `json:"field_name"` + `validate:"rules"` on DTOs
- Interfaces defined where they are **consumed** (services define store interfaces)

---

## Validation Tags Reference

| Rule | Example | Notes |
| --- | --- | --- |
| `required` | `validate:"required"` | Non-zero value |
| `email` | `validate:"required,email"` | RFC 5322 |
| `min`/`max` | `validate:"required,min=8"` | String length or numeric |
| `oneof` | `validate:"oneof=baseline api full"` | Enum values |
| `url` | `validate:"required,url"` | Valid URL format |

---

## Adding a New Feature (Checklist)

1. [ ] Define DTOs in `internal/dtos/`
2. [ ] Add store interface + implementation in `internal/stores/` (embed schema)
3. [ ] Add service in `internal/services/` (inject store interface)
4. [ ] Add handler in `internal/handlers/` (inject service)
5. [ ] Wire route in `internal/routes/routes.go`
6. [ ] Update `cmd/app/main.go` wiring
7. [ ] Add unit tests for token/logic in `internal/auth/*_test.go`
8. [ ] Update OpenAPI spec if applicable (`api/openapi.yaml`)
9. [ ] Update relevant docs in `docs/`

---

## See Also

- Architecture: [architecture.md](../architecture.md)
- Development setup: [development.md](../development.md)
- Error handling details: [error_handling.md](../error_handling.md)
- Git workflow: [git_workflow.md](../git_workflow.md)