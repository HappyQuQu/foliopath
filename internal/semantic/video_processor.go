package semantic

import (
	"context"
	"errors"
	"time"
)

type VideoProcessor struct {
	storyboards  CompleteStoryboardSource
	preprocessor ImagePreprocessor
	encoder      ImageEncoder
	repository   VideoEmbeddingRepository
	now          func() time.Time
}

func NewVideoProcessor(
	storyboards CompleteStoryboardSource,
	preprocessor ImagePreprocessor,
	encoder ImageEncoder,
	repository VideoEmbeddingRepository,
	now func() time.Time,
) (*VideoProcessor, error) {
	if storyboards == nil || preprocessor == nil || encoder == nil || repository == nil {
		return nil, errors.New("video semantic processor dependencies are required")
	}
	if now == nil {
		now = time.Now
	}
	return &VideoProcessor{storyboards: storyboards, preprocessor: preprocessor, encoder: encoder, repository: repository, now: now}, nil
}

// Process embeds every frame from one complete published storyboard. Any
// frame failure aborts the plan, so partial plans can be retained as job
// diagnostics but never become searchable repository state.
func (processor *VideoProcessor) Process(ctx context.Context, generationID string, libraryID, assetID int64) error {
	plan, err := processor.BuildPlan(ctx, generationID, libraryID, assetID)
	if err != nil {
		return err
	}
	return processor.repository.ReplaceVideoEmbeddingPlan(ctx, plan)
}

// BuildPlan performs media I/O and inference without holding a database
// transaction. Durable workers hand the returned complete plan to the queue
// owner so frames, progress, and checkpoint can be committed atomically.
func (processor *VideoProcessor) BuildPlan(ctx context.Context, generationID string, libraryID, assetID int64) (VideoEmbeddingPlan, error) {
	if len(generationID) < 8 || len(generationID) > 128 || libraryID < 1 || assetID < 1 {
		return VideoEmbeddingPlan{}, ErrInvalidVideoSemantic
	}
	storyboard, err := processor.storyboards.OpenCompleteStoryboard(ctx, libraryID, assetID)
	if err != nil {
		return VideoEmbeddingPlan{}, err
	}
	if err := ValidateCompleteStoryboard(storyboard); err != nil || storyboard.LibraryID != libraryID || storyboard.AssetID != assetID {
		closeStoryboardFrames(storyboard.Frames)
		return VideoEmbeddingPlan{}, ErrInvalidVideoSemantic
	}
	defer closeStoryboardFrames(storyboard.Frames)
	frames := make([]VideoFrameEmbedding, 0, storyboard.PlanSize)
	for _, frame := range storyboard.Frames {
		if err := ctx.Err(); err != nil {
			return VideoEmbeddingPlan{}, err
		}
		tensor, err := processor.preprocessor.PrepareSemanticImage(ctx, frame.Image, frame.Format)
		if err != nil {
			return VideoEmbeddingPlan{}, err
		}
		vector, err := processor.encoder.EncodeSemanticImage(ctx, generationID, tensor)
		if err != nil {
			return VideoEmbeddingPlan{}, err
		}
		encoded, err := EncodeEmbedding(vector, len(vector))
		if err != nil {
			return VideoEmbeddingPlan{}, err
		}
		frames = append(frames, VideoFrameEmbedding{Ordinal: frame.Ordinal, TimestampMS: frame.TimestampMS, Vector: encoded})
	}
	return VideoEmbeddingPlan{
		GenerationID: generationID, LibraryID: libraryID, AssetID: assetID,
		SourceFingerprint: storyboard.SourceFingerprint, StoryboardFingerprint: storyboard.StoryboardFingerprint,
		TransformVersion: storyboard.TransformVersion, PlanSize: storyboard.PlanSize,
		Frames: frames, CreatedAt: processor.now().UTC(),
	}, nil
}

func closeStoryboardFrames(frames []StoryboardFrame) {
	for _, frame := range frames {
		if frame.Image != nil {
			_ = frame.Image.Close()
		}
	}
}
