# API Naming Standards

## Base Path

All API endpoints are under `/api/v1`.

---

## Resources (Nouns, Plural)

| Resource | Path |
| --- | --- |
| Security scans | `/api/v1/scans` |
| Scan findings | `/api/v1/scans/{scanId}/findings` |
| Load tests | `/api/v1/load-tests` |
| Authentication | `/api/v1/auth` |

---

## Collection Endpoints

| Operation | Method | Path | Notes |
| --- | --- | --- | --- |
| List | `GET` | `/scans` | Query: `limit`, `offset` |
| Create | `POST` | `/scans` | Body: CreateScanRequest |
| List | `GET` | `/load-tests` | Query: `limit`, `offset` |
| Create | `POST` | `/load-tests` | Body: CreateLoadTestRequest |

---

## Single Resource Endpoints

| Operation | Method | Path | Notes |
| --- | --- | --- | --- |
| Get | `GET` | `/scans/{scanId}` | |
| Get findings | `GET` | `/scans/{scanId}/findings` | |
| Get | `GET` | `/load-tests/{id}` | |
| Cancel | `POST` | `/load-tests/{id}/cancel` | Action endpoint (verb in path) |

---

## Auth Endpoints

| Operation | Method | Path | Auth Required |
| --- | --- | --- | --- |
| Check setup status | `GET` | `/auth/status` | No |
| Initial setup | `POST` | `/auth/setup` | No |
| Login | `POST` | `/auth/login` | No |
| Refresh token | `POST` | `/auth/refresh` | No |
| Logout | `POST` | `/auth/logout` | **Yes** |
| Current user | `GET` | `/auth/me` | **Yes** |

---

## Naming Rules

| Rule | Correct | Incorrect |
| --- | --- | --- |
| Plural nouns | `/scans`, `/load-tests` | `/scan`, `/load_test` |
| Kebab-case | `/load-tests` | `/loadTests`, `/load_tests` |
| Nested collections | `/scans/{id}/findings` | `/findings?scan_id={id}` |
| Action verbs for non-CRUD | `POST /load-tests/{id}/cancel` | `DELETE /load-tests/{id}` (semantically different) |
| Auth group | `/auth/*` | `/login`, `/setup` at root |

---

## Query Parameters (List Endpoints)

| Parameter | Type | Default | Description |
| --- | --- | --- | --- |
| `limit` | int | 20 | Max items (max 100) |
| `offset` | int | 0 | Pagination offset |

Response envelope:
```json
{
  "data": [...],
  "meta": {
    "limit": 20,
    "offset": 0,
    "total": 42
  }
}
```

---

## Request/Response Naming (DTOs)

| Operation | Request Struct | Response Struct |
| --- | --- | --- |
| Create scan | `CreateScanRequest` | `ScanResponse` |
| List scans | (query params) | `[]ScanResponse` (wrapped) |
| Get scan | (path param) | `ScanResponse` |
| Get findings | (path param) | `[]FindingResponse` |
| Create load test | `CreateLoadTestRequest` | `LoadTestResponse` |

---

## HTTP Status Codes

See [http_status.md](http_status.md) for full mapping.

---

## Versioning

- Current: `v1` in path (`/api/v1/...`)
- Breaking changes → new version `v2` in parallel
- Deprecation: 90-day notice via `Deprecation` header + changelog