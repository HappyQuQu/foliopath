package face

import (
	"context"
	"errors"
)

const MaxMergeConflictExamples = 3

type MergeConflictExample struct{ SourceLibraryID, SourceAssetID, TargetLibraryID, TargetAssetID int64 }
type PersonMergeCheck struct {
	SourcePersonID, TargetPersonID            string
	SourceRevision, TargetRevision            int64
	CannotLinkCount, DuplicateAssignmentCount int64
	SourceLimitExceeded                       bool
	Examples                                  []MergeConflictExample
}

func (c PersonMergeCheck) CanMerge() bool {
	return c.CannotLinkCount == 0 && c.DuplicateAssignmentCount == 0 && !c.SourceLimitExceeded
}

type PersonMergeCheckRepository interface {
	GetPersonMergeCheck(context.Context, string, string) (PersonMergeCheck, error)
}
type PersonMergeCheckService struct{ repository PersonMergeCheckRepository }

func NewPersonMergeCheckService(repository PersonMergeCheckRepository) (*PersonMergeCheckService, error) {
	if repository == nil {
		return nil, errors.New("person merge check repository is required")
	}
	return &PersonMergeCheckService{repository: repository}, nil
}
func (s *PersonMergeCheckService) Check(ctx context.Context, source, target string) (PersonMergeCheck, error) {
	if !ValidReviewPairGuard(source, 1, target, 1) {
		return PersonMergeCheck{}, ErrInvalidPerson
	}
	return s.repository.GetPersonMergeCheck(ctx, source, target)
}
