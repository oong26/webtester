# List API Standards

## Pagination

All list endpoints support pagination via query parameters.

### Request

```
GET /api/v1/scans?limit=20&offset=0
GET /api/v1/load-tests?limit=50&offset=100
```

| Param | Type | Default | Min | Max |
| --- | --- | --- | --- | --- |
| `limit` | int | 20 | 1 | 100 |
| `offset` | int | 0 | 0 | — |

### Response Envelope

```json
{
  "data": [
    { "id": "...", "target_url": "...", "status": "completed", ... },
    { "id": "...", "target_url": "...", "status": "running", ... }
  ],
  "meta": {
    "limit": 20,
    "offset": 0,
    "total": 42
  }
}
```

---

## Filtering (Future)

When filters are added, use query parameters:

```
GET /api/v1/scans?status=completed&scan_type=full
GET /api/v1/load-tests?status=running
```

| Filter | Example |
| --- | --- |
| Exact match | `?status=completed` |
| Multiple values | `?status=queued,running` (comma-separated) |
| Date range | `?created_after=2024-01-01&created_before=2024-12-31` |

---

## Sorting (Future)

```
GET /api/v1/scans?sort=-created_at
GET /api/v1/load-tests?sort=status,created_at
```

| Format | Meaning |
| --- | --- |
| `sort=field` | Ascending |
| `sort=-field` | Descending |
| `sort=field1,-field2` | Multi-field |

Default sort: `-created_at` (newest first).

---

## Current Implementation

### Scan List

```go
// internal/handlers/scan_handler.go
func (h *ScanHandler) List(w http.ResponseWriter, r *http.Request) {
    limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
    offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
    res, err := h.svc.List(r.Context(), limit, offset)
    ...
}

// internal/services/scan_service.go
func (s *ScanService) List(ctx context.Context, limit, offset int) ([]dtos.ScanResponse, error) {
    if limit <= 0 { limit = 20 }
    if limit > 100 { limit = 100 }
    if offset < 0 { offset = 0 }
    scans, err := s.store.List(ctx, stores.ListScansParams{Limit: limit, Offset: offset})
    ...
}

// internal/stores/scan_store.go
func (s *sqliteScanStore) List(ctx context.Context, p ListScansParams) ([]Scan, error) {
    rows, err := s.db.QueryContext(ctx,
        `SELECT ... FROM scans ORDER BY created_at DESC LIMIT ? OFFSET ?`,
        p.Limit, p.Offset)
    ...
}
```

### Load Test List

Same pattern in `loadtest_handler.go`, `loadtest_service.go`, `loadtest_store.go`.

---

## Rules

1. **Always paginate.** No unbounded `SELECT *`.
2. **Default limit 20, max 100.** Enforced in service layer.
3. **Return `meta` with total count.** Enables client-side pagination UI.
4. **Order by `created_at DESC`** by default (newest first).
5. **No cursor pagination yet.** Offset-based is fine for current scale.
6. **Filters/sorting not implemented.** Add when needed, follow query param conventions above.

---

## Future: Export Endpoint

```
GET /api/v1/scans/export?format=csv&status=completed
GET /api/v1/load-tests/export?format=json
```

Returns file download (`Content-Disposition: attachment`).