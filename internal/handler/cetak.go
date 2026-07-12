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

	"surat-waris/internal/surat"
)

// cetakData adalah view-model untuk ketiga surat.
type cetakData struct {
	B               berkasDetail
	Peng            pengaturanView
	Lurah           *pejabatView
	Camat           *pejabatView
	PenerimaKuasa   *ahliWarisView
	PemberiKuasa    []ahliWarisView
	PewarisFrasa    string // "Almarhum X (Suami) dan Almarhumah Y (Istri)"
	JumlahAhliWaris int
	Terbilang       string // ejaan jumlah ahli waris, mis. "Empat"
	TanggalID       string // "10 Juli 2026"
	TempatTanggal   string // "Dumai, 10 Juli 2026"
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

	tglID := formatTanggalID(detail.TanggalSurat)
	refs := make([]surat.PewarisRef, 0, len(detail.Pewaris))
	for _, p := range detail.Pewaris {
		refs = append(refs, surat.PewarisRef{Nama: p.Nama, Status: p.Status})
	}

	data := cetakData{
		B:               detail,
		Peng:            pv,
		Lurah:           h.pejabatAktif(r, "lurah"),
		Camat:           h.pejabatAktif(r, "camat"),
		PewarisFrasa:    surat.PewarisFrasa(refs),
		JumlahAhliWaris: len(detail.AhliWaris),
		Terbilang:       surat.Terbilang(len(detail.AhliWaris)),
		TanggalID:       tglID,
		TempatTanggal:   tempatTanggal(pv.Kota, tglID),
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
		"gelar": surat.Gelar,
		"statusLabel": func(s string) string {
			if strings.EqualFold(strings.TrimSpace(s), "istri") {
				return "Istri"
			}
			return "Suami"
		},
		"orDash": func(s string) string {
			if strings.TrimSpace(s) == "" {
				return "................."
			}
			return s
		},
	}
}
