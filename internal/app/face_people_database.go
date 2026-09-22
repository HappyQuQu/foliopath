package app

import (
	"context"
	"errors"

	"github.com/HappyQuQu/foliopath/internal/face"
)

func (service *databaseService) GetFaceClusterSnapshot(ctx context.Context, libraryID int64) (face.FaceClusterSnapshot, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return face.FaceClusterSnapshot{}, errors.New("face repository is not ready")
	}
	return service.store.GetFaceClusterSnapshot(ctx, libraryID)
}

func (service *databaseService) ListFaceClusterViews(ctx context.Context, query face.FaceClusterListQuery) ([]face.FaceClusterView, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return nil, errors.New("face repository is not ready")
	}
	return service.store.ListFaceClusterViews(ctx, query)
}

func (service *databaseService) GetPeopleSnapshot(ctx context.Context) (face.PeopleSnapshot, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return face.PeopleSnapshot{}, errors.New("face repository is not ready")
	}
	return service.store.GetPeopleSnapshot(ctx)
}

func (service *databaseService) ListPeoplePage(ctx context.Context, query face.PeopleQuery) ([]face.Person, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return nil, errors.New("face repository is not ready")
	}
	return service.store.ListPeoplePage(ctx, query)
}

func (service *databaseService) GetPerson(ctx context.Context, id string) (face.Person, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return face.Person{}, errors.New("face repository is not ready")
	}
	return service.store.GetPerson(ctx, id)
}

func (service *databaseService) ListPeople(ctx context.Context, limit int) ([]face.Person, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return nil, errors.New("face repository is not ready")
	}
	return service.store.ListPeople(ctx, limit)
}

func (service *databaseService) CreatePerson(ctx context.Context, command face.CreatePersonCommand) (face.Person, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return face.Person{}, errors.New("face repository is not ready")
	}
	return service.store.CreatePerson(ctx, command)
}

func (service *databaseService) RenamePerson(ctx context.Context, command face.RenamePersonCommand) (face.Person, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return face.Person{}, errors.New("face repository is not ready")
	}
	return service.store.RenamePerson(ctx, command)
}

func (service *databaseService) DeletePerson(ctx context.Context, command face.DeletePersonCommand) error {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return errors.New("face repository is not ready")
	}
	return service.store.DeletePerson(ctx, command)
}

func (service *databaseService) ListCoreClusterFaceIDs(ctx context.Context, id string, revision int64, limit int) ([]string, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return nil, errors.New("face repository is not ready")
	}
	return service.store.ListCoreClusterFaceIDs(ctx, id, revision, limit)
}

func (service *databaseService) ListAssetFaceViews(ctx context.Context, id int64) ([]face.AssetFaceView, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return nil, errors.New("face repository is not ready")
	}
	return service.store.ListAssetFaceViews(ctx, id)
}

func (service *databaseService) GetPersonAssetSnapshot(ctx context.Context, id string) (face.PersonAssetSnapshot, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return face.PersonAssetSnapshot{}, errors.New("face repository is not ready")
	}
	return service.store.GetPersonAssetSnapshot(ctx, id)
}

func (service *databaseService) ListPersonAssetViews(ctx context.Context, query face.PersonAssetQuery) ([]face.PersonAssetView, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return nil, errors.New("face repository is not ready")
	}
	return service.store.ListPersonAssetViews(ctx, query)
}

func (service *databaseService) GetFaceClusterDetailSnapshot(ctx context.Context, libraryID int64, id string) (face.FaceClusterDetailSnapshot, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return face.FaceClusterDetailSnapshot{}, errors.New("face repository is not ready")
	}
	return service.store.GetFaceClusterDetailSnapshot(ctx, libraryID, id)
}

func (service *databaseService) ListFaceClusterMemberViews(ctx context.Context, query face.FaceClusterMemberQuery) ([]face.FaceClusterMemberView, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return nil, errors.New("face repository is not ready")
	}
	return service.store.ListFaceClusterMemberViews(ctx, query)
}

func (service *databaseService) CreatePersonFromCluster(ctx context.Context, command face.CreatePersonFromClusterCommand) (face.ReviewResult, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return face.ReviewResult{}, errors.New("face repository is not ready")
	}
	return service.store.CreatePersonFromCluster(ctx, command)
}

func (service *databaseService) AssignFace(ctx context.Context, command face.AssignFaceCommand) (face.ReviewResult, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return face.ReviewResult{}, errors.New("face repository is not ready")
	}
	return service.store.AssignFace(ctx, command)
}

func (service *databaseService) AssignCluster(ctx context.Context, command face.AssignClusterCommand) (face.ReviewResult, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return face.ReviewResult{}, errors.New("face repository is not ready")
	}
	return service.store.AssignCluster(ctx, command)
}

func (service *databaseService) ExcludeFace(ctx context.Context, command face.ExcludeFaceCommand) (face.ReviewResult, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return face.ReviewResult{}, errors.New("face repository is not ready")
	}
	return service.store.ExcludeFace(ctx, command)
}

func (service *databaseService) CannotLinkFaces(ctx context.Context, command face.CannotLinkCommand) (face.ReviewResult, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return face.ReviewResult{}, errors.New("face repository is not ready")
	}
	return service.store.CannotLinkFaces(ctx, command)
}

func (service *databaseService) MergePeople(ctx context.Context, command face.MergePeopleCommand) (face.ReviewResult, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return face.ReviewResult{}, errors.New("face repository is not ready")
	}
	return service.store.MergePeople(ctx, command)
}

func (service *databaseService) SplitFace(ctx context.Context, command face.SplitFaceCommand) (face.ReviewResult, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return face.ReviewResult{}, errors.New("face repository is not ready")
	}
	return service.store.SplitFace(ctx, command)
}

func (service *databaseService) UndoFaceReview(ctx context.Context, command face.UndoReviewCommand) (face.ReviewResult, error) {
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	if service.store == nil {
		return face.ReviewResult{}, errors.New("face repository is not ready")
	}
	return service.store.UndoFaceReview(ctx, command)
}
