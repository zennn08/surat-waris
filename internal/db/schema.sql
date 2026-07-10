-- Skema Surat Waris (sesuai SPEC §6). Semua DDL idempotent (IF NOT EXISTS)
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

-- PEJABAT (Lurah / Camat) — nama + NIP
CREATE TABLE IF NOT EXISTS pejabat (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    jabatan    TEXT NOT NULL,               -- 'lurah' | 'camat'
    nama       TEXT NOT NULL,
    nip        TEXT NOT NULL,
    aktif      INTEGER NOT NULL DEFAULT 1,   -- pejabat aktif yang dipakai di surat
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- PENGATURAN (identitas kelurahan; satu baris). Dipakai di isi surat & nomor,
-- BUKAN sebagai kop dekoratif. Tidak ada logo.
CREATE TABLE IF NOT EXISTS pengaturan (
    id             INTEGER PRIMARY KEY CHECK (id = 1),
    nama_kelurahan TEXT,
    kecamatan      TEXT,
    kabupaten      TEXT,
    provinsi       TEXT,
    format_nomor   TEXT           -- template format nomor surat
);

-- BERKAS WARIS (induk; 1 nomor untuk 3 surat)
CREATE TABLE IF NOT EXISTS berkas_waris (
    id                           INTEGER PRIMARY KEY AUTOINCREMENT,
    nomor_surat                  TEXT NOT NULL UNIQUE,
    tahun                        INTEGER NOT NULL,
    urutan                       INTEGER NOT NULL,           -- counter per tahun
    tanggal                      TEXT NOT NULL,
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
    nama               TEXT NOT NULL,
    nik                TEXT NOT NULL UNIQUE,     -- lock 1x per NIK pewaris
    tgl_meninggal      TEXT NOT NULL,
    no_surat_kematian  TEXT NOT NULL,
    tgl_surat_kematian TEXT NOT NULL
);

-- AHLI WARIS (list dinamis)
CREATE TABLE IF NOT EXISTS ahli_waris (
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
CREATE TABLE IF NOT EXISTS saksi (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    berkas_id INTEGER NOT NULL REFERENCES berkas_waris(id),
    nama      TEXT NOT NULL,
    ttl       TEXT,               -- tempat, tanggal lahir
    alamat    TEXT,
    nik       TEXT,
    hubungan  TEXT                -- hubungan dengan almarhum
);

-- HARTA / yang dikuasakan (bagian Surat Kuasa, EDITABLE)
CREATE TABLE IF NOT EXISTS harta (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    berkas_id INTEGER NOT NULL REFERENCES berkas_waris(id),
    deskripsi TEXT NOT NULL
);

-- NOMOR URUT AWAL per tahun (migrasi manual→digital). urutan_awal = nomor
-- terakhir yang sudah dipakai manual; generator mulai dari urutan_awal+1.
CREATE TABLE IF NOT EXISTS nomor_awal (
    tahun       INTEGER PRIMARY KEY,
    urutan_awal INTEGER NOT NULL
);
