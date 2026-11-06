# Error Handling Standards

## AppError Type

Defined in `pkg/errors/app_error.go`:

```go
type AppError struct {
    Status  int    // HTTP status code
    Code    string // Machine-readable code (snake_case)
    Message string // Human-readable message
    Err     error  // Wrapped error (optional)
}

func (e *AppError) Error() string { ... }
func (e *AppError) Unwrap() error { return e.Err }
```

---

## Constructors

| Function | Status | Code | Use For |
| --- | --- | --- | --- |
| `Validation(msg)` | 422 | `validation_failed` | Invalid input, struct validation failure |
| `NotFound(msg)` | 404 | `not_found` | Resource not found |
| `Internal(msg, err)` | 500 | `internal_error` | Unexpected errors, DB failures, external API errors |
| `Unavailable(msg, err)` | 503 | `service_unavailable` | Dependency temporarily down (Redis, ZAP, DB) |
| `Unauthorized(msg)` | 401 | `unauthorized` | Missing/invalid/expired token, wrong credentials |
| `Conflict(msg)` | 409 | `conflict` | Duplicate resource, state conflict |

---

## Error Flow

```
Store/Client
    │
    ▼ returns error (sql.ErrNoRows, redis.Nil, etc.)
Service
    │
    ├─ Wrap with apperr.Internal/NotFound/Conflict
    │
    ▼ returns *AppError
Handler
    │
    ├─ response.Fail(w, err)  // writes JSON error envelope
    │
    ▼ HTTP response
```

---

## Wrapping Rules

### 1. Not Found (from store)

```go
// internal/services/scan_service.go
scan, err := s.store.Get(ctx, id)
if err != nil {
    if errors.Is(err, stores.ErrNotFound) {
        return nil, apperr.NotFound("scan not found")
    }
    return nil, apperr.Internal("get scan", err)
}
```

### 2. Internal (unexpected)

```go
// internal/services/scan_service.go
scan, err := s.store.Create(ctx, params)
if err != nil {
    return nil, apperr.Internal("create scan", err)  // wraps original
}
```

### 3. Conflict (business rule)

```go
// internal/services/auth_service.go
count, _ := s.userStore.Count(ctx)
if count > 0 {
    return nil, apperr.Conflict("initial setup already completed")
}
```

### 4. Validation (handler)

```go
// internal/handlers/scan_handler.go
if err := validate.Struct(req); err != nil {
    response.Fail(w, apperr.Validation(err.Error()))
    return
}
```

### 5. Unavailable (dependency down)

```go
// internal/auth/middleware.go
blacklisted, err := m.store.IsTokenBlacklisted(r.Context(), token)
if err != nil {
    response.Fail(w, apperr.Unavailable("failed to check token", err))
    return
}
```

---

## Response Format

All errors via `pkg/response.Fail`:

```json
{
  "error": {
    "code": "validation_failed",
    "message": "Key: 'CreateScanRequest.TargetURL' Error:Field validation for 'TargetURL' failed on the 'url' tag"
  }
}
```

Success via `pkg/response.JSON`:

```json
{
  "data": { "id": "...", "status": "queued", ... }
}
```

---

## Logging

- **Handlers:** Do not log errors (middleware does access log).
- **Services:** Log unexpected errors at `slog.Error` level with context.
- **Stores/Clients:** Return errors; let service decide logging.
- **Middleware:** `Recover` logs panics; `AccessLog` logs every request.

```go
// Example service logging
func (s *ScanService) Execute(scanID string) {
    ctx := context.Background()
    if err := s.runChecks(ctx, scanID); err != nil {
        slog.Error("scan execution failed", "scan_id", scanID, "error", err)
        s.store.UpdateStatus(ctx, scanID, "failed", 100, err.Error())
    }
}
```

---

## Error Codes Reference

| Code | Status | Source |
| --- | --- | --- |
| `validation_failed` | 422 | `apperr.Validation` |
| `not_found` | 404 | `apperr.NotFound` |
| `internal_error` | 500 | `apperr.Internal` |
| `service_unavailable` | 503 | `apperr.Unavailable` |
| `unauthorized` | 401 | `apperr.Unauthorized` |
| `conflict` | 409 | `apperr.Conflict` |
| `precondition_failed` | 412 | (future) `apperr.PreconditionFailed` |
| `too_many_requests` | 429 | (future) `apperr.TooManyRequests` |

---

## Do Not

| Anti-pattern | Correct |
| --- | --- |
| `return fmt.Errorf("failed: %w", err)` | Use `apperr.Internal("context", err)` |
| Return raw `error` from service | Return `*AppError` or typed sentinel (`ErrNotFound`) |
| Log at store level | Log at service level with context |
| Panic in business logic | Return error; only `Recover` middleware panics |
| Expose internal error details to client | `message` is human-readable; `code` for machine |