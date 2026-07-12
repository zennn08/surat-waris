# Spesifikasi Lengkap — Aplikasi Surat Waris (v2, FINAL)

> Dokumen mandiri. Kerjakan dari **folder kosong**. Tidak ada kode/konteks sebelumnya.
> Kerjakan **bertahap per fase** (Bagian 14), konfirmasi tiap fase sebelum lanjut.
> Teks ketiga surat pada Bagian 11 diambil dari **blangko resmi yang sedang dipakai** — pertahankan kata demi kata, jangan diubah/diperbaiki bahasanya.

---

## 1. Ringkasan

Aplikasi desktop **standalone** untuk membuat & mencetak surat waris di kantor kelurahan. Petugas mengisi data **sekali**, aplikasi menyimpan ke database lokal dan menghasilkan **3 surat siap cetak**:

1. **Surat Keterangan Ahli Waris**
2. **Surat Kuasa Ahli Waris**
3. **Surat Pernyataan Ahli Waris**

Akan dipasang di **5 kelurahan berbeda, masing-masing DB sendiri**. Distribusi = salin satu `.exe`. Identitas kelurahan diisi lewat menu Pengaturan.

---

## 2. Batasan Non-Negosiasi

- **Zero-install.** Deliverable = satu file `.exe`, tinggal double-click. Tanpa installer/runtime/DLL.
- **Mesin spek rendah.** Idle RAM < 50 MB.
- **Database lokal file tunggal** (SQLite) di samping exe. Backup = copy 1 file.
- **`CGO_ENABLED=0` wajib bisa** (cross-compile ke Windows tanpa C compiler).
- **Surat POLOS** — tanpa kop/letterhead/logo. Hanya judul surat + isi + blok tanda tangan.

---

## 3. Stack (fixed — jangan diganti)

| Layer | Pilihan |
|---|---|
| Backend | **Go** (single static binary) |
| Router | **`github.com/go-chi/chi/v5`** |
| Database | **`modernc.org/sqlite`** (pure-Go). JANGAN `mattn/go-sqlite3` |
| Query | **`sqlc`** |
| Auth | **`golang.org/x/crypto/bcrypt`** + session cookie |
| UI input | **Svelte** → `dist/`, di-embed via `//go:embed` |
| Halaman cetak | **Go `html/template`** (server-rendered), TERPISAH dari Svelte |
| Kertas | **F4 / Folio (215 × 330 mm)**, CSS `@media print`, font serif (Arial/Times — [VERIFIKASI]) |

---

## 4. Arsitektur

```
surat-waris.exe (satu binary)
├── //go:embed frontend/dist  → UI input (Svelte SPA)
├── html/template             → 3 halaman cetak (F4, polos)
├── chi + JSON API            → /api/* (auth, pejabat, pengaturan, berkas)
├── modernc.org/sqlite        → surat-waris.db (auto-create di samping exe)
└── auto-open browser         → http://localhost:8080
```

- `GET /` + aset → Svelte dari `embed.FS`
- `/api/...` → JSON API
- `GET /berkas/{id}/cetak` → 3 surat via `html/template`, tab baru, siap Ctrl+P
- Halaman cetak **wajib** `html/template` polos (bukan komponen Svelte) — kontrol `@media print`, F4, page-break lebih andal.

---

## 5. Keputusan Final

- **Satu berkas = satu kali input → 3 surat.**
- **Lock:** hanya boleh **1× per NIK pewaris**. Data dasar dibekukan setelah dibuat.
- **Setelah dibuat, yang boleh diedit HANYA bagian Surat Kuasa** (penerima kuasa + daftar kuasa/harta). Sisanya read-only, **enforce di server-side**.
- Pewaris **min 1, max 2** (suami-istri). Saksi **tepat 2**.
- **Penerima kuasa** = salah satu ahli waris; ahli waris lainnya otomatis menjadi **pemberi kuasa**.
- **Surat polos**, tanpa kop.

---

## 6. Data Model

