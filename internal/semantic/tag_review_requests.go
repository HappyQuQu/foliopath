package semantic

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var ErrTagReviewRequestConflict = errors.New("tag review idempotency conflict")

type TagReviewRequestItemState struct {
	Item    TagReviewItem
	Outcome *TagReviewOutcome
}

type TagReviewRequestRecord struct {
	IdempotencyKeyHash string
	RequestHash        string
	State              string
	Items              []TagReviewRequestItemState
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type TagReviewRequestRepository interface {
	BeginTagReviewRequest(context.Context, TagReviewRequestRecord) (TagReviewRequestRecord, bool, error)
	CommitTagReviewRequestOutcome(context.Context, string, int, TagReviewOutcome, time.Time) error
	CompleteTagReviewRequest(context.Context, string, time.Time) error
}

type IdempotentTagReviewResult struct {
	Items    []TagReviewOutcome
	Replayed bool
}

type IdempotentTagReviewService struct {
	reviews    *TagReviewService
	repository TagReviewRequestRepository
	now        func() time.Time
}

func NewIdempotentTagReviewService(reviews *TagReviewService, repository TagReviewRequestRepository, now func() time.Time) (*IdempotentTagReviewService, error) {
	if reviews == nil || repository == nil {
		return nil, errors.New("idempotent tag review dependencies are required")
	}
	if now == nil {
		now = time.Now
	}
	return &IdempotentTagReviewService{reviews: reviews, repository: repository, now: now}, nil
}

func (service *IdempotentTagReviewService) Review(ctx context.Context, idempotencyKey string, items []TagReviewItem) (IdempotentTagReviewResult, error) {
	if !semanticKeyPattern.MatchString(idempotencyKey) || validateTagReviewItems(items) != nil {
		return IdempotentTagReviewResult{}, ErrInvalidTagSuggestion
	}
	encoded, err := json.Marshal(items)
	if err != nil {
		return IdempotentTagReviewResult{}, err
	}
	keyHash := digestSemanticValue("foliopath:tag-review-key:v1\x00" + idempotencyKey)
	requestHash := digestSemanticValue("foliopath:tag-review-request:v1\x00" + string(encoded))
	now := service.now().UTC()
	states := make([]TagReviewRequestItemState, len(items))
	for index, item := range items {
		states[index].Item = item
	}
	record, created, err := service.repository.BeginTagReviewRequest(ctx, TagReviewRequestRecord{
		IdempotencyKeyHash: keyHash, RequestHash: requestHash, State: "running", Items: states, CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		return IdempotentTagReviewResult{}, err
	}
	if record.RequestHash != requestHash {
		return IdempotentTagReviewResult{}, ErrTagReviewRequestConflict
	}
	result := IdempotentTagReviewResult{Items: make([]TagReviewOutcome, len(record.Items)), Replayed: !created}
	for index, state := range record.Items {
		if state.Outcome != nil {
			result.Items[index] = *state.Outcome
			continue
		}
		if existing, found, lookupErr := service.reviews.repository.GetTagReviewBySuggestion(ctx, state.Item.SuggestionID); lookupErr != nil {
			return IdempotentTagReviewResult{}, lookupErr
		} else if found {
			result.Items[index] = TagReviewOutcome{SuggestionID: state.Item.SuggestionID, Outcome: existing.Decision, Revision: existing.Revision}
		} else {
			outcome, reviewErr := service.reviews.reviewOne(ctx, state.Item)
			if reviewErr != nil {
				return IdempotentTagReviewResult{}, reviewErr
			}
			result.Items[index] = outcome
		}
		if err := service.repository.CommitTagReviewRequestOutcome(ctx, keyHash, index, result.Items[index], service.now().UTC()); err != nil {
			return IdempotentTagReviewResult{}, err
		}
	}
	if err := service.repository.CompleteTagReviewRequest(ctx, keyHash, service.now().UTC()); err != nil {
		return IdempotentTagReviewResult{}, err
	}
	return result, nil
}
