# Spesifikasi Lengkap — Aplikasi Surat Waris

> Dokumen ini adalah spesifikasi utuh dan mandiri. Kerjakan dari **folder kosong**. Tidak ada kode atau konteks sebelumnya. Ikuti batasan teknis dengan ketat, dan kerjakan **bertahap per fase** (lihat bagian "Urutan Pengerjaan"), konfirmasi tiap fase sebelum lanjut.

---

## 1. Ringkasan

Aplikasi desktop **standalone** untuk membuat dan mencetak surat waris di kantor kelurahan. Petugas mengisi data sekali, aplikasi menyimpan ke database lokal dan menghasilkan **3 surat siap cetak**.

Aplikasi ini akan dipasang di **5 kelurahan berbeda, masing-masing memakai databasenya sendiri**. Distribusinya cukup dengan menyalin satu file `.exe`; identitas tiap kelurahan diisi lewat menu Pengaturan di dalam aplikasi.

Tiga surat yang dihasilkan dari satu input:
1. **Surat Keterangan Ahli Waris**
2. **Surat Kuasa Ahli Waris**
3. **Surat Pernyataan Ahli Waris**

---

## 2. Batasan Non-Negosiasi

- **Zero-install.** Hasil akhir HARUS satu file `.exe` yang tinggal double-click. Tidak ada installer, runtime terpisah, atau dependency native (DLL / C library).
- **Mesin spek rendah.** Idle RAM target < 50 MB. Hindari solusi berat.
- **Database lokal, file tunggal** (SQLite) di samping exe. Backup = copy satu file.
- **Cross-compile ke Windows harus jalan tanpa toolchain C** → `CGO_ENABLED=0` wajib bisa.
- **Surat dibuat POLOS** — tanpa kop/letterhead/logo di atas. Blok tanda tangan pejabat tetap ada di bawah.

---

## 3. Stack (sudah diputuskan — jangan diganti)

| Layer | Pilihan |
|---|---|
| Bahasa/Backend | **Go** (single static binary) |
| HTTP router | **`github.com/go-chi/chi/v5`** |
| Database | **`modernc.org/sqlite`** (pure-Go, TANPA CGO). JANGAN pakai `mattn/go-sqlite3` |
| Query layer | **`sqlc`** (typed) |
| Auth hashing | **`golang.org/x/crypto/bcrypt`** (pure-Go) + session cookie |
| Frontend input | **Svelte**, build → `dist/`, di-embed via `//go:embed` |
| Halaman cetak | **Go `html/template`** (server-rendered), TERPISAH dari Svelte |
| Ukuran kertas | **F4 / Folio (215 × 330 mm)**, CSS `@media print` |

---

## 4. Arsitektur

```
Satu binary Go (surat-waris.exe)
├── //go:embed frontend/dist   → UI input (Svelte SPA)
├── html/template               → 3 halaman cetak surat (F4, polos)
├── chi router + JSON API       → /api/*  (auth, pejabat, pengaturan, berkas)
├── modernc.org/sqlite          → surat-waris.db (dibuat otomatis di samping exe)
└── auto-open browser           → http://localhost:8080 saat start
```

- `GET /` + aset → serve Svelte `dist` dari `embed.FS`.
- `GET|POST /api/...` → JSON API.
- `GET /berkas/{id}/cetak` → halaman cetak (`html/template`), dibuka di tab baru, siap Ctrl+P.
- Halaman cetak **wajib** pakai `html/template` polos, bukan komponen Svelte (kontrol `@media print`, F4, page-break lebih andal).

---

## 5. Keputusan Final (fixed)

- **Satu berkas waris = satu kali input → 3 surat.**
- **Penomoran: SATU nomor surat dipakai untuk ketiga surat.**
- **Lock:** Surat Keterangan hanya bisa diproses **1× per NIK pewaris**. Data dasar berkas **dibekukan** setelah dibuat.
- **Setelah berkas dibuat, yang masih bisa diedit HANYA bagian Surat Kuasa** (penerima kuasa + daftar harta). Sisanya read-only.
- Pewaris **minimal 1, maksimal 2** (suami-istri). Saksi **tepat 2 orang**.
- **Penerima kuasa** = salah satu ahli waris yang dipilih (diberi kuasa oleh ahli waris lain).
- **Surat polos** — tanpa kop/logo. Blok TTD pejabat tetap di bawah.

---

## 6. Data Model

```sql
-- AUTH
CREATE TABLE users (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    username      TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    nama          TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'petugas',
    created_at    TEXT NOT NULL DEFAULT (datetime('now'))
);

-- PEJABAT (Lurah / Camat) — nama + NIP
CREATE TABLE pejabat (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    jabatan    TEXT NOT NULL,               -- 'lurah' | 'camat'
    nama       TEXT NOT NULL,
    nip        TEXT NOT NULL,
    aktif      INTEGER NOT NULL DEFAULT 1,   -- pejabat aktif yang dipakai di surat
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- PENGATURAN (identitas kelurahan; satu baris). Dipakai di isi surat & nomor,
-- BUKAN sebagai kop dekoratif. Tidak ada logo.
CREATE TABLE pengaturan (
    id             INTEGER PRIMARY KEY CHECK (id = 1),
    nama_kelurahan TEXT,
    kecamatan      TEXT,
    kabupaten      TEXT,
    provinsi       TEXT,
    format_nomor   TEXT           -- [VERIFIKASI] template format nomor surat
);

-- BERKAS WARIS (induk; 1 nomor untuk 3 surat)
CREATE TABLE berkas_waris (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    nomor_surat  TEXT NOT NULL UNIQUE,
    tahun        INTEGER NOT NULL,
    urutan       INTEGER NOT NULL,           -- counter per tahun
    tanggal      TEXT NOT NULL,
    tempat_tinggal_pewaris TEXT NOT NULL,
    penerima_kuasa_ahli_waris_id INTEGER REFERENCES ahli_waris(id),  -- EDITABLE
    status       TEXT NOT NULL DEFAULT 'terbit',
    created_by   INTEGER REFERENCES users(id),
    created_at   TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at   TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE UNIQUE INDEX idx_berkas_urutan ON berkas_waris(tahun, urutan);

-- PEWARIS (1-2 per berkas). NIK UNIQUE global = penegak LOCK.
CREATE TABLE pewaris (
    id                 INTEGER PRIMARY KEY AUTOINCREMENT,
    berkas_id          INTEGER NOT NULL REFERENCES berkas_waris(id),
    nama               TEXT NOT NULL,
    nik                TEXT NOT NULL UNIQUE,     -- <-- lock 1x per NIK pewaris
    tgl_meninggal      TEXT NOT NULL,
    no_surat_kematian  TEXT NOT NULL,
    tgl_surat_kematian TEXT NOT NULL
);

-- AHLI WARIS (list dinamis)
CREATE TABLE ahli_waris (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    berkas_id     INTEGER NOT NULL REFERENCES berkas_waris(id),
    nama          TEXT NOT NULL,
    nik           TEXT NOT NULL,
    umur          INTEGER,
    jenis_kelamin TEXT,           -- 'L' | 'P'
    agama         TEXT,
    alamat        TEXT,
    keterangan    TEXT            -- contoh: "Anak", "Istri"
);

-- SAKSI (tepat 2 per berkas)
CREATE TABLE saksi (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    berkas_id INTEGER NOT NULL REFERENCES berkas_waris(id),
    nama      TEXT NOT NULL,
    ttl       TEXT,               -- tempat, tanggal lahir
    alamat    TEXT,
    nik       TEXT,
    hubungan  TEXT                -- hubungan dengan almarhum
);

-- HARTA / yang dikuasakan (bagian Surat Kuasa, EDITABLE)
CREATE TABLE harta (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    berkas_id INTEGER NOT NULL REFERENCES berkas_waris(id),
    deskripsi TEXT NOT NULL
);
```

---

## 7. Aturan Bisnis

### 7.1 Lock (1× per NIK pewaris)
- `pewaris.nik` di-set `UNIQUE` global sebagai penegak utama di DB.
- Saat submit berkas baru: dalam satu transaksi, cek apakah salah satu NIK pewaris sudah ada. Jika ada → tolak dengan pesan jelas: `"Pewaris dengan NIK {nik} sudah pernah dibuatkan Surat Keterangan Ahli Waris."` dan jangan buat berkas.
- Karena 3 surat lahir dari 1 berkas, mengunci berkas otomatis mengunci Surat Keterangan.

### 7.2 Editability
- Setelah berkas dibuat, **hanya** ini yang boleh diubah:
  - `berkas_waris.penerima_kuasa_ahli_waris_id`
  - isi tabel `harta` (tambah / edit / hapus)
- Field lain (pewaris, ahli waris, saksi, tempat tinggal, nomor) **read-only**. **Enforce di server** (handler), bukan hanya di UI.

### 7.3 Nomor surat (1 nomor untuk 3 surat)
- Digenerate saat berkas dibuat, sekuensial per tahun (`MAX(urutan)+1` untuk tahun berjalan, dalam transaksi).
- `nomor_surat` `UNIQUE` sebagai pengaman; jika bentrok, retry.
- Format diambil dari `pengaturan.format_nomor`. **[VERIFIKASI] format resminya** (bulan biasanya angka Romawi).

### 7.4 Pejabat / tanda tangan
- Surat memakai pejabat `aktif = 1`. **[VERIFIKASI]** siapa TTD di masing-masing surat (mis. Lurah menerangkan, Camat mengetahui).

---

## 8. Halaman / Flow

1. **Login** → sesi.
2. **Daftar Berkas** — list berkas, cari by NIK/nama pewaris, tombol "Buat Baru".
3. **Form Buat Berkas** (Svelte):
   - Pewaris (1–2): Nama, NIK, Tgl Meninggal, No. Surat Kematian, Tgl Surat Kematian.
   - Ahli Waris (dinamis, tambah/hapus): Nama, NIK, Umur, Jenis Kelamin, Agama, Alamat, Keterangan.
   - Tempat tinggal pewaris.
   - Saksi (tepat 2): Nama, TTL, Alamat, NIK, Hubungan dgn alm.
   - Bagian Surat Kuasa: pilih **penerima kuasa** dari daftar ahli waris + daftar **harta** (dinamis).
   - Submit → validasi lock → simpan → generate nomor.
4. **Detail Berkas** — data read-only, KECUALI bagian Surat Kuasa (penerima kuasa + harta) yang bisa diedit. Tombol **Cetak**.
5. **Cetak** — `GET /berkas/{id}/cetak` merender **3 surat, masing-masing 1 halaman F4** dengan `page-break-after: always`. Siap Ctrl+P / Save PDF.
6. **Menu Pejabat** — CRUD lurah/camat (nama + NIP + status aktif).
7. **Menu Pengaturan** — identitas kelurahan + format nomor.

---

## 9. Auth & Seeder

- **Seeder** (jalan saat DB kosong, atau via flag `--seed`):
  - Buat user admin default (mis. username `admin`, password default) — **wajibkan ganti password saat login pertama** [VERIFIKASI].
  - Buat baris `pengaturan` (id=1) kosong siap diisi.
- Password disimpan sebagai **hash bcrypt** (jangan plaintext).
- Middleware chi: semua route kecuali `/login` wajib sesi valid.
- Auth bersifat lokal per-deployment (tiap kelurahan DB sendiri). **[VERIFIKASI]** perlu >1 role atau cukup satu.

---

## 10. Cetak (surat POLOS)

- 3 template `html/template` terpisah: Keterangan, Kuasa, Pernyataan. Digabung di satu halaman cetak dengan `page-break-after: always` antar surat.
- **Tanpa kop/logo di atas.** Boleh judul surat (mis. "SURAT KETERANGAN AHLI WARIS") + nomor. Isi memakai merge field: `{{.Pewaris}}`, `{{.AhliWaris}}`, `{{.Saksi}}`, `{{.Harta}}`, `{{.PenerimaKuasa}}`, `{{.Pengaturan}}`, `{{.Pejabat}}`, dll.
- **Blok tanda tangan di bawah** dari `pejabat` aktif (nama + NIP + jabatan).
- CSS: ukuran F4, margin wajar, font surat resmi (mis. Times/Serif). **[VERIFIKASI] margin/detail.**
- **Teks/legal isi surat = PLACEHOLDER.** JANGAN mengarang narasi hukum. Sediakan struktur + merge field; bagian narasi diisi user dari blangko resmi.

---

## 11. Build & Deployment

- Module: `surat-waris`. Database: `surat-waris.db`. Port default: `8080` (fallback ke port bebas jika terpakai). Binary: `surat-waris` / `surat-waris.exe`.
- Saat start: buka/buat `surat-waris.db` di direktori exe, `PRAGMA journal_mode=WAL;`, jalankan migrasi idempotent, lalu **auto-open browser default** ke `http://localhost:{PORT}`.
  - Windows: `exec.Command("cmd", "/c", "start", url)` · macOS: `open` · Linux: `xdg-open` (deteksi via `runtime.GOOS`).
- Build:
  ```bash
  # dev lokal
  go build -o surat-waris .

  # target Windows (deliverable) — wajib sukses tanpa C compiler
  CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-H windowsgui -s -w" -o surat-waris.exe .
  ```
  - `-H windowsgui` → tanpa jendela terminal hitam. `-s -w` → perkecil binary.
- Migrasi idempotent (cek versi skema) supaya upgrade exe tidak menyentuh data.
- Sediakan `Makefile` untuk build/dev/build-frontend.

---

## 12. Struktur Folder yang Diharapkan

```
surat-waris/
├── main.go
├── go.mod
├── Makefile
├── internal/
│   ├── db/          # schema.sql, queries.sql, sqlc output, migrasi
│   ├── auth/        # login, session, bcrypt
│   ├── handler/     # HTTP handlers (API + cetak)
│   ├── surat/       # business logic: nomor generator, lock, editability
│   └── server/      # router, embed, auto-open browser
├── frontend/        # project Svelte
│   └── dist/        # hasil build (di-embed)
└── templates/       # html/template 3 surat (F4, polos)
```

Jangan over-engineer di awal; boleh mulai ringkas lalu dirapikan.

---

## 13. Daftar [VERIFIKASI] (tanyakan / tandai sebagai asumsi)

- Format nomor surat resmi.
- Siapa pejabat penanda tangan tiap surat (Lurah / Camat / keduanya).
- Perlu lebih dari satu role user atau cukup satu.
- Wajib ganti password default saat login pertama atau tidak.
- Margin & font halaman cetak F4.
- Teks/narasi hukum ketiga surat (diisi user, jangan dikarang).

---

## 14. Urutan Pengerjaan (kerjakan per fase, konfirmasi tiap fase)

**Fase 0 — Scaffold**
1. `go mod init surat-waris`; tambah chi + `modernc.org/sqlite`.
2. Buka/buat DB (tanpa tabel dulu), `PRAGMA journal_mode=WAL`, `db.Ping()`.
3. HTTP server chi + route `GET /` "Hello World" + auto-open browser.
4. Pastikan `go build` dan cross-compile `.exe` (CGO off) sukses.

**Fase A — Fondasi data & auth**
5. Skema DB penuh + migrasi idempotent + sqlc.
6. Seeder (admin default + baris pengaturan).
7. Auth: login, logout, session middleware, bcrypt.

**Fase B — Master data**
8. CRUD Pejabat (lurah/camat, nama, NIP, aktif).
9. Menu Pengaturan (identitas kelurahan + format nomor).

**Fase C — Inti**
10. Data model berkas + generator nomor + **enforce lock** (uji via API dulu).
11. Form Svelte buat berkas (ahli waris & harta dinamis, pilih penerima kuasa).
12. Detail berkas + edit terbatas (hanya penerima kuasa + harta, enforce server-side).

**Fase D — Output**
13. 3 template cetak F4 polos (placeholder teks) + route `/berkas/{id}/cetak`.
14. Polish: pencarian berkas, validasi, pesan error lock yang jelas.

Selesaikan Fase 0 dulu dan tunjukkan hasilnya sebelum lanjut ke Fase A.