```sql
CREATE TABLE users (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    username      TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    nama          TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'petugas',
    created_at    TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Pejabat penanda tangan (Lurah & Camat)
CREATE TABLE pejabat (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    jabatan    TEXT NOT NULL,              -- 'lurah' | 'camat'
    nama       TEXT NOT NULL,              -- termasuk gelar, mis. "SYAFRIANDI,S.Sos.,M.Si"
    nip        TEXT NOT NULL,              -- mis. "19740223 200112 1 005"
    aktif      INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Identitas wilayah (1 baris). Dipakai di ISI surat & nomor register, BUKAN kop.
CREATE TABLE pengaturan (
    id             INTEGER PRIMARY KEY CHECK (id = 1),
    nama_kelurahan TEXT,    -- "Teluk Binjai"
    kecamatan      TEXT,    -- "Dumai Timur"
    kota           TEXT,    -- "Dumai"  (dipakai utk "Dumai, 22 Juni 2026" & "Dibuat di")
    kode_kecamatan TEXT,    -- "DT"     (utk Reg. No Camat)
    kode_kelurahan TEXT,    -- "TB"     (utk Reg. No Lurah)
    instansi_kematian TEXT  -- default: "Dinas Kependudukan dan Pencatatan Sipil Kota Dumai"
);

-- BERKAS induk
CREATE TABLE berkas_waris (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    tahun         INTEGER NOT NULL,
    urutan        INTEGER NOT NULL,        -- counter per tahun
    reg_no_camat  TEXT NOT NULL,           -- "88/SKAW/DT/2026"
    reg_no_lurah  TEXT NOT NULL,           -- "88/SKAW/TB-DT/2026"
    tanggal_reg_camat TEXT,
    tanggal_reg_lurah TEXT,
    tanggal_surat TEXT NOT NULL,           -- "22 Juni 2026" (tgl di blok TTD & Surat Pernyataan)
    tempat_tinggal_pewaris TEXT NOT NULL,  -- "Jl. Merdeka Baru RT.007"
    penerima_kuasa_id INTEGER REFERENCES ahli_waris(id),   -- EDITABLE
    status        TEXT NOT NULL DEFAULT 'terbit',
    created_by    INTEGER REFERENCES users(id),
    created_at    TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at    TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE UNIQUE INDEX idx_berkas_urutan ON berkas_waris(tahun, urutan);

-- PEWARIS (1-2). NIK UNIQUE = penegak LOCK.
CREATE TABLE pewaris (
    id                 INTEGER PRIMARY KEY AUTOINCREMENT,
    berkas_id          INTEGER NOT NULL REFERENCES berkas_waris(id),
    nama               TEXT NOT NULL,          -- "FAOZIDUHU TAFONAO"
    nik                TEXT NOT NULL UNIQUE,   -- <-- LOCK 1x per NIK
    status             TEXT NOT NULL,          -- 'suami' | 'istri'  → label "(Suami)"/"(Istri)"
    tgl_meninggal      TEXT NOT NULL,          -- "27 Desember 2023"
    instansi_kematian  TEXT NOT NULL,          -- default dari pengaturan, bisa dioverride
    no_surat_kematian  TEXT NOT NULL,          -- "1472-KM-05012024-0005"
    tgl_surat_kematian TEXT NOT NULL,          -- "5 Januari 2024"
    urutan             INTEGER NOT NULL DEFAULT 1   -- urutan tampil (suami dulu, lalu istri)
);

-- AHLI WARIS
CREATE TABLE ahli_waris (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    berkas_id     INTEGER NOT NULL REFERENCES berkas_waris(id),
    urutan        INTEGER NOT NULL,
    nama          TEXT NOT NULL,
    nik           TEXT NOT NULL,
    umur          INTEGER,
    jenis_kelamin TEXT,        -- 'L' | 'P'
    agama         TEXT,
    alamat        TEXT,
    keterangan    TEXT,        -- "Anak" | "Istri" | dll
    -- field TAMBAHAN, hanya wajib bila jadi penerima kuasa (Surat 2):
    tempat_lahir  TEXT,        -- "Doli-doli"
    tgl_lahir     TEXT,        -- "08-12-1994"
    pekerjaan     TEXT         -- "Pelajar/Mahasiswa"
);

-- SAKSI (tepat 2)
CREATE TABLE saksi (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    berkas_id    INTEGER NOT NULL REFERENCES berkas_waris(id),
    urutan       INTEGER NOT NULL,   -- 1 | 2
    nama         TEXT NOT NULL,
    tempat_lahir TEXT,
    tgl_lahir    TEXT,
    alamat       TEXT,               -- bisa multi-baris
    nik          TEXT,
    hubungan     TEXT                -- "Tetangga" | "Famili"
);

-- ISI KUASA (bagian Surat Kuasa — EDITABLE)
CREATE TABLE kuasa_item (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    berkas_id INTEGER NOT NULL REFERENCES berkas_waris(id),
    urutan    INTEGER NOT NULL,
    deskripsi TEXT NOT NULL   -- teks bebas, lihat contoh di Bagian 11.2
);
```

---

## 7. Aturan Bisnis

### 7.1 Lock (1× per NIK pewaris)
- `pewaris.nik` `UNIQUE` global = penegak di level DB.
- Saat submit berkas baru, dalam satu transaksi: cek NIK pewaris. Jika sudah ada → **tolak**, jangan buat berkas. Pesan: `"Pewaris dengan NIK {nik} sudah pernah dibuatkan Surat Keterangan Ahli Waris (Reg. No. {reg_no_camat})."`
- Berlaku walau hanya salah satu dari 2 NIK yang bentrok.

### 7.2 Editability
- Setelah berkas dibuat, **hanya** ini yang boleh diubah:
  - `berkas_waris.penerima_kuasa_id`
  - isi `kuasa_item` (tambah/edit/hapus)
  - field pelengkap penerima kuasa (`tempat_lahir`, `tgl_lahir`, `pekerjaan`) pada ahli waris yang terpilih
- **Semua field lain read-only.** Enforce di handler, bukan hanya di UI.

### 7.3 Nomor register (DUA nomor, dari SATU urutan)
Satu berkas menghasilkan satu `urutan` per tahun, dipakai untuk membentuk **dua** nomor register:

- **Camat:** `{urutan}/SKAW/{kode_kecamatan}/{tahun}` → contoh `88/SKAW/DT/2026`
- **Lurah:** `{urutan}/SKAW/{kode_kelurahan}-{kode_kecamatan}/{tahun}` → contoh `88/SKAW/TB-DT/2026`

Digenerate saat berkas dibuat: `MAX(urutan)+1` untuk tahun berjalan, di dalam transaksi. Index `UNIQUE(tahun, urutan)` sebagai pengaman; jika bentrok, retry.

**[VERIFIKASI]** Apakah nomor Camat & Lurah selalu sama angkanya, atau punya counter terpisah? Di blangko contoh keduanya `88`. Asumsikan **sama** sampai dikoreksi.

### 7.4 Penanda tangan tiap surat
| Surat | Camat | Lurah |
|---|---|---|
| 1. Keterangan Ahli Waris | ✅ | ✅ |
| 2. Kuasa Ahli Waris | ✅ | ✅ |
| 3. Pernyataan Ahli Waris | ❌ | ✅ |

Ambil dari `pejabat` dengan `aktif = 1`.

### 7.5 Turunan otomatis (jangan minta input)
- **Jumlah anak + terbilang**: `dikaruniai 4 (Empat) orang anak` → hitung dari jumlah ahli waris, konversi ke terbilang Indonesia (Satu, Dua, Tiga, Empat, …). Buat helper `terbilang(n int) string`.
- **Pemberi kuasa** (Surat 2) = semua ahli waris **KECUALI** penerima kuasa.
- **Tanggal Surat Pernyataan** yang dirujuk di Surat 1 = `tanggal_surat` berkas yang sama.

---

## 8. Flow Aplikasi

1. **Login**
2. **Daftar Berkas** — list, cari by NIK/nama pewaris, tombol "Buat Baru"
3. **Form Buat Berkas** (Svelte, satu form panjang):
   - Pewaris (1–2): Nama, NIK, status (Suami/Istri), Tgl Meninggal, Instansi penerbit, No. Surat Kematian, Tgl Surat Kematian
   - Tempat tinggal terakhir pewaris
   - Ahli Waris (dinamis): Nama, NIK, Umur, L/P, Agama, Alamat, Keterangan
   - Saksi (tepat 2): Nama, Tempat & Tgl Lahir, Alamat, NIK, Hub dengan Alm
   - Bagian Surat Kuasa: pilih **penerima kuasa** dari daftar ahli waris → munculkan field tambahan (Tempat/Tgl Lahir, Pekerjaan) + daftar **item kuasa** (dinamis, textarea)
   - Tanggal surat
   - Submit → validasi lock → simpan → generate `urutan` + 2 reg no
4. **Detail Berkas** — read-only, KECUALI bagian Surat Kuasa yang bisa diedit. Tombol **Cetak**.
5. **Cetak** — `GET /berkas/{id}/cetak` → 3 surat, tiap surat 1 halaman F4, `page-break-after: always`.
6. **Menu Pejabat** — CRUD Lurah/Camat (nama, NIP, aktif)
7. **Menu Pengaturan** — identitas wilayah + kode + instansi kematian default

---

## 9. Auth & Seeder

- **Seeder** (jalan saat DB kosong / flag `--seed`):
  - User admin default (`admin` + password default) — **[VERIFIKASI]** wajib ganti saat login pertama?
  - Baris `pengaturan` (id=1) kosong siap diisi
- Password = **hash bcrypt**, jangan plaintext.
- Middleware chi: semua route kecuali `/login` wajib sesi valid.
- **[VERIFIKASI]** perlu >1 role atau cukup satu.

---

## 10. Build & Deployment

- Module `surat-waris` · DB `surat-waris.db` · Port default `8080` (fallback jika terpakai)
- Startup: buka/buat DB → `PRAGMA journal_mode=WAL` → migrasi **idempotent** → auto-open browser
  - Windows `cmd /c start` · macOS `open` · Linux `xdg-open` (deteksi `runtime.GOOS`)
- Build:
  ```bash
  go build -o surat-waris .
  CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-H windowsgui -s -w" -o surat-waris.exe .
  ```
- Migrasi idempotent → upgrade exe tidak menyentuh data.
- Sediakan `Makefile`.

---

## 11. TEMPLATE SURAT (teks resmi — JANGAN diubah kata-katanya)

Aturan umum:
- **Tanpa kop/logo.** Judul di tengah, **bold + underline**, huruf kapital.
- Font serif, ukuran ~11pt, isi **justify**. Tabel bergaris penuh.
- Merge field ditandai `{{...}}` di bawah — implementasikan dengan `html/template`.
- **Pertahankan typo/inkonsistensi asli** (lihat Bagian 12) kecuali user menyuruh sebaliknya.

Notasi bantu:
- `{{P1}}` = pewaris ke-1 (mis. Suami), `{{P2}}` = pewaris ke-2 (mis. Istri)
- `{{PEWARIS_FRASA}}` = frasa gabungan pewaris. Bila 2 pewaris:
  `Almarhum {{P1.Nama}} (Suami) dan Almarhumah {{P2.Nama}} (Istri)`
  Bila 1 pewaris: `Almarhum {{P1.Nama}}` (atau `Almarhumah` bila perempuan).
  **Buat helper agar frasa ini konsisten di seluruh surat.**

---

### 11.1 SURAT KETERANGAN AHLI WARIS

**Judul:** `SURAT KETERANGAN AHLI WARIS`

**Paragraf 1:**
> Yang bertanda tangan dibawah ini, ahli waris dari {{PEWARIS_FRASA}} sanggup diangkat sumpah, bahwa {{PEWARIS_FRASA}} tempat tinggal terakhir di {{TempatTinggal}} Kelurahan {{Kelurahan}} Kecamatan {{Kecamatan}}, Almarhum {{P1.Nama}} (Suami) meninggal dunia tepatnya pada tanggal {{P1.TglMeninggal}} sesuai dengan Surat Kematian yang dikeluarkan oleh {{P1.InstansiKematian}} Nomor : {{P1.NoSuratKematian}} Tanggal {{P1.TglSuratKematian}}. dan Almarhumah {{P2.Nama}} (Istri) meninggal dunia tepatnya pada tanggal {{P2.TglMeninggal}} sesuai dengan Surat Kematian yang dikeluarkan oleh {{P2.InstansiKematian}} Nomor : {{P2.NoSuratKematian}} tanggal {{P2.TglSuratKematian}}.

> *(Bila hanya 1 pewaris, hilangkan kalimat pewaris ke-2.)*

**Paragraf 2:**
> Dari perkawinan {{PEWARIS_FRASA}}, dikaruniai {{JumlahAhliWaris}} ({{Terbilang}}) orang anak yaitu, dengan susunan anggota keluarga yaitu :

**Tabel ahli waris (SEMUA ahli waris):**

| No | N a m a | No. Identitas/NIK | Umur | L/P | Agama | Alamat | Ket |
|----|---------|-------------------|------|-----|-------|--------|-----|
| 1. | ANGERAGO TAFONAO | 1472021010950014 | 31 | L | Kristen | Jl. Sabar Menanti Kel. Bumi Ayu Kec. Dumai Selatan | Anak |

**Paragraf 3:**
> Selain dari nama-nama diatas, tidak ada ahli waris lain dari {{PEWARIS_FRASA}}, berdasarkan Surat Pernyataan Ahli Waris {{TanggalSurat}}.

**Paragraf 4:**
> Surat Keterangan Ahli Waris dibuat dihadapan 2 (dua) orang saksi yang turut menandatangani yaitu :

**Blok saksi (2 kolom berdampingan):**
```
1. N a m a          : {{Saksi1.Nama}}          2. Nama             : {{Saksi2.Nama}}
   Tempat/Tgl. Lahir: {{Saksi1.TTL}}              Tempat/Tgl. Lahir: {{Saksi2.TTL}}
   Alamat           : {{Saksi1.Alamat}}           Alamat           : {{Saksi2.Alamat}}
   NIK              : {{Saksi1.NIK}}              NIK              : {{Saksi2.NIK}}
   Hub dengan Alm   : {{Saksi1.Hubungan}}         Hub dengan Alm   : {{Saksi2.Hubungan}}
```

**Paragraf penutup:**
> Demikian Surat Keterangan ini kami buat dan apabila dikemudian hari ternyata keterangan kami ini tidak benar, maka Surat Keterangan ini dapat dijadikan dasar penuntutan hukum baik secara Pidana maupun Perdata berdasarkan Peraturan Perundang-undangan yang berlaku di Negara Republik Indonesia tanpa melibatkan unsur pemerintah dan atau pihak lainnya.

**Blok tanda tangan (2 kolom):**
```
Kami para Saksi-saksi :                {{Kota}}, {{TanggalSurat}}
                                       Kami Para Ahli Waris,
1. {{Saksi1.Nama}}
2. {{Saksi2.Nama}}                     1. {{AhliWaris1.Nama}}
                                       2. {{AhliWaris2.Nama}}
                                       ... (semua ahli waris)
```
*(Beri jarak vertikal cukup untuk tanda tangan basah & meterai.)*

**Garis pemisah** (baris `====...` melintang penuh)

**Dasar hukum:**
> Diketahui oleh Camat {{Kecamatan}} dan Lurah {{Kelurahan}} berdasarkan Peraturan Menteri Negara Agraria Kepala BPN Nomor : 03 Tahun 1997 Bagian Kelima Pasal 111 Angka (1) Huruf C.

**Blok pejabat (2 kolom):**
```
Mengetahui :                           Mengetahui :
Reg. No. : {{RegNoCamat}}              Reg. No. : {{RegNoLurah}}
Tanggal  : {{TanggalRegCamat}}         Tanggal  : {{TanggalRegLurah}}
        C A M A T,                             L U R A H,


{{Camat.Nama}}                         {{Lurah.Nama}}
NIP. {{Camat.NIP}}                     NIP. {{Lurah.NIP}}
```

---

### 11.2 SURAT KUASA AHLI WARIS

**Judul:** `SURAT KUASA AHLI WARIS`

**Paragraf 1:**
> Kami yang bertanda tangan dibawah ini adalah Ahli Waris {{PEWARIS_FRASA}}, bersama ini menerangkan dengan sesungguhnya tanpa ada paksaan dari pihak manapun dan sanggup diangkat sumpahnya {{PEWARIS_FRASA}}, bertempat tinggal terakhir di {{TempatTinggal}}, Kelurahan {{Kelurahan}} Kecamatan {{Kecamatan}}, dengan data penerima kuasa sebagai berikut :

**Tabel — PENTING: berisi ahli waris PEMBERI KUASA saja (semua ahli waris KECUALI penerima kuasa).** Kolom sama dengan Surat 1: `No | N a m a | No. Identitas/NIK | Umur | L/P | Agama | Alamat | Ket`

**Blok penerima kuasa:**
```
Pihak Pertama Penerima Kuasa kepada :
    Nama                     : {{PenerimaKuasa.Nama}}
    Tempat / Tgl. Lahir      : {{PenerimaKuasa.TempatLahir}},{{PenerimaKuasa.TglLahir}}
    Nomor Identitas          : {{PenerimaKuasa.NIK}}
    Pekerjaan                : {{PenerimaKuasa.Pekerjaan}}
    Alamat                   : {{PenerimaKuasa.Alamat}}
    Hubungan dengan Almarhum : {{PenerimaKuasa.Keterangan}}
    Dalam hal ini disebut Pihak Pertama (Penerima Kuasa).
```
*("Dalam hal ini disebut Pihak Pertama (Penerima Kuasa)." — teks ini bergaris bawah pada bagian "Pihak Pertama".)*

**Paragraf kuasa:**
> Pihak Pertama dengan akal sehat dan pikiran yang sehat tanpa paksaan dan tidak dipengaruhi dari pihak manapun juga, berdasarkan hasil musyawarah/mufakat dari ahli waris, dengan ini memberikan kuasa untuk:

**Daftar item kuasa** (dari `kuasa_item`, bisa >1, teks bebas). Contoh isi asli:
> Pengurusan administrasi kartu BPJS Ketenagakerjaan dengan Nomor 23137224459 an. **Almarhummah SARITISA TAFONAO** Kepada an. **ANGERAGO TAFONAO (Anak)**

**Blok saksi** — sama persis dengan Surat 1, didahului kalimat:
> Surat Keterangan Ahli Waris dibuat dihadapan 2 (dua) orang saksi yang turut menandatangani yaitu :

*(Ya, di blangko asli Surat Kuasa memang tertulis "Surat Keterangan Ahli Waris". Pertahankan.)*

**Paragraf penutup:**
> Demikian Surat Kuasa ini kami buat dan ditandatangani oleh masing-masing pihak dihadapan 2 (dua) orang saksi.

**Blok tempat/tanggal (rata kanan):**
```
Dibuat di    : {{Kota}}
Pada tanggal : {{TanggalSurat}}
```

**Blok saksi + tanda tangan:**
```
Kami para Saksi :
1. {{Saksi1.Nama}}
2. {{Saksi2.Nama}}


Yang Menerima Kuasa,                   Yang Memberi Kuasa,
      PIHAK II                               PIHAK I

{{PenerimaKuasa.Nama}}                 1. {{PemberiKuasa1.Nama}}
                                       2. {{PemberiKuasa2.Nama}}
                                       ... (semua pemberi kuasa)
```

**Blok pejabat** — sama dengan Surat 1 (Camat kiri + Lurah kanan, dengan Reg. No & Tanggal).

---

### 11.3 SURAT PERNYATAAN AHLI WARIS

**Judul:** `SURAT PERNYATAAN AHLI WARIS`

**Pembuka:**
> Saya yang bertanda tangan dibawah ini :

**Tabel ahli waris** — SEMUA ahli waris, kolom sama dengan Surat 1.

**Paragraf 1:**
> Dengan ini menyatakan bahwa memang benar dari {{PEWARIS_FRASA}}, dikaruniai {{JumlahAhliWaris}} ({{Terbilang}}) orang anak yaitu :

**Daftar bernomor nama ahli waris:**
```
1. {{AhliWaris1.Nama}}
2. {{AhliWaris2.Nama}}
...
```

**Paragraf 2:**
> Selain dari Ahli Waris tersebut, tidak ada lagi Ahli Waris yang lain dari {{PEWARIS_FRASA}}.

**Paragraf penutup:**
> Demikian Surat Keterangan ini kami buat dan apabila dikemudian hari ternyata keterangan kami ini tidak benar, maka Surat Keterangan ini dapat dijadikan dasar penuntutan hukum baik secara Pidana maupun Perdata berdasarkan Peraturan Perundang-undangan yang berlaku di Negara Republik Indonesia tanpa melibatkan unsur pemerintah dan atau pihak lainnya.

*(Ya, Surat Pernyataan pun memakai frasa "Surat Keterangan" di penutup. Pertahankan.)*

