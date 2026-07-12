package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"surat-waris/internal/db"
)

// Catatan enforcement editability (SPEC §7.2):
// Setelah berkas dibuat, HANYA yang berikut boleh diubah:
//   - penerima_kuasa (berkas_waris.penerima_kuasa_ahli_waris_id)
//   - item kuasa (tambah/edit/hapus)
//   - field pelengkap penerima kuasa (tempat_lahir, tgl_lahir, pekerjaan)
// Ditegakkan struktural: TIDAK ADA endpoint untuk mengubah pewaris / ahli waris
// lain / saksi / tempat tinggal / nomor. Handler di bawah adalah satu-satunya
// jalur edit yang tersedia.

func berkasIDParam(r *http.Request) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
}

// SetPenerimaKuasa: PUT /api/berkas/{id}/penerima-kuasa  {ahli_waris_id}
func (h *Handler) SetPenerimaKuasa(w http.ResponseWriter, r *http.Request) {
	id, err := berkasIDParam(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "id tidak valid")
		return
	}
	if _, err := h.q.GetBerkas(r.Context(), id); errors.Is(err, sql.ErrNoRows) {
		writeErr(w, http.StatusNotFound, "berkas tidak ditemukan")
		return
	} else if err != nil {
		writeErr(w, http.StatusInternalServerError, "kesalahan server")
		return
	}

	var req struct {
		AhliWarisID *int64 `json:"ahli_waris_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "body tidak valid")
		return
	}

	param := db.SetBerkasPenerimaKuasaParams{ID: id}
	if req.AhliWarisID != nil {
		aw, err := h.q.GetAhliWaris(r.Context(), *req.AhliWarisID)
		if errors.Is(err, sql.ErrNoRows) || (err == nil && aw.BerkasID != id) {
			writeErr(w, http.StatusBadRequest, "ahli waris tidak ditemukan di berkas ini")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "kesalahan server")
			return
		}
		param.PenerimaKuasaAhliWarisID = sql.NullInt64{Int64: *req.AhliWarisID, Valid: true}
	}
	if err := h.q.SetBerkasPenerimaKuasa(r.Context(), param); err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal menyimpan penerima kuasa")
		return
	}
	detail, err := h.loadDetail(r.Context(), h.q, id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal memuat berkas")
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

// UpdateAhliWarisPelengkap: PUT /api/berkas/{id}/ahli-waris/{ahliId}/pelengkap
// {tempat_lahir, tgl_lahir, pekerjaan} — field pelengkap penerima kuasa.
func (h *Handler) UpdateAhliWarisPelengkap(w http.ResponseWriter, r *http.Request) {
	id, err := berkasIDParam(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "id tidak valid")
		return
	}
	ahliID, err := strconv.ParseInt(chi.URLParam(r, "ahliId"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "id ahli waris tidak valid")
		return
	}
	aw, err := h.q.GetAhliWaris(r.Context(), ahliID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && aw.BerkasID != id) {
		writeErr(w, http.StatusNotFound, "ahli waris tidak ditemukan di berkas ini")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "kesalahan server")
		return
	}
	var req struct {
		TempatLahir string `json:"tempat_lahir"`
		TglLahir    string `json:"tgl_lahir"`
		Pekerjaan   string `json:"pekerjaan"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "body tidak valid")
		return
	}
	if err := h.q.UpdateAhliWarisPelengkap(r.Context(), db.UpdateAhliWarisPelengkapParams{
		TempatLahir: nullStr(strings.TrimSpace(req.TempatLahir)),
		TglLahir:    nullStr(strings.TrimSpace(req.TglLahir)),
		Pekerjaan:   nullStr(strings.TrimSpace(req.Pekerjaan)),
		ID:          ahliID,
		BerkasID:    id,
	}); err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal menyimpan data penerima kuasa")
		return
	}
	_ = h.q.TouchBerkas(r.Context(), id)
	detail, err := h.loadDetail(r.Context(), h.q, id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal memuat berkas")
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

// ListKuasa: GET /api/berkas/{id}/kuasa
func (h *Handler) ListKuasa(w http.ResponseWriter, r *http.Request) {
	id, err := berkasIDParam(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "id tidak valid")
		return
	}
	rows, err := h.q.ListKuasaItemByBerkas(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal memuat item kuasa")
		return
	}
	writeJSON(w, http.StatusOK, toKuasaViews(rows))
}

