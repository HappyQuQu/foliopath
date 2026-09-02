package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	modernsqlite "modernc.org/sqlite"
)

func (s *Store) FindAIModelInstall(ctx context.Context, key string) (aimodel.InstallWork, bool, error) {
	work, err := scanAIModelInstall(s.db.QueryRowContext(ctx, aiModelInstallSelect+` WHERE request.idempotency_key = ?`, key))
	if errors.Is(err, sql.ErrNoRows) {
		return aimodel.InstallWork{}, false, nil
	}
	return work, err == nil, err
}

func (s *Store) CreateAIModelInstall(ctx context.Context, work aimodel.InstallWork) (aimodel.InstallWork, bool, error) {
	packageJSON, manifestJSON, err := encodeAIModelInstall(work)
	if err != nil {
		return aimodel.InstallWork{}, false, err
	}
	created := false
	err = s.withWriteTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `
            INSERT INTO ai_model_operations(
                id, kind, state, phase, completed_items, revision, created_at_ms, updated_at_ms
            ) VALUES(?, ?, ?, ?, 0, 1, ?, ?)`,
			work.Operation.ID, work.Operation.Kind, work.Operation.State, work.Operation.Phase,
			work.Operation.CreatedAt.UTC().UnixMilli(), work.Operation.UpdatedAt.UTC().UnixMilli(),
		); err != nil {
			return fmt.Errorf("create AI model install operation: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
            INSERT INTO ai_model_install_requests(
                idempotency_key, request_hash, operation_id, candidate_id, storage_mode,
                package_json, manifest_json, source_identity, created_at_ms
            ) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			work.IdempotencyKey, work.RequestHash, work.Operation.ID, work.CandidateID,
			work.StorageMode, packageJSON, manifestJSON, work.Candidate.SourceIdentity,
			work.Operation.CreatedAt.UTC().UnixMilli(),
		); err != nil {
			if isUniqueConstraint(err) {
				return err
			}
			return fmt.Errorf("create AI model install request: %w", err)
		}
		created = true
		return nil
	})
	if err != nil {
		if !isUniqueConstraint(err) {
			return aimodel.InstallWork{}, false, err
		}
		existing, found, findErr := s.FindAIModelInstall(ctx, work.IdempotencyKey)
		return existing, false, firstErrorIfMissing(found, findErr)
	}
	createdWork, found, err := s.FindAIModelInstall(ctx, work.IdempotencyKey)
	return createdWork, created, firstErrorIfMissing(found, err)
}

func (s *Store) ClaimAIModelInstall(ctx context.Context, now time.Time) (aimodel.InstallWork, bool, error) {
	var claimed aimodel.InstallWork
	found := false
	err := s.withWriteTx(ctx, func(tx *sql.Tx) error {
		work, err := scanAIModelInstall(tx.QueryRowContext(ctx, aiModelInstallSelect+`
            WHERE operation.state = 'queued'
            ORDER BY operation.created_at_ms, operation.id LIMIT 1`))
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		updatedAt := now.UTC()
		if updatedAt.Before(work.Operation.CreatedAt) {
			updatedAt = work.Operation.CreatedAt
		}
		result, err := tx.ExecContext(ctx, `
            UPDATE ai_model_operations
			SET state = 'running', phase = 'verifying', revision = revision + 1, updated_at_ms = MAX(created_at_ms, ?)
            WHERE id = ? AND state = 'queued' AND revision = ?`,
			updatedAt.UnixMilli(), work.Operation.ID, work.Operation.Revision,
		)
		if err != nil {
			return fmt.Errorf("claim AI model install: %w", err)
		}
		rows, err := result.RowsAffected()
		if err != nil || rows != 1 {
			return fmt.Errorf("claim AI model install contention: %w", aimodel.ErrPreconditionFailed)
		}
		work.Operation.State = aimodel.OperationRunning
		work.Operation.Phase = aimodel.PhaseVerifying
		work.Operation.Revision++
		work.Operation.UpdatedAt = updatedAt
		claimed, found = work, true
		return nil
	})
	return claimed, found, err
}

