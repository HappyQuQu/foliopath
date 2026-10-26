package semantic

import (
	"container/heap"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"time"

	"github.com/HappyQuQu/foliopath/internal/media"
)

var (
	ErrInvalidVideoSemantic    = errors.New("invalid video semantic state")
	ErrStoryboardNotReady      = errors.New("complete storyboard not ready")
	ErrStoryboardSourceChanged = errors.New("storyboard source changed")
)

type StoryboardFrame struct {
	Ordinal     int
	TimestampMS int64
	Format      media.Format
	Image       io.ReadSeekCloser
}

type CompleteStoryboard struct {
	LibraryID             int64
	AssetID               int64
	SourceFingerprint     string
	StoryboardFingerprint string
	TransformVersion      int
	PlanSize              int
	Frames                []StoryboardFrame
}

// CompleteStoryboardSource is implemented by the thumbnail/media boundary.
// It returns cells decoded from one already-published storyboard and never
// invokes FFmpeg or accepts a path from semantic code.
type CompleteStoryboardSource interface {
	OpenCompleteStoryboard(context.Context, int64, int64) (CompleteStoryboard, error)
}

type VideoFrameEmbedding struct {
	Ordinal     int
	TimestampMS int64
	Vector      []byte
}

type VideoEmbeddingPlan struct {
	GenerationID          string
	LibraryID             int64
	AssetID               int64
	SourceFingerprint     string
	StoryboardFingerprint string
	TransformVersion      int
	PlanSize              int
	Frames                []VideoFrameEmbedding
	CreatedAt             time.Time
}

type VideoEmbeddingRepository interface {
	ReplaceVideoEmbeddingPlan(context.Context, VideoEmbeddingPlan) error
}

func StoryboardFingerprint(sourceFingerprint string, transformVersion, planSize int) (string, error) {
	if len(sourceFingerprint) < 1 || len(sourceFingerprint) > 256 || transformVersion < 1 ||
		(planSize != 4 && planSize != 10) {
		return "", ErrInvalidVideoSemantic
	}
	digest := sha256.Sum256([]byte(fmt.Sprintf(
		"foliopath:storyboard-semantic:v1\x00%s\x00%d\x00%d",
		sourceFingerprint, transformVersion, planSize,
	)))
	return "sb_" + hex.EncodeToString(digest[:]), nil
}

func ValidateCompleteStoryboard(value CompleteStoryboard) error {
	if value.LibraryID < 1 || value.AssetID < 1 || len(value.SourceFingerprint) < 1 ||
		len(value.SourceFingerprint) > 256 || len(value.StoryboardFingerprint) < 8 ||
		len(value.StoryboardFingerprint) > 256 || value.TransformVersion < 1 ||
		(value.PlanSize != 4 && value.PlanSize != 10) || len(value.Frames) != value.PlanSize {
		return ErrInvalidVideoSemantic
	}
	for index, frame := range value.Frames {
		if frame.Ordinal != index || frame.TimestampMS < 0 || frame.Format != media.FormatWebP || frame.Image == nil ||
			index > 0 && value.Frames[index-1].TimestampMS >= frame.TimestampMS {
			return ErrInvalidVideoSemantic
		}
	}
	return nil
}

func ValidateVideoEmbeddingPlan(value VideoEmbeddingPlan, dimension int) error {
	if len(value.GenerationID) < 8 || len(value.GenerationID) > 128 || value.CreatedAt.IsZero() ||
		dimension < 1 || dimension > 65536 || value.LibraryID < 1 || value.AssetID < 1 ||
		len(value.SourceFingerprint) < 1 || len(value.SourceFingerprint) > 256 ||
		len(value.StoryboardFingerprint) < 8 || len(value.StoryboardFingerprint) > 256 ||
		value.TransformVersion < 1 || (value.PlanSize != 4 && value.PlanSize != 10) || len(value.Frames) != value.PlanSize {
		return ErrInvalidVideoSemantic
	}
	for index, frame := range value.Frames {
		if frame.Ordinal != index || frame.TimestampMS < 0 || len(frame.Vector) != dimension*2 ||
			index > 0 && value.Frames[index-1].TimestampMS >= frame.TimestampMS {
			return ErrInvalidVideoSemantic
		}
		if _, err := DecodeEmbedding(frame.Vector, dimension); err != nil {
			return ErrInvalidVideoSemantic
		}
	}
	return nil
}

type VideoVectorCandidate struct {
	LibraryID   int64
	AssetID     int64
	PlanSize    int
	Ordinal     int
	TimestampMS int64
	Score       float32
}

type VideoVectorMatch struct {
	LibraryID   int64
	AssetID     int64
	PlanSize    int
	Ordinal     int
	TimestampMS int64
	Score       float32
}

type VideoVectorSearchRequest struct {
	GenerationID string
	LibraryID    int64
	DirectoryID  int64
	Recursive    bool
	Query        []float32
	After        *SearchPosition
	Limit        int
}

