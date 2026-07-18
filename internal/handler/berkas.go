package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"surat-waris/internal/auth"
	"surat-waris/internal/db"
	"surat-waris/internal/surat"
)

// ---------- Views ----------

type berkasView struct {
	ID                       int64  `json:"id"`
	Tahun                    int64  `json:"tahun"`
	Urutan                   int64  `json:"urutan"`
	RegNoCamat               string `json:"reg_no_camat"`
	RegNoLurah               string `json:"reg_no_lurah"`
	TanggalRegCamat          string `json:"tanggal_reg_camat"`
	TanggalRegLurah          string `json:"tanggal_reg_lurah"`
	TanggalSurat             string `json:"tanggal_surat"`
	TempatTinggalPewaris     string `json:"tempat_tinggal_pewaris"`
	PenerimaKuasaAhliWarisID *int64 `json:"penerima_kuasa_ahli_waris_id"`
	Status                   string `json:"status"`
	CreatedAt                string `json:"created_at"`
	UpdatedAt                string `json:"updated_at"`
}

func toBerkasView(b db.BerkasWaris) berkasView {
	v := berkasView{
		ID:                   b.ID,
		Tahun:                b.Tahun,
		Urutan:               b.Urutan,
		RegNoCamat:           b.RegNoCamat,
		RegNoLurah:           b.RegNoLurah,
		TanggalRegCamat:      strOrEmpty(b.TanggalRegCamat),
		TanggalRegLurah:      strOrEmpty(b.TanggalRegLurah),
		TanggalSurat:         b.TanggalSurat,
		TempatTinggalPewaris: b.TempatTinggalPewaris,
		Status:               b.Status,
		CreatedAt:            b.CreatedAt,
		UpdatedAt:            b.UpdatedAt,
	}
	if b.PenerimaKuasaAhliWarisID.Valid {
		id := b.PenerimaKuasaAhliWarisID.Int64
		v.PenerimaKuasaAhliWarisID = &id
	}
	return v
}

type pewarisView struct {
	ID               int64  `json:"id"`
	Urutan           int64  `json:"urutan"`
	Nama             string `json:"nama"`
	Nik              string `json:"nik"`
	Status           string `json:"status"`
	TglMeninggal     string `json:"tgl_meninggal"`
	InstansiKematian string `json:"instansi_kematian"`
	NoSuratKematian  string `json:"no_surat_kematian"`
	TglSuratKematian string `json:"tgl_surat_kematian"`
}

type ahliWarisView struct {
	ID           int64  `json:"id"`
	Urutan       int64  `json:"urutan"`
	Nama         string `json:"nama"`
	Nik          string `json:"nik"`
	Umur         *int64 `json:"umur"`
	JenisKelamin string `json:"jenis_kelamin"`
	Agama        string `json:"agama"`
	Alamat       string `json:"alamat"`
	Keterangan   string `json:"keterangan"`
	TempatLahir  string `json:"tempat_lahir"`
	TglLahir     string `json:"tgl_lahir"`
	Pekerjaan    string `json:"pekerjaan"`
}

type saksiView struct {
	ID          int64  `json:"id"`
	Urutan      int64  `json:"urutan"`
	Nama        string `json:"nama"`
	TempatLahir string `json:"tempat_lahir"`
	TglLahir    string `json:"tgl_lahir"`
	Alamat      string `json:"alamat"`
	Nik         string `json:"nik"`
	Hubungan    string `json:"hubungan"`
}

type kuasaView struct {
	ID        int64  `json:"id"`
	Urutan    int64  `json:"urutan"`
	Deskripsi string `json:"deskripsi"`
}

type berkasDetail struct {
	berkasView
	Pewaris   []pewarisView   `json:"pewaris"`
	AhliWaris []ahliWarisView `json:"ahli_waris"`
	Saksi     []saksiView     `json:"saksi"`
	Kuasa     []kuasaView     `json:"kuasa"`
}

