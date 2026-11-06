# Unit Testing Standards

## Test File Naming

| Source | Test File |
| --- | --- |
| `internal/auth/token.go` | `internal/auth/token_test.go` |
| `internal/auth/user_store.go` | `internal/auth/user_store_test.go` |
| `internal/services/scan_service.go` | `internal/services/scan_service_test.go` |
| `internal/handlers/scan_handler.go` | `internal/handlers/scan_handler_test.go` |

---

## What to Test Per Layer

### 1. `internal/auth` (Pure logic + SQLite/Redis integration)

**Existing:**
- `token_test.go`: JWT generation, parsing, expiry, wrong secret
- `user_store_test.go`: SQLite CRUD, bcrypt, schema migration

**Add:**
- `middleware_test.go`: Token extraction, blacklist check, context injection
- `store_test.go`: Redis token save/get/delete, blacklist, session

### 2. `internal/services` (Mock stores)

Test orchestration logic, error mapping:

```go
type fakeScanStore struct {
    scans map[string]stores.Scan
    mu    sync.Mutex
}

func (f *fakeScanStore) Create(ctx context.Context, p stores.CreateScanParams) (stores.Scan, error) {
    f.mu.Lock(); defer f.mu.Unlock()
    scan := stores.Scan{ID: "test-id", TargetURL: p.TargetURL, Status: p.Status, ...}
    f.scans[scan.ID] = scan
    return scan, nil
}

func (f *fakeScanStore) Get(ctx context.Context, id string) (stores.Scan, error) {
    f.mu.Lock(); defer f.mu.Unlock()
    s, ok := f.scans[id]
    if !ok { return stores.Scan{}, stores.ErrNotFound }
    return s, nil
}

func TestScanService_Create(t *testing.T) {
    fake := &fakeScanStore{scans: make(map[string]stores.Scan)}
    svc := services.NewScanService(fake, nil) // nil zap client for baseline

    req := dtos.CreateScanRequest{TargetURL: "http://example.com", ScanType: "baseline", ConfirmOwnership: true}
    res, err := svc.Create(context.Background(), req)

    assert.NoError(t, err)
    assert.Equal(t, "queued", res.Status)
    assert.Len(t, fake.scans, 1)
}
```

### 3. `internal/handlers` (httptest)

```go
func TestScanHandler_Create_Validation(t *testing.T) {
    svc := services.NewScanService(&fakeScanStore{}, nil)
    h := handlers.NewScanHandler(svc)

    req := httptest.NewRequest("POST", "/api/v1/scans", strings.NewReader(`{"target_url":"bad"}`))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()

    h.Create(w, req)

    assert.Equal(t, 422, w.Code)
    var resp map[string]any
    json.Unmarshal(w.Body.Bytes(), &resp)
    assert.Equal(t, "validation_failed", resp["error"].(map[string]any)["code"])
}
```

### 4. `internal/loadtest` (Timing & Stats)

- Test stats aggregation (min/max/mean/p50/p95/p99)
- Test cancellation via context timeout
- Test status breakdown counting

---

## Test File Layout

```
internal/auth/
├── token.go
├── token_test.go           # Unit tests
├── user_store.go
├── user_store_test.go      # SQLite integration tests
├── store.go
├── store_test.go           # (future) Redis integration
└── middleware.go
    middleware_test.go      # (future)

internal/services/
├── scan_service.go
├── scan_service_test.go    # With fake stores
├── loadtest_service.go
├── loadtest_service_test.go
├── auth_service.go
└── auth_service_test.go

internal/handlers/
├── scan_handler.go
├── scan_handler_test.go
├── loadtest_handler.go
├── loadtest_handler_test.go
├── auth_handler.go
└── auth_handler_test.go
```

---

## Test Helpers

### Temp SQLite

```go
func newTestStore(t *testing.T) *sqliteScanStore {
    t.Helper()
    tmp := t.TempDir()
    path := filepath.Join(tmp, "test.db")
    store, err := NewSQLiteStore(path)
    require.NoError(t, err)
    return store
}
```

### Fake Store (in `_test.go`)

Define fakes **in the test file** (not exported). Use `sync.Mutex` for concurrent access.

### Redis (via testcontainers, future)

```go
func TestRedisStore(t *testing.T) {
    container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: testcontainers.ContainerRequest{Image: "redis:7-alpine"},
    })
    defer container.Terminate(ctx)
    port, _ := container.MappedPort(ctx, "6379")
    store, err := auth.NewRedisStore("localhost:"+port.Port(), "", 0)
    ...
}
```

---

## Running Tests

```bash
go test ./...                    # All tests
go test ./internal/auth -v      # Single package verbose
go test -run TestToken ./...    # Single test
go test -cover ./...            # Coverage
go test -bench=. ./internal/... # Benchmarks
```

---

## Assertions (Minimal)

No testify dependency yet. Use standard library:

```go
import "testing"

func TestToken(t *testing.T) {
    if got != want {
        t.Errorf("got %q, want %q", got, want)
    }
}
```

When testify is added:
```go
require.NoError(t, err)
assert.Equal(t, expected, actual)
assert.Contains(t, output, "substring")
```

---

## Test Data

| Kind | Location | Notes |
| --- | --- | --- |
| Fixtures | `testdata/` in package | JSON files, OpenAPI specs |
| Seeds | Inline in `_test.go` | Small structs |
| Real data | Never in tests | Use fake URLs (`example.com`) |

---

## Coverage Goals

| Layer | Target |
| --- | --- |
| `pkg/errors`, `pkg/response`, `pkg/middleware` | 90%+ |
| `internal/auth` (token, user_store) | 85%+ |
| `internal/services` | 70%+ (with fakes) |
| `internal/handlers` | 60%+ (httptest) |
| `internal/stores` (SQL) | 50%+ (SQLite temp) |

---

## CI Integration

```makefile
test:
	go test ./... -race -count=1 -cover

test-auth:
	go test ./internal/auth -v

test-short:
	go test ./... -short   # skip integration (Redis, ZAP)
```

---

## Forbidden

| Anti-pattern | Why |
| --- | --- |
| Test network (Docker, HTTP) | Flaky, slow. Use fakes. |
| Test with real credentials | Use fake secrets. |
| Global test state | Each test independent. |
| Sleep in tests | Use channels/context instead. |
| Skip without `-short` guard | Mark integration tests `if testing.Short() { t.Skip() }` |