package surat

import "testing"

func TestRegNo(t *testing.T) {
	if got := RegNoCamat(88, 2026, "DT"); got != "88/SKAW/DT/2026" {
		t.Errorf("RegNoCamat = %q", got)
	}
	if got := RegNoLurah(88, 2026, "TB", "DT"); got != "88/SKAW/TB-DT/2026" {
		t.Errorf("RegNoLurah = %q", got)
	}
}

func TestTerbilang(t *testing.T) {
	cases := map[int]string{
		0: "Nol", 1: "Satu", 4: "Empat", 10: "Sepuluh", 11: "Sebelas",
		12: "Dua Belas", 19: "Sembilan Belas", 20: "Dua Puluh", 21: "Dua Puluh Satu",
		100: "Seratus", 101: "Seratus Satu", 250: "Dua Ratus Lima Puluh",
		1000: "Seribu", 2026: "Dua Ribu Dua Puluh Enam",
	}
	for in, want := range cases {
		if got := Terbilang(in); got != want {
			t.Errorf("Terbilang(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestPewarisFrasa(t *testing.T) {
	one := []PewarisRef{{Nama: "BUDI", Status: "suami"}}
	if got := PewarisFrasa(one); got != "Almarhum BUDI" {
		t.Errorf("frasa 1 = %q", got)
	}
	two := []PewarisRef{{Nama: "BUDI", Status: "suami"}, {Nama: "ANI", Status: "istri"}}
	want := "Almarhum BUDI (Suami) dan Almarhumah ANI (Istri)"
	if got := PewarisFrasa(two); got != want {
		t.Errorf("frasa 2 = %q, want %q", got, want)
	}
}