// AddKuasa: POST /api/berkas/{id}/kuasa  {deskripsi}
func (h *Handler) AddKuasa(w http.ResponseWriter, r *http.Request) {
	id, err := berkasIDParam(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "id tidak valid")
		return
	}
	if _, err := h.q.GetBerkas(r.Context(), id); errors.Is(err, sql.ErrNoRows) {
		writeErr(w, http.StatusNotFound, "berkas tidak ditemukan")
		return
	} else if err != nil {
		writeErr(w, http.StatusInternalServerError, "kesalahan server")
		return
	}
	var req struct {
		Deskripsi string `json:"deskripsi"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "body tidak valid")
		return
	}
	req.Deskripsi = strings.TrimSpace(req.Deskripsi)
	if req.Deskripsi == "" {
		writeErr(w, http.StatusBadRequest, "deskripsi item kuasa wajib diisi")
		return
	}
	// urutan berikutnya = jumlah item + 1.
	existing, _ := h.q.ListKuasaItemByBerkas(r.Context(), id)
	created, err := h.q.CreateKuasaItem(r.Context(), db.CreateKuasaItemParams{
		BerkasID: id, Urutan: int64(len(existing) + 1), Deskripsi: req.Deskripsi,
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal menambah item kuasa")
		return
	}
	_ = h.q.TouchBerkas(r.Context(), id)
	writeJSON(w, http.StatusCreated, kuasaView{ID: created.ID, Urutan: created.Urutan, Deskripsi: created.Deskripsi})
}

// UpdateKuasa: PUT /api/berkas/{id}/kuasa/{kuasaId}  {deskripsi}
func (h *Handler) UpdateKuasa(w http.ResponseWriter, r *http.Request) {
	id, err := berkasIDParam(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "id tidak valid")
		return
	}
	kuasaID, err := strconv.ParseInt(chi.URLParam(r, "kuasaId"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "id item kuasa tidak valid")
		return
	}
	item, ok := h.kuasaBelongsTo(r, kuasaID, id, w)
	if !ok {
		return
	}
	var req struct {
		Deskripsi string `json:"deskripsi"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "body tidak valid")
		return
	}
	req.Deskripsi = strings.TrimSpace(req.Deskripsi)
	if req.Deskripsi == "" {
		writeErr(w, http.StatusBadRequest, "deskripsi item kuasa wajib diisi")
		return
	}
	if err := h.q.UpdateKuasaItem(r.Context(), db.UpdateKuasaItemParams{Deskripsi: req.Deskripsi, ID: kuasaID}); err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal menyimpan item kuasa")
		return
	}
	_ = h.q.TouchBerkas(r.Context(), id)
	writeJSON(w, http.StatusOK, kuasaView{ID: kuasaID, Urutan: item.Urutan, Deskripsi: req.Deskripsi})
}

// DeleteKuasa: DELETE /api/berkas/{id}/kuasa/{kuasaId}
func (h *Handler) DeleteKuasa(w http.ResponseWriter, r *http.Request) {
	id, err := berkasIDParam(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "id tidak valid")
		return
	}
	kuasaID, err := strconv.ParseInt(chi.URLParam(r, "kuasaId"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "id item kuasa tidak valid")
		return
	}
	if _, ok := h.kuasaBelongsTo(r, kuasaID, id, w); !ok {
		return
	}
	if err := h.q.DeleteKuasaItem(r.Context(), kuasaID); err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal menghapus item kuasa")
		return
	}
	_ = h.q.TouchBerkas(r.Context(), id)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// kuasaBelongsTo memastikan item kuasa ada dan milik berkas yang dimaksud.
func (h *Handler) kuasaBelongsTo(r *http.Request, kuasaID, berkasID int64, w http.ResponseWriter) (db.KuasaItem, bool) {
	item, err := h.q.GetKuasaItem(r.Context(), kuasaID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && item.BerkasID != berkasID) {
		writeErr(w, http.StatusNotFound, "item kuasa tidak ditemukan di berkas ini")
		return db.KuasaItem{}, false
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "kesalahan server")
		return db.KuasaItem{}, false
	}
	return item, true
}
