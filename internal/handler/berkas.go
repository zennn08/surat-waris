package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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
	NomorSurat               string `json:"nomor_surat"`
	Tahun                    int64  `json:"tahun"`
	Urutan                   int64  `json:"urutan"`
	Tanggal                  string `json:"tanggal"`
	TempatTinggalPewaris     string `json:"tempat_tinggal_pewaris"`
	PenerimaKuasaAhliWarisID *int64 `json:"penerima_kuasa_ahli_waris_id"`
	Status                   string `json:"status"`
	CreatedAt                string `json:"created_at"`
	UpdatedAt                string `json:"updated_at"`
}

func toBerkasView(b db.BerkasWaris) berkasView {
	v := berkasView{
		ID:                   b.ID,
		NomorSurat:           b.NomorSurat,
		Tahun:                b.Tahun,
		Urutan:               b.Urutan,
		Tanggal:              b.Tanggal,
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
	Nama             string `json:"nama"`
	Nik              string `json:"nik"`
	TglMeninggal     string `json:"tgl_meninggal"`
	NoSuratKematian  string `json:"no_surat_kematian"`
	TglSuratKematian string `json:"tgl_surat_kematian"`
}

type ahliWarisView struct {
	ID           int64  `json:"id"`
	Nama         string `json:"nama"`
	Nik          string `json:"nik"`
	Umur         *int64 `json:"umur"`
	JenisKelamin string `json:"jenis_kelamin"`
	Agama        string `json:"agama"`
	Alamat       string `json:"alamat"`
	Keterangan   string `json:"keterangan"`
}

type saksiView struct {
	ID       int64  `json:"id"`
	Nama     string `json:"nama"`
	Ttl      string `json:"ttl"`
	Alamat   string `json:"alamat"`
	Nik      string `json:"nik"`
	Hubungan string `json:"hubungan"`
}

type hartaView struct {
	ID        int64  `json:"id"`
	Deskripsi string `json:"deskripsi"`
}

type berkasDetail struct {
	berkasView
	Pewaris   []pewarisView   `json:"pewaris"`
	AhliWaris []ahliWarisView `json:"ahli_waris"`
	Saksi     []saksiView     `json:"saksi"`
	Harta     []hartaView     `json:"harta"`
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
	TglMeninggal     string `json:"tgl_meninggal"`
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
}

type saksiInput struct {
	Nama     string `json:"nama"`
	Ttl      string `json:"ttl"`
	Alamat   string `json:"alamat"`
	Nik      string `json:"nik"`
	Hubungan string `json:"hubungan"`
}

type createBerkasReq struct {
	Tanggal              string           `json:"tanggal"`
	TempatTinggalPewaris string           `json:"tempat_tinggal_pewaris"`
	Pewaris              []pewarisInput   `json:"pewaris"`
	AhliWaris            []ahliWarisInput `json:"ahli_waris"`
	Saksi                []saksiInput     `json:"saksi"`
	PenerimaKuasaIndex   *int             `json:"penerima_kuasa_index"`
	Harta                []string         `json:"harta"`
}

func (r *createBerkasReq) validate() (time.Time, error) {
	tgl, err := time.Parse("2006-01-02", strings.TrimSpace(r.Tanggal))
	if err != nil {
		return time.Time{}, errors.New("tanggal tidak valid (format YYYY-MM-DD)")
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
		if strings.TrimSpace(p.TglMeninggal) == "" || strings.TrimSpace(p.NoSuratKematian) == "" || strings.TrimSpace(p.TglSuratKematian) == "" {
			return time.Time{}, fmt.Errorf("pewaris #%d: tanggal meninggal, no & tgl surat kematian wajib diisi", i+1)
		}
	}
	if len(r.AhliWaris) < 1 {
		return time.Time{}, errors.New("ahli waris minimal 1 orang")
	}
	for i, a := range r.AhliWaris {
		if strings.TrimSpace(a.Nama) == "" || strings.TrimSpace(a.Nik) == "" {
			return time.Time{}, fmt.Errorf("ahli waris #%d: nama dan NIK wajib diisi", i+1)
		}
	}
	if len(r.Saksi) != 2 {
		return time.Time{}, errors.New("saksi harus tepat 2 orang")
	}
	for i, s := range r.Saksi {
		if strings.TrimSpace(s.Nama) == "" {
			return time.Time{}, fmt.Errorf("saksi #%d: nama wajib diisi", i+1)
		}
	}
	if r.PenerimaKuasaIndex != nil {
		if *r.PenerimaKuasaIndex < 0 || *r.PenerimaKuasaIndex >= len(r.AhliWaris) {
			return time.Time{}, errors.New("penerima kuasa tidak valid")
		}
	}
	return tgl, nil
}

// CreateBerkas: POST /api/berkas
func (h *Handler) CreateBerkas(w http.ResponseWriter, r *http.Request) {
	uid, _ := auth.UserIDFromContext(r.Context())

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
type lockError struct{ nik string }

func (e lockError) Error() string {
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
		n, err := qtx.CountPewarisByNik(ctx, strings.TrimSpace(p.Nik))
		if err != nil {
			return 0, err
		}
		if n > 0 {
			return 0, lockError{nik: strings.TrimSpace(p.Nik)}
		}
	}

	// 2. Generate nomor: urutan per tahun + template pengaturan.
	// urutan = max(MAX(existing)+1, urutan_awal+1) — hormati nomor urut awal
	// (migrasi manual→digital) tanpa menabrak berkas yang sudah ada.
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
	nomor := surat.FormatNomor(strOrEmpty(peng.FormatNomor), surat.NomorData{
		Urutan:    int(urutan),
		Bulan:     int(tgl.Month()),
		Tahun:     int(tahun),
		Kelurahan: strOrEmpty(peng.NamaKelurahan),
		Kecamatan: strOrEmpty(peng.Kecamatan),
	})

	// 3. Buat berkas.
	b, err := qtx.CreateBerkas(ctx, db.CreateBerkasParams{
		NomorSurat:           nomor,
		Tahun:                tahun,
		Urutan:               urutan,
		Tanggal:              tgl.Format("2006-01-02"),
		TempatTinggalPewaris: strings.TrimSpace(req.TempatTinggalPewaris),
		CreatedBy:            sql.NullInt64{Int64: uid, Valid: uid != 0},
	})
	if err != nil {
		return 0, err
	}

	// 4. Pewaris (UNIQUE nik = pengaman lock tingkat DB).
	for _, p := range req.Pewaris {
		if _, err := qtx.CreatePewaris(ctx, db.CreatePewarisParams{
			BerkasID:         b.ID,
			Nama:             strings.TrimSpace(p.Nama),
			Nik:              strings.TrimSpace(p.Nik),
			TglMeninggal:     strings.TrimSpace(p.TglMeninggal),
			NoSuratKematian:  strings.TrimSpace(p.NoSuratKematian),
			TglSuratKematian: strings.TrimSpace(p.TglSuratKematian),
		}); err != nil {
			return 0, err
		}
	}

	// 5. Ahli waris — simpan ID untuk resolusi penerima kuasa.
	ahliIDs := make([]int64, 0, len(req.AhliWaris))
	for _, a := range req.AhliWaris {
		created, err := qtx.CreateAhliWaris(ctx, db.CreateAhliWarisParams{
			BerkasID:     b.ID,
			Nama:         strings.TrimSpace(a.Nama),
			Nik:          strings.TrimSpace(a.Nik),
			Umur:         nullInt(a.Umur),
			JenisKelamin: nullStr(strings.TrimSpace(a.JenisKelamin)),
			Agama:        nullStr(strings.TrimSpace(a.Agama)),
			Alamat:       nullStr(strings.TrimSpace(a.Alamat)),
			Keterangan:   nullStr(strings.TrimSpace(a.Keterangan)),
		})
		if err != nil {
			return 0, err
		}
		ahliIDs = append(ahliIDs, created.ID)
	}

	// 6. Saksi.
	for _, s := range req.Saksi {
		if err := qtx.CreateSaksi(ctx, db.CreateSaksiParams{
			BerkasID: b.ID,
			Nama:     strings.TrimSpace(s.Nama),
			Ttl:      nullStr(strings.TrimSpace(s.Ttl)),
			Alamat:   nullStr(strings.TrimSpace(s.Alamat)),
			Nik:      nullStr(strings.TrimSpace(s.Nik)),
			Hubungan: nullStr(strings.TrimSpace(s.Hubungan)),
		}); err != nil {
			return 0, err
		}
	}

	// 7. Harta.
	for _, d := range req.Harta {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		if _, err := qtx.CreateHarta(ctx, db.CreateHartaParams{BerkasID: b.ID, Deskripsi: d}); err != nil {
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

// loadDetail memuat berkas + semua anak. Menerima *db.Queries agar bisa dipakai
// baik di luar maupun (jika perlu) di dalam transaksi.
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
	ht, err := q.ListHartaByBerkas(ctx, id)
	if err != nil {
		return berkasDetail{}, err
	}
	return berkasDetail{
		berkasView: toBerkasView(b),
		Pewaris:    toPewarisViews(pw),
		AhliWaris:  toAhliWarisViews(aw),
		Saksi:      toSaksiViews(sk),
		Harta:      toHartaViews(ht),
	}, nil
}

// ---------- konversi slice ----------

func toPewarisViews(rows []db.Pewaris) []pewarisView {
	out := make([]pewarisView, 0, len(rows))
	for _, p := range rows {
		out = append(out, pewarisView{
			ID: p.ID, Nama: p.Nama, Nik: p.Nik,
			TglMeninggal: p.TglMeninggal, NoSuratKematian: p.NoSuratKematian, TglSuratKematian: p.TglSuratKematian,
		})
	}
	return out
}

func toAhliWarisViews(rows []db.AhliWaris) []ahliWarisView {
	out := make([]ahliWarisView, 0, len(rows))
	for _, a := range rows {
		v := ahliWarisView{
			ID: a.ID, Nama: a.Nama, Nik: a.Nik,
			JenisKelamin: strOrEmpty(a.JenisKelamin), Agama: strOrEmpty(a.Agama),
			Alamat: strOrEmpty(a.Alamat), Keterangan: strOrEmpty(a.Keterangan),
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
			ID: s.ID, Nama: s.Nama, Ttl: strOrEmpty(s.Ttl),
			Alamat: strOrEmpty(s.Alamat), Nik: strOrEmpty(s.Nik), Hubungan: strOrEmpty(s.Hubungan),
		})
	}
	return out
}

func toHartaViews(rows []db.Harta) []hartaView {
	out := make([]hartaView, 0, len(rows))
	for _, h := range rows {
		out = append(out, hartaView{ID: h.ID, Deskripsi: h.Deskripsi})
	}
	return out
}
