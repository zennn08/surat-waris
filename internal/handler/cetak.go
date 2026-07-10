package handler

import (
	"database/sql"
	"errors"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

// cetakData adalah view-model untuk ketiga surat.
type cetakData struct {
	B             berkasDetail
	Peng          pengaturanView
	Lurah         *pejabatView
	Camat         *pejabatView
	PenerimaKuasa *ahliWarisView
	PemberiKuasa  []ahliWarisView
	TanggalID     string // "10 Juli 2026"
	TempatTanggal string // "Sukamaju, 10 Juli 2026"
}

var bulanID = [...]string{
	"", "Januari", "Februari", "Maret", "April", "Mei", "Juni",
	"Juli", "Agustus", "September", "Oktober", "November", "Desember",
}

// formatTanggalID mengubah "2006-01-02" → "2 Januari 2006". Bila gagal parse,
// kembalikan apa adanya.
func formatTanggalID(s string) string {
	t, err := time.Parse("2006-01-02", strings.TrimSpace(s))
	if err != nil {
		return s
	}
	return strconv.Itoa(t.Day()) + " " + bulanID[int(t.Month())] + " " + strconv.Itoa(t.Year())
}

// Cetak: GET /berkas/{id}/cetak — render 3 surat A4 dalam satu halaman.
func (h *Handler) Cetak(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "id tidak valid", http.StatusBadRequest)
		return
	}
	detail, err := h.loadDetail(r.Context(), h.q, id)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "berkas tidak ditemukan", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "gagal memuat berkas", http.StatusInternalServerError)
		return
	}

	peng, err := h.q.GetPengaturan(r.Context())
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "gagal memuat pengaturan", http.StatusInternalServerError)
		return
	}
	pv := toPengaturanView(peng)

	data := cetakData{
		B:             detail,
		Peng:          pv,
		Lurah:         h.pejabatAktif(r, "lurah"),
		Camat:         h.pejabatAktif(r, "camat"),
		TanggalID:     formatTanggalID(detail.Tanggal),
		TempatTanggal: tempatTanggal(pv.NamaKelurahan, formatTanggalID(detail.Tanggal)),
	}

	// Bagi ahli waris menjadi penerima & pemberi kuasa (untuk Surat Kuasa).
	if detail.PenerimaKuasaAhliWarisID != nil {
		pkID := *detail.PenerimaKuasaAhliWarisID
		for i := range detail.AhliWaris {
			a := detail.AhliWaris[i]
			if a.ID == pkID {
				cp := a
				data.PenerimaKuasa = &cp
			} else {
				data.PemberiKuasa = append(data.PemberiKuasa, a)
			}
		}
	} else {
		// belum dipilih → semua jadi pemberi kuasa
		data.PemberiKuasa = append(data.PemberiKuasa, detail.AhliWaris...)
	}

	if err := h.tmpl.ExecuteTemplate(w, "cetak", data); err != nil {
		http.Error(w, "gagal merender surat: "+err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) pejabatAktif(r *http.Request, jabatan string) *pejabatView {
	p, err := h.q.GetPejabatAktif(r.Context(), jabatan)
	if err != nil {
		return nil
	}
	v := toPejabatView(p)
	return &v
}

func tempatTanggal(kelurahan, tanggal string) string {
	if strings.TrimSpace(kelurahan) == "" {
		return tanggal
	}
	return kelurahan + ", " + tanggal
}

// UmurStr menampilkan umur untuk template (pointer → string).
func (a ahliWarisView) UmurStr() string {
	if a.Umur == nil {
		return "-"
	}
	return strconv.FormatInt(*a.Umur, 10)
}

// JKStr menampilkan jenis kelamin lengkap.
func (a ahliWarisView) JKStr() string {
	switch a.JenisKelamin {
	case "L":
		return "Laki-laki"
	case "P":
		return "Perempuan"
	default:
		return "-"
	}
}

// TemplateFuncs adalah helper untuk parsing template di main.
func TemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"add":   func(a, b int) int { return a + b },
		"tglID": formatTanggalID,
		"orDash": func(s string) string {
			if strings.TrimSpace(s) == "" {
				return "................."
			}
			return s
		},
	}
}
