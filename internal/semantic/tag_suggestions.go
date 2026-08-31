package semantic

import (
	"container/heap"
	"context"
	"errors"
	"math"
	"slices"
	"time"
)

const MaxTagSuggestionsPerAsset = 5

var ErrInvalidTagSuggestion = errors.New("invalid semantic tag suggestion")

type TagEmbedding struct {
	TagID  int64
	Vector []byte
}

type TagEmbeddingBatch struct {
	GenerationID string
	SnapshotID   string
	CreatedAt    time.Time
	Items        []TagEmbedding
}

type TagSuggestion struct {
	ID                string
	GenerationID      string
	LibraryID         int64
	AssetID           int64
	SnapshotID        string
	TagID             int64
	SourceFingerprint string
	Confidence        float32
	State             string
	Revision          int64
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type TagSuggestionRepository interface {
	PutTagEmbeddingBatch(context.Context, TagEmbeddingBatch) error
	ReplacePendingTagSuggestions(context.Context, int64, int64, string, string, string, []TagSuggestion, time.Time) error
}

type TagSuggestionInputSource interface {
	LoadTagSuggestionInputs(context.Context, string, string, int64, int64) (string, []float32, map[int64][]float32, error)
}

type TagEmbeddingInput struct {
	TagID int64
	Text  string
}
type TagEmbeddingInputRepository interface {
	ListMissingTagEmbeddingInputs(context.Context, string, string, int) ([]TagEmbeddingInput, error)
	PutTagEmbeddingBatch(context.Context, TagEmbeddingBatch) error
}

type ControlledTagEmbeddingBuilder struct {
	repository TagEmbeddingInputRepository
	encoder    TextEncoder
	now        func() time.Time
	batchSize  int
}

func NewControlledTagEmbeddingBuilder(repository TagEmbeddingInputRepository, encoder TextEncoder, now func() time.Time, batchSize int) (*ControlledTagEmbeddingBuilder, error) {
	if repository == nil || encoder == nil {
		return nil, ErrInvalidTagSuggestion
	}
	if now == nil {
		now = time.Now
	}
	if batchSize == 0 {
		batchSize = 100
	}
	if batchSize < 1 || batchSize > 500 {
		return nil, ErrInvalidTagSuggestion
	}
	return &ControlledTagEmbeddingBuilder{repository: repository, encoder: encoder, now: now, batchSize: batchSize}, nil
}

func (builder *ControlledTagEmbeddingBuilder) EnsureTagEmbeddings(ctx context.Context, generationID, snapshotID string) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		inputs, err := builder.repository.ListMissingTagEmbeddingInputs(ctx, generationID, snapshotID, builder.batchSize)
		if err != nil {
			return err
		}
		if len(inputs) == 0 {
			return nil
		}
		items := make([]TagEmbedding, len(inputs))
		for index, input := range inputs {
			vector, err := builder.encoder.EncodeSemanticText(ctx, generationID, input.Text)
			if err != nil {
				return err
			}
			encoded, err := EncodeEmbedding(vector, len(vector))
			if err != nil {
				return err
			}
			items[index] = TagEmbedding{TagID: input.TagID, Vector: encoded}
		}
		if err := builder.repository.PutTagEmbeddingBatch(ctx, TagEmbeddingBatch{GenerationID: generationID, SnapshotID: snapshotID, CreatedAt: builder.now().UTC(), Items: items}); err != nil {
			return err
		}
	}
}

var _ TagEmbeddingBuilder = (*ControlledTagEmbeddingBuilder)(nil)

type ControlledTagPlanBuilder struct {
	source    TagSuggestionInputSource
	now       func() time.Time
	newID     func(string) (string, error)
	threshold float32
}

func NewControlledTagPlanBuilder(source TagSuggestionInputSource, now func() time.Time, newID func(string) (string, error), threshold float32) (*ControlledTagPlanBuilder, error) {
	if source == nil || math.IsNaN(float64(threshold)) || math.IsInf(float64(threshold), 0) || threshold < 0 || threshold > 1 {
		return nil, ErrInvalidTagSuggestion
	}
	if now == nil {
		now = time.Now
	}
	if newID == nil {
		newID = randomSemanticID
	}
	return &ControlledTagPlanBuilder{source: source, now: now, newID: newID, threshold: threshold}, nil
}

func (builder *ControlledTagPlanBuilder) BuildTagPlan(ctx context.Context, generationID, snapshotID string, libraryID, assetID int64) (TagSuggestionPlan, error) {
	fingerprint, asset, tags, err := builder.source.LoadTagSuggestionInputs(ctx, generationID, snapshotID, libraryID, assetID)
	if err != nil {
		return TagSuggestionPlan{}, err
	}
	items, err := RankControlledTagSuggestions(asset, tags, builder.threshold, MaxTagSuggestionsPerAsset)
	if err != nil {
		return TagSuggestionPlan{}, err
	}
	now := builder.now().UTC()
	for index := range items {
		id, err := builder.newID("ais")
		if err != nil {
			return TagSuggestionPlan{}, err
		}
		items[index].ID, items[index].GenerationID, items[index].SnapshotID = id, generationID, snapshotID
		items[index].LibraryID, items[index].AssetID, items[index].SourceFingerprint = libraryID, assetID, fingerprint
		items[index].CreatedAt, items[index].UpdatedAt = now, now
	}
	return TagSuggestionPlan{GenerationID: generationID, VocabularySnapshotID: snapshotID, SourceFingerprint: fingerprint, LibraryID: libraryID, AssetID: assetID, Suggestions: items}, nil
}

var _ TagPlanBuilder = (*ControlledTagPlanBuilder)(nil)

