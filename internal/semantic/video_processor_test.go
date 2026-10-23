package semantic

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/media"
)

type videoReadSeekCloser struct{ *bytes.Reader }

func (videoReadSeekCloser) Close() error { return nil }

type storyboardSourceStub struct{ value CompleteStoryboard }

func (stub storyboardSourceStub) OpenCompleteStoryboard(context.Context, int64, int64) (CompleteStoryboard, error) {
	return stub.value, nil
}

type videoPreprocessorStub struct {
	calls int
	fail  int
}

func (stub *videoPreprocessorStub) PrepareSemanticImage(_ context.Context, input io.ReadSeeker, format media.Format) ([]float32, error) {
	stub.calls++
	if format != media.FormatWebP {
		return nil, ErrInvalidImageInput
	}
	if stub.fail == stub.calls {
		return nil, errors.New("frame failed")
	}
	return []float32{1, 0}, nil
}

type videoEncoderStub struct{}

func (videoEncoderStub) EncodeSemanticImage(context.Context, string, []float32) ([]float32, error) {
	return []float32{1, 0}, nil
}

type videoRepositoryStub struct {
	commits int
	plan    VideoEmbeddingPlan
}

func (stub *videoRepositoryStub) ReplaceVideoEmbeddingPlan(_ context.Context, plan VideoEmbeddingPlan) error {
	stub.commits++
	stub.plan = plan
	return nil
}

func TestVideoProcessorCommitsOnlyCompletePublishedPlan(t *testing.T) {
	storyboard := completeStoryboardFixture(t)
	preprocessor := &videoPreprocessorStub{}
	repository := &videoRepositoryStub{}
	processor, err := NewVideoProcessor(storyboardSourceStub{value: storyboard}, preprocessor, videoEncoderStub{}, repository,
		func() time.Time { return time.Date(2026, 8, 29, 13, 0, 0, 0, time.UTC) })
	if err != nil {
		t.Fatal(err)
	}
	if err := processor.Process(context.Background(), "aig_video_generation", 1, 2); err != nil {
		t.Fatal(err)
	}
	if repository.commits != 1 || len(repository.plan.Frames) != 4 || preprocessor.calls != 4 {
		t.Fatalf("commits=%d frames=%d calls=%d", repository.commits, len(repository.plan.Frames), preprocessor.calls)
	}
}

func TestVideoProcessorDoesNotCommitPartialPlan(t *testing.T) {
	storyboard := completeStoryboardFixture(t)
	preprocessor := &videoPreprocessorStub{fail: 3}
	repository := &videoRepositoryStub{}
	processor, err := NewVideoProcessor(storyboardSourceStub{value: storyboard}, preprocessor, videoEncoderStub{}, repository, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	if err := processor.Process(context.Background(), "aig_video_generation", 1, 2); err == nil {
		t.Fatal("partial plan succeeded")
	}
	if repository.commits != 0 {
		t.Fatalf("partial commits = %d", repository.commits)
	}
}

func completeStoryboardFixture(t *testing.T) CompleteStoryboard {
	t.Helper()
	fingerprint, err := StoryboardFingerprint("v1:42:100", 1, 4)
	if err != nil {
		t.Fatal(err)
	}
	frames := make([]StoryboardFrame, 4)
	for ordinal := range frames {
		frames[ordinal] = StoryboardFrame{
			Ordinal: ordinal, TimestampMS: int64(ordinal+1) * 1000, Format: media.FormatWebP,
			Image: videoReadSeekCloser{bytes.NewReader([]byte{byte(ordinal)})},
		}
	}
	return CompleteStoryboard{LibraryID: 1, AssetID: 2, SourceFingerprint: "v1:42:100",
		StoryboardFingerprint: fingerprint, TransformVersion: 1, PlanSize: 4, Frames: frames}
}