**Blok tanda tangan (2 kolom):**
```
Kami para Saksi-saksi :                {{Kota}}, {{TanggalSurat}}
                                       Kami Para Ahli Waris,
1. {{Saksi1.Nama}}
2. {{Saksi2.Nama}}                     1. {{AhliWaris1.Nama}}
                                       2. {{AhliWaris2.Nama}}
                                       ...
```

**Blok pejabat:** **HANYA LURAH** (tanpa Camat, tanpa Reg. No pada blangko contoh).
```
                    Mengetahui,
                     L U R A H,


                  {{Lurah.Nama}}
                  NIP. {{Lurah.NIP}}
```
**[VERIFIKASI]** apakah Surat Pernyataan benar-benar tanpa Reg. No.

---

## 12. Inkonsistensi pada Blangko Asli (SENGAJA dipertahankan)

Jangan "memperbaiki" hal-hal ini tanpa persetujuan user — ini muncul di blangko resmi yang sedang dipakai:

1. **Surat Kuasa & Surat Pernyataan** memakai frasa **"Surat Keterangan"** di beberapa kalimat. Biarkan.
2. **Surat Kuasa**: penerima kuasa disebut **"Pihak Pertama"** di narasi, tetapi di blok tanda tangan tertulis **"PIHAK II"** (dan pemberi kuasa = "PIHAK I"). Biarkan seperti aslinya.
3. Penulisan "Almarhummah" (dobel m) muncul di isi kuasa. Itu teks bebas dari user, bukan template.
4. Tabel Surat Kuasa **mengecualikan** penerima kuasa — ini disengaja, bukan bug.

**[VERIFIKASI]** Konfirmasi ke user apakah 1 & 2 memang ingin dipertahankan atau diperbaiki di aplikasi baru.

---

## 13. Daftar [VERIFIKASI]

- Reg. No Camat & Lurah selalu angka sama? (asumsi: ya)
- Surat Pernyataan benar tanpa Reg. No?
- Inkonsistensi "Surat Keterangan" di Surat Kuasa/Pernyataan → pertahankan atau perbaiki?
- "PIHAK I / PIHAK II" → pertahankan atau perbaiki?
- Font & margin cetak F4 yang tepat.
- Wajib ganti password default saat login pertama?
- Perlu >1 role user?
- Bagaimana narasi bila **hanya 1 pewaris** (belum ada contoh blangkonya)?
- Bagaimana bila ahli waris bukan hanya "Anak" (mis. ada Istri) — frasa "dikaruniai N orang anak" masih cocok?

---

## 14. Urutan Pengerjaan (per fase, konfirmasi tiap fase)

**Fase 0 — Scaffold**
1. `go mod init surat-waris`; tambah chi + `modernc.org/sqlite`.
2. Buka/buat DB (belum ada tabel), `PRAGMA journal_mode=WAL`, `db.Ping()`.
3. Server chi + `GET /` "Hello World" + auto-open browser.
4. Pastikan `go build` **dan** cross-compile `.exe` (CGO off) sukses.

**Fase A — Fondasi**
5. Skema DB penuh + migrasi idempotent + sqlc.
6. Seeder (admin default + baris pengaturan).
7. Auth: login, logout, session middleware, bcrypt.

**Fase B — Master data**
8. CRUD Pejabat (lurah/camat).
9. Menu Pengaturan (identitas wilayah + kode + instansi kematian).

**Fase C — Inti**
10. Model berkas + generator `urutan` & 2 reg no + **enforce lock** + helper `terbilang()`. Uji via API dulu.
11. Form Svelte buat berkas (ahli waris & item kuasa dinamis, pilih penerima kuasa → munculkan field TTL/Pekerjaan).
12. Detail berkas + edit terbatas (penerima kuasa + item kuasa), enforce server-side.

**Fase D — Cetak**
13. 3 template `html/template` sesuai Bagian 11, F4, `page-break-after: always`, route `/berkas/{id}/cetak`.
14. Uji cetak/Save-as-PDF, banding-kan dengan blangko asli, rapikan margin & jarak TTD.
15. Polish: pencarian berkas, validasi, pesan error lock.

**Selesaikan Fase 0 dan tunjukkan hasilnya sebelum lanjut.**