// berkasListItem = ringkas untuk daftar berkas.
type berkasListItem struct {
	berkasView
	Pewaris []pewarisView `json:"pewaris"`
}

// ---------- Input ----------

type pewarisInput struct {
	Nama             string `json:"nama"`
	Nik              string `json:"nik"`
	Status           string `json:"status"` // 'suami' | 'istri'
	TglMeninggal     string `json:"tgl_meninggal"`
	InstansiKematian string `json:"instansi_kematian"`
	NoSuratKematian  string `json:"no_surat_kematian"`
	TglSuratKematian string `json:"tgl_surat_kematian"`
}

type ahliWarisInput struct {
	Nama         string `json:"nama"`
	Nik          string `json:"nik"`
	Umur         *int64 `json:"umur"`
	JenisKelamin string `json:"jenis_kelamin"`
	Agama        string `json:"agama"`
	Alamat       string `json:"alamat"`
	Keterangan   string `json:"keterangan"`
	TempatLahir  string `json:"tempat_lahir"`
	TglLahir     string `json:"tgl_lahir"`
	Pekerjaan    string `json:"pekerjaan"`
}

type saksiInput struct {
	Nama        string `json:"nama"`
	TempatLahir string `json:"tempat_lahir"`
	TglLahir    string `json:"tgl_lahir"`
	Alamat      string `json:"alamat"`
	Nik         string `json:"nik"`
	Hubungan    string `json:"hubungan"`
}

type createBerkasReq struct {
	TanggalSurat         string           `json:"tanggal_surat"`
	TempatTinggalPewaris string           `json:"tempat_tinggal_pewaris"`
	Pewaris              []pewarisInput   `json:"pewaris"`
	AhliWaris            []ahliWarisInput `json:"ahli_waris"`
	Saksi                []saksiInput     `json:"saksi"`
	PenerimaKuasaIndex   *int             `json:"penerima_kuasa_index"`
	Kuasa                []string         `json:"kuasa"`
}

var nik16Re = regexp.MustCompile(`^\d{16}$`)

func (r *createBerkasReq) validate() (time.Time, error) {
	tgl, err := time.Parse("2006-01-02", strings.TrimSpace(r.TanggalSurat))
	if err != nil {
		return time.Time{}, errors.New("tanggal surat tidak valid (format YYYY-MM-DD)")
	}
	if strings.TrimSpace(r.TempatTinggalPewaris) == "" {
		return time.Time{}, errors.New("tempat tinggal pewaris wajib diisi")
	}
	if len(r.Pewaris) < 1 || len(r.Pewaris) > 2 {
		return time.Time{}, errors.New("pewaris minimal 1, maksimal 2")
	}
	for i, p := range r.Pewaris {
		if strings.TrimSpace(p.Nama) == "" || strings.TrimSpace(p.Nik) == "" {
			return time.Time{}, fmt.Errorf("pewaris #%d: nama dan NIK wajib diisi", i+1)
		}
		if !nik16Re.MatchString(strings.TrimSpace(p.Nik)) {
			return time.Time{}, fmt.Errorf("pewaris #%d: NIK harus 16 digit angka", i+1)
		}
		st := strings.ToLower(strings.TrimSpace(p.Status))
		if st != "suami" && st != "istri" {
			return time.Time{}, fmt.Errorf("pewaris #%d: status harus 'suami' atau 'istri'", i+1)
		}
		if strings.TrimSpace(p.TglMeninggal) == "" || strings.TrimSpace(p.NoSuratKematian) == "" || strings.TrimSpace(p.TglSuratKematian) == "" {
			return time.Time{}, fmt.Errorf("pewaris #%d: tanggal meninggal, no & tgl surat kematian wajib diisi", i+1)
		}
		// Surat kematian tidak mungkin terbit sebelum orangnya meninggal.
		tm, errM := time.Parse("2006-01-02", strings.TrimSpace(p.TglMeninggal))
		ts, errS := time.Parse("2006-01-02", strings.TrimSpace(p.TglSuratKematian))
		if errM == nil && errS == nil && ts.Before(tm) {
			return time.Time{}, fmt.Errorf("pewaris #%d: tanggal surat kematian tidak boleh lebih awal dari tanggal meninggal", i+1)
		}
	}
	if len(r.AhliWaris) < 1 {
		return time.Time{}, errors.New("ahli waris minimal 1 orang")
	}
	for i, a := range r.AhliWaris {
		if strings.TrimSpace(a.Nama) == "" || strings.TrimSpace(a.Nik) == "" {
			return time.Time{}, fmt.Errorf("ahli waris #%d: nama dan NIK wajib diisi", i+1)
		}
		if !nik16Re.MatchString(strings.TrimSpace(a.Nik)) {
			return time.Time{}, fmt.Errorf("ahli waris #%d: NIK harus 16 digit angka", i+1)
		}
	}
	if len(r.Saksi) != 2 {
		return time.Time{}, errors.New("saksi harus tepat 2 orang")
	}
	for i, s := range r.Saksi {
		if strings.TrimSpace(s.Nama) == "" {
			return time.Time{}, fmt.Errorf("saksi #%d: nama wajib diisi", i+1)
		}
		if nik := strings.TrimSpace(s.Nik); nik != "" && !nik16Re.MatchString(nik) {
			return time.Time{}, fmt.Errorf("saksi #%d: NIK harus 16 digit angka (atau kosongkan)", i+1)
		}
	}
	if r.PenerimaKuasaIndex != nil {
		if *r.PenerimaKuasaIndex < 0 || *r.PenerimaKuasaIndex >= len(r.AhliWaris) {
			return time.Time{}, errors.New("penerima kuasa tidak valid")
		}
	}
	return tgl, nil
}

