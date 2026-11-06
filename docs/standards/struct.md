# Struct & DTO Standards

## DTO Location

All data transfer objects live in `internal/dtos/`:

```
internal/dtos/
├── scan_dto.go       # Scan requests/responses
├── loadtest_dto.go   # Load test requests/responses
├── auth_dto.go       # Auth requests/responses
```

---

## Naming Conventions

| Kind | Suffix | Example |
| --- | --- | --- |
| Create request | `Create{Entity}Request` | `CreateScanRequest` |
| Update request | `Update{Entity}Request` | (not used yet) |
| Query params | `List{Entity}Params` | `ListScansParams` (store) |
| Single response | `{Entity}Response` | `ScanResponse` |
| List response | `[]Response` (wrapped) | `[]ScanResponse` in envelope |
| Auth tokens | `TokenResponse` | `TokenResponse` |
| Nested object | Inline in parent | `AuthConfig` inside `CreateScanRequest` |

---

## Scan DTOs (scan_dto.go)

```go
type CreateScanRequest struct {
    TargetURL       string      `json:"target_url" validate:"required,url"`
    ScanType        string      `json:"scan_type" validate:"required,oneof=baseline api full"`
    OpenAPIURL      string      `json:"openapi_url,omitempty" validate:"omitempty,url"`
    ConfirmOwnership bool       `json:"confirm_ownership" validate:"required"`
    Auth            *AuthConfig `json:"auth,omitempty"`
    BOLAProbeURLs   []string    `json:"bola_probe_urls,omitempty"`
}

type AuthConfig struct {
    HeaderName string `json:"header_name" validate:"required"`
    UserA      string `json:"user_a" validate:"required"`
    UserB      string `json:"user_b,omitempty"`
}

type ScanResponse struct {
    ID            string    `json:"id"`
    TargetURL     string    `json:"target_url"`
    ScanType      string    `json:"scan_type"`
    Status        string    `json:"status"`
    Progress      int       `json:"progress"`
    ErrorMessage  string    `json:"error_message,omitempty"`
    CreatedAt     time.Time `json:"created_at"`
    UpdatedAt     time.Time `json:"updated_at"`
}

type FindingResponse struct {
    ID          string `json:"id"`
    RuleID      string `json:"rule_id"`
    Name        string `json:"name"`
    Severity    string `json:"severity"`
    Confidence  string `json:"confidence"`
    URL         string `json:"url"`
    Evidence    string `json:"evidence"`
    CWE         int    `json:"cwe"`
    Source      string `json:"source"`
    CreatedAt   time.Time `json:"created_at"`
}
```

---

## Load Test DTOs (loadtest_dto.go)

```go
type CreateLoadTestRequest struct {
    TargetURL      string            `json:"target_url" validate:"required,url"`
    Method         string            `json:"method" validate:"required,oneof=GET HEAD OPTIONS POST PUT PATCH DELETE"`
    DurationSeconds int              `json:"duration_seconds" validate:"required,min=1,max=600"`
    RequestsPerSec  int              `json:"requests_per_sec" validate:"required,min=1,max=5000"`
    Concurrency     int              `json:"concurrency" validate:"required,min=1,max=500"`
    Headers         map[string]string `json:"headers,omitempty"`
    Body            string            `json:"body,omitempty"`
    ConfirmOwnership bool            `json:"confirm_ownership" validate:"required"`
}

type LoadTestResponse struct {
    ID              string             `json:"id"`
    TargetURL       string             `json:"target_url"`
    Method          string             `json:"method"`
    DurationSeconds int                `json:"duration_seconds"`
    RequestsPerSec  int                `json:"requests_per_sec"`
    Concurrency     int                `json:"concurrency"`
    Status          string             `json:"status"`
    ErrorMessage    string             `json:"error_message,omitempty"`
    CreatedAt       time.Time          `json:"created_at"`
    StartedAt       *time.Time         `json:"started_at,omitempty"`
    FinishedAt      *time.Time         `json:"finished_at,omitempty"`
    TotalRequests   int64              `json:"total_requests"`
    Successful      int64              `json:"successful"`
    Failed          int64              `json:"failed"`
    ErrorRate       float64            `json:"error_rate"`
    RPSAchieved     float64            `json:"rps_achieved"`
    MinMs           float64            `json:"min_ms"`
    MaxMs           float64            `json:"max_ms"`
    MeanMs          float64            `json:"mean_ms"`
    P50Ms           float64            `json:"p50_ms"`
    P90Ms           float64            `json:"p90_ms"`
    P95Ms           float64            `json:"p95_ms"`
    P99Ms           float64            `json:"p99_ms"`
    StatusBreakdown map[string]int64   `json:"status_breakdown"`
    ErrorBreakdown  map[string]int64   `json:"error_breakdown"`
}
```

