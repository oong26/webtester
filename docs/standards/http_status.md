# HTTP Status Code Standards

## Quick Reference

| Code | Name | When to Use |
| --- | --- | --- |
| 200 | OK | Successful GET, PUT, PATCH, POST (non-create) |
| 201 | Created | Successful POST that creates a resource (return resource in body) |
| 204 | No Content | Successful DELETE, or POST with no response body |
| 400 | Bad Request | Malformed JSON, missing required fields (generic) |
| 401 | Unauthorized | Missing/invalid/expired token, wrong credentials |
| 403 | Forbidden | Valid token but insufficient permissions (role, ownership) |
| 404 | Not Found | Resource ID not found |
| 409 | Conflict | Resource already exists (e.g., duplicate email, setup already done) |
| 412 | Precondition Failed | Business rule violation (e.g., cannot cancel completed scan) |
| 422 | Unprocessable Entity | Validation failed (struct validation, field rules) |
| 429 | Too Many Requests | Rate limit exceeded |
| 500 | Internal Server Error | Unexpected error (DB down, external API timeout, panic) |
| 503 | Service Unavailable | Dependency temporarily down (ZAP not ready, Redis down) |

---

## Mapping in This Project

| Scenario | Status | Error Code | Handler |
| --- | --- | --- | --- |
| Invalid JSON body | 422 | `validation_failed` | `apperr.Validation` |
| Validator struct error | 422 | `validation_failed` | `apperr.Validation` |
| Missing auth token | 401 | `unauthorized` | Auth middleware |
| Invalid/expired token | 401 | `unauthorized` | Auth middleware |
| Token blacklisted | 401 | `unauthorized` | Auth middleware |
| Wrong email/password | 401 | `unauthorized` | AuthService.Login |
| Scan not found | 404 | `not_found` | ScanService.Get |
| Load test not found | 404 | `not_found` | LoadTestService.Get |
| Setup already completed | 409 | `conflict` | AuthService.Setup |
| Duplicate email (if added) | 409 | `conflict` | UserStore.Create |
| Cannot cancel finished load test | 412 | `precondition_failed` | LoadTestService.Cancel |
| ZAP container start failed | 503 | `service_unavailable` | ScanService.Execute |
| Redis connection failed | 503 | `service_unavailable` | AuthMiddleware/Service |
| SQLite error | 500 | `internal_error` | Store methods |
| Panic recovered | 500 | `internal_error` | pkg/middleware.Recover |

---

## Decision Tree

```
Is it a client error (4xx) or server error (5xx)?
├─ Client sent bad data → 422 (validation_failed)
├─ Client not authenticated → 401 (unauthorized)
├─ Client authenticated but not allowed → 403 (forbidden)
├─ Resource doesn't exist → 404 (not_found)
├─ Resource already exists → 409 (conflict)
├─ Business rule prevents action → 412 (precondition_failed)
└─ Rate limit → 429 (too_many_requests)

Server side:
├─ Dependency down (Redis, ZAP, DB) → 503 (service_unavailable)
└─ Unexpected bug/panic → 500 (internal_error)
```

---

## 409 vs 412 vs 422

| Code | Meaning | Example |
| --- | --- | --- |
| **422** | Request data is syntactically valid but semantically invalid | Email format wrong, password too short, scan_type not in enum |
| **409** | Resource conflict with current state | Setup already done, email already registered, concurrent modification |
| **412** | Precondition in request headers/state not met, OR business rule violation | Cancel a completed load test, start scan without ownership confirmation |

**In this project:**
- `Setup` after first user → **409 Conflict** ("initial setup already completed")
- `Cancel` on non-running load test → **412 Precondition Failed** (if implemented)
- Validator errors → **422 Unprocessable Entity**

---

## Response Format

All errors via `pkg/response.Fail(w, err)`:

```json
{
  "error": {
    "code": "unauthorized",
    "message": "missing authorization token"
  }
}
```

Success via `pkg/response.JSON(w, status, data)`:

```json
{
  "data": { "id": "...", "status": "queued", ... }
}
```

---

## Headers

| Header | Usage |
| --- | --- |
| `Content-Type: application/json` | All responses |
| `Retry-After: <seconds>` | 429, 503 (optional) |
| `Deprecation: true` | Deprecated endpoints |
| `Link: <...>; rel="next"` | Pagination (future) |

---

## Special Cases

### Auth Middleware (401)

```go
// internal/auth/middleware.go
if token == "" {
    response.Fail(w, apperr.Unauthorized("missing authorization token"))
    return
}
if blacklisted {
    response.Fail(w, apperr.Unauthorized("token is blacklisted"))
    return
}
```

### Validator (422)

```go
// internal/handlers/scan_handler.go
if err := validate.Struct(req); err != nil {
    response.Fail(w, apperr.Validation(err.Error()))
    return
}
```

### Not Found (404)

```go
// internal/services/scan_service.go
if errors.Is(err, stores.ErrNotFound) {
    return nil, apperr.NotFound("scan not found")
}
```

### Conflict (409)

```go
// internal/services/auth_service.go
count, _ := s.userStore.Count(ctx)
if count > 0 {
    return nil, apperr.Conflict("initial setup already completed")
}
```

---

## Never Use

| Code | Reason |
| --- | --- |
| 400 Bad Request | Too generic. Use 422 for validation, 412 for business rules. |
| 403 Forbidden | Not implemented yet (no RBAC). When roles added, use for permission checks. |
| 500 for known dependency failure | Use 503 with `service_unavailable`. |