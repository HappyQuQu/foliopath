package semantic

import (
	"context"
	"errors"
	"time"

	"github.com/HappyQuQu/foliopath/internal/curation"
)

const MaxTagSuggestionReviewBatch = 100

type TagReviewDecision string

const (
	TagReviewAccept  TagReviewDecision = "accepted"
	TagReviewDismiss TagReviewDecision = "dismissed"
)

type TagReview struct {
	LibraryID                int64
	AssetID                  int64
	TagID                    int64
	Decision                 TagReviewDecision
	SourceSuggestionID       string
	AcceptedCurationRevision int64
	Revision                 int64
	ReviewedAt               time.Time
}

type TagReviewItem struct {
	SuggestionID               string
	Action                     TagReviewDecision
	ExpectedSuggestionRevision int64
	ExpectedCurationRevision   int64
}

type TagReviewOutcome struct {
	SuggestionID string
	Outcome      TagReviewDecision
	Revision     int64
	Conflict     bool
}

type TagReviewRepository interface {
	GetTagSuggestion(context.Context, string) (TagSuggestion, bool, error)
	GetTagReviewBySuggestion(context.Context, string) (TagReview, bool, error)
	CommitTagReview(context.Context, string, int64, TagReviewDecision, int64, time.Time) (TagReview, error)
}

type SuggestedTagCuration interface {
	AddAssetTag(context.Context, int64, int64, int64) (curation.AssetState, error)
}

type TagReviewService struct {
	repository TagReviewRepository
	curation   SuggestedTagCuration
	now        func() time.Time
}

func NewTagReviewService(repository TagReviewRepository, curationService SuggestedTagCuration, now func() time.Time) (*TagReviewService, error) {
	if repository == nil || curationService == nil {
		return nil, errors.New("tag review dependencies are required")
	}
	if now == nil {
		now = time.Now
	}
	return &TagReviewService{repository: repository, curation: curationService, now: now}, nil
}

func (service *TagReviewService) Review(ctx context.Context, items []TagReviewItem) ([]TagReviewOutcome, error) {
	if err := validateTagReviewItems(items); err != nil {
		return nil, err
	}
	result := make([]TagReviewOutcome, 0, len(items))
	for _, item := range items {
		outcome, err := service.reviewOne(ctx, item)
		if err != nil {
			return nil, err
		}
		result = append(result, outcome)
	}
	return result, nil
}

func (service *TagReviewService) reviewOne(ctx context.Context, item TagReviewItem) (TagReviewOutcome, error) {
	suggestion, found, err := service.repository.GetTagSuggestion(ctx, item.SuggestionID)
	if err != nil {
		return TagReviewOutcome{}, err
	}
	if !found || suggestion.State != "pending" || suggestion.Revision != item.ExpectedSuggestionRevision {
		return TagReviewOutcome{SuggestionID: item.SuggestionID, Revision: item.ExpectedSuggestionRevision, Conflict: true}, nil
	}
	curationRevision := int64(0)
	if item.Action == TagReviewAccept {
		state, addErr := service.curation.AddAssetTag(ctx, suggestion.AssetID, item.ExpectedCurationRevision, suggestion.TagID)
		if errors.Is(addErr, curation.ErrPreconditionFailed) || errors.Is(addErr, curation.ErrAssetNotFound) ||
			errors.Is(addErr, curation.ErrTagNotFound) || errors.Is(addErr, curation.ErrInvalidRequest) {
			return TagReviewOutcome{SuggestionID: item.SuggestionID, Revision: item.ExpectedSuggestionRevision, Conflict: true}, nil
		}
		if addErr != nil {
			return TagReviewOutcome{}, addErr
		}
		curationRevision = state.Revision
	}
	review, err := service.repository.CommitTagReview(
		ctx, item.SuggestionID, item.ExpectedSuggestionRevision, item.Action, curationRevision, service.now().UTC(),
	)
	if errors.Is(err, ErrInvalidTagSuggestion) {
		return TagReviewOutcome{SuggestionID: item.SuggestionID, Revision: item.ExpectedSuggestionRevision, Conflict: true}, nil
	}
	if err != nil {
		return TagReviewOutcome{}, err
	}
	return TagReviewOutcome{SuggestionID: item.SuggestionID, Outcome: review.Decision, Revision: review.Revision}, nil
}

func validateTagReviewItems(items []TagReviewItem) error {
	if len(items) < 1 || len(items) > MaxTagSuggestionReviewBatch {
		return ErrInvalidTagSuggestion
	}
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		if len(item.SuggestionID) < 8 || len(item.SuggestionID) > 128 || item.ExpectedSuggestionRevision < 1 ||
			(item.Action != TagReviewAccept && item.Action != TagReviewDismiss) ||
			(item.Action == TagReviewAccept && item.ExpectedCurationRevision < 1) || item.ExpectedCurationRevision < 0 {
			return ErrInvalidTagSuggestion
		}
		if _, exists := seen[item.SuggestionID]; exists {
			return ErrInvalidTagSuggestion
		}
		seen[item.SuggestionID] = struct{}{}
	}
	return nil
}
