package handler

// Ekspor daftar berkas ke .xlsx. File xlsx adalah zip berisi beberapa XML;
// ditulis langsung dengan archive/zip agar tanpa dependensi baru.
// ponytail: semua sel bertipe teks (inlineStr) — reg no & NIK memang harus
// teks agar Excel tidak memotong digit; ganti ke library bila kelak butuh
// styling/formula.

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"surat-waris/internal/db"
)

// ExportBerkasExcel: GET /api/berkas/export-excel?q=...
// Mengunduh daftar berkas (mengikuti kata kunci pencarian bila ada) sebagai xlsx.
func (h *Handler) ExportBerkasExcel(w http.ResponseWriter, r *http.Request) {
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

	data := [][]string{{
		"No", "Reg. No. Kelurahan", "Tgl Reg. Lurah", "Reg. No. Camat", "Tgl Reg. Camat",
		"Pewaris", "NIK Pewaris", "Tanggal Surat", "Tempat Tinggal Terakhir", "Status",
	}}
	for i, b := range rows {
		pw, err := h.q.ListPewarisByBerkas(r.Context(), b.ID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "gagal memuat pewaris")
			return
		}
		var nama, nik []string
		for _, p := range pw {
			nama = append(nama, p.Nama)
			nik = append(nik, p.Nik)
		}
		data = append(data, []string{
			fmt.Sprint(i + 1),
			b.RegNoLurah, strOrEmpty(b.TanggalRegLurah),
			b.RegNoCamat, strOrEmpty(b.TanggalRegCamat),
			strings.Join(nama, ", "), strings.Join(nik, ", "),
			b.TanggalSurat, b.TempatTinggalPewaris, b.Status,
		})
	}

	name := "siwaris-daftar-berkas-" + time.Now().Format("2006-01-02") + ".xlsx"
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", name))
	widths := []int{5, 24, 13, 22, 13, 28, 22, 13, 30, 10}
	if err := writeXLSX(w, "Daftar Berkas", widths, data); err != nil {
		// header sudah terkirim; cukup catat di log via panic-recoverer? cukup abaikan.
		return
	}
}

var xmlEsc = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")

// writeXLSX menulis workbook satu sheet, semua sel teks (inlineStr).
func writeXLSX(out io.Writer, sheetName string, colWidths []int, rows [][]string) error {
	z := zip.NewWriter(out)

	static := map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Default Extension="xml" ContentType="application/xml"/>
<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
<Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
</Types>`,
		"_rels/.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>
</Relationships>`,
		"xl/workbook.xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
<sheets><sheet name="` + xmlEsc.Replace(sheetName) + `" sheetId="1" r:id="rId1"/></sheets>
</workbook>`,
		"xl/_rels/workbook.xml.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>
</Relationships>`,
	}
	for path, content := range static {
		f, err := z.Create(path)
		if err != nil {
			return err
		}
		if _, err := io.WriteString(f, content); err != nil {
			return err
		}
	}

	f, err := z.Create("xl/worksheets/sheet1.xml")
	if err != nil {
		return err
	}
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">`)
	if len(colWidths) > 0 {
		sb.WriteString("<cols>")
		for i, wd := range colWidths {
			fmt.Fprintf(&sb, `<col min="%d" max="%d" width="%d" customWidth="1"/>`, i+1, i+1, wd)
		}
		sb.WriteString("</cols>")
	}
	sb.WriteString("<sheetData>")
	for _, row := range rows {
		sb.WriteString("<row>")
		for _, cell := range row {
			sb.WriteString(`<c t="inlineStr"><is><t xml:space="preserve">`)
			sb.WriteString(xmlEsc.Replace(cell))
			sb.WriteString(`</t></is></c>`)
		}
		sb.WriteString("</row>")
	}
	sb.WriteString("</sheetData></worksheet>")
	if _, err := io.WriteString(f, sb.String()); err != nil {
		return err
	}

	return z.Close()
}
