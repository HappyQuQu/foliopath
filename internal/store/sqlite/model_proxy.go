package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"github.com/HappyQuQu/foliopath/internal/aimodel"
)

func (s *Store) GetModelProxy(ctx context.Context) (aimodel.ProxySettings, error) {
	var v aimodel.ProxySettings
	err := s.db.QueryRowContext(ctx, `SELECT mode,proxy_url,revision FROM model_proxy_settings WHERE id=1`).Scan(&v.Mode, &v.URL, &v.Revision)
	return v, err
}
func (s *Store) SaveModelProxy(ctx context.Context, v aimodel.ProxySettings) (aimodel.ProxySettings, error) {
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		err := tx.QueryRowContext(ctx, `UPDATE model_proxy_settings SET mode=?,proxy_url=?,revision=revision+1 WHERE id=1 AND revision=? RETURNING revision`, v.Mode, v.URL, v.Revision).Scan(&v.Revision)
		if errors.Is(err, sql.ErrNoRows) {
			return aimodel.ErrPreconditionFailed
		}
		return err
	})
	return v, err
}
