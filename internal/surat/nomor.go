// Package surat berisi business logic surat waris: generator nomor register,
// terbilang, dan frasa pewaris.
package surat

import (
	"fmt"
	"strings"
)

// RegNoCamat: "{urutan}/SKAW/{kode_kecamatan}/{tahun}" → "88/SKAW/DT/2026".
func RegNoCamat(urutan, tahun int, kodeKecamatan string) string {
	return fmt.Sprintf("%d/SKAW/%s/%d", urutan, kodeKecamatan, tahun)
}

// RegNoLurah: "{urutan}/SKAW/{kode_kelurahan}-{kode_kecamatan}/{tahun}"
// → "88/SKAW/TB-DT/2026".
func RegNoLurah(urutan, tahun int, kodeKelurahan, kodeKecamatan string) string {
	return fmt.Sprintf("%d/SKAW/%s-%s/%d", urutan, kodeKelurahan, kodeKecamatan, tahun)
}

// ---------- Terbilang ----------

var satuan = [...]string{
	"Nol", "Satu", "Dua", "Tiga", "Empat", "Lima", "Enam",
	"Tujuh", "Delapan", "Sembilan", "Sepuluh", "Sebelas",
}

// Terbilang mengubah bilangan bulat ke ejaan Indonesia (Title Case), mis.
// 4 → "Empat", 21 → "Dua Puluh Satu". ponytail: cukup s/d jutaan; jumlah anak
// tak akan melampaui itu.
func Terbilang(n int) string {
	switch {
	case n < 0:
		return "Minus " + Terbilang(-n)
	case n < 12:
		return satuan[n]
	case n < 20:
		return Terbilang(n-10) + " Belas"
	case n < 100:
		return Terbilang(n/10) + " Puluh" + spaceWord(Terbilang(n%10))
	case n < 200:
		return "Seratus" + spaceWord(Terbilang(n-100))
	case n < 1000:
		return Terbilang(n/100) + " Ratus" + spaceWord(Terbilang(n%100))
	case n < 2000:
		return "Seribu" + spaceWord(Terbilang(n-1000))
	case n < 1000000:
		return Terbilang(n/1000) + " Ribu" + spaceWord(Terbilang(n%1000))
	default:
		return Terbilang(n/1000000) + " Juta" + spaceWord(Terbilang(n%1000000))
	}
}

// spaceWord memberi spasi di depan bila w non-kosong & bukan "Nol".
func spaceWord(w string) string {
	if w == "" || w == "Nol" {
		return ""
	}
	return " " + w
}

// ---------- Frasa pewaris ----------

// PewarisRef adalah data minimal pewaris untuk membentuk frasa surat.
type PewarisRef struct {
	Nama   string
	Status string // "suami" | "istri"
}

// Gelar mengembalikan "Almarhum" (suami) atau "Almarhumah" (istri).
func Gelar(status string) string {
	if strings.EqualFold(strings.TrimSpace(status), "istri") {
		return "Almarhumah"
	}
	return "Almarhum"
}

func labelStatus(status string) string {
	if strings.EqualFold(strings.TrimSpace(status), "istri") {
		return "Istri"
	}
	return "Suami"
}

// PewarisFrasa membentuk frasa gabungan pewaris (SPEC §11):
//   - 1 pewaris: "Almarhum {Nama}" / "Almarhumah {Nama}"
//   - 2 pewaris: "Almarhum {P1} (Suami) dan Almarhumah {P2} (Istri)"
func PewarisFrasa(ps []PewarisRef) string {
	switch len(ps) {
	case 0:
		return ""
	case 1:
		return Gelar(ps[0].Status) + " " + ps[0].Nama
	default:
		parts := make([]string, 0, len(ps))
		for _, p := range ps {
			parts = append(parts, fmt.Sprintf("%s %s (%s)", Gelar(p.Status), p.Nama, labelStatus(p.Status)))
		}
		return strings.Join(parts, " dan ")
	}
}
