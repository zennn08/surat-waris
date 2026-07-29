package main

// Pembaruan otomatis dari GitHub Releases.
// Cek versi: baca redirect /releases/latest (tanpa API token/JSON).
// Pasang: unduh exe baru, rename exe berjalan ke .old (Windows mengizinkan),
// taruh exe baru, jalankan proses baru di port yang sama, proses lama keluar.

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	releaseLatestURL = "https://github.com/zennn08/surat-waris/releases/latest"
	releaseExeURL    = releaseLatestURL + "/download/siwaris.exe"
)

type updater struct {
	sqldb   *sql.DB
	restart func() // di-set di main setelah server berdiri
}

// latestVersion membaca tag rilis terbaru: coba header redirect GitHub dulu
// (tanpa batas laju), lalu fallback ke REST API. Redirect terbukti bisa
// sementara menunjuk halaman indeks (bukan /tag/...) sesaat setelah rilis
// dihapus atau ditata ulang.
func latestVersion(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, releaseLatestURL, nil)
	if err != nil {
		return "", err
	}
	client := &http.Client{
		Timeout:       8 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	resp.Body.Close()
	loc := resp.Header.Get("Location")
	if i := strings.LastIndex(loc, "/tag/"); i >= 0 {
		return loc[i+len("/tag/"):], nil
	}
	return latestVersionAPI(ctx)
}

func latestVersionAPI(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://api.github.com/repos/zennn08/surat-waris/releases/latest", nil)
	if err != nil {
		return "", err
	}
	resp, err := (&http.Client{Timeout: 8 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API rilis: HTTP %d", resp.StatusCode)
	}
	var out struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&out); err != nil {
		return "", err
	}
	if out.TagName == "" {
		return "", fmt.Errorf("API rilis: tag kosong")
	}
	return out.TagName, nil
}

func parseVer(v string) (out [3]int, ok bool) {
	parts := strings.SplitN(strings.TrimPrefix(v, "v"), ".", 3)
	if len(parts) != 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

// newerThan: true bila latest lebih baru dari current. Versi tanpa format
// vX.Y.Z (mis. "dev") tidak pernah ditawari pembaruan.
func newerThan(latest, current string) bool {
	l, ok1 := parseVer(latest)
	c, ok2 := parseVer(current)
	if !ok1 || !ok2 {
		return false
	}
	for i := 0; i < 3; i++ {
		if l[i] != c[i] {
			return l[i] > c[i]
		}
	}
	return false
}

// check: GET /api/update/check
func (u *updater) check(w http.ResponseWriter, r *http.Request) {
	latest, err := latestVersion(r.Context())
	if err != nil {
		writeJSONErr(w, http.StatusBadGateway, "tidak bisa memeriksa versi terbaru (butuh koneksi internet)")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"current":%q,"latest":%q,"available":%v}`, version, latest, newerThan(latest, version))
}

// apply: POST /api/update/apply
func (u *updater) apply(w http.ResponseWriter, r *http.Request) {
	latest, err := latestVersion(r.Context())
	if err != nil || !newerThan(latest, version) {
		writeJSONErr(w, http.StatusBadRequest, "tidak ada versi baru untuk dipasang")
		return
	}

	exe, err := os.Executable()
	if err != nil {
		writeJSONErr(w, http.StatusInternalServerError, "gagal menentukan lokasi aplikasi")
		return
	}
	tmp := exe + ".update"
	if err := downloadFile(r.Context(), releaseExeURL, tmp); err != nil {
		os.Remove(tmp)
		log.Printf("update: unduh gagal: %v", err)
		writeJSONErr(w, http.StatusBadGateway, "gagal mengunduh pembaruan, coba lagi nanti")
		return
	}
	if err := validExe(tmp); err != nil {
		os.Remove(tmp)
		log.Printf("update: file tidak valid: %v", err)
		writeJSONErr(w, http.StatusBadGateway, "file pembaruan tidak valid, coba lagi nanti")
		return
	}

	old := exe + ".old"
	os.Remove(old)
	if err := os.Rename(exe, old); err != nil {
		os.Remove(tmp)
		log.Printf("update: rename exe lama: %v", err)
		writeJSONErr(w, http.StatusInternalServerError, "gagal menyiapkan pembaruan")
		return
	}
	if err := os.Rename(tmp, exe); err != nil {
		os.Rename(old, exe) // pulihkan
		os.Remove(tmp)
		log.Printf("update: pasang exe baru: %v", err)
		writeJSONErr(w, http.StatusInternalServerError, "gagal memasang pembaruan")
		return
	}

	log.Printf("pembaruan %s terpasang, memulai ulang aplikasi...", latest)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"ok":true,"latest":%q}`, latest)
	go u.restart()
}

func downloadFile(ctx context.Context, url, dst string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := (&http.Client{Timeout: 10 * time.Minute}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// validExe: cek minimal file adalah executable Windows (header MZ) dan tidak
// terpotong (unduhan gagal di tengah biasanya jauh lebih kecil).
func validExe(path string) error {
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	if fi.Size() < 5<<20 {
		return fmt.Errorf("ukuran terlalu kecil: %d byte", fi.Size())
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	head := make([]byte, 2)
	if _, err := io.ReadFull(f, head); err != nil {
		return err
	}
	if string(head) != "MZ" {
		return fmt.Errorf("bukan executable Windows")
	}
	return nil
}

// cleanupOldExe menghapus sisa exe lama pasca-pembaruan. Saat proses baru
// mulai, proses lama mungkin masih hidup sebentar dan mengunci file .old,
// jadi dicoba berulang di latar belakang.
func cleanupOldExe() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	old := exe + ".old"
	go func() {
		for i := 0; i < 15; i++ {
			err := os.Remove(old)
			if err == nil || errors.Is(err, fs.ErrNotExist) {
				return
			}
			time.Sleep(2 * time.Second)
		}
	}()
}