// prasyaratBerkas memastikan pejabat aktif (Camat & Lurah) dan pengaturan sudah
// terisi sebelum berkas boleh dibuat. Balikan pesan kosong bila lolos.
func (h *Handler) prasyaratBerkas(ctx context.Context) string {
	if _, err := h.q.GetPejabatAktif(ctx, "camat"); errors.Is(err, sql.ErrNoRows) {
		return "Pejabat Camat aktif belum diisi. Lengkapi dulu di halaman Pejabat."
	}
	if _, err := h.q.GetPejabatAktif(ctx, "lurah"); errors.Is(err, sql.ErrNoRows) {
		return "Pejabat Lurah aktif belum diisi. Lengkapi dulu di halaman Pejabat."
	}
	peng, err := h.q.GetPengaturan(ctx)
	if err != nil {
		return "Pengaturan belum diisi. Lengkapi dulu di halaman Pengaturan."
	}
	wajib := []struct{ label, val string }{
		{"Nama Kelurahan", strOrEmpty(peng.NamaKelurahan)},
		{"Kecamatan", strOrEmpty(peng.Kecamatan)},
		{"Kota", strOrEmpty(peng.Kota)},
		{"Kode Kecamatan", strOrEmpty(peng.KodeKecamatan)},
		{"Kode Kelurahan", strOrEmpty(peng.KodeKelurahan)},
		{"Instansi Penerbit Surat Kematian", strOrEmpty(peng.InstansiKematian)},
	}
	var kosong []string
	for _, f := range wajib {
		if strings.TrimSpace(f.val) == "" {
			kosong = append(kosong, f.label)
		}
	}
	if len(kosong) > 0 {
		return "Pengaturan belum lengkap (" + strings.Join(kosong, ", ") + "). Lengkapi dulu di halaman Pengaturan."
	}
	return ""
}

