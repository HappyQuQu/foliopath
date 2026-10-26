package semantic

import (
	"context"
	"errors"
	"slices"
	"time"
)

const MaxControlledVocabularyEntries = 1000

var (
	ErrTagVocabularyUnavailable = errors.New("controlled tag vocabulary unavailable")
	ErrTagVocabularyConflict    = errors.New("controlled tag vocabulary revision conflict")
)

type TagVocabularyEntry struct {
	TagID int64
	Name  string
}

type TagVocabulary struct {
	ID          string
	Revision    int64
	Entries     []TagVocabularyEntry
	PublishedAt time.Time
}

type TagVocabularyRepository interface {
	GetActiveTagVocabulary(context.Context) (TagVocabulary, error)
	PublishTagVocabulary(context.Context, string, int64, []int64, time.Time) (TagVocabulary, error)
}

type TagVocabularyService struct {
	repository TagVocabularyRepository
	now        func() time.Time
	newID      func(string) (string, error)
}

func NewTagVocabularyService(repository TagVocabularyRepository, now func() time.Time, newID func(string) (string, error)) (*TagVocabularyService, error) {
	if repository == nil {
		return nil, errors.New("controlled tag vocabulary repository is required")
	}
	if now == nil {
		now = time.Now
	}
	if newID == nil {
		newID = randomSemanticID
	}
	return &TagVocabularyService{repository: repository, now: now, newID: newID}, nil
}

func (service *TagVocabularyService) Get(ctx context.Context) (TagVocabulary, error) {
	value, err := service.repository.GetActiveTagVocabulary(ctx)
	if err != nil {
		return TagVocabulary{}, err
	}
	if err := ValidateTagVocabulary(value); err != nil {
		return TagVocabulary{}, err
	}
	return value, nil
}

func (service *TagVocabularyService) Publish(ctx context.Context, expectedRevision int64, tagIDs []int64) (TagVocabulary, error) {
	if expectedRevision < 1 || len(tagIDs) > MaxControlledVocabularyEntries {
		return TagVocabulary{}, ErrInvalidTagSuggestion
	}
	ids := slices.Clone(tagIDs)
	slices.Sort(ids)
	for index, id := range ids {
		if id < 1 || index > 0 && ids[index-1] == id {
			return TagVocabulary{}, ErrInvalidTagSuggestion
		}
	}
	id, err := service.newID("aivocab")
	if err != nil {
		return TagVocabulary{}, err
	}
	value, err := service.repository.PublishTagVocabulary(ctx, id, expectedRevision, ids, service.now().UTC())
	if err != nil {
		return TagVocabulary{}, err
	}
	if err := ValidateTagVocabulary(value); err != nil || value.ID != id || value.Revision != expectedRevision+1 {
		if err != nil {
			return TagVocabulary{}, err
		}
		return TagVocabulary{}, ErrInvalidTagSuggestion
	}
	return value, nil
}

func ValidateTagVocabulary(value TagVocabulary) error {
	if len(value.ID) < 8 || len(value.ID) > 128 || value.Revision < 1 || value.PublishedAt.IsZero() || len(value.Entries) > MaxControlledVocabularyEntries {
		return ErrInvalidTagSuggestion
	}
	for index, entry := range value.Entries {
		if entry.TagID < 1 || entry.Name == "" || len(entry.Name) > 400 || index > 0 && value.Entries[index-1].TagID >= entry.TagID {
			return ErrInvalidTagSuggestion
		}
	}
	return nil
}
