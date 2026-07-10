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
	NamaKelurahan string `json:"nama_kelurahan"`
	Kecamatan     string `json:"kecamatan"`
	Kabupaten     string `json:"kabupaten"`
	Provinsi      string `json:"provinsi"`
	FormatNomor   string `json:"format_nomor"`
}

func toPengaturanView(p db.Pengaturan) pengaturanView {
	return pengaturanView{
		NamaKelurahan: strOrEmpty(p.NamaKelurahan),
		Kecamatan:     strOrEmpty(p.Kecamatan),
		Kabupaten:     strOrEmpty(p.Kabupaten),
		Provinsi:      strOrEmpty(p.Provinsi),
		FormatNomor:   strOrEmpty(p.FormatNomor),
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
	in.Kabupaten = strings.TrimSpace(in.Kabupaten)
	in.Provinsi = strings.TrimSpace(in.Provinsi)
	in.FormatNomor = strings.TrimSpace(in.FormatNomor)

	if err := h.q.UpsertPengaturan(r.Context(), db.UpsertPengaturanParams{
		NamaKelurahan: nullStr(in.NamaKelurahan),
		Kecamatan:     nullStr(in.Kecamatan),
		Kabupaten:     nullStr(in.Kabupaten),
		Provinsi:      nullStr(in.Provinsi),
		FormatNomor:   nullStr(in.FormatNomor),
	}); err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal menyimpan pengaturan")
		return
	}
	writeJSON(w, http.StatusOK, in)
}
