package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"surat-waris/internal/db"
)

type nomorAwalView struct {
	Tahun      int64 `json:"tahun"`
	UrutanAwal int64 `json:"urutan_awal"`
}

// ListNomorAwal: GET /api/nomor-awal
func (h *Handler) ListNomorAwal(w http.ResponseWriter, r *http.Request) {
	rows, err := h.q.ListNomorAwal(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal memuat nomor urut awal")
		return
	}
	out := make([]nomorAwalView, 0, len(rows))
	for _, n := range rows {
		out = append(out, nomorAwalView{Tahun: n.Tahun, UrutanAwal: n.UrutanAwal})
	}
	writeJSON(w, http.StatusOK, out)
}

// UpsertNomorAwal: PUT /api/nomor-awal  {tahun, urutan_awal}
func (h *Handler) UpsertNomorAwal(w http.ResponseWriter, r *http.Request) {
	var in nomorAwalView
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "body tidak valid")
		return
	}
	nowYear := int64(time.Now().Year())
	if in.Tahun < 2000 || in.Tahun > nowYear+5 {
		writeErr(w, http.StatusBadRequest, "tahun tidak wajar")
		return
	}
	if in.UrutanAwal < 0 {
		writeErr(w, http.StatusBadRequest, "nomor urut awal tidak boleh negatif")
		return
	}
	if err := h.q.UpsertNomorAwal(r.Context(), db.UpsertNomorAwalParams{
		Tahun:      in.Tahun,
		UrutanAwal: in.UrutanAwal,
	}); err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal menyimpan nomor urut awal")
		return
	}
	writeJSON(w, http.StatusOK, in)
}

// DeleteNomorAwal: DELETE /api/nomor-awal/{tahun}
func (h *Handler) DeleteNomorAwal(w http.ResponseWriter, r *http.Request) {
	tahun, err := strconv.ParseInt(chi.URLParam(r, "tahun"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "tahun tidak valid")
		return
	}
	if err := h.q.DeleteNomorAwal(r.Context(), tahun); err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal menghapus")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
