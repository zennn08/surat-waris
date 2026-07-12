package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"surat-waris/internal/db"
)

type pengaturanView struct {
	NamaKelurahan    string `json:"nama_kelurahan"`
	Kecamatan        string `json:"kecamatan"`
	Kota             string `json:"kota"`
	KodeKecamatan    string `json:"kode_kecamatan"`
	KodeKelurahan    string `json:"kode_kelurahan"`
	InstansiKematian string `json:"instansi_kematian"`
}

func toPengaturanView(p db.Pengaturan) pengaturanView {
	return pengaturanView{
		NamaKelurahan:    strOrEmpty(p.NamaKelurahan),
		Kecamatan:        strOrEmpty(p.Kecamatan),
		Kota:             strOrEmpty(p.Kota),
		KodeKecamatan:    strOrEmpty(p.KodeKecamatan),
		KodeKelurahan:    strOrEmpty(p.KodeKelurahan),
		InstansiKematian: strOrEmpty(p.InstansiKematian),
	}
}

// GetPengaturan: GET /api/pengaturan
func (h *Handler) GetPengaturan(w http.ResponseWriter, r *http.Request) {
	p, err := h.q.GetPengaturan(r.Context())
	if errors.Is(err, sql.ErrNoRows) {
		// Baris id=1 belum ada (seharusnya sudah di-seed); balikan kosong.
		writeJSON(w, http.StatusOK, pengaturanView{})
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal memuat pengaturan")
		return
	}
	writeJSON(w, http.StatusOK, toPengaturanView(p))
}

// UpdatePengaturan: PUT /api/pengaturan
func (h *Handler) UpdatePengaturan(w http.ResponseWriter, r *http.Request) {
	var in pengaturanView
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "body tidak valid")
		return
	}
	in.NamaKelurahan = strings.TrimSpace(in.NamaKelurahan)
	in.Kecamatan = strings.TrimSpace(in.Kecamatan)
	in.Kota = strings.TrimSpace(in.Kota)
	in.KodeKecamatan = strings.TrimSpace(in.KodeKecamatan)
	in.KodeKelurahan = strings.TrimSpace(in.KodeKelurahan)
	in.InstansiKematian = strings.TrimSpace(in.InstansiKematian)

	if err := h.q.UpsertPengaturan(r.Context(), db.UpsertPengaturanParams{
		NamaKelurahan:    nullStr(in.NamaKelurahan),
		Kecamatan:        nullStr(in.Kecamatan),
		Kota:             nullStr(in.Kota),
		KodeKecamatan:    nullStr(in.KodeKecamatan),
		KodeKelurahan:    nullStr(in.KodeKelurahan),
		InstansiKematian: nullStr(in.InstansiKematian),
	}); err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal menyimpan pengaturan")
		return
	}
	writeJSON(w, http.StatusOK, in)
}
