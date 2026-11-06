# WebTester — Dokumentasi Rekayasa Perangkat Lunak

WebTester adalah layanan pengujian keamanan web otomatis dan pengujian beban (load testing) yang ditulis dalam Go. Layanan ini menyediakan pemindaian keamanan (baseline pasif, fuzzing OpenAPI, full scan aktif via OWASP ZAP, pengujian BOLA), pengujian beban HTTP yang dapat dikonfigurasi, cache sesi Redis, autentikasi JWT, dan antarmuka web satu halaman (SPA) bawaan.

Spesifikasi produk tersedia di [specification.md](specification.md). Standar rekayasa dan arsitektur kode tersedia dalam direktori ini.

---

## Mulai Dari Sini

| Tujuan | Dokumen |
| --- | --- |
| Memahami fitur & spesifikasi produk | [specification.md](specification.md) |
| Konfigurasi & menjalankan aplikasi lokal | [development.id.md](development.id.md) |
| Arsitektur & batasan layer | [architecture.id.md](architecture.id.md) |
| Konvensi & aturan basis kode | [standards/standard.md](standards/standard.md) |
| Penanganan eror & kode status HTTP | [error_handling.id.md](error_handling.id.md), [standards/http_status.md](standards/http_status.md) |
| Alur kerja Git & commit | [git_workflow.id.md](git_workflow.id.md) |
| Aturan dokumentasi | [standards/documentation.md](standards/documentation.md) |

---

## Stack Teknologi

| Domain | Teknologi |
| --- | --- |
| Bahasa | Go ≥ 1.22 |
| Router | `go-chi/chi` v5 |
| Basis Data | SQLite (`modernc.org/sqlite` - bebas CGO) |
| Cache & Sesi | Redis (`github.com/redis/go-redis/v9`) |
| Autentikasi | JWT (`github.com/golang-jwt/jwt/v5`) + `bcrypt` |
| Pemindai Keamanan | OWASP ZAP (integrasi Docker) + probe bawaan |
| Validasi | `go-playground/validator/v10` |
| Antarmuka Web | Embedded SPA HTML/JS (`web/embed.go`) |

---

## Struktur Direktori

```
webtester/
├── cmd/
│   └── app/                  # Entrypoint aplikasi (injeksi DB, Redis, service, router)
├── internal/
│   ├── auth/                 # Token JWT, token store Redis, user store SQLite, middleware
│   ├── checks/               # Aturan pemindaian keamanan (pasif, probe BOLA)
│   ├── clients/zap/          # Klien runner OWASP ZAP Docker
│   ├── configs/              # Pemuat konfigurasi dari environment
│   ├── dtos/                 # Data transfer object request/response API
│   ├── handlers/             # Handler HTTP (Scan, LoadTest, Auth)
│   ├── loadtest/             # Mesin runner load test konkruen
│   ├── routes/               # Setup router Chi, grup rute, proteksi auth
│   ├── services/             # Logika bisnis (ScanService, LoadTestService, AuthService)
│   └── stores/               # Lapisan akses data (store SQLite scan, loadtest)
├── pkg/                      # Paket mandiri yang dapat digunakan kembali
│   ├── errors/               # Definisi & konstruktor AppError
│   ├── middleware/           # Access logging, panic recovery
│   └── response/             # Pembungkus standar response JSON
├── web/                      # Frontend SPA yang di-embed ke dalam binary
└── docs/                     # Dokumentasi teknis rekayasa
```
