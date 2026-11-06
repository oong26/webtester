# Arsitektur WebTester

## Struktur Lapisan Tingkat Tinggi

```
┌─────────────────────────────────────────────────────────────────┐
│                         HTTP Request                            │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  pkg/middleware (Recover, AccessLog)                            │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  internal/routes (Router Chi, grup rute)                        │
│   ├─ /healthz (publik)                                          │
│   ├─ /api/v1/auth/* (publik: status, setup, login, refresh)     │
│   └─ /api/v1/* (dilindungi oleh auth.Middleware.Handler)        │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  internal/handlers (HTTP → Validasi DTO → Panggil Service)      │
│   ├─ AuthHandler    (Status, Setup, Login, Refresh, Logout, Me)│
│   ├─ ScanHandler    (List, Create, Get, Findings)              │
│   └─ LoadTestHandler(List, Create, Get, Cancel)                │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  internal/services (Logika bisnis & orkestrasi)                 │
│   ├─ AuthService   (user store + redis token store + JWT)      │
│   ├─ ScanService   (scan store + client ZAP + check engine)    │
│   └─ LoadTestService (loadtest store + runner konkuren)        │
└─────────────────────────────────────────────────────────────────┘
                              │
          ┌───────────────────┼───────────────────┐
          ▼                   ▼                   ▼
┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
│ internal/stores │ │ internal/clients│ │  internal/auth  │
│                 │ │                 │ │                 │
│ • ScanStore     │ │ • ZapClient     │ │ • UserStore     │
│ • LoadTestStore │ │                 │ │   (SQLite)      │
│                 │ │                 │ │ • RedisStore    │
│                 │ │                 │ │   (token)       │
│                 │ │                 │ │ • Middleware    │
└─────────────────┘ └─────────────────┘ └─────────────────┘
          │
          ▼
┌─────────────────────────────────────────────────────────────────┐
│  SQLite (webtester.db)  +  Redis (localhost:6379)              │
└─────────────────────────────────────────────────────────────────┘
```

---

## Aturan Impor Antarpaket

| Paket | Boleh Mengimpor |
| --- | --- |
| `cmd/app` | `internal/*`, `pkg/*`, `configs` |
| `internal/routes` | `internal/handlers`, `internal/services`, `internal/auth`, `pkg/middleware`, `web` |
| `internal/handlers` | `internal/services`, `internal/dtos`, `pkg/errors`, `pkg/response` |
| `internal/services` | `internal/stores`, `internal/clients`, `internal/auth`, `internal/dtos`, `pkg/errors` |
| `internal/auth` | `pkg/errors`, `pkg/response`, **TIDAK BUKAN** paket internal lainnya |
| `internal/stores` | **TIDAK BUKAN** paket internal lainnya (hanya `internal/dtos`, `modernc.org/sqlite`) |
| `internal/clients` | **TIDAK BUKAN** paket internal lainnya |
| `internal/dtos` | **TIDAK BUKAN** paket internal lainnya |
| `internal/checks` | **TIDAK BUKAN** paket internal lainnya |
| `internal/loadtest` | **TIDAK BUKAN** paket internal lainnya |
| `pkg/*` | **TIDAK BUKAN** paket internal apa pun (ketergantungan nol pada aplikasi) |

---

## Konfigurasi (Environment Variables)

| Variabel | Bawaan | Deskripsi |
| --- | --- | --- |
| `HTTP_ADDR` | `:9090` | Port listen HTTP |
| `DB_PATH` | `webtester.db` | Lokasi berkas SQLite |
| `ZAP_IMAGE` | `ghcr.io/zaproxy/zaproxy:stable` | Image container Docker ZAP |
| `ZAP_TIMEOUT_SECONDS` | `1800` | Batas waktu pemindaian aktif ZAP |
| `REDIS_ADDR` | `localhost:6379` | Alamat host Redis |
| `REDIS_PASSWORD` | (kosong) | Kata sandi Redis |
| `REDIS_DB` | `0` | Nomor indeks database logikal Redis |
