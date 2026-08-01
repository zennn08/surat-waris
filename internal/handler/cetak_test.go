package handler

import "testing"

// Aturan yang gampang rusak diam-diam: istri (hidup maupun almarhumah) tidak
// boleh ikut terhitung sebagai "orang anak", dan frasa istri harus menggabung
// istri dari daftar pewaris + daftar ahli waris.
func TestIstriFrasaDanJumlahAnak(t *testing.T) {
	tests := []struct {
		nama       string
		detail     berkasDetail
		wantFrasa  string
		wantJumlah int
		wantAnak   int
	}{
		{
			nama: "satu istri masih hidup",
			detail: berkasDetail{
				Pewaris: []pewarisView{{Nama: "BUDI", Status: "suami"}},
				AhliWaris: []ahliWarisView{
					{Nama: "MARYATI", Keterangan: "Istri"},
					{Nama: "BAGUS", Keterangan: "Anak"},
				},
			},
			wantFrasa: "MARYATI", wantJumlah: 1, wantAnak: 1,
		},
		{
			nama: "dua istri masih hidup",
			detail: berkasDetail{
				Pewaris: []pewarisView{{Nama: "BUDI", Status: "suami"}},
				AhliWaris: []ahliWarisView{
					{Nama: "SITI", Keterangan: "Istri"},
					{Nama: "ANDI", Keterangan: "Anak"},
					{Nama: "RATNA", Keterangan: "Istri"},
					{Nama: "EKA", Keterangan: "anak"},
				},
			},
			wantFrasa: "SITI dan RATNA", wantJumlah: 2, wantAnak: 2,
		},
		{
			nama: "satu istri sudah meninggal, satu masih hidup",
			detail: berkasDetail{
				Pewaris: []pewarisView{{Nama: "BUDI", Status: "suami"}, {Nama: "SITI", Status: "istri"}},
				AhliWaris: []ahliWarisView{
					{Nama: "ANDI", Keterangan: "Anak"},
					{Nama: "RATNA", Keterangan: "Istri"},
				},
			},
			wantFrasa: "Almarhumah SITI dan RATNA", wantJumlah: 2, wantAnak: 1,
		},
		{
			// Istri pertama meninggal lebih dulu: bukan pewaris berkas ini dan
			// bukan ahli waris, namanya hanya ada di kolom "Dari Istri".
			nama: "istri yang hanya tercatat di kolom Dari Istri",
			detail: berkasDetail{
				Pewaris: []pewarisView{{Nama: "ZAINAL", Status: "suami"}},
				AhliWaris: []ahliWarisView{
					{Nama: "NURHAYATI", Keterangan: "Istri"},
					{Nama: "RIZKI", Keterangan: "Anak", DariIstri: "HALIMAH"},
					{Nama: "SALMA", Keterangan: "Anak", DariIstri: "HALIMAH"},
					{Nama: "TAUFIK", Keterangan: "Anak", DariIstri: "NURHAYATI"},
				},
			},
			wantFrasa: "NURHAYATI dan HALIMAH", wantJumlah: 2, wantAnak: 3,
		},
		{
			nama: "tanpa istri di daftar (data berkas lama)",
			detail: berkasDetail{
				Pewaris:   []pewarisView{{Nama: "BUDI", Status: "suami"}},
				AhliWaris: []ahliWarisView{{Nama: "ANDI", Keterangan: "Anak"}, {Nama: "EKA"}},
			},
			wantFrasa: "", wantJumlah: 0, wantAnak: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.nama, func(t *testing.T) {
			frasa, jumlah := istriFrasa(tt.detail)
			if frasa != tt.wantFrasa {
				t.Errorf("frasa = %q, mau %q", frasa, tt.wantFrasa)
			}
			if jumlah != tt.wantJumlah {
				t.Errorf("jumlah istri = %d, mau %d", jumlah, tt.wantJumlah)
			}
			anak := 0
			for _, a := range tt.detail.AhliWaris {
				if pasanganStatus(a.Keterangan) == "" {
					anak++
				}
			}
			if anak != tt.wantAnak {
				t.Errorf("jumlah anak = %d, mau %d", anak, tt.wantAnak)
			}
		})
	}
}
