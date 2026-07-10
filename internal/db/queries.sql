-- ============================================================
-- USERS
-- ============================================================

-- name: GetUserByUsername :one
SELECT id, username, password_hash, nama, role, must_change_password, created_at
FROM users
WHERE username = ?;

-- name: GetUserByID :one
SELECT id, username, password_hash, nama, role, must_change_password, created_at
FROM users
WHERE id = ?;

-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: CreateUser :one
INSERT INTO users (username, password_hash, nama, role, must_change_password)
VALUES (?, ?, ?, ?, ?)
RETURNING id, username, password_hash, nama, role, must_change_password, created_at;

-- name: UpdateUserPassword :exec
UPDATE users
SET password_hash = ?, must_change_password = 0
WHERE id = ?;

-- ============================================================
-- PENGATURAN (satu baris, id = 1)
-- ============================================================

-- name: GetPengaturan :one
SELECT id, nama_kelurahan, kecamatan, kabupaten, provinsi, format_nomor
FROM pengaturan
WHERE id = 1;

-- name: EnsurePengaturanRow :exec
INSERT OR IGNORE INTO pengaturan (id) VALUES (1);

-- name: UpsertPengaturan :exec
INSERT INTO pengaturan (id, nama_kelurahan, kecamatan, kabupaten, provinsi, format_nomor)
VALUES (1, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    nama_kelurahan = excluded.nama_kelurahan,
    kecamatan      = excluded.kecamatan,
    kabupaten      = excluded.kabupaten,
    provinsi       = excluded.provinsi,
    format_nomor   = excluded.format_nomor;

-- ============================================================
-- PEJABAT (Lurah / Camat)
-- ============================================================

-- name: ListPejabat :many
SELECT id, jabatan, nama, nip, aktif, created_at
FROM pejabat
ORDER BY jabatan, aktif DESC, created_at DESC;

-- name: GetPejabat :one
SELECT id, jabatan, nama, nip, aktif, created_at
FROM pejabat
WHERE id = ?;

-- name: GetPejabatAktif :one
SELECT id, jabatan, nama, nip, aktif, created_at
FROM pejabat
WHERE jabatan = ? AND aktif = 1
ORDER BY created_at DESC
LIMIT 1;

-- name: CreatePejabat :one
INSERT INTO pejabat (jabatan, nama, nip, aktif)
VALUES (?, ?, ?, ?)
RETURNING id, jabatan, nama, nip, aktif, created_at;

-- name: UpdatePejabat :exec
UPDATE pejabat
SET jabatan = ?, nama = ?, nip = ?, aktif = ?
WHERE id = ?;

-- name: DeletePejabat :exec
DELETE FROM pejabat WHERE id = ?;

-- name: DeactivatePejabatByJabatan :exec
UPDATE pejabat SET aktif = 0 WHERE jabatan = ?;

-- ============================================================
-- BERKAS WARIS + anak-anaknya
-- ============================================================

-- name: NextUrutan :one
SELECT COALESCE(MAX(urutan), 0) + 1 AS next_urutan
FROM berkas_waris
WHERE tahun = ?;

-- name: GetNomorAwal :one
SELECT urutan_awal FROM nomor_awal WHERE tahun = ?;

-- name: ListNomorAwal :many
SELECT tahun, urutan_awal FROM nomor_awal ORDER BY tahun DESC;

-- name: UpsertNomorAwal :exec
INSERT INTO nomor_awal (tahun, urutan_awal) VALUES (?, ?)
ON CONFLICT(tahun) DO UPDATE SET urutan_awal = excluded.urutan_awal;

-- name: DeleteNomorAwal :exec
DELETE FROM nomor_awal WHERE tahun = ?;

-- name: CreateBerkas :one
INSERT INTO berkas_waris (nomor_surat, tahun, urutan, tanggal, tempat_tinggal_pewaris, created_by)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING id, nomor_surat, tahun, urutan, tanggal, tempat_tinggal_pewaris,
          penerima_kuasa_ahli_waris_id, status, created_by, created_at, updated_at;

-- name: GetBerkas :one
SELECT id, nomor_surat, tahun, urutan, tanggal, tempat_tinggal_pewaris,
       penerima_kuasa_ahli_waris_id, status, created_by, created_at, updated_at
FROM berkas_waris
WHERE id = ?;

-- name: ListBerkas :many
SELECT id, nomor_surat, tahun, urutan, tanggal, tempat_tinggal_pewaris,
       penerima_kuasa_ahli_waris_id, status, created_by, created_at, updated_at
FROM berkas_waris
ORDER BY created_at DESC;

-- name: SearchBerkas :many
SELECT b.id, b.nomor_surat, b.tahun, b.urutan, b.tanggal, b.tempat_tinggal_pewaris,
       b.penerima_kuasa_ahli_waris_id, b.status, b.created_by, b.created_at, b.updated_at
FROM berkas_waris b
WHERE EXISTS (
    SELECT 1 FROM pewaris p
    WHERE p.berkas_id = b.id
      AND (p.nama LIKE '%' || ?1 || '%' OR p.nik LIKE '%' || ?1 || '%')
)
OR b.nomor_surat LIKE '%' || ?1 || '%'
ORDER BY b.created_at DESC;

-- name: SetBerkasPenerimaKuasa :exec
UPDATE berkas_waris
SET penerima_kuasa_ahli_waris_id = ?, updated_at = datetime('now')
WHERE id = ?;

-- name: TouchBerkas :exec
UPDATE berkas_waris SET updated_at = datetime('now') WHERE id = ?;

-- PEWARIS
-- name: CountPewarisByNik :one
SELECT COUNT(*) FROM pewaris WHERE nik = ?;

-- name: CreatePewaris :one
INSERT INTO pewaris (berkas_id, nama, nik, tgl_meninggal, no_surat_kematian, tgl_surat_kematian)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING id, berkas_id, nama, nik, tgl_meninggal, no_surat_kematian, tgl_surat_kematian;

-- name: ListPewarisByBerkas :many
SELECT id, berkas_id, nama, nik, tgl_meninggal, no_surat_kematian, tgl_surat_kematian
FROM pewaris
WHERE berkas_id = ?
ORDER BY id;

-- AHLI WARIS
-- name: CreateAhliWaris :one
INSERT INTO ahli_waris (berkas_id, nama, nik, umur, jenis_kelamin, agama, alamat, keterangan)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id, berkas_id, nama, nik, umur, jenis_kelamin, agama, alamat, keterangan;

-- name: ListAhliWarisByBerkas :many
SELECT id, berkas_id, nama, nik, umur, jenis_kelamin, agama, alamat, keterangan
FROM ahli_waris
WHERE berkas_id = ?
ORDER BY id;

-- name: GetAhliWaris :one
SELECT id, berkas_id, nama, nik, umur, jenis_kelamin, agama, alamat, keterangan
FROM ahli_waris
WHERE id = ?;

-- SAKSI
-- name: CreateSaksi :exec
INSERT INTO saksi (berkas_id, nama, ttl, alamat, nik, hubungan)
VALUES (?, ?, ?, ?, ?, ?);

-- name: ListSaksiByBerkas :many
SELECT id, berkas_id, nama, ttl, alamat, nik, hubungan
FROM saksi
WHERE berkas_id = ?
ORDER BY id;

-- HARTA (editable)
-- name: CreateHarta :one
INSERT INTO harta (berkas_id, deskripsi)
VALUES (?, ?)
RETURNING id, berkas_id, deskripsi;

-- name: GetHarta :one
SELECT id, berkas_id, deskripsi FROM harta WHERE id = ?;

-- name: ListHartaByBerkas :many
SELECT id, berkas_id, deskripsi
FROM harta
WHERE berkas_id = ?
ORDER BY id;

-- name: UpdateHarta :exec
UPDATE harta SET deskripsi = ? WHERE id = ?;

-- name: DeleteHarta :exec
DELETE FROM harta WHERE id = ?;
