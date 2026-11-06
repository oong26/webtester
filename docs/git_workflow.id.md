# Alur Kerja Git & Kontribusi

## Model Percabangan

- `main` / `master`: Rilis produksi. Bersih, stabil, siap deploy.
- `develop`: Cabang integrasi pengembangan utama.
- `feat/*`: Cabang fitur baru (contoh: `feat/auth-redis`).
- `fix/*`: Cabang perbaikan bug (contoh: `fix/loadtest-duration`).
- `docs/*`: Cabang dokumentasi (contoh: `docs/architecture`).

---

## Konvensi Pesan Commit

Mengikuti standar Conventional Commits:

```
<type>(<scope>): <ringkasan singkat dalam bentuk imperatif>

[penjelasan opsional mengenai alasan dan detail perubahan]
```

Tipe umum:
- `feat`: Fungsionalitas baru
- `fix`: Perbaikan bug
- `docs`: Perubahan dokumentasi
- `test`: Penambahan pengujian unit
- `chore`: Konfigurasi atau dependensi
