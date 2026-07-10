package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"surat-waris/internal/db"
)

type pejabatView struct {
	ID        int64  `json:"id"`
	Jabatan   string `json:"jabatan"`
	Nama      string `json:"nama"`
	Nip       string `json:"nip"`
	Aktif     bool   `json:"aktif"`
	CreatedAt string `json:"created_at"`
}

func toPejabatView(p db.Pejabat) pejabatView {
	return pejabatView{
		ID:        p.ID,
		Jabatan:   p.Jabatan,
		Nama:      p.Nama,
		Nip:       p.Nip,
		Aktif:     p.Aktif != 0,
		CreatedAt: p.CreatedAt,
	}
}

type pejabatInput struct {
	Jabatan string `json:"jabatan"`
	Nama    string `json:"nama"`
	Nip     string `json:"nip"`
	Aktif   bool   `json:"aktif"`
}

// validate menormalkan dan memvalidasi input pejabat.
func (in *pejabatInput) validate() error {
	in.Jabatan = strings.ToLower(strings.TrimSpace(in.Jabatan))
	in.Nama = strings.TrimSpace(in.Nama)
	in.Nip = strings.TrimSpace(in.Nip)
	if in.Jabatan != "lurah" && in.Jabatan != "camat" {
		return errors.New("jabatan harus 'lurah' atau 'camat'")
	}
	if in.Nama == "" {
		return errors.New("nama wajib diisi")
	}
	if in.Nip == "" {
		return errors.New("NIP wajib diisi")
	}
	return nil
}

func boolToInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

// ListPejabat: GET /api/pejabat
func (h *Handler) ListPejabat(w http.ResponseWriter, r *http.Request) {
	rows, err := h.q.ListPejabat(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal memuat pejabat")
		return
	}
	out := make([]pejabatView, 0, len(rows))
	for _, p := range rows {
		out = append(out, toPejabatView(p))
	}
	writeJSON(w, http.StatusOK, out)
}

// CreatePejabat: POST /api/pejabat
func (h *Handler) CreatePejabat(w http.ResponseWriter, r *http.Request) {
	var in pejabatInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "body tidak valid")
		return
	}
	if err := in.validate(); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	created, err := h.createPejabatTx(r.Context(), in)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal menyimpan pejabat")
		return
	}
	writeJSON(w, http.StatusCreated, toPejabatView(created))
}

// createPejabatTx menyimpan pejabat; jika aktif, menonaktifkan pejabat lain
// dengan jabatan sama lebih dulu (satu aktif per jabatan) — dalam satu transaksi.
func (h *Handler) createPejabatTx(ctx context.Context, in pejabatInput) (db.Pejabat, error) {
	tx, err := h.sqldb.BeginTx(ctx, nil)
	if err != nil {
		return db.Pejabat{}, err
	}
	defer tx.Rollback()
	qtx := h.q.WithTx(tx)

	if in.Aktif {
		if err := qtx.DeactivatePejabatByJabatan(ctx, in.Jabatan); err != nil {
			return db.Pejabat{}, err
		}
	}
	created, err := qtx.CreatePejabat(ctx, db.CreatePejabatParams{
		Jabatan: in.Jabatan,
		Nama:    in.Nama,
		Nip:     in.Nip,
		Aktif:   boolToInt(in.Aktif),
	})
	if err != nil {
		return db.Pejabat{}, err
	}
	return created, tx.Commit()
}

// UpdatePejabat: PUT /api/pejabat/{id}
func (h *Handler) UpdatePejabat(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "id tidak valid")
		return
	}
	var in pejabatInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeErr(w, http.StatusBadRequest, "body tidak valid")
		return
	}
	if err := in.validate(); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	// Pastikan ada.
	if _, err := h.q.GetPejabat(r.Context(), id); errors.Is(err, sql.ErrNoRows) {
		writeErr(w, http.StatusNotFound, "pejabat tidak ditemukan")
		return
	} else if err != nil {
		writeErr(w, http.StatusInternalServerError, "kesalahan server")
		return
	}

	if err := h.updatePejabatTx(r.Context(), id, in); err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal menyimpan perubahan")
		return
	}
	updated, err := h.q.GetPejabat(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "kesalahan server")
		return
	}
	writeJSON(w, http.StatusOK, toPejabatView(updated))
}

func (h *Handler) updatePejabatTx(ctx context.Context, id int64, in pejabatInput) error {
	tx, err := h.sqldb.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	qtx := h.q.WithTx(tx)

	if in.Aktif {
		if err := qtx.DeactivatePejabatByJabatan(ctx, in.Jabatan); err != nil {
			return err
		}
	}
	if err := qtx.UpdatePejabat(ctx, db.UpdatePejabatParams{
		Jabatan: in.Jabatan,
		Nama:    in.Nama,
		Nip:     in.Nip,
		Aktif:   boolToInt(in.Aktif),
		ID:      id,
	}); err != nil {
		return err
	}
	return tx.Commit()
}

// DeletePejabat: DELETE /api/pejabat/{id}
func (h *Handler) DeletePejabat(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "id tidak valid")
		return
	}
	if _, err := h.q.GetPejabat(r.Context(), id); errors.Is(err, sql.ErrNoRows) {
		writeErr(w, http.StatusNotFound, "pejabat tidak ditemukan")
		return
	} else if err != nil {
		writeErr(w, http.StatusInternalServerError, "kesalahan server")
		return
	}
	if err := h.q.DeletePejabat(r.Context(), id); err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal menghapus pejabat")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