func ValidateTagEmbeddingBatch(batch TagEmbeddingBatch, dimension, maximumItems int) error {
	if len(batch.GenerationID) < 8 || len(batch.GenerationID) > 128 ||
		len(batch.SnapshotID) < 8 || len(batch.SnapshotID) > 128 || batch.CreatedAt.IsZero() ||
		dimension < 1 || dimension > 65536 || maximumItems < 1 || len(batch.Items) < 1 || len(batch.Items) > maximumItems {
		return ErrInvalidTagSuggestion
	}
	seen := make(map[int64]struct{}, len(batch.Items))
	for _, item := range batch.Items {
		if item.TagID < 1 || len(item.Vector) != dimension*2 {
			return ErrInvalidTagSuggestion
		}
		if _, exists := seen[item.TagID]; exists {
			return ErrInvalidTagSuggestion
		}
		seen[item.TagID] = struct{}{}
		if _, err := DecodeEmbedding(item.Vector, dimension); err != nil {
			return ErrInvalidTagSuggestion
		}
	}
	return nil
}

type tagScoreHeap []TagSuggestion

func (values tagScoreHeap) Len() int { return len(values) }
func (values tagScoreHeap) Less(i, j int) bool {
	if values[i].Confidence != values[j].Confidence {
		return values[i].Confidence < values[j].Confidence
	}
	return values[i].TagID > values[j].TagID
}
func (values tagScoreHeap) Swap(i, j int)   { values[i], values[j] = values[j], values[i] }
func (values *tagScoreHeap) Push(value any) { *values = append(*values, value.(TagSuggestion)) }
func (values *tagScoreHeap) Pop() any {
	old := *values
	value := old[len(old)-1]
	*values = old[:len(old)-1]
	return value
}

// RankControlledTagSuggestions scores only caller-supplied controlled tag IDs.
// Cosine is mapped from [-1,1] to the public finite confidence range [0,1].
func RankControlledTagSuggestions(
	assetVector []float32,
	tagVectors map[int64][]float32,
	threshold float32,
	limit int,
) ([]TagSuggestion, error) {
	if len(assetVector) < 1 || len(tagVectors) < 1 || limit < 1 || limit > MaxTagSuggestionsPerAsset ||
		math.IsNaN(float64(threshold)) || math.IsInf(float64(threshold), 0) || threshold < 0 || threshold > 1 {
		return nil, ErrInvalidTagSuggestion
	}
	asset, err := NormalizeEmbedding(assetVector, len(assetVector))
	if err != nil {
		return nil, ErrInvalidTagSuggestion
	}
	values := make(tagScoreHeap, 0, limit)
	for tagID, vector := range tagVectors {
		if tagID < 1 || len(vector) != len(asset) {
			return nil, ErrInvalidTagSuggestion
		}
		normalized, err := NormalizeEmbedding(vector, len(asset))
		if err != nil {
			return nil, ErrInvalidTagSuggestion
		}
		var cosine float64
		for index := range asset {
			cosine += float64(asset[index]) * float64(normalized[index])
		}
		confidence := float32((math.Max(-1, math.Min(1, cosine)) + 1) / 2)
		if confidence < threshold {
			continue
		}
		candidate := TagSuggestion{TagID: tagID, Confidence: confidence, State: "pending", Revision: 1}
		if len(values) < limit {
			heap.Push(&values, candidate)
			continue
		}
		worst := values[0]
		if confidence > worst.Confidence || confidence == worst.Confidence && tagID < worst.TagID {
			heap.Pop(&values)
			heap.Push(&values, candidate)
		}
	}
	result := make([]TagSuggestion, len(values))
	for index := len(result) - 1; index >= 0; index-- {
		result[index] = heap.Pop(&values).(TagSuggestion)
	}
	return result, nil
}

func ValidatePendingTagSuggestions(items []TagSuggestion, libraryID, assetID int64, generationID, snapshotID, sourceFingerprint string) error {
	if libraryID < 1 || assetID < 1 || len(generationID) < 8 || len(generationID) > 128 ||
		len(snapshotID) < 8 || len(snapshotID) > 128 || len(sourceFingerprint) < 1 || len(sourceFingerprint) > 256 ||
		len(items) > MaxTagSuggestionsPerAsset {
		return ErrInvalidTagSuggestion
	}
	seen := make(map[int64]struct{}, len(items))
	for index, item := range items {
		if len(item.ID) < 8 || len(item.ID) > 128 || item.GenerationID != generationID || item.LibraryID != libraryID ||
			item.AssetID != assetID || item.SnapshotID != snapshotID || item.TagID < 1 || item.SourceFingerprint != sourceFingerprint ||
			item.State != "pending" || item.Revision != 1 || item.CreatedAt.IsZero() || item.UpdatedAt.IsZero() ||
			math.IsNaN(float64(item.Confidence)) || math.IsInf(float64(item.Confidence), 0) || item.Confidence < 0 || item.Confidence > 1 {
			return ErrInvalidTagSuggestion
		}
		if index > 0 && (items[index-1].Confidence < item.Confidence ||
			items[index-1].Confidence == item.Confidence && items[index-1].TagID >= item.TagID) {
			return ErrInvalidTagSuggestion
		}
		if _, exists := seen[item.TagID]; exists {
			return ErrInvalidTagSuggestion
		}
		seen[item.TagID] = struct{}{}
	}
	return nil
}

func ControlledVocabularyTagIDs(items []TagEmbedding) ([]int64, error) {
	ids := make([]int64, len(items))
	for index, item := range items {
		if item.TagID < 1 {
			return nil, ErrInvalidTagSuggestion
		}
		ids[index] = item.TagID
	}
	slices.Sort(ids)
	if len(slices.Compact(ids)) != len(ids) {
		return nil, ErrInvalidTagSuggestion
	}
	return ids, nil
}
