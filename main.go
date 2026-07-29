package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"surat-waris/internal/auth"
	"surat-waris/internal/db"
	"surat-waris/internal/handler"
)

const (
	defaultPort = "8080"
	dbFileName  = "surat-waris.db"
)

// version diisi saat build via -ldflags "-X main.version=v1.x.x" (lihat CI/Makefile).
var version = "dev"

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	seedOnly := flag.Bool("seed", false, "jalankan seeder lalu keluar")
	portFlag := flag.String("port", defaultPort, "port HTTP (dipakai internal saat pembaruan otomatis)")
	flag.Parse()
	// --port eksplisit = proses hasil pembaruan otomatis: tunggu port lama
	// dilepas proses sebelumnya dan jangan buka tab browser baru.
	spawned := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "port" {
			spawned = true
		}
	})

	cleanupOldExe()

	sqldb, err := openDB()
	if err != nil {
		log.Fatalf("gagal membuka database: %v", err)
	}
	defer sqldb.Close()

	dir, err := exeDir()
	if err != nil {
		log.Fatalf("gagal menentukan folder aplikasi: %v", err)
	}
	if err := db.Migrate(sqldb, dir); err != nil {
		if errors.Is(err, db.ErrDBNewer) {
			sqldb.Close()
			fatalPage("Database dari versi lebih baru",
				"File data <b>surat-waris.db</b> dibuat oleh versi aplikasi yang lebih baru, "+
					"sehingga aplikasi versi lama ini tidak berani membukanya (mencegah kerusakan data). "+
					"Silakan jalankan <b>siwaris.exe versi terbaru</b>. Bila memang ingin kembali ke versi lama, "+
					"impor file cadangan <b>surat-waris-pra-migrasi-*.db</b> yang ada di folder aplikasi.")
		}
		log.Fatalf("gagal migrasi: %v", err)
	}

	q := db.New(sqldb)

	if err := auth.Seed(context.Background(), q); err != nil {
		log.Fatalf("gagal seed: %v", err)
	}
	if *seedOnly {
		log.Println("seeder selesai")
		return
	}

	mgr := auth.NewManager()
	u := &updater{sqldb: sqldb}
	r := newRouter(sqldb, q, mgr, u)

	ln, port, err := listen(*portFlag, spawned)
	if err != nil {
		log.Fatalf("gagal membuka listener: %v", err)
	}

	url := fmt.Sprintf("http://localhost:%s", port)
	srv := &http.Server{Handler: r}

	// restart pasca-pembaruan: matikan server & DB, jalankan exe baru di port
	// yang sama, lalu keluar. Proses baru menunggu port dilepas (flag --port).
	u.restart = func() {
		time.Sleep(500 * time.Millisecond) // beri waktu respons /apply terkirim
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		sqldb.Close()
		exe, err := os.Executable()
		if err == nil {
			if err := exec.Command(exe, "--port", port).Start(); err != nil {
				log.Printf("gagal menjalankan exe baru: %v", err)
			}
		}
		os.Exit(0)
	}

	go func() {
		log.Printf("SIWARIS berjalan di %s", url)
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	time.Sleep(300 * time.Millisecond)
	if !spawned {
		openBrowser(url)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	log.Println("server berhenti")
}

// openDB membuka/membuat surat-waris.db di direktori exe (WAL, foreign keys).
func openDB() (*sql.DB, error) {
	dir, err := exeDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, dbFileName)
	sqldb, err := db.Open(path)
	if err != nil {
		return nil, err
	}
	log.Printf("database siap: %s (WAL)", path)
	return sqldb, nil
}

// exeDir mengembalikan direktori binary agar DB tersimpan di samping exe.
func exeDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(exe), nil
}

func newRouter(sqldb *sql.DB, q *db.Queries, mgr *auth.Manager, u *updater) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	authH := auth.NewHandler(q, mgr)
	apiH := handler.New(sqldb, q, parseTemplates())

	// Publik
	r.Post("/api/login", authH.Login)
	r.Get("/api/version", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"version":%q}`, version)
	})
	r.Get("/healthz", func(w http.ResponseWriter, req *http.Request) {
		if err := sqldb.PingContext(req.Context()); err != nil {
			http.Error(w, "db down", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"ok"}`)
	})

	// Butuh sesi
	r.Group(func(pr chi.Router) {
		pr.Use(mgr.RequireAuth)
		pr.Post("/api/logout", authH.Logout)
		pr.Get("/api/me", authH.Me)
		pr.Post("/api/change-password", authH.ChangePassword)

		// Master data — Pejabat
		pr.Get("/api/pejabat", apiH.ListPejabat)
		pr.Post("/api/pejabat", apiH.CreatePejabat)
		pr.Put("/api/pejabat/{id}", apiH.UpdatePejabat)
		pr.Delete("/api/pejabat/{id}", apiH.DeletePejabat)

		// Master data — Pengaturan
		pr.Get("/api/pengaturan", apiH.GetPengaturan)
		pr.Put("/api/pengaturan", apiH.UpdatePengaturan)

		// Nomor urut awal per tahun (migrasi manual→digital)
		pr.Get("/api/nomor-awal", apiH.ListNomorAwal)
		pr.Put("/api/nomor-awal", apiH.UpsertNomorAwal)
		pr.Delete("/api/nomor-awal/{tahun}", apiH.DeleteNomorAwal)

		// Berkas waris (inti)
		pr.Get("/api/berkas", apiH.ListBerkas)
		pr.Get("/api/berkas/export-excel", apiH.ExportBerkasExcel)
		pr.Post("/api/berkas", apiH.CreateBerkas)
		pr.Get("/api/berkas/{id}", apiH.GetBerkas)
		pr.Put("/api/berkas/{id}", apiH.UpdateBerkas)

		// Edit terbatas (SPEC §7.2): penerima kuasa + item kuasa + pelengkap penerima kuasa
		pr.Put("/api/berkas/{id}/penerima-kuasa", apiH.SetPenerimaKuasa)
		pr.Put("/api/berkas/{id}/ahli-waris/{ahliId}/pelengkap", apiH.UpdateAhliWarisPelengkap)
		pr.Get("/api/berkas/{id}/kuasa", apiH.ListKuasa)
		pr.Post("/api/berkas/{id}/kuasa", apiH.AddKuasa)
		pr.Put("/api/berkas/{id}/kuasa/{kuasaId}", apiH.UpdateKuasa)
		pr.Delete("/api/berkas/{id}/kuasa/{kuasaId}", apiH.DeleteKuasa)

		// Cadangan & pindah data
		pr.Get("/api/database/export", exportDatabase(sqldb))
		pr.Post("/api/database/import", importDatabase(sqldb))

		// Pembaruan aplikasi
		pr.Get("/api/update/check", u.check)
		pr.Post("/api/update/apply", u.apply)

		// Halaman cetak (html/template, A4) — dibuka di tab baru.
		pr.Get("/berkas/{id}/cetak", apiH.Cetak)
	})

	// UI Svelte (embed) — publik; auth ditangani di dalam SPA via /api/me.
	r.Handle("/*", spaHandler())

	return r
}

// listen membuka port pilihan; wait=true (proses hasil pembaruan) menunggu
// sampai 10 detik agar proses lama sempat melepas port yang sama.
func listen(preferred string, wait bool) (net.Listener, string, error) {
	deadline := time.Now().Add(10 * time.Second)
	for {
		ln, err := net.Listen("tcp", "127.0.0.1:"+preferred)
		if err == nil {
			return ln, preferred, nil
		}
		if !wait || time.Now().After(deadline) {
			break
		}
		time.Sleep(300 * time.Millisecond)
	}
	log.Printf("port %s terpakai, mencari port bebas...", preferred)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, "", err
	}
	_, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		ln.Close()
		return nil, "", err
	}
	return ln, port, nil
}

// fatalPage menampilkan pesan fatal lewat browser (build windowsgui tidak
// punya konsol, log.Fatal tidak akan terlihat pengguna), lalu menunggu ditutup.
func fatalPage(title, msgHTML string) {
	ln, port, err := listen(defaultPort, false)
	if err != nil {
		log.Fatalf("%s", title)
	}
	page := fmt.Sprintf(`<!doctype html><html lang="id"><head><meta charset="utf-8"><title>SIWARIS</title></head>
<body style="font-family:'Segoe UI',Arial,sans-serif;max-width:560px;margin:4rem auto;padding:0 1rem;color:#1a2433;line-height:1.6">
<h2 style="color:#c62828">%s</h2><p>%s</p>
<p style="color:#55606d;font-size:.9em">Aplikasi tidak melanjutkan proses apa pun; data Anda tidak disentuh. Tutup jendela ini setelah selesai membaca.</p>
</body></html>`, title, msgHTML)
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, page)
	})}
	go srv.Serve(ln)
	openBrowser(fmt.Sprintf("http://localhost:%s", port))
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
	os.Exit(1)
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		log.Printf("tidak bisa membuka browser otomatis (%v). Buka manual: %s", err, url)
	}
}
