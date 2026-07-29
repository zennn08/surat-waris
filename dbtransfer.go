package main

// Ekspor/impor database untuk pindah komputer atau cadangan.
// Ekspor: VACUUM INTO -> unduh satu file .db snapshot konsisten.
// Impor: validasi file, cadangkan DB lama, lalu salin seluruh isi
// via ATTACH dalam satu transaksi (tanpa restart aplikasi).

import (
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"surat-waris/internal/db"
)

const sqliteHeader = "SQLite format 3\x00"

func writeJSONErr(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	fmt.Fprintf(w, `{"error":%q}`, msg)
}

// exportDatabase mengunduh snapshot database sebagai satu file .db.
func exportDatabase(sqldb *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dir, err := exeDir()
		if err != nil {
			writeJSONErr(w, 500, "gagal menentukan folder aplikasi")
			return
		}
		tmp := filepath.Join(dir, fmt.Sprintf("siwaris-export-%d.tmp", time.Now().UnixNano()))
		defer os.Remove(tmp)

		if _, err := sqldb.ExecContext(r.Context(), "VACUUM INTO ?", tmp); err != nil {
			log.Printf("ekspor db: %v", err)
			writeJSONErr(w, 500, "gagal membuat cadangan database")
			return
		}

		name := "siwaris-cadangan-" + time.Now().Format("2006-01-02") + ".db"
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", name))
		http.ServeFile(w, r, tmp)
	}
}

// importDatabase mengganti seluruh isi database dengan file cadangan yang diunggah.
func importDatabase(sqldb *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 200<<20)
		file, _, err := r.FormFile("file")
		if err != nil {
			writeJSONErr(w, 400, "file cadangan tidak ditemukan pada unggahan")
			return
		}
		defer file.Close()

		dir, err := exeDir()
		if err != nil {
			writeJSONErr(w, 500, "gagal menentukan folder aplikasi")
			return
		}

		// Simpan unggahan ke file sementara di samping exe (satu volume dengan DB).
		tmp := filepath.Join(dir, fmt.Sprintf("siwaris-import-%d.tmp", time.Now().UnixNano()))
		if err := saveUpload(file, tmp); err != nil {
			log.Printf("impor db, simpan unggahan: %v", err)
			writeJSONErr(w, 500, "gagal menyimpan file unggahan")
			return
		}
		defer func() {
			os.Remove(tmp)
			os.Remove(tmp + "-wal")
			os.Remove(tmp + "-shm")
		}()

		if err := validateImport(tmp); err != nil {
			writeJSONErr(w, 400, err.Error())
			return
		}

		// Cadangkan data saat ini sebelum ditimpa.
		backup := filepath.Join(dir, "siwaris-backup-sebelum-impor-"+time.Now().Format("2006-01-02-150405")+".db")
		if _, err := sqldb.ExecContext(r.Context(), "VACUUM INTO ?", backup); err != nil {
			log.Printf("impor db, backup: %v", err)
			writeJSONErr(w, 500, "gagal mencadangkan data lama, impor dibatalkan")
			return
		}

		if err := copyAllTables(r, sqldb, tmp); err != nil {
			log.Printf("impor db, salin data: %v", err)
			writeJSONErr(w, 500, "gagal menyalin data, database lama tidak berubah")
			return
		}

		log.Printf("impor database berhasil, data lama dicadangkan ke %s", backup)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"ok":true,"backup":%q}`, filepath.Base(backup))
	}
}

func saveUpload(src io.Reader, dst string) error {
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, src)
	return err
}

// validateImport memastikan file adalah database SIWARIS dan menaikkan
// skemanya ke versi terkini (agar cadangan dari versi lama tetap bisa diimpor).
func validateImport(path string) error {
	head := make([]byte, len(sqliteHeader))
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("gagal membaca file unggahan")
	}
	n, _ := io.ReadFull(f, head)
	f.Close()
	if n < len(sqliteHeader) || !bytes.Equal(head, []byte(sqliteHeader)) {
		return fmt.Errorf("file bukan database SIWARIS (.db hasil ekspor)")
	}

	up, err := db.Open(path)
	if err != nil {
		return fmt.Errorf("file tidak bisa dibuka sebagai database")
	}
	defer up.Close()
	if err := db.Migrate(up, ""); err != nil {
		return fmt.Errorf("file bukan database SIWARIS yang valid")
	}
	for _, t := range []string{"users", "pengaturan", "berkas_waris"} {
		var n int
		if err := up.QueryRow("SELECT count(*) FROM " + t).Scan(&n); err != nil {
			return fmt.Errorf("file bukan database SIWARIS yang valid (tabel %s hilang)", t)
		}
	}
	return nil
}

// copyAllTables menyalin seluruh isi src ke database utama dalam satu transaksi.
// Daftar tabel dan kolom diambil dari sqlite_master milik database utama sendiri,
// jadi tabel baru di skema mendatang otomatis ikut tersalin.
func copyAllTables(r *http.Request, sqldb *sql.DB, srcPath string) error {
	ctx := r.Context()
	conn, err := sqldb.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, "PRAGMA foreign_keys=OFF"); err != nil {
		return err
	}
	defer conn.ExecContext(ctx, "PRAGMA foreign_keys=ON")

	if _, err := conn.ExecContext(ctx, "ATTACH ? AS src", srcPath); err != nil {
		return err
	}
	defer conn.ExecContext(ctx, "DETACH src")

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx,
		"SELECT name FROM main.sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'")
	if err != nil {
		return err
	}
	var tables []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			rows.Close()
			return err
		}
		tables = append(tables, t)
	}
	rows.Close()

	for _, t := range tables {
		cols, err := tableColumns(tx, t)
		if err != nil {
			return err
		}
		q := quoteIdent(t)
		if _, err := tx.ExecContext(ctx, "DELETE FROM main."+q); err != nil {
			return fmt.Errorf("kosongkan %s: %w", t, err)
		}
		ins := fmt.Sprintf("INSERT INTO main.%s (%s) SELECT %s FROM src.%s", q, cols, cols, q)
		if _, err := tx.ExecContext(ctx, ins); err != nil {
			return fmt.Errorf("salin %s: %w", t, err)
		}
	}

	// Selaraskan penghitung AUTOINCREMENT agar id baru tidak menabrak data impor.
	// sqlite_sequence baru ada setelah insert pertama, jadi cek dulu di kedua sisi.
	for _, side := range []string{"main", "src"} {
		var n int
		err := tx.QueryRowContext(ctx,
			fmt.Sprintf("SELECT count(*) FROM %s.sqlite_master WHERE name='sqlite_sequence'", side)).Scan(&n)
		if err != nil {
			return err
		}
		if n == 0 {
			continue
		}
		if side == "main" {
			_, err = tx.ExecContext(ctx, "DELETE FROM main.sqlite_sequence")
		} else {
			_, err = tx.ExecContext(ctx, "INSERT INTO main.sqlite_sequence SELECT * FROM src.sqlite_sequence")
		}
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// tableColumns mengembalikan daftar kolom ter-quote, dipisah koma, urut sesuai
// skema utama. Kolom eksplisit membuat salinan aman walau urutan kolom src berbeda.
func tableColumns(tx *sql.Tx, table string) (string, error) {
	rows, err := tx.Query("SELECT name FROM pragma_table_info(?)", table)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var cols []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return "", err
		}
		cols = append(cols, quoteIdent(c))
	}
	return strings.Join(cols, ", "), rows.Err()
}

func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}
