# Panduan Pengembangan WebTester

## Prasyarat

- Go ≥ 1.22 (disarankan Go 1.26 toolchain)
- Server Redis (lokal atau Docker)
- Docker (opsional, hanya dibutuhkan untuk Full Scan OWASP ZAP)
- SQLite3 CLI (opsional untuk memeriksa berkas basis data)

---

## Memulai Cepat

### 1. Jalankan Redis

```bash
docker run -d --name webtester-redis -p 6379:6379 redis:7-alpine
```

### 2. Jalankan Aplikasi

```bash
go run ./cmd/app
```

Output:
```
2026/11/07 02:00:00 webtester listening on :9090
```

### 3. Buka UI di Peramban

Kunjungi `http://localhost:9090`.
- Jika pertama kali dijalankan: Modal setup administrator awal akan muncul.
- Masukkan Nama, Email, dan Password (minimal 8 karakter) untuk membuat akun pertama.

---

## Perintah Pengujian & Kompilasi

```bash
# Build binary
go build -o bin/webtester ./cmd/app

# Jalankan seluruh unit test
go test -v ./...

# Jalankan pengujian auth saja
go test -v ./internal/auth

# Jalankan dengan race detector
go test -race ./...
```
