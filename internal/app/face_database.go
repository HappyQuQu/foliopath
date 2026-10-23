package app

import (
	"context"
	"errors"
	"github.com/HappyQuQu/foliopath/internal/face"
	"github.com/HappyQuQu/foliopath/internal/jobs"
	"time"
)

func (service *databaseService) GetActiveFaceGeneration(ctx context.Context) (string, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return "", errors.New("face repository is not ready")
	}
	return service.store.GetActiveFaceGeneration(ctx)
}

func (service *databaseService) GetFaceDeploymentGenerationRuntime(ctx context.Context, id string) (face.DeploymentGenerationRuntime, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		err := errors.New("face repository is not ready")
		return face.DeploymentGenerationRuntime{}, err
	}
	return service.store.GetFaceDeploymentGenerationRuntime(ctx, id)
}
func (service *databaseService) FaceJobLibraryState(ctx context.Context, libraryID int64, generationID string) (face.JobLibraryState, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		err := errors.New("face repository is not ready")
		return "", err
	}
	return service.store.FaceJobLibraryState(ctx, libraryID, generationID)
}
func (service *databaseService) CountFaceJobCandidates(ctx context.Context, libraryID int64, generationID string, mode face.JobMode) (face.JobCandidateCounts, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		err := errors.New("face repository is not ready")
		return face.JobCandidateCounts{}, err
	}
	return service.store.CountFaceJobCandidates(ctx, libraryID, generationID, mode)
}
func (service *databaseService) ListFaceJobCandidates(ctx context.Context, libraryID int64, generationID string, mode face.JobMode, checkpoint int64, limit int) (face.JobCandidatePage, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		err := errors.New("face repository is not ready")
		return face.JobCandidatePage{}, err
	}
	return service.store.ListFaceJobCandidates(ctx, libraryID, generationID, mode, checkpoint, limit)
}
func (service *databaseService) FindFaceJob(ctx context.Context, key string) (face.JobAdmission, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		err := errors.New("face repository is not ready")
		return face.JobAdmission{}, false, err
	}
	return service.store.FindFaceJob(ctx, key)
}
func (service *databaseService) CreateFaceJob(ctx context.Context, value face.JobAdmission) (face.JobAdmission, bool, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		err := errors.New("face repository is not ready")
		return face.JobAdmission{}, false, false, err
	}
	return service.store.CreateFaceJob(ctx, value)
}
func (service *databaseService) ClaimFaceJob(ctx context.Context, now time.Time, lease time.Duration) (face.AnalysisJob, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		err := errors.New("face repository is not ready")
		return face.AnalysisJob{}, false, err
	}
	return service.store.ClaimFaceJob(ctx, now, lease)
}
func (service *databaseService) RefreshFaceJobLease(ctx context.Context, job face.AnalysisJob, now time.Time, lease time.Duration) (bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		err := errors.New("face repository is not ready")
		return false, err
	}
	return service.store.RefreshFaceJobLease(ctx, job, now, lease)
}
func (service *databaseService) GetFaceJobProgress(ctx context.Context, generationID string, libraryID int64) (face.JobProgress, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		err := errors.New("face repository is not ready")
		return face.JobProgress{}, false, err
	}
	return service.store.GetFaceJobProgress(ctx, generationID, libraryID)
}
func (service *databaseService) CommitFaceJobProgress(ctx context.Context, commit face.JobProgressCommit) (face.JobProgress, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		err := errors.New("face repository is not ready")
		return face.JobProgress{}, err
	}
	return service.store.CommitFaceJobProgress(ctx, commit)
}
func (service *databaseService) FinishFaceJob(ctx context.Context, job face.AnalysisJob, succeeded bool, code string, now time.Time) (face.AnalysisJob, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		err := errors.New("face repository is not ready")
		return face.AnalysisJob{}, err
	}
	return service.store.FinishFaceJob(ctx, job, succeeded, code, now)
}
func (service *databaseService) CancelFaceJobOperation(ctx context.Context, id string, revision int64, now time.Time) (face.AnalysisJob, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		err := errors.New("face repository is not ready")
		return face.AnalysisJob{}, err
	}
	return service.store.CancelFaceJobOperation(ctx, id, revision, now)
}
func (service *databaseService) RecoverExpiredFaceJobs(ctx context.Context, now time.Time) (jobs.RecoverySummary, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		err := errors.New("face repository is not ready")
		return jobs.RecoverySummary{}, err
	}
	return service.store.RecoverExpiredFaceJobs(ctx, now)
}
func (service *databaseService) RebuildFaceClusters(ctx context.Context, generationID string, libraryID int64, jobID string, revision int64, profile face.ClusterProfile, now time.Time) error {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		err := errors.New("face repository is not ready")
		return err
	}
	return service.store.RebuildFaceClusters(ctx, generationID, libraryID, jobID, revision, profile, now)
}
func (service *databaseService) GetFaceLibrarySettings(ctx context.Context, libraryID int64) (face.LibrarySettings, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		err := errors.New("face repository is not ready")
		return face.LibrarySettings{}, err
	}
	return service.store.GetFaceLibrarySettings(ctx, libraryID)
}
func (service *databaseService) UpdateFaceLibrarySettings(ctx context.Context, libraryID int64, enabled bool, revision int64, now time.Time) (face.LibrarySettings, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		err := errors.New("face repository is not ready")
		return face.LibrarySettings{}, err
	}
	return service.store.UpdateFaceLibrarySettings(ctx, libraryID, enabled, revision, now)
}
func (service *databaseService) FindFaceClear(ctx context.Context, key string) (face.ClearAdmission, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		err := errors.New("face repository is not ready")
		return face.ClearAdmission{}, false, err
	}
	return service.store.FindFaceClear(ctx, key)
}
func (service *databaseService) CreateFaceClear(ctx context.Context, value face.ClearAdmission) (face.ClearAdmission, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		err := errors.New("face repository is not ready")
		return face.ClearAdmission{}, false, err
	}
	return service.store.CreateFaceClear(ctx, value)
}
func (service *databaseService) ClaimFaceClear(ctx context.Context, now time.Time, lease time.Duration) (face.ClearJob, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		err := errors.New("face repository is not ready")
		return face.ClearJob{}, false, err
	}
	return service.store.ClaimFaceClear(ctx, now, lease)
}
func (service *databaseService) RefreshFaceClearLease(ctx context.Context, job face.ClearJob, now time.Time, lease time.Duration) (bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		err := errors.New("face repository is not ready")
		return false, err
	}
	return service.store.RefreshFaceClearLease(ctx, job, now, lease)
}
func (service *databaseService) DeleteFaceClearBatch(ctx context.Context, job face.ClearJob, limit int, now time.Time) (int64, bool, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		err := errors.New("face repository is not ready")
		return 0, false, err
	}
	return service.store.DeleteFaceClearBatch(ctx, job, limit, now)
}
func (service *databaseService) FinishFaceClear(ctx context.Context, job face.ClearJob, succeeded bool, code string, now time.Time) (face.ClearJob, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		err := errors.New("face repository is not ready")
		return face.ClearJob{}, err
	}
	return service.store.FinishFaceClear(ctx, job, succeeded, code, now)
}
func (service *databaseService) CancelFaceClearOperation(ctx context.Context, id string, revision int64, now time.Time) (face.ClearJob, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		err := errors.New("face repository is not ready")
		return face.ClearJob{}, err
	}
	return service.store.CancelFaceClearOperation(ctx, id, revision, now)
}
func (service *databaseService) RecoverExpiredFaceClears(ctx context.Context, now time.Time) (jobs.RecoverySummary, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		err := errors.New("face repository is not ready")
		return jobs.RecoverySummary{}, err
	}
	return service.store.RecoverExpiredFaceClears(ctx, now)
}
