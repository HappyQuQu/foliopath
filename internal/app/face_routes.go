package app

import (
	"github.com/HappyQuQu/foliopath/internal/api"
	"github.com/HappyQuQu/foliopath/internal/face"
)

// composeFaceResults connects existing contracts only when the face deployment is enabled.
func composeFaceResults(routes *api.RouteDependencies, database *databaseService) error {
	var err error
	routes.FaceClusters, err = face.NewFaceClusterListService(database, nil)
	if err != nil {
		return err
	}
	routes.FaceMergeCheck, err = face.NewPersonMergeCheckService(database)
	if err != nil {
		return err
	}
	routes.FacePeople, err = face.NewPeopleListService(database, nil)
	if err != nil {
		return err
	}
	routes.FaceAssetFaces, err = face.NewAssetFacesService(database)
	if err != nil {
		return err
	}
	routes.FacePersonAssets, err = face.NewPersonAssetService(database, nil)
	if err != nil {
		return err
	}
	routes.FaceClusterDetails, err = face.NewFaceClusterDetailService(database, nil)
	if err != nil {
		return err
	}
	groupGate := face.UncalibratedGroupReview{}
	routes.FaceReviews, err = face.NewReviewService(database, groupGate, nil)
	if err != nil {
		return err
	}
	people, err := face.NewPersonMutationService(database, database, groupGate, nil)
	if err != nil {
		return err
	}
	routes.FacePerson = people
	routes.FacePersonMutations = people
	return nil
}