// CreateBerkas: POST /api/berkas
func (h *Handler) CreateBerkas(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromContext(r.Context())

	if msg := h.prasyaratBerkas(r.Context()); msg != "" {
		writeErr(w, http.StatusUnprocessableEntity, msg)
		return
	}

	var req createBerkasReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "body tidak valid")
		return
	}
	tgl, err := req.validate()
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	berkasID, err := h.createBerkasTx(r.Context(), req, tgl, uid)
	if err != nil {
		var lockErr lockError
		if errors.As(err, &lockErr) {
			writeErr(w, http.StatusConflict, lockErr.Error())
			return
		}
		writeErr(w, http.StatusInternalServerError, "gagal menyimpan berkas: "+err.Error())
		return
	}

	detail, err := h.loadDetail(r.Context(), h.q, berkasID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "berkas tersimpan tapi gagal dimuat")
		return
	}
	writeJSON(w, http.StatusCreated, detail)
}

// lockError menandai NIK pewaris yang sudah pernah dibuatkan surat.
type lockError struct {
	nik      string
	regCamat string
}

func (e lockError) Error() string {
	if e.regCamat != "" {
		return fmt.Sprintf("Pewaris dengan NIK %s sudah pernah dibuatkan Surat Keterangan Ahli Waris (Reg. No. %s).", e.nik, e.regCamat)
	}
	return fmt.Sprintf("Pewaris dengan NIK %s sudah pernah dibuatkan Surat Keterangan Ahli Waris.", e.nik)
}

