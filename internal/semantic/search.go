package semantic

import (
	"container/heap"
	"context"
	"errors"
	"math"
)

const (
	MaxSemanticSearchLimit = 200
	MaxSemanticCursorBytes = 2048
)

var ErrInvalidSemanticSearch = errors.New("invalid semantic search")

type SearchPosition struct {
	Score   float32
	AssetID int64
}

type VectorSearchRequest struct {
	GenerationID string
	LibraryID    int64
	DirectoryID  int64
	Recursive    bool
	Query        []float32
	After        *SearchPosition
	Limit        int
}

type VectorMatch struct {
	LibraryID int64
	AssetID   int64
	Score     float32
}

type VectorSearchRepository interface {
	SearchSemanticVectors(context.Context, VectorSearchRequest) ([]VectorMatch, error)
}

func ValidateVectorSearchRequest(request VectorSearchRequest) error {
	if len(request.GenerationID) < 8 || len(request.GenerationID) > 128 || request.LibraryID < 0 || request.DirectoryID < 0 ||
		request.Limit < 1 || request.Limit > MaxSemanticSearchLimit || len(request.Query) < 1 {
		return ErrInvalidSemanticSearch
	}
	if request.DirectoryID > 0 && request.LibraryID == 0 {
		return ErrInvalidSemanticSearch
	}
	for _, value := range request.Query {
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			return ErrInvalidSemanticSearch
		}
	}
	if request.After != nil && (request.After.AssetID < 1 || math.IsNaN(float64(request.After.Score)) || math.IsInf(float64(request.After.Score), 0)) {
		return ErrInvalidSemanticSearch
	}
	return nil
}

type matchHeap []VectorMatch

func (values matchHeap) Len() int { return len(values) }
func (values matchHeap) Less(i, j int) bool {
	if values[i].Score != values[j].Score {
		return values[i].Score < values[j].Score
	}
	return values[i].AssetID > values[j].AssetID
}
func (values matchHeap) Swap(i, j int)   { values[i], values[j] = values[j], values[i] }
func (values *matchHeap) Push(value any) { *values = append(*values, value.(VectorMatch)) }
func (values *matchHeap) Pop() any {
	old := *values
	value := old[len(old)-1]
	*values = old[:len(old)-1]
	return value
}

func BoundedVectorMatches(limit int, candidates func(func(VectorMatch) error) error) ([]VectorMatch, error) {
	if limit < 1 || limit > MaxSemanticSearchLimit || candidates == nil {
		return nil, ErrInvalidSemanticSearch
	}
	values := make(matchHeap, 0, limit)
	if err := candidates(func(value VectorMatch) error {
		if value.LibraryID < 1 || value.AssetID < 1 || math.IsNaN(float64(value.Score)) || math.IsInf(float64(value.Score), 0) {
			return ErrInvalidSemanticSearch
		}
		if len(values) < limit {
			heap.Push(&values, value)
			return nil
		}
		worst := values[0]
		if value.Score > worst.Score || value.Score == worst.Score && value.AssetID < worst.AssetID {
			heap.Pop(&values)
			heap.Push(&values, value)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	result := make([]VectorMatch, len(values))
	for index := len(result) - 1; index >= 0; index-- {
		result[index] = heap.Pop(&values).(VectorMatch)
	}
	return result, nil
}
