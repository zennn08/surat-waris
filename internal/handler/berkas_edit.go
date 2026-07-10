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
// Setelah berkas dibuat, HANYA penerima_kuasa & harta yang bisa diubah.
// Ini ditegakkan secara struktural: TIDAK ADA endpoint untuk mengubah
// pewaris / ahli waris / saksi / tempat tinggal / nomor. Handler di bawah
// adalah satu-satunya jalur edit yang tersedia.

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
		// Validasi: ahli waris harus milik berkas ini.
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

// ListHarta: GET /api/berkas/{id}/harta
func (h *Handler) ListHarta(w http.ResponseWriter, r *http.Request) {
	id, err := berkasIDParam(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "id tidak valid")
		return
	}
	rows, err := h.q.ListHartaByBerkas(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal memuat harta")
		return
	}
	writeJSON(w, http.StatusOK, toHartaViews(rows))
}

// AddHarta: POST /api/berkas/{id}/harta  {deskripsi}
func (h *Handler) AddHarta(w http.ResponseWriter, r *http.Request) {
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
		writeErr(w, http.StatusBadRequest, "deskripsi harta wajib diisi")
		return
	}
	created, err := h.q.CreateHarta(r.Context(), db.CreateHartaParams{BerkasID: id, Deskripsi: req.Deskripsi})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal menambah harta")
		return
	}
	_ = h.q.TouchBerkas(r.Context(), id)
	writeJSON(w, http.StatusCreated, hartaView{ID: created.ID, Deskripsi: created.Deskripsi})
}

// UpdateHarta: PUT /api/berkas/{id}/harta/{hartaId}  {deskripsi}
func (h *Handler) UpdateHarta(w http.ResponseWriter, r *http.Request) {
	id, err := berkasIDParam(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "id tidak valid")
		return
	}
	hartaID, err := strconv.ParseInt(chi.URLParam(r, "hartaId"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "id harta tidak valid")
		return
	}
	if !h.hartaBelongsTo(r, hartaID, id, w) {
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
		writeErr(w, http.StatusBadRequest, "deskripsi harta wajib diisi")
		return
	}
	if err := h.q.UpdateHarta(r.Context(), db.UpdateHartaParams{Deskripsi: req.Deskripsi, ID: hartaID}); err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal menyimpan harta")
		return
	}
	_ = h.q.TouchBerkas(r.Context(), id)
	writeJSON(w, http.StatusOK, hartaView{ID: hartaID, Deskripsi: req.Deskripsi})
}

// DeleteHarta: DELETE /api/berkas/{id}/harta/{hartaId}
func (h *Handler) DeleteHarta(w http.ResponseWriter, r *http.Request) {
	id, err := berkasIDParam(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "id tidak valid")
		return
	}
	hartaID, err := strconv.ParseInt(chi.URLParam(r, "hartaId"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "id harta tidak valid")
		return
	}
	if !h.hartaBelongsTo(r, hartaID, id, w) {
		return
	}
	if err := h.q.DeleteHarta(r.Context(), hartaID); err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal menghapus harta")
		return
	}
	_ = h.q.TouchBerkas(r.Context(), id)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// hartaBelongsTo memastikan harta ada dan milik berkas yang dimaksud.
// Menuliskan error response & mengembalikan false bila tidak.
func (h *Handler) hartaBelongsTo(r *http.Request, hartaID, berkasID int64, w http.ResponseWriter) bool {
	ht, err := h.q.GetHarta(r.Context(), hartaID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && ht.BerkasID != berkasID) {
		writeErr(w, http.StatusNotFound, "harta tidak ditemukan di berkas ini")
		return false
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "kesalahan server")
		return false
	}
	return true
}
