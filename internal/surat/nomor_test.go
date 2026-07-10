package surat

import "testing"

func TestFormatNomor(t *testing.T) {
	d := NomorData{Urutan: 7, Bulan: 7, Tahun: 2026, Kelurahan: "Sukamaju", Kecamatan: "Cibinong"}

	tests := []struct {
		tmpl string
		want string
	}{
		{"470/{urutan3}/{bulan_romawi}/{tahun}", "470/007/VII/2026"},
		{"{urutan}/SKW/{bulan}/{tahun}", "7/SKW/7/2026"},
		{"", "007/SKW/VII/2026"}, // default
		{"Kel {kelurahan} Kec {kecamatan}", "Kel Sukamaju Kec Cibinong"},
		{"{urutan3}-{urutan}", "007-7"}, // {urutan3} tidak ketimpa {urutan}
	}
	for _, tt := range tests {
		if got := FormatNomor(tt.tmpl, d); got != tt.want {
			t.Errorf("FormatNomor(%q) = %q, want %q", tt.tmpl, got, tt.want)
		}
	}
}

func TestRomawi(t *testing.T) {
	cases := map[int]string{1: "I", 4: "IV", 9: "IX", 12: "XII", 0: "", 13: ""}
	for in, want := range cases {
		if got := Romawi(in); got != want {
			t.Errorf("Romawi(%d) = %q, want %q", in, got, want)
		}
	}
}
