package face

import (
	"context"
	"errors"
	"time"
)

type PersonMutationService struct {
	people   PeopleRepository
	reviews  ReviewRepository
	clusters CoreClusterFaceRepository
	now      func() time.Time
}

func NewPersonMutationService(people PeopleRepository, reviews ReviewRepository, clusters CoreClusterFaceRepository, now func() time.Time) (*PersonMutationService, error) {
	if people == nil || reviews == nil || clusters == nil {
		return nil, errors.New("person mutation dependencies are required")
	}
	if now == nil {
		now = time.Now
	}
	return &PersonMutationService{people: people, reviews: reviews, clusters: clusters, now: now}, nil
}

func (service *PersonMutationService) Create(ctx context.Context, idempotencyKey, name, clusterID string, expectedClusterRevision *int64) (Person, bool, error) {
	normalized, err := NormalizePersonName(name)
	if err != nil || (clusterID == "") != (expectedClusterRevision == nil) {
		return Person{}, false, ErrInvalidPerson
	}
	request := struct {
		Action                  string `json:"action"`
		Name                    string `json:"name"`
		ClusterID               string `json:"clusterId,omitempty"`
		ExpectedClusterRevision *int64 `json:"expectedClusterRevision,omitempty"`
	}{"create_person", normalized, clusterID, expectedClusterRevision}
	eventID, requestHash, err := ReviewIdentity(idempotencyKey, request)
	if err != nil {
		return Person{}, false, err
	}
	personID := derivedReviewResourceID("person", eventID, "person")
	now := service.now().UTC()
	if clusterID == "" {
		existing, getErr := service.people.GetPerson(ctx, personID)
		if getErr == nil {
			if existing.Name != normalized {
				return Person{}, false, ErrReviewConflict
			}
			return existing, true, nil
		}
		if !errors.Is(getErr, ErrPersonNotFound) {
			return Person{}, false, getErr
		}
		created, err := service.people.CreatePerson(ctx, CreatePersonCommand{ID: personID, Name: normalized, CreatedAt: now})
		return created, false, err
	}
	faceIDs, err := service.clusters.ListCoreClusterFaceIDs(ctx, clusterID, *expectedClusterRevision, MaxGroupReviewFaces+1)
	if err != nil {
		return Person{}, false, err
	}
	if len(faceIDs) < 2 || len(faceIDs) > MaxGroupReviewFaces {
		return Person{}, false, ErrReviewConflict
	}
	anchors := make([]string, len(faceIDs))
	for index, faceID := range faceIDs {
		anchors[index] = derivedReviewResourceID("fanchor", eventID, faceID)
	}
	result, err := service.reviews.CreatePersonFromCluster(ctx, CreatePersonFromClusterCommand{
		EventID: eventID, RequestHash: requestHash, PersonID: personID, Name: normalized, ClusterID: clusterID,
		AnchorIDs: anchors, ExpectedClusterRevision: *expectedClusterRevision, CreatedAt: now,
	})
	if err != nil {
		return Person{}, false, err
	}
	person, err := service.people.GetPerson(ctx, personID)
	return person, result.Replayed, err
}

func (service *PersonMutationService) Rename(ctx context.Context, id, name string, expectedRevision int64) (Person, error) {
	return service.people.RenamePerson(ctx, RenamePersonCommand{ID: id, Name: name, ExpectedRevision: expectedRevision, UpdatedAt: service.now().UTC()})
}

func (service *PersonMutationService) Delete(ctx context.Context, id string, expectedRevision int64) error {
	return service.people.DeletePerson(ctx, DeletePersonCommand{ID: id, ExpectedRevision: expectedRevision, DeletedAt: service.now().UTC()})
}
