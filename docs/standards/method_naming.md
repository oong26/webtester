# Method Naming Standards

## Store Interfaces (Data Access)

| Operation | Method Name | Signature |
| --- | --- | --- |
| Create | `Create` | `Create(ctx, params) (Entity, error)` |
| Get by ID | `Get` | `Get(ctx, id) (Entity, error)` |
| Get by unique field | `GetBy{Field}` | `GetByEmail(ctx, email) (Entity, error)` |
| List (paginated) | `List` | `List(ctx, params ListParams) ([]Entity, error)` |
| Update (full) | `Update` | `Update(ctx, id, params) error` |
| Update (partial/status) | `Update{Field}` | `UpdateStatus(ctx, id, status) error` |
| Delete | `Delete` | `Delete(ctx, id) error` |
| Count | `Count` | `Count(ctx) (int, error)` |
| Bulk insert | `Save{Plural}` | `SaveFindings(ctx, scanID, findings) error` |

### Examples from Codebase

```go
// internal/stores/scan_store.go
type ScanStore interface {
    Create(ctx context.Context, p CreateScanParams) (Scan, error)
    Get(ctx context.Context, id string) (Scan, error)
    List(ctx context.Context, p ListScansParams) ([]Scan, error)
    UpdateStatus(ctx context.Context, id, status string, progress int, errMsg string) error
    UpdateProgress(ctx context.Context, id string, progress int) error
    SaveFindings(ctx context.Context, scanID string, findings []Finding) error
}

// internal/stores/loadtest_store.go
type LoadTestStore interface {
    Create(ctx context.Context, p CreateLoadTestParams) (LoadTest, error)
    Get(ctx context.Context, id string) (LoadTest, error)
    List(ctx context.Context, p ListLoadTestsParams) ([]LoadTest, error)
    UpdateStatus(ctx context.Context, id, status, errMsg string) error
    UpdateStats(ctx context.Context, id string, stats LoadTestStats) error
}

// internal/auth/user_store.go
type UserStore interface {
    Create(ctx context.Context, email, password, name string) (*User, error)
    GetByEmail(ctx context.Context, email string) (*User, error)
    GetByID(ctx context.Context, id string) (*User, error)
    UpdatePassword(ctx context.Context, id, newPasswordHash string) error
    UpdateLastLogin(ctx context.Context, id string) error
    Count(ctx context.Context) (int, error)
}
```

---

## Service Methods (Business Logic)

| Operation | Method Name | Signature |
| --- | --- | --- |
| Create/orchestrate | `Create` | `Create(ctx, req DTO) (ResponseDTO, error)` |
| Get by ID | `Get` | `Get(ctx, id) (ResponseDTO, error)` |
| List | `List` | `List(ctx, limit, offset) ([]ResponseDTO, error)` |
| Execute async job | `Execute` | `Execute(ctx, id) error` (fire-and-forget) |
| Auth-specific | `Login`, `Refresh`, `Logout`, `Me`, `Setup`, `IsSetupRequired` | Domain-specific names |

### Examples from Codebase

```go
// internal/services/scan_service.go
func (s *ScanService) Create(ctx context.Context, req dtos.CreateScanRequest) (*dtos.ScanResponse, error)
func (s *ScanService) Get(ctx context.Context, id string) (*dtos.ScanResponse, error)
func (s *ScanService) List(ctx context.Context, limit, offset int) ([]dtos.ScanResponse, error)
func (s *ScanService) Execute(ctx context.Context, scanID string) // async

// internal/services/loadtest_service.go
func (s *LoadTestService) Create(ctx context.Context, req dtos.CreateLoadTestRequest) (*dtos.LoadTestResponse, error)
func (s *LoadTestService) Get(ctx context.Context, id string) (*dtos.LoadTestResponse, error)
func (s *LoadTestService) List(ctx context.Context, limit, offset int) ([]dtos.LoadTestResponse, error)
func (s *LoadTestService) Cancel(ctx context.Context, id string) error

// internal/services/auth_service.go
func (s *AuthService) IsSetupRequired(ctx context.Context) (bool, error)
func (s *AuthService) Setup(ctx context.Context, req dtos.SetupRequest) (*dtos.TokenResponse, error)
func (s *AuthService) Login(ctx context.Context, req dtos.LoginRequest) (*dtos.TokenResponse, error)
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*dtos.TokenResponse, error)
func (s *AuthService) Logout(ctx context.Context, accessToken, userID string) error
func (s *AuthService) Me(ctx context.Context, userID string) (*dtos.UserResponse, error)
```

---

## Handler Methods (HTTP Layer)

| Operation | Method Name | Signature |
| --- | --- | --- |
| List | `List` | `List(w http.ResponseWriter, r *http.Request)` |
| Create | `Create` | `Create(w http.ResponseWriter, r *http.Request)` |
| Get | `Get` | `Get(w http.ResponseWriter, r *http.Request)` |
| Update | `Update` | `Update(w http.ResponseWriter, r *http.Request)` |
| Delete/Cancel | `Delete` / `Cancel` | `Cancel(w http.ResponseWriter, r *http.Request)` |
| Special actions | `{Action}` | `Findings`, `Refresh`, `Logout`, `Me`, `Status`, `Setup`, `Login` |

### Examples from Codebase

```go
// internal/handlers/scan_handler.go
func (h *ScanHandler) List(w http.ResponseWriter, r *http.Request)
func (h *ScanHandler) Create(w http.ResponseWriter, r *http.Request)
func (h *ScanHandler) Get(w http.ResponseWriter, r *http.Request)
func (h *ScanHandler) Findings(w http.ResponseWriter, r *http.Request)

// internal/handlers/loadtest_handler.go
func (h *LoadTestHandler) List(w http.ResponseWriter, r *http.Request)
func (h *LoadTestHandler) Create(w http.ResponseWriter, r *http.Request)
func (h *LoadTestHandler) Get(w http.ResponseWriter, r *http.Request)
func (h *LoadTestHandler) Cancel(w http.ResponseWriter, r *http.Request)

// internal/handlers/auth_handler.go
func (h *AuthHandler) Status(w http.ResponseWriter, r *http.Request)
func (h *AuthHandler) Setup(w http.ResponseWriter, r *http.Request)
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request)
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request)
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request)
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request)
```

---

## Helper Functions (Package-Private)

| Pattern | Example |
| --- | --- |
| Scanners | `scanUser(row)`, `scanLoadTest(rows)` |
| ID generators | `newID() string` |
| Mappers | `toUserResponse(u *User) *dtos.UserResponse` |
| Validators | `isValidHeaderName(name string) bool` |

---

## Rules

1. **Receiver name:** Single letter matching struct (`s *ScanService`, `h *ScanHandler`, `st *sqliteScanStore`).
2. **Context first:** `ctx context.Context` always first parameter.
3. **Error last:** Return `(T, error)` or `error` only.
4. **No GetAll:** Use `List` with pagination params.
5. **No FindBy:** Use `GetBy{Field}` for unique lookups.
6. **Async jobs:** `Execute` returns immediately; status polled via `Get`.