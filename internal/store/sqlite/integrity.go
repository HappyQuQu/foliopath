package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
)

var ErrIntegrityCheck = errors.New("database integrity check failed")

func integrityDSN(filename string) string {
	dsn := &url.URL{Scheme: "file", Path: filename}
	query := dsn.Query()
	query.Set("mode", "ro")
	query.Add("_pragma", "busy_timeout(5000)")
	dsn.RawQuery = query.Encode()
	return dsn.String()
}

// CheckIntegrity does not initialize, migrate, checkpoint, or write the database.
// Call it inside the same kernel/lock domain as the owning application, not on
// a host opening a guest's live bind-mounted WAL database.
func CheckIntegrity(ctx context.Context, filename string) error {
	if filename == "" || filename == ":memory:" {
		return ErrIntegrityCheck
	}
	db, err := sql.Open("sqlite", integrityDSN(filename))
	if err != nil {
		return ErrIntegrityCheck
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return ErrIntegrityCheck
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, "PRAGMA quick_check")
	if err != nil {
		return ErrIntegrityCheck
	}
	count := 0
	for rows.Next() {
		var result string
		if err := rows.Scan(&result); err != nil || result != "ok" {
			rows.Close()
			return ErrIntegrityCheck
		}
		count++
	}
	rowErr := rows.Err()
	rows.Close()
	if rowErr != nil || count != 1 {
		return ErrIntegrityCheck
	}
	rows, err = tx.QueryContext(ctx, "PRAGMA foreign_key_check")
	if err != nil {
		return ErrIntegrityCheck
	}
	failure := rows.Next()
	rowErr = rows.Err()
	rows.Close()
	if failure || rowErr != nil {
		return ErrIntegrityCheck
	}
	if err := tx.Commit(); err != nil {
		return ErrIntegrityCheck
	}
	return nil
}
