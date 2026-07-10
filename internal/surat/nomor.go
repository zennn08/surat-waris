// Package surat berisi business logic surat waris: generator nomor, dll.
package surat

import (
	"fmt"
	"strconv"
	"strings"
)

// DefaultFormatNomor dipakai bila pengaturan.format_nomor kosong.
const DefaultFormatNomor = "{urutan3}/SKW/{bulan_romawi}/{tahun}"

var bulanRomawi = [...]string{
	"", "I", "II", "III", "IV", "V", "VI",
	"VII", "VIII", "IX", "X", "XI", "XII",
}

// NomorData adalah nilai yang disubstitusikan ke template nomor surat.
type NomorData struct {
	Urutan    int
	Bulan     int // 1-12
	Tahun     int
	Kelurahan string
	Kecamatan string
}

// Romawi mengembalikan angka Romawi untuk bulan 1-12 (kosong jika di luar rentang).
func Romawi(bulan int) string {
	if bulan < 1 || bulan > 12 {
		return ""
	}
	return bulanRomawi[bulan]
}

// FormatNomor merender template dengan mengganti placeholder yang didukung:
//
//	{urutan} {urutan3} {bulan} {bulan_romawi} {tahun} {kelurahan} {kecamatan}
//
// Bila template kosong, dipakai DefaultFormatNomor.
func FormatNomor(template string, d NomorData) string {
	if strings.TrimSpace(template) == "" {
		template = DefaultFormatNomor
	}
	r := strings.NewReplacer(
		"{urutan3}", fmt.Sprintf("%03d", d.Urutan),
		"{urutan}", strconv.Itoa(d.Urutan),
		"{bulan_romawi}", Romawi(d.Bulan),
		"{bulan}", strconv.Itoa(d.Bulan),
		"{tahun}", strconv.Itoa(d.Tahun),
		"{kelurahan}", d.Kelurahan,
		"{kecamatan}", d.Kecamatan,
	)
	return r.Replace(template)
}
