# Standar Penanganan Eror

## Tipe AppError

Didefinisikan di `pkg/errors/app_error.go`:

```go
type AppError struct {
    Status  int    // Kode status HTTP
    Code    string // Kode eror mesin (snake_case)
    Message string // Pesan eror manusiawi
    Err     error  // Eror asli (opsional)
}
```

---

## Fungsi Konstruktor

| Fungsi | Status | Kode | Digunakan Untuk |
| --- | --- | --- | --- |
| `Validation(msg)` | 422 | `validation_failed` | Validasi input / payload gagal |
| `NotFound(msg)` | 404 | `not_found` | Resource tidak ditemukan |
| `Internal(msg, err)` | 500 | `internal_error` | Eror tak terduga / basis data |
| `Unavailable(msg, err)` | 503 | `service_unavailable` | Ketergantungan mati sementara (Redis, ZAP) |
| `Unauthorized(msg)` | 401 | `unauthorized` | Token tidak ada, tidak valid, atau kedaluwarsa |
| `Conflict(msg)` | 409 | `conflict` | Konflik state (setup sudah dilakukan, data duplikat) |