type VideoVectorSearchRepository interface {
	SearchVideoSemanticVectors(context.Context, VideoVectorSearchRequest) ([]VideoVectorMatch, error)
}

func ValidateVideoVectorSearchRequest(request VideoVectorSearchRequest) error {
	if len(request.GenerationID) < 8 || len(request.GenerationID) > 128 || request.LibraryID < 0 ||
		request.DirectoryID < 0 || request.DirectoryID > 0 && request.LibraryID == 0 ||
		request.Limit < 1 || request.Limit > MaxSemanticSearchLimit || len(request.Query) < 1 {
		return ErrInvalidVideoSemantic
	}
	for _, value := range request.Query {
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			return ErrInvalidVideoSemantic
		}
	}
	if request.After != nil && (request.After.AssetID < 1 || math.IsNaN(float64(request.After.Score)) || math.IsInf(float64(request.After.Score), 0)) {
		return ErrInvalidVideoSemantic
	}
	return nil
}

type videoMatchHeap []VideoVectorMatch

func (values videoMatchHeap) Len() int { return len(values) }
func (values videoMatchHeap) Less(i, j int) bool {
	if values[i].Score != values[j].Score {
		return values[i].Score < values[j].Score
	}
	return values[i].AssetID > values[j].AssetID
}
func (values videoMatchHeap) Swap(i, j int)   { values[i], values[j] = values[j], values[i] }
func (values *videoMatchHeap) Push(value any) { *values = append(*values, value.(VideoVectorMatch)) }
func (values *videoMatchHeap) Pop() any {
	old := *values
	value := old[len(old)-1]
	*values = old[:len(old)-1]
	return value
}

func BoundedVideoMatches(limit int, candidates func(func(VideoVectorMatch) error) error) ([]VideoVectorMatch, error) {
	if limit < 1 || limit > MaxSemanticSearchLimit || candidates == nil {
		return nil, ErrInvalidVideoSemantic
	}
	values := make(videoMatchHeap, 0, limit)
	if err := candidates(func(value VideoVectorMatch) error {
		if value.LibraryID < 1 || value.AssetID < 1 || (value.PlanSize != 4 && value.PlanSize != 10) ||
			value.Ordinal < 0 || value.Ordinal >= value.PlanSize || value.TimestampMS < 0 ||
			math.IsNaN(float64(value.Score)) || math.IsInf(float64(value.Score), 0) {
			return ErrInvalidVideoSemantic
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
	result := make([]VideoVectorMatch, len(values))
	for index := len(result) - 1; index >= 0; index-- {
		result[index] = heap.Pop(&values).(VideoVectorMatch)
	}
	return result, nil
}

// BestVideoMatches applies the frozen max-frame aggregation. The input may be
// in any order, but every asset must contribute exactly one complete 4/10 plan.
func BestVideoMatches(candidates []VideoVectorCandidate, limit int) ([]VideoVectorMatch, error) {
	if limit < 1 || limit > MaxSemanticSearchLimit {
		return nil, ErrInvalidVideoSemantic
	}
	type aggregate struct {
		planSize int
		seen     map[int]struct{}
		best     VideoVectorMatch
	}
	groups := make(map[int64]*aggregate)
	for _, value := range candidates {
		if value.LibraryID < 1 || value.AssetID < 1 || (value.PlanSize != 4 && value.PlanSize != 10) ||
			value.Ordinal < 0 || value.Ordinal >= value.PlanSize || value.TimestampMS < 0 ||
			math.IsNaN(float64(value.Score)) || math.IsInf(float64(value.Score), 0) {
			return nil, ErrInvalidVideoSemantic
		}
		group := groups[value.AssetID]
		if group == nil {
			group = &aggregate{planSize: value.PlanSize, seen: make(map[int]struct{}), best: VideoVectorMatch{
				LibraryID: value.LibraryID, AssetID: value.AssetID, PlanSize: value.PlanSize,
				Ordinal: value.Ordinal, TimestampMS: value.TimestampMS, Score: value.Score,
			}}
			groups[value.AssetID] = group
		}
		if group.planSize != value.PlanSize || group.best.LibraryID != value.LibraryID {
			return nil, ErrInvalidVideoSemantic
		}
		if _, duplicate := group.seen[value.Ordinal]; duplicate {
			return nil, ErrInvalidVideoSemantic
		}
		group.seen[value.Ordinal] = struct{}{}
		if value.Score > group.best.Score || value.Score == group.best.Score && value.Ordinal < group.best.Ordinal {
			group.best.Ordinal, group.best.TimestampMS, group.best.Score = value.Ordinal, value.TimestampMS, value.Score
		}
	}
	result := make([]VideoVectorMatch, 0, len(groups))
	for _, group := range groups {
		if len(group.seen) == group.planSize {
			result = append(result, group.best)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Score != result[j].Score {
			return result[i].Score > result[j].Score
		}
		return result[i].AssetID < result[j].AssetID
	})
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}