const aiModelInstallSelect = `
    SELECT request.idempotency_key, request.request_hash, request.candidate_id,
           request.storage_mode, request.package_json, request.manifest_json, request.source_identity,
           operation.id, operation.kind, operation.state, operation.phase,
           operation.completed_items, operation.total_items, operation.error_code,
           operation.revision, operation.created_at_ms, operation.updated_at_ms, operation.finished_at_ms
    FROM ai_model_install_requests AS request
    JOIN ai_model_operations AS operation ON operation.id = request.operation_id`

func scanAIModelInstall(row aiModelRowScanner) (aimodel.InstallWork, error) {
	var work aimodel.InstallWork
	var packageJSON, manifestJSON []byte
	var kind, state, phase string
	var totalItems, finishedAt sql.NullInt64
	var errorCode sql.NullString
	var createdAt, updatedAt int64
	if err := row.Scan(
		&work.IdempotencyKey, &work.RequestHash, &work.CandidateID, &work.StorageMode,
		&packageJSON, &manifestJSON, &work.Candidate.SourceIdentity,
		&work.Operation.ID, &kind, &state, &phase, &work.Operation.CompletedItems,
		&totalItems, &errorCode, &work.Operation.Revision, &createdAt, &updatedAt, &finishedAt,
	); err != nil {
		return aimodel.InstallWork{}, err
	}
	if err := json.Unmarshal(packageJSON, &work.Candidate.Package); err != nil {
		return aimodel.InstallWork{}, fmt.Errorf("decode AI model install package: %w", err)
	}
	if err := json.Unmarshal(manifestJSON, &work.Candidate.Manifest); err != nil {
		return aimodel.InstallWork{}, fmt.Errorf("decode AI model install manifest: %w", err)
	}
	work.Candidate.ID, work.Candidate.Compatibility = work.CandidateID, "compatible"
	work.Operation.Kind = aimodel.OperationKind(kind)
	work.Operation.State = aimodel.OperationState(state)
	work.Operation.Phase = aimodel.OperationPhase(phase)
	work.Operation.ErrorCode = errorCode.String
	work.Operation.CreatedAt = time.UnixMilli(createdAt).UTC()
	work.Operation.UpdatedAt = time.UnixMilli(updatedAt).UTC()
	if totalItems.Valid {
		work.Operation.TotalItems = &totalItems.Int64
	}
	if finishedAt.Valid {
		value := time.UnixMilli(finishedAt.Int64).UTC()
		work.Operation.FinishedAt = &value
	}
	return work, nil
}

func encodeAIModelInstall(work aimodel.InstallWork) ([]byte, []byte, error) {
	if work.IdempotencyKey == "" || work.RequestHash == "" || work.CandidateID == "" ||
		work.Candidate.ID != work.CandidateID || work.Candidate.Compatibility != "compatible" ||
		aimodel.ValidatePackage(work.Candidate.Package) != nil || work.Candidate.SourceIdentity == "" ||
		(work.StorageMode != aimodel.StorageManaged && work.StorageMode != aimodel.StorageDirect) {
		return nil, nil, aimodel.ErrInvalidModel
	}
	packageJSON, err := json.Marshal(work.Candidate.Package)
	if err != nil {
		return nil, nil, err
	}
	manifestJSON, err := json.Marshal(work.Candidate.Manifest)
	return packageJSON, manifestJSON, err
}

func firstErrorIfMissing(found bool, err error) error {
	if err != nil {
		return err
	}
	if !found {
		return aimodel.ErrRepositoryState
	}
	return nil
}

func isUniqueConstraint(err error) bool {
	var sqliteError *modernsqlite.Error
	if !errors.As(err, &sqliteError) {
		return false
	}
	// SQLite extended result codes retain SQLITE_CONSTRAINT (19) in the low byte.
	return sqliteError.Code()&0xff == 19
}
