package db

import (
	_ "embed"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

// migrations diterapkan berurutan; index+1 = versi. Migrasi hanya dijalankan
// jika PRAGMA user_version < versinya, lalu user_version dinaikkan. Ini membuat
// upgrade exe idempotent dan tidak menyentuh data yang sudah ada.
var migrations = []string{
	schemaSQL, // v1: skema penuh (model v2 spek)
}

// Open membuka/membuat SQLite di path yang diberikan dengan WAL + foreign keys.
func Open(path string) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)",
		path,
	)
	sqldb, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	sqldb.SetMaxOpenConns(1) // SQLite single-writer; sederhanakan konkuransi.
	if err := sqldb.Ping(); err != nil {
		sqldb.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return sqldb, nil
}

// Migrate menjalankan migrasi yang belum diterapkan berdasarkan user_version.
func Migrate(sqldb *sql.DB) error {
	var current int
	if err := sqldb.QueryRow("PRAGMA user_version").Scan(&current); err != nil {
		return fmt.Errorf("baca user_version: %w", err)
	}

	for i := current; i < len(migrations); i++ {
		version := i + 1
		if err := applyMigration(sqldb, migrations[i], version); err != nil {
			return fmt.Errorf("migrasi v%d: %w", version, err)
		}
	}
	return nil
}

func applyMigration(sqldb *sql.DB, script string, version int) error {
	tx, err := sqldb.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(script); err != nil {
		return err
	}
	// PRAGMA user_version tidak menerima parameter bind → format literal (int aman).
	if _, err := tx.Exec(fmt.Sprintf("PRAGMA user_version = %d", version)); err != nil {
		return err
	}
	return tx.Commit()
}
