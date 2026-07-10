package auth

import (
	"context"
	"log"

	"surat-waris/internal/db"
)

// Default kredensial admin awal. Sesuai asumsi: wajib ganti saat login pertama.
const (
	defaultAdminUsername = "admin"
	defaultAdminPassword = "admin123"
	defaultAdminNama     = "Administrator"
	defaultAdminRole     = "admin"
)

// Seed memastikan baris pengaturan (id=1) ada dan membuat user admin default
// bila belum ada user sama sekali. Idempotent: aman dipanggil tiap start.
func Seed(ctx context.Context, q *db.Queries) error {
	if err := q.EnsurePengaturanRow(ctx); err != nil {
		return err
	}

	n, err := q.CountUsers(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}

	hash, err := HashPassword(defaultAdminPassword)
	if err != nil {
		return err
	}
	if _, err := q.CreateUser(ctx, db.CreateUserParams{
		Username:           defaultAdminUsername,
		PasswordHash:       hash,
		Nama:               defaultAdminNama,
		Role:               defaultAdminRole,
		MustChangePassword: 1,
	}); err != nil {
		return err
	}
	log.Printf("seeder: user admin default dibuat (username=%q, password=%q) — WAJIB ganti saat login pertama",
		defaultAdminUsername, defaultAdminPassword)
	return nil
}
