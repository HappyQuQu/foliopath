package semantic

import (
	"context"
	"errors"
	"io"
	"testing"
)

type trackedVideoStream struct {
	io.ReadSeekCloser
	closed int
}

func (s *trackedVideoStream) Close() error { s.closed++; return s.ReadSeekCloser.Close() }

func TestStreamVideoUsesBoundEncoderAndClosesAllFrames(t *testing.T) {
	for _, scenario := range []string{"success", "retired-before", "retired-during", "invalid-vector", "cancelled"} {
		t.Run(scenario, func(t *testing.T) {
			encoder, fixture := newBoundFixture(t)
			storyboard := completeStoryboardFixture(t)
			streams := make([]*trackedVideoStream, len(storyboard.Frames))
			for i := range storyboard.Frames {
				streams[i] = &trackedVideoStream{ReadSeekCloser: storyboard.Frames[i].Image}
				storyboard.Frames[i].Image = streams[i]
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch scenario {
			case "retired-before":
				fixture.generation.State = "retired"
			case "retired-during":
				fixture.onSend = func() {
					if fixture.sent == 2 {
						fixture.generation.State = "retired"
					}
				}
			case "invalid-vector":
				fixture.onSend = func() {
					if fixture.sent == 2 {
						fixture.vector = []float32{0}
					}
				}
			case "cancelled":
				fixture.onSend = func() {
					if fixture.sent == 2 {
						cancel()
					}
				}
			}
			repository := &videoRepositoryStub{}
			processor, err := NewStreamVideoProcessor(storyboardSourceStub{storyboard}, encoder, repository, nil)
			if err != nil {
				t.Fatal(err)
			}
			err = processor.Process(ctx, "generation", 1, 2)
			if scenario == "success" {
				if err != nil || repository.commits != 1 || len(repository.plan.Frames) != 4 || fixture.sent != 4 {
					t.Fatalf("complete plan failed: %v %+v", err, repository)
				}
				for _, frame := range repository.plan.Frames {
					vector, err := DecodeEmbedding(frame.Vector, 512)
					if err != nil || len(vector) != 512 || vector[0] != 1 {
						t.Fatal("invalid persisted frame vector")
					}
				}
			} else if err == nil || repository.commits != 0 {
				t.Fatalf("partial plan committed: %v commits=%d", err, repository.commits)
			}
			if scenario == "retired-before" && fixture.sent != 0 {
				t.Fatal("retired generation inferred frames")
			}
			if scenario == "cancelled" && !errors.Is(err, context.Canceled) {
				t.Fatalf("cancellation lost: %v", err)
			}
			for _, stream := range streams {
				if stream.closed != 1 {
					t.Fatalf("closed %d times", stream.closed)
				}
			}
		})
	}
}

func TestStreamVideoRejectsMissingDependencies(t *testing.T) {
	encoder, _ := newBoundFixture(t)
	source := storyboardSourceStub{completeStoryboardFixture(t)}
	repository := &videoRepositoryStub{}
	for _, tc := range []struct {
		source     CompleteStoryboardSource
		encoder    StreamImageEncoder
		repository VideoEmbeddingRepository
	}{
		{nil, encoder, repository}, {source, nil, repository}, {source, encoder, nil},
	} {
		if _, err := NewStreamVideoProcessor(tc.source, tc.encoder, tc.repository, nil); err == nil {
			t.Fatal("accepted missing dependency")
		}
	}
}