---

## Auth DTOs (auth_dto.go)

```go
type SetupStatusResponse struct {
    SetupRequired bool `json:"setup_required"`
}

type SetupRequest struct {
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
    Name     string `json:"name" validate:"required"`
}

type LoginRequest struct {
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required"`
}

type RefreshTokenRequest struct {
    RefreshToken string `json:"refresh_token" validate:"required"`
}

type TokenResponse struct {
    AccessToken  string       `json:"access_token"`
    RefreshToken string       `json:"refresh_token"`
    TokenType    string       `json:"token_type"` // "Bearer"
    ExpiresIn    int64        `json:"expires_in"` // seconds
    User         UserResponse `json:"user"`
}

type UserResponse struct {
    ID        string     `json:"id"`
    Email     string     `json:"email"`
    Name      string     `json:"name"`
    Role      string     `json:"role"`
    CreatedAt time.Time  `json:"created_at"`
    LastLogin *time.Time `json:"last_login,omitempty"`
}
```

---

## Rules

1. **JSON tags required:** Every exported field must have `json:"field_name"` (snake_case).
2. **Validation tags on requests:** All create/login/setup requests use `validate:"..."` tags.
3. **Omitempty on optional:** Use `omitempty` for truly optional fields.
4. **No pointers for primitives:** Use `string`, `int`, `bool` — not `*string`. Only pointers for `time.Time` nilability and optional nested structs.
5. **Time format:** `time.Time` serializes to RFC3339 (JSON default). Store in UTC.
6. **Map for flexible headers:** `map[string]string` for dynamic headers.
7. **Response ≠ Store model:** DTOs may flatten, omit, or rename store fields. Never return store structs directly.
8. **Envelope always:** HTTP responses wrapped via `pkg/response.JSON(w, status, data)` → `{"data": ...}`.

---

## Validation Tags Quick Reference

```go
validate:"required"                    // non-zero
validate:"required,email"              // valid email
validate:"required,url"                // valid URL
validate:"required,min=8"              // min length/value
validate:"required,oneof=a b c"`       // enum
validate:"omitempty,url"`              // optional but if present, valid URL
validate:"required,min=1,max=5000"`    // numeric range
```

---

## Example: Handler → DTO Flow

```go
// Handler
func (h *ScanHandler) Create(w http.ResponseWriter, r *http.Request) {
    var req dtos.CreateScanRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        response.Fail(w, apperr.Validation("invalid JSON body"))
        return
    }
    if err := validate.Struct(req); err != nil {
        response.Fail(w, apperr.Validation(err.Error()))
        return
    }
    res, err := h.svc.Create(r.Context(), req)
    ...
}

// Service
func (s *ScanService) Create(ctx context.Context, req dtos.CreateScanRequest) (*dtos.ScanResponse, error) {
    // Map DTO → Store params
    params := stores.CreateScanParams{
        TargetURL:  req.TargetURL,
        ScanType:   req.ScanType,
        OpenAPIURL: req.OpenAPIURL,
        Status:     "queued",
    }
    scan, err := s.store.Create(ctx, params)
    ...
    return toScanResponse(scan), nil
}
```

---

## Forbidden Patterns

| Anti-pattern | Why |
| --- | --- |
| Returning store struct from handler | Couples API to DB schema |
| Sharing DTO between request and response | Different validation/fields |
| Using `interface{}` in DTOs | Loses type safety |
| Embedding store struct in DTO | Same as first row |