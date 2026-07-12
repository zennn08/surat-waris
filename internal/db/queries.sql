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
SELECT id, nama_kelurahan, kecamatan, kota, kode_kecamatan, kode_kelurahan, instansi_kematian
FROM pengaturan
WHERE id = 1;

-- name: EnsurePengaturanRow :exec
INSERT OR IGNORE INTO pengaturan (id) VALUES (1);

-- name: UpsertPengaturan :exec
INSERT INTO pengaturan (id, nama_kelurahan, kecamatan, kota, kode_kecamatan, kode_kelurahan, instansi_kematian)
VALUES (1, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    nama_kelurahan    = excluded.nama_kelurahan,
    kecamatan         = excluded.kecamatan,
    kota              = excluded.kota,
    kode_kecamatan    = excluded.kode_kecamatan,
    kode_kelurahan    = excluded.kode_kelurahan,
    instansi_kematian = excluded.instansi_kematian;

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
INSERT INTO berkas_waris (tahun, urutan, reg_no_camat, reg_no_lurah, tanggal_surat, tempat_tinggal_pewaris, created_by)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING id, tahun, urutan, reg_no_camat, reg_no_lurah, tanggal_reg_camat, tanggal_reg_lurah,
          tanggal_surat, tempat_tinggal_pewaris, penerima_kuasa_ahli_waris_id, status,
          created_by, created_at, updated_at;

-- name: GetBerkas :one
SELECT id, tahun, urutan, reg_no_camat, reg_no_lurah, tanggal_reg_camat, tanggal_reg_lurah,
       tanggal_surat, tempat_tinggal_pewaris, penerima_kuasa_ahli_waris_id, status,
       created_by, created_at, updated_at
FROM berkas_waris
WHERE id = ?;

-- name: ListBerkas :many
SELECT id, tahun, urutan, reg_no_camat, reg_no_lurah, tanggal_reg_camat, tanggal_reg_lurah,
       tanggal_surat, tempat_tinggal_pewaris, penerima_kuasa_ahli_waris_id, status,
       created_by, created_at, updated_at
FROM berkas_waris
ORDER BY created_at DESC;

-- name: SearchBerkas :many
SELECT b.id, b.tahun, b.urutan, b.reg_no_camat, b.reg_no_lurah, b.tanggal_reg_camat, b.tanggal_reg_lurah,
       b.tanggal_surat, b.tempat_tinggal_pewaris, b.penerima_kuasa_ahli_waris_id, b.status,
       b.created_by, b.created_at, b.updated_at
FROM berkas_waris b
WHERE EXISTS (
    SELECT 1 FROM pewaris p
    WHERE p.berkas_id = b.id
      AND (p.nama LIKE '%' || ?1 || '%' OR p.nik LIKE '%' || ?1 || '%')
)
OR b.reg_no_camat LIKE '%' || ?1 || '%'
OR b.reg_no_lurah LIKE '%' || ?1 || '%'
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
INSERT INTO pewaris (berkas_id, urutan, nama, nik, status, tgl_meninggal, instansi_kematian, no_surat_kematian, tgl_surat_kematian)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id, berkas_id, urutan, nama, nik, status, tgl_meninggal, instansi_kematian, no_surat_kematian, tgl_surat_kematian;

-- name: ListPewarisByBerkas :many
SELECT id, berkas_id, urutan, nama, nik, status, tgl_meninggal, instansi_kematian, no_surat_kematian, tgl_surat_kematian
FROM pewaris
WHERE berkas_id = ?
ORDER BY urutan, id;

-- AHLI WARIS
-- name: CreateAhliWaris :one
INSERT INTO ahli_waris (berkas_id, urutan, nama, nik, umur, jenis_kelamin, agama, alamat, keterangan, tempat_lahir, tgl_lahir, pekerjaan)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id, berkas_id, urutan, nama, nik, umur, jenis_kelamin, agama, alamat, keterangan, tempat_lahir, tgl_lahir, pekerjaan;

-- name: ListAhliWarisByBerkas :many
SELECT id, berkas_id, urutan, nama, nik, umur, jenis_kelamin, agama, alamat, keterangan, tempat_lahir, tgl_lahir, pekerjaan
FROM ahli_waris
WHERE berkas_id = ?
ORDER BY urutan, id;

-- name: GetAhliWaris :one
SELECT id, berkas_id, urutan, nama, nik, umur, jenis_kelamin, agama, alamat, keterangan, tempat_lahir, tgl_lahir, pekerjaan
FROM ahli_waris
WHERE id = ?;

-- name: UpdateAhliWarisPelengkap :exec
UPDATE ahli_waris
SET tempat_lahir = ?, tgl_lahir = ?, pekerjaan = ?
WHERE id = ? AND berkas_id = ?;

-- SAKSI
-- name: CreateSaksi :exec
INSERT INTO saksi (berkas_id, urutan, nama, tempat_lahir, tgl_lahir, alamat, nik, hubungan)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: ListSaksiByBerkas :many
SELECT id, berkas_id, urutan, nama, tempat_lahir, tgl_lahir, alamat, nik, hubungan
FROM saksi
WHERE berkas_id = ?
ORDER BY urutan, id;

-- KUASA ITEM (editable)
-- name: CreateKuasaItem :one
INSERT INTO kuasa_item (berkas_id, urutan, deskripsi)
VALUES (?, ?, ?)
RETURNING id, berkas_id, urutan, deskripsi;

-- name: GetKuasaItem :one
SELECT id, berkas_id, urutan, deskripsi FROM kuasa_item WHERE id = ?;

-- name: ListKuasaItemByBerkas :many
SELECT id, berkas_id, urutan, deskripsi
FROM kuasa_item
WHERE berkas_id = ?
ORDER BY urutan, id;

-- name: UpdateKuasaItem :exec
UPDATE kuasa_item SET deskripsi = ? WHERE id = ?;

-- name: DeleteKuasaItem :exec
DELETE FROM kuasa_item WHERE id = ?;