func (h *Handler) createBerkasTx(ctx context.Context, req createBerkasReq, tgl time.Time, uid int64) (int64, error) {
	tx, err := h.sqldb.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	qtx := h.q.WithTx(tx)

	// 1. Enforce lock: cek tiap NIK pewaris.
	for _, p := range req.Pewaris {
		nik := strings.TrimSpace(p.Nik)
		n, err := qtx.CountPewarisByNik(ctx, nik)
		if err != nil {
			return 0, err
		}
		if n > 0 {
			return 0, lockError{nik: nik, regCamat: h.lookupRegByPewarisNik(ctx, qtx, nik)}
		}
	}

	// 2. Generate urutan per tahun (hormati nomor urut awal migrasi manual→digital).
	tahun := int64(tgl.Year())
	urutan, err := qtx.NextUrutan(ctx, tahun)
	if err != nil {
		return 0, err
	}
	if awal, err := qtx.GetNomorAwal(ctx, tahun); err == nil {
		if awal+1 > urutan {
			urutan = awal + 1
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	peng, err := qtx.GetPengaturan(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	kodeKec := strOrEmpty(peng.KodeKecamatan)
	kodeKel := strOrEmpty(peng.KodeKelurahan)
	regCamat := surat.RegNoCamat(int(urutan), int(tahun), kodeKec)
	regLurah := surat.RegNoLurah(int(urutan), int(tahun), kodeKel, kodeKec)

	// 3. Buat berkas.
	b, err := qtx.CreateBerkas(ctx, db.CreateBerkasParams{
		Tahun:                tahun,
		Urutan:               urutan,
		RegNoCamat:           regCamat,
		RegNoLurah:           regLurah,
		TanggalSurat:         tgl.Format("2006-01-02"),
		TempatTinggalPewaris: strings.TrimSpace(req.TempatTinggalPewaris),
		CreatedBy:            sql.NullInt64{Int64: uid, Valid: uid != 0},
	})
	if err != nil {
		return 0, err
	}

	// 4. Pewaris (UNIQUE nik = pengaman lock tingkat DB). instansi_kematian
	// default dari pengaturan bila kosong.
	defaultInstansi := strOrEmpty(peng.InstansiKematian)
	for i, p := range req.Pewaris {
		instansi := strings.TrimSpace(p.InstansiKematian)
		if instansi == "" {
			instansi = defaultInstansi
		}
		if _, err := qtx.CreatePewaris(ctx, db.CreatePewarisParams{
			BerkasID:         b.ID,
			Urutan:           int64(i + 1),
			Nama:             strings.TrimSpace(p.Nama),
			Nik:              strings.TrimSpace(p.Nik),
			Status:           strings.ToLower(strings.TrimSpace(p.Status)),
			TglMeninggal:     strings.TrimSpace(p.TglMeninggal),
			InstansiKematian: instansi,
			NoSuratKematian:  strings.TrimSpace(p.NoSuratKematian),
			TglSuratKematian: strings.TrimSpace(p.TglSuratKematian),
		}); err != nil {
			return 0, err
		}
	}

	// 5. Ahli waris — simpan ID untuk resolusi penerima kuasa.
	ahliIDs := make([]int64, 0, len(req.AhliWaris))
	for i, a := range req.AhliWaris {
		created, err := qtx.CreateAhliWaris(ctx, db.CreateAhliWarisParams{
			BerkasID:     b.ID,
			Urutan:       int64(i + 1),
			Nama:         strings.TrimSpace(a.Nama),
			Nik:          strings.TrimSpace(a.Nik),
			Umur:         nullInt(a.Umur),
			JenisKelamin: nullStr(strings.TrimSpace(a.JenisKelamin)),
			Agama:        nullStr(strings.TrimSpace(a.Agama)),
			Alamat:       nullStr(strings.TrimSpace(a.Alamat)),
			Keterangan:   nullStr(strings.TrimSpace(a.Keterangan)),
			TempatLahir:  nullStr(strings.TrimSpace(a.TempatLahir)),
			TglLahir:     nullStr(strings.TrimSpace(a.TglLahir)),
			Pekerjaan:    nullStr(strings.TrimSpace(a.Pekerjaan)),
		})
		if err != nil {
			return 0, err
		}
		ahliIDs = append(ahliIDs, created.ID)
	}

	// 6. Saksi.
	for i, s := range req.Saksi {
		if err := qtx.CreateSaksi(ctx, db.CreateSaksiParams{
			BerkasID:    b.ID,
			Urutan:      int64(i + 1),
			Nama:        strings.TrimSpace(s.Nama),
			TempatLahir: nullStr(strings.TrimSpace(s.TempatLahir)),
			TglLahir:    nullStr(strings.TrimSpace(s.TglLahir)),
			Alamat:      nullStr(strings.TrimSpace(s.Alamat)),
			Nik:         nullStr(strings.TrimSpace(s.Nik)),
			Hubungan:    nullStr(strings.TrimSpace(s.Hubungan)),
		}); err != nil {
			return 0, err
		}
	}

	// 7. Item kuasa.
	urut := 0
	for _, d := range req.Kuasa {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		urut++
		if _, err := qtx.CreateKuasaItem(ctx, db.CreateKuasaItemParams{BerkasID: b.ID, Urutan: int64(urut), Deskripsi: d}); err != nil {
			return 0, err
		}
	}

	// 8. Penerima kuasa (opsional).
	if req.PenerimaKuasaIndex != nil {
		if err := qtx.SetBerkasPenerimaKuasa(ctx, db.SetBerkasPenerimaKuasaParams{
			PenerimaKuasaAhliWarisID: sql.NullInt64{Int64: ahliIDs[*req.PenerimaKuasaIndex], Valid: true},
			ID:                       b.ID,
		}); err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return b.ID, nil
}

// lookupRegByPewarisNik mencari reg_no_camat berkas yang memuat NIK pewaris ini
// (best effort untuk pesan lock; abaikan error).
func (h *Handler) lookupRegByPewarisNik(ctx context.Context, q *db.Queries, nik string) string {
	rows, err := q.SearchBerkas(ctx, nullStr(nik))
	if err != nil || len(rows) == 0 {
		return ""
	}
	return rows[0].RegNoCamat
}

// ListBerkas: GET /api/berkas?q=...
func (h *Handler) ListBerkas(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	var rows []db.BerkasWaris
	var err error
	if q == "" {
		rows, err = h.q.ListBerkas(r.Context())
	} else {
		rows, err = h.q.SearchBerkas(r.Context(), nullStr(q))
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal memuat berkas")
		return
	}
	out := make([]berkasListItem, 0, len(rows))
	for _, b := range rows {
		pw, err := h.q.ListPewarisByBerkas(r.Context(), b.ID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "gagal memuat pewaris")
			return
		}
		out = append(out, berkasListItem{berkasView: toBerkasView(b), Pewaris: toPewarisViews(pw)})
	}
	writeJSON(w, http.StatusOK, out)
}

// GetBerkas: GET /api/berkas/{id}
func (h *Handler) GetBerkas(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "id tidak valid")
		return
	}
	detail, err := h.loadDetail(r.Context(), h.q, id)
	if errors.Is(err, sql.ErrNoRows) {
		writeErr(w, http.StatusNotFound, "berkas tidak ditemukan")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal memuat berkas")
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

// loadDetail memuat berkas + semua anak.
func (h *Handler) loadDetail(ctx context.Context, q *db.Queries, id int64) (berkasDetail, error) {
	b, err := q.GetBerkas(ctx, id)
	if err != nil {
		return berkasDetail{}, err
	}
	pw, err := q.ListPewarisByBerkas(ctx, id)
	if err != nil {
		return berkasDetail{}, err
	}
	aw, err := q.ListAhliWarisByBerkas(ctx, id)
	if err != nil {
		return berkasDetail{}, err
	}
	sk, err := q.ListSaksiByBerkas(ctx, id)
	if err != nil {
		return berkasDetail{}, err
	}
	ki, err := q.ListKuasaItemByBerkas(ctx, id)
	if err != nil {
		return berkasDetail{}, err
	}
	return berkasDetail{
		berkasView: toBerkasView(b),
		Pewaris:    toPewarisViews(pw),
		AhliWaris:  toAhliWarisViews(aw),
		Saksi:      toSaksiViews(sk),
		Kuasa:      toKuasaViews(ki),
	}, nil
}

// ---------- konversi slice ----------

func toPewarisViews(rows []db.Pewaris) []pewarisView {
	out := make([]pewarisView, 0, len(rows))
	for _, p := range rows {
		out = append(out, pewarisView{
			ID: p.ID, Urutan: p.Urutan, Nama: p.Nama, Nik: p.Nik, Status: p.Status,
			TglMeninggal: p.TglMeninggal, InstansiKematian: p.InstansiKematian,
			NoSuratKematian: p.NoSuratKematian, TglSuratKematian: p.TglSuratKematian,
		})
	}
	return out
}

func toAhliWarisViews(rows []db.AhliWaris) []ahliWarisView {
	out := make([]ahliWarisView, 0, len(rows))
	for _, a := range rows {
		v := ahliWarisView{
			ID: a.ID, Urutan: a.Urutan, Nama: a.Nama, Nik: a.Nik,
			JenisKelamin: strOrEmpty(a.JenisKelamin), Agama: strOrEmpty(a.Agama),
			Alamat: strOrEmpty(a.Alamat), Keterangan: strOrEmpty(a.Keterangan),
			TempatLahir: strOrEmpty(a.TempatLahir), TglLahir: strOrEmpty(a.TglLahir),
			Pekerjaan: strOrEmpty(a.Pekerjaan),
		}
		if a.Umur.Valid {
			u := a.Umur.Int64
			v.Umur = &u
		}
		out = append(out, v)
	}
	return out
}

func toSaksiViews(rows []db.Saksi) []saksiView {
	out := make([]saksiView, 0, len(rows))
	for _, s := range rows {
		out = append(out, saksiView{
			ID: s.ID, Urutan: s.Urutan, Nama: s.Nama,
			TempatLahir: strOrEmpty(s.TempatLahir), TglLahir: strOrEmpty(s.TglLahir),
			Alamat: strOrEmpty(s.Alamat), Nik: strOrEmpty(s.Nik), Hubungan: strOrEmpty(s.Hubungan),
		})
	}
	return out
}

func toKuasaViews(rows []db.KuasaItem) []kuasaView {
	out := make([]kuasaView, 0, len(rows))
	for _, k := range rows {
		out = append(out, kuasaView{ID: k.ID, Urutan: k.Urutan, Deskripsi: k.Deskripsi})
	}
	return out
}
