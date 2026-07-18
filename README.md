# SIWARIS — Sistem Informasi Surat Ahli Waris

Aplikasi desktop **standalone** untuk membuat & mencetak surat waris di kantor kelurahan.
Satu file `.exe` tinggal double-click — server Go + UI Svelte (embedded) + database SQLite lokal.
Satu kali input menghasilkan **3 surat** siap cetak:

1. Surat Keterangan Ahli Waris
2. Surat Kuasa Ahli Waris
3. Surat Pernyataan Ahli Waris

## Stack

| Layer | Pilihan |
|---|---|
| Backend | Go (single static binary, `CGO_ENABLED=0`) |
| Router | `github.com/go-chi/chi/v5` |
| Database | `modernc.org/sqlite` (pure-Go, tanpa CGO) — file `surat-waris.db` di samping exe |
| Query | `sqlc` (typed) — hasil generate di-commit |
| Frontend | Svelte 4 + Vite 5, build → `frontend/dist`, di-embed via `//go:embed` |
| Cetak | Go `html/template`, kertas **A4**, Times New Roman 12pt |

## Menjalankan (end user)

**Unduh `siwaris.exe` dari halaman [Releases](../../releases/latest)**, lalu double-click.
Browser default terbuka otomatis ke `http://localhost:8080` (otomatis pindah port bila
8080 terpakai). Database dibuat otomatis di samping exe.

Login awal: **admin** / **admin123** (wajib ganti password saat login pertama).
Panduan lengkap bergambar: [docs/panduan-penggunaan.md](docs/panduan-penggunaan.md).

## Build dari source

Butuh **Go 1.25+** dan **Node 20 + Yarn**. Frontend harus dibuild lebih dulu karena di-embed.

```bash
# 1. Build frontend → frontend/dist
cd frontend && yarn install && yarn build && cd ..

# 2a. Build lokal (ada jendela konsol untuk log)
go build -o siwaris.exe .

# 2b. Deliverable Windows (tanpa jendela konsol, binary kecil, tanpa C compiler)
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-H windowsgui -s -w" -o siwaris.exe .
```

Atau lewat Makefile: `make build-frontend && make build-win`.

### Regenerate query (sqlc)

Hasil sqlc sudah di-commit; regenerate hanya bila mengubah `schema.sql`/`queries.sql`:

```bash
make generate   # menjalankan ./.tools/sqlc.exe generate
```

## CI/CD & Rilis

`.github/workflows/build.yml` menjalankan test + build frontend + cross-compile Windows exe
(`siwaris.exe`) di setiap push/PR ke `main` (artifact, bisa juga dipicu manual via
*workflow_dispatch*).

**Merilis versi baru** (agar user bisa unduh langsung tanpa login GitHub):

```bash
git tag v1.0.0
git push origin v1.0.0
```

Push tag `v*` otomatis membuat **GitHub Release** berisi `siwaris.exe` beserta
catatan rilis — tautan unduhnya: `https://github.com/zennn08/surat-waris/releases/latest`.

## Struktur

```
main.go / web.go / templates.go   # entrypoint, embed frontend, embed template cetak
internal/db/                      # schema.sql, queries.sql, migrasi, hasil sqlc
internal/auth/                    # login, session, bcrypt, seeder
internal/handler/                 # API + halaman cetak
internal/surat/                   # generator nomor surat
frontend/                         # Svelte SPA (build → dist, di-embed)
templates/                        # 3 template cetak A4 (html/template)
```
