-- Skema Surat Waris v2 (sesuai SPEC-surat-waris-v2 §6). DDL idempotent (IF NOT EXISTS)
-- agar upgrade exe tidak menyentuh data. Versi skema dilacak via PRAGMA user_version.

-- AUTH
CREATE TABLE IF NOT EXISTS users (
    id                   INTEGER PRIMARY KEY AUTOINCREMENT,
    username             TEXT NOT NULL UNIQUE,
    password_hash        TEXT NOT NULL,
    nama                 TEXT NOT NULL,
    role                 TEXT NOT NULL DEFAULT 'petugas',
    must_change_password INTEGER NOT NULL DEFAULT 0,   -- wajib ganti password saat login pertama
    created_at           TEXT NOT NULL DEFAULT (datetime('now'))
);

-- PEJABAT (Lurah / Camat) — nama (termasuk gelar) + NIP
CREATE TABLE IF NOT EXISTS pejabat (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    jabatan    TEXT NOT NULL,               -- 'lurah' | 'camat'
    nama       TEXT NOT NULL,
    nip        TEXT NOT NULL,
    aktif      INTEGER NOT NULL DEFAULT 1,   -- pejabat aktif yang dipakai di surat
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- PENGATURAN (identitas wilayah; satu baris). Dipakai di ISI surat & nomor register,
-- BUKAN kop. Tidak ada logo.
CREATE TABLE IF NOT EXISTS pengaturan (
    id                INTEGER PRIMARY KEY CHECK (id = 1),
    nama_kelurahan    TEXT,   -- "Teluk Binjai"
    kecamatan         TEXT,   -- "Dumai Timur"
    kota              TEXT,   -- "Dumai"  (utk "Dumai, 22 Juni 2026" & "Dibuat di")
    kode_kecamatan    TEXT,   -- "DT"     (Reg. No Camat)
    kode_kelurahan    TEXT,   -- "TB"     (Reg. No Lurah)
    instansi_kematian TEXT    -- default instansi penerbit surat kematian
);

-- BERKAS WARIS (induk; 1 urutan → 2 reg no untuk 3 surat)
CREATE TABLE IF NOT EXISTS berkas_waris (
    id                           INTEGER PRIMARY KEY AUTOINCREMENT,
    tahun                        INTEGER NOT NULL,
    urutan                       INTEGER NOT NULL,           -- counter per tahun
    reg_no_camat                 TEXT NOT NULL,              -- "88/SKAW/DT/2026"
    reg_no_lurah                 TEXT NOT NULL,              -- "88/SKAW/TB-DT/2026"
    tanggal_reg_camat            TEXT,
    tanggal_reg_lurah            TEXT,
    tanggal_surat                TEXT NOT NULL,              -- tgl di blok TTD & Surat Pernyataan
    tempat_tinggal_pewaris       TEXT NOT NULL,
    penerima_kuasa_ahli_waris_id INTEGER REFERENCES ahli_waris(id),  -- EDITABLE
    status                       TEXT NOT NULL DEFAULT 'terbit',
    created_by                   INTEGER REFERENCES users(id),
    created_at                   TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at                   TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_berkas_urutan ON berkas_waris(tahun, urutan);

-- PEWARIS (1-2 per berkas). NIK UNIQUE global = penegak LOCK.
CREATE TABLE IF NOT EXISTS pewaris (
    id                 INTEGER PRIMARY KEY AUTOINCREMENT,
    berkas_id          INTEGER NOT NULL REFERENCES berkas_waris(id),
    urutan             INTEGER NOT NULL DEFAULT 1,   -- suami dulu, lalu istri
    nama               TEXT NOT NULL,
    nik                TEXT NOT NULL UNIQUE,         -- lock 1x per NIK pewaris
    status             TEXT NOT NULL,                -- 'suami' | 'istri'
    tgl_meninggal      TEXT NOT NULL,
    instansi_kematian  TEXT NOT NULL,                -- default dari pengaturan, bisa dioverride
    no_surat_kematian  TEXT NOT NULL,
    tgl_surat_kematian TEXT NOT NULL
);

-- AHLI WARIS (list dinamis)
CREATE TABLE IF NOT EXISTS ahli_waris (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    berkas_id     INTEGER NOT NULL REFERENCES berkas_waris(id),
    urutan        INTEGER NOT NULL DEFAULT 1,
    nama          TEXT NOT NULL,
    nik           TEXT NOT NULL,
    umur          INTEGER,
    jenis_kelamin TEXT,           -- 'L' | 'P'
    agama         TEXT,
    alamat        TEXT,
    keterangan    TEXT,           -- "Anak" | "Istri" | dll
    -- field tambahan, hanya wajib bila jadi penerima kuasa (Surat 2):
    tempat_lahir  TEXT,
    tgl_lahir     TEXT,
    pekerjaan     TEXT
);

-- SAKSI (tepat 2 per berkas)
CREATE TABLE IF NOT EXISTS saksi (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    berkas_id    INTEGER NOT NULL REFERENCES berkas_waris(id),
    urutan       INTEGER NOT NULL DEFAULT 1,   -- 1 | 2
    nama         TEXT NOT NULL,
    tempat_lahir TEXT,
    tgl_lahir    TEXT,
    alamat       TEXT,
    nik          TEXT,
    hubungan     TEXT                          -- hubungan dengan almarhum
);

-- KUASA ITEM (isi Surat Kuasa, EDITABLE)
CREATE TABLE IF NOT EXISTS kuasa_item (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    berkas_id INTEGER NOT NULL REFERENCES berkas_waris(id),
    urutan    INTEGER NOT NULL DEFAULT 1,
    deskripsi TEXT NOT NULL
);

-- NOMOR URUT AWAL per tahun (migrasi manual→digital). urutan_awal = nomor
-- terakhir yang sudah dipakai manual; generator mulai dari urutan_awal+1.
CREATE TABLE IF NOT EXISTS nomor_awal (
    tahun       INTEGER PRIMARY KEY,
    urutan_awal INTEGER NOT NULL
);
