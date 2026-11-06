# Service & Store Layer Standards

## Layer Responsibilities

| Layer | Responsibility | Must Not |
| --- | --- | --- |
| **Handler** | HTTP decode, validate, call service, encode response | Business logic, DB queries, token generation |
| **Service** | Business rules, orchestration, cross-store transactions, async job launch | HTTP types, SQL, HTML rendering |
| **Store** | Data access (SQL), schema migrations, row scanning | Business rules, HTTP, external API calls |
| **Client** | Outbound I/O (Docker, HTTP, email) | Business rules, DB access |

---

## Interface Ownership

**Rule:** Interfaces are defined in the **consumer** package.

```go
// internal/services/scan_service.go (consumer defines)
type ScanStore interface {
    Create(ctx context.Context, p CreateScanParams) (Scan, error)
    Get(ctx context.Context, id string) (Scan, error)
    List(ctx context.Context, p ListScansParams) ([]Scan, error)
    UpdateStatus(ctx context.Context, id, status string, progress int, errMsg string) error
    SaveFindings(ctx context.Context, scanID string, findings []Finding) error
}

func NewScanService(store ScanStore, zapClient *zap.Client) *ScanService {
    return &ScanService{store: store, zap: zapClient}
}

// internal/stores/scan_store.go (implements)
type sqliteScanStore struct { db *sql.DB }

func NewSQLiteStore(path string) (ScanStore, error) { ... }

func (s *sqliteScanStore) Create(ctx context.Context, p CreateScanParams) (Scan, error) { ... }
```

This allows:
- Services to be tested with fake stores (no DB needed).
- Multiple store implementations (SQLite, Postgres, mock).
- Stores to be reused by different services.

---

## Dependency Injection Pattern

```go
// cmd/app/main.go
func main() {
    cfg := configs.Load()

    // 1. Stores (leaf dependencies)
    scanStore, _ := stores.NewSQLiteStore(cfg.Database.Path)
    loadStore, _ := stores.NewLoadTestStore(cfg.Database.Path)
    userStore, _ := auth.NewUserStore(cfg.Database.Path)
    redisStore, _ := auth.NewRedisStore(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)

    // 2. Clients (external I/O)
    zapClient := zap.NewDockerClient(cfg.Zap.Image, cfg.Zap.Timeout)

    // 3. Token config (shared)
    tokenConfig := auth.DefaultTokenConfig()

    // 4. Auth (store + client + config)
    authSvc := services.NewAuthService(userStore, redisStore, tokenConfig)
    authMiddleware := auth.NewMiddleware(redisStore, tokenConfig)

    // 5. Domain services (inject stores + clients)
    scanSvc := services.NewScanService(scanStore, zapClient)
    loadSvc := services.NewLoadTestService(loadStore)

    // 6. Router (inject all services + middleware)
    r := routes.New(cfg, scanSvc, loadSvc, authSvc, authMiddleware)

    http.ListenAndServe(cfg.HTTPAddr, r)
}
```

---

## Store Patterns

### 1. Embedded Schema Migration

```go
// internal/stores/scan_store.go
//go:embed schema.sql
var schemaFS embed.FS

func NewSQLiteStore(path string) (*sqliteScanStore, error) {
    db, err := sql.Open("sqlite", path)
    ...
    schema, _ := schemaFS.ReadFile("schema.sql")
    db.Exec(string(schema))  // idempotent CREATE TABLE IF NOT EXISTS
    return &sqliteScanStore{db: db}, nil
}
```

### 2. Row Scanning Helper (Private)

```go
func scanScan(row interface{ Scan(...interface{}) error }) (Scan, error) {
    var s Scan
    var createdStr, updatedStr string
    err := row.Scan(&s.ID, &s.TargetURL, ..., &createdStr, &updatedStr)
    s.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdStr)
    s.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedStr)
    return s, err
}
```

### 3. Param Structs for Create/Update

