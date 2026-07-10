package handler

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"net/http"

	"surat-waris/internal/db"
)

// Handler menampung dependency untuk semua HTTP handler API + halaman cetak.
type Handler struct {
	sqldb *sql.DB
	q     *db.Queries
	tmpl  *template.Template // template cetak (html/template)
}

func New(sqldb *sql.DB, q *db.Queries, tmpl *template.Template) *Handler {
	return &Handler{sqldb: sqldb, q: q, tmpl: tmpl}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// nullStr membungkus string ke sql.NullString (kosong = NULL).
func nullStr(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

// strOrEmpty membuka sql.NullString menjadi string biasa untuk JSON.
func strOrEmpty(n sql.NullString) string {
	if n.Valid {
		return n.String
	}
	return ""
}

// nullInt membungkus *int64 ke sql.NullInt64 (nil = NULL).
func nullInt(p *int64) sql.NullInt64 {
	if p == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *p, Valid: true}
}
