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

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	seedOnly := flag.Bool("seed", false, "jalankan seeder lalu keluar")
	flag.Parse()

	sqldb, err := openDB()
	if err != nil {
		log.Fatalf("gagal membuka database: %v", err)
	}
	defer sqldb.Close()

	if err := db.Migrate(sqldb); err != nil {
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
	r := newRouter(sqldb, q, mgr)

	ln, port, err := listen(defaultPort)
	if err != nil {
		log.Fatalf("gagal membuka listener: %v", err)
	}

	url := fmt.Sprintf("http://localhost:%s", port)
	srv := &http.Server{Handler: r}

	go func() {
		log.Printf("SIWARIS berjalan di %s", url)
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	time.Sleep(300 * time.Millisecond)
	openBrowser(url)

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

func newRouter(sqldb *sql.DB, q *db.Queries, mgr *auth.Manager) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	authH := auth.NewHandler(q, mgr)
	apiH := handler.New(sqldb, q, parseTemplates())

	// Publik
	r.Post("/api/login", authH.Login)
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
		pr.Post("/api/berkas", apiH.CreateBerkas)
		pr.Get("/api/berkas/{id}", apiH.GetBerkas)

		// Edit terbatas (SPEC §7.2): penerima kuasa + item kuasa + pelengkap penerima kuasa
		pr.Put("/api/berkas/{id}/penerima-kuasa", apiH.SetPenerimaKuasa)
		pr.Put("/api/berkas/{id}/ahli-waris/{ahliId}/pelengkap", apiH.UpdateAhliWarisPelengkap)
		pr.Get("/api/berkas/{id}/kuasa", apiH.ListKuasa)
		pr.Post("/api/berkas/{id}/kuasa", apiH.AddKuasa)
		pr.Put("/api/berkas/{id}/kuasa/{kuasaId}", apiH.UpdateKuasa)
		pr.Delete("/api/berkas/{id}/kuasa/{kuasaId}", apiH.DeleteKuasa)

		// Halaman cetak (html/template, A4) — dibuka di tab baru.
		pr.Get("/berkas/{id}/cetak", apiH.Cetak)
	})

	// UI Svelte (embed) — publik; auth ditangani di dalam SPA via /api/me.
	r.Handle("/*", spaHandler())

	return r
}

func listen(preferred string) (net.Listener, string, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:"+preferred)
	if err == nil {
		return ln, preferred, nil
	}
	log.Printf("port %s terpakai, mencari port bebas...", preferred)
	ln, err = net.Listen("tcp", "127.0.0.1:0")
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