```go
type CreateScanParams struct {
    TargetURL  string
    OpenAPIURL string
    ScanType   string
    Status     string
}

func (s *sqliteScanStore) Create(ctx context.Context, p CreateScanParams) (Scan, error) {
    id := newID()
    now := time.Now().UTC().Format(time.RFC3339Nano)
    _, err := s.db.ExecContext(ctx, `INSERT INTO scans ...`, id, p.TargetURL, p.OpenAPIURL, p.ScanType, p.Status, now, now)
    return s.Get(ctx, id)
}
```

### 4. Context in Every Method

```go
func (s *sqliteScanStore) List(ctx context.Context, p ListScansParams) ([]Scan, error) {
    rows, err := s.db.QueryContext(ctx, `SELECT ... LIMIT ? OFFSET ?`, p.Limit, p.Offset)
    ...
}
```

---

## Service Patterns

### 1. Orchestration (Not Just Pass-Through)

```go
func (s *ScanService) Create(ctx context.Context, req dtos.CreateScanRequest) (*dtos.ScanResponse, error) {
    // Validate business rules (ownership, scan type compatibility)
    if req.ScanType == "api" && req.OpenAPIURL == "" {
        return nil, apperr.Validation("openapi_url required for api scan")
    }

    // Map DTO → Store params
    params := stores.CreateScanParams{...}

    // Persist
    scan, err := s.store.Create(ctx, params)
    if err != nil { return nil, apperr.Internal("create scan", err) }

    // Launch async execution
    go s.Execute(scan.ID)  // fire-and-forget, status polled via GET

    return toScanResponse(scan), nil
}
```

### 2. Error Wrapping

```go
func (s *ScanService) Get(ctx context.Context, id string) (*dtos.ScanResponse, error) {
    scan, err := s.store.Get(ctx, id)
    if err != nil {
        if errors.Is(err, stores.ErrNotFound) {
            return nil, apperr.NotFound("scan not found")
        }
        return nil, apperr.Internal("get scan", err)
    }
    return toScanResponse(scan), nil
}
```

### 3. Async Job Execution (Background)

```go
func (s *ScanService) Execute(scanID string) {
    ctx := context.Background()
    s.store.UpdateStatus(ctx, scanID, "running", 10, "")

    // Run checks
    findings, err := s.runChecks(ctx, scanID)

    if err != nil {
        s.store.UpdateStatus(ctx, scanID, "failed", 100, err.Error())
        return
    }

    s.store.SaveFindings(ctx, scanID, findings)
    s.store.UpdateStatus(ctx, scanID, "completed", 100, "")
}
```

---

## Transaction Pattern (When Needed)

Currently SQLite has no explicit transactions in code. If multi-table atomicity required:

```go
func (s *sqliteStore) CreateWithFindings(ctx context.Context, scan Scan, findings []Finding) error {
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil { return err }
    defer tx.Rollback()

    if _, err := tx.ExecContext(ctx, `INSERT INTO scans ...`); err != nil { return err }
    for _, f := range findings {
        if _, err := tx.ExecContext(ctx, `INSERT INTO findings ...`); err != nil { return err }
    }
    return tx.Commit()
}
```

---

## Testing with Fakes

```go
// internal/services/scan_service_test.go
type fakeScanStore struct {
    scans map[string]stores.Scan
    mu    sync.Mutex
}

func (f *fakeScanStore) Create(ctx context.Context, p stores.CreateScanParams) (stores.Scan, error) {
    f.mu.Lock(); defer f.mu.Unlock()
    scan := stores.Scan{ID: "test-id", TargetURL: p.TargetURL, ...}
    f.scans[scan.ID] = scan
    return scan, nil
}

func TestScanService_Create(t *testing.T) {
    fake := &fakeScanStore{scans: make(map[string]stores.Scan)}
    svc := services.NewScanService(fake, nil)
    res, err := svc.Create(context.Background(), dtos.CreateScanRequest{...})
    ...
}
```

---

## Rules Summary

1. **Store interface in service package**, implementation in store package.
2. **Service constructor takes interfaces**, not concrete types.
3. **All store methods take `context.Context` first**.
4. **Schema embedded and auto-migrated** on store construction.
5. **Services orchestrate; stores only CRUD**.
6. **Errors wrapped with `apperr.Internal/NotFound/Validation`** at service boundary.
7. **Async jobs launched via goroutine**; status polled via existing GET endpoints.