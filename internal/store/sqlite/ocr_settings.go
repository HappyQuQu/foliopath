package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/HappyQuQu/foliopath/internal/ocr"
)

var _ ocr.Repository = (*Store)(nil)

func getOCRSettings(ctx context.Context, q queryRower, id int64) (ocr.Settings, error) {
	v := ocr.Settings{LibraryID: id, Revision: 1, Epoch: 1}
	err := q.QueryRowContext(ctx, `SELECT COALESCE(s.enabled,0),COALESCE(s.clearing,0),COALESCE(s.revision,1),COALESCE(s.epoch,1),COALESCE(s.fingerprint,'') FROM libraries l LEFT JOIN ocr_library_settings s ON s.library_id=l.id WHERE l.id=?`, id).Scan(&v.Enabled, &v.Clearing, &v.Revision, &v.Epoch, &v.Fingerprint)
	if errors.Is(err, sql.ErrNoRows) {
		return v, ocr.ErrNotFound
	}
	return v, err
}
func (s *Store) GetOCRSettings(ctx context.Context, id int64) (ocr.Settings, error) {
	if id < 1 {
		return ocr.Settings{}, ocr.ErrInvalidPersistence
	}
	return getOCRSettings(ctx, s.db, id)
}
func (s *Store) ConfigureOCR(ctx context.Context, id, revision int64, enabled bool, identity ocr.Identity, now time.Time) (ocr.Settings, error) {
	var v ocr.Settings
	if id < 1 || revision < 1 || now.UnixMilli() < 1 {
		return v, ocr.ErrInvalidPersistence
	}
	fingerprint := ""
	var raw []byte
	if enabled {
		var err error
		fingerprint, err = identity.Fingerprint()
		if err != nil || identity.InputContractSHA256 != ocr.InputContractSHA256() || identity.OutputContractSHA256 != ocr.OutputContractSHA256() {
			return v, ocr.ErrInvalidIdentity
		}
		raw, err = json.Marshal(identity)
		if err != nil {
			return v, err
		}
	}
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		current, err := getOCRSettings(ctx, tx, id)
		if err != nil {
			return err
		}
		if current.Revision != revision {
			return ocr.ErrRevision
		}
		if current.Clearing {
			return ocr.ErrConflict
		}
		if !enabled {
			fingerprint = current.Fingerprint
		}
		if enabled {
			if _, err = tx.ExecContext(ctx, `INSERT INTO ocr_generations(fingerprint,identity_json,created_at_ms) VALUES(?,?,?) ON CONFLICT(fingerprint) DO NOTHING`, fingerprint, string(raw), now.UnixMilli()); err != nil {
				return err
			}
			var stored string
			if err = tx.QueryRowContext(ctx, `SELECT identity_json FROM ocr_generations WHERE fingerprint=?`, fingerprint).Scan(&stored); err != nil {
				return err
			}
			if stored != string(raw) {
				return ocr.ErrConflict
			}
		}
		// Even a repeated explicit setting invalidates old claims; valid same-model
		// results remain available after re-enabling without needless reprocessing.
		if err = cancelOCRLibraryTx(ctx, tx, id, now); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO ocr_library_settings(library_id,enabled,revision,epoch,fingerprint) VALUES(?,?,?,?,NULLIF(?,'')) ON CONFLICT(library_id) DO UPDATE SET enabled=excluded.enabled,revision=excluded.revision,epoch=excluded.epoch,fingerprint=excluded.fingerprint`, id, enabled, current.Revision+1, current.Epoch+1, fingerprint)
		if err != nil {
			return err
		}
		v, err = getOCRSettings(ctx, tx, id)
		return err
	})
	return v, err
}
func cancelOCRLibraryTx(ctx context.Context, tx *sql.Tx, id int64, now time.Time) error {
	_, err := tx.ExecContext(ctx, `UPDATE ocr_jobs SET state=CASE state WHEN 'queued' THEN 'cancelled' ELSE 'cancelling' END,updated_at_ms=MAX(created_at_ms,?) WHERE library_id=? AND state IN ('queued','running')`, now.UnixMilli(), id)
	return err
}
func (s *Store) BeginOCRClear(ctx context.Context, id, revision int64, now time.Time) (ocr.Settings, error) {
	var v ocr.Settings
	if id < 1 || revision < 1 || now.UnixMilli() < 1 {
		return v, ocr.ErrInvalidPersistence
	}
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		current, err := getOCRSettings(ctx, tx, id)
		if err != nil {
			return err
		}
		if current.Revision != revision {
			return ocr.ErrRevision
		}
		if current.Clearing {
			return ocr.ErrConflict
		}
		if err = cancelOCRLibraryTx(ctx, tx, id, now); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO ocr_library_settings(library_id,enabled,clearing,revision,epoch,fingerprint) VALUES(?,0,1,?,?,NULLIF(?,'')) ON CONFLICT(library_id) DO UPDATE SET enabled=0,clearing=1,revision=excluded.revision,epoch=excluded.epoch`, id, current.Revision+1, current.Epoch+1, current.Fingerprint)
		if err != nil {
			return err
		}
		v, err = getOCRSettings(ctx, tx, id)
		return err
	})
	return v, err
}
func (s *Store) DrainOCRClear(ctx context.Context, id, epoch int64, limit int, now time.Time) (bool, error) {
	if id < 1 || epoch < 1 || limit < 1 || limit > ocr.MaximumClearBatch || now.UnixMilli() < 1 {
		return false, ocr.ErrInvalidPersistence
	}
	done := false
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		current, err := getOCRSettings(ctx, tx, id)
		if err != nil {
			return err
		}
		if current.Epoch != epoch {
			return ocr.ErrConflict
		}
		if !current.Clearing {
			done = true
			return nil
		}
		if _, err = tx.ExecContext(ctx, `DELETE FROM ocr_asset_results WHERE rowid IN (SELECT rowid FROM ocr_asset_results WHERE library_id=? ORDER BY fingerprint,asset_id LIMIT ?)`, id, limit); err != nil {
			return err
		}
		var remaining bool
		if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM ocr_asset_results WHERE library_id=?)`, id).Scan(&remaining); err != nil {
			return err
		}
		if !remaining {
			_, err = tx.ExecContext(ctx, `UPDATE ocr_library_settings SET clearing=0,revision=revision+1 WHERE library_id=?`, id)
			done = err == nil
		}
		return err
	})
	return done, err
}
