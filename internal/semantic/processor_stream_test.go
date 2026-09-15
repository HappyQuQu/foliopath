package semantic

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/media"
)

type trackedStream struct {
	*bytes.Reader
	closed bool
}

func (s *trackedStream) Close() error { s.closed = true; return nil }

type streamAssetFixture struct {
	file        *trackedStream
	fingerprint string
}

func (f streamAssetFixture) OpenSemanticAsset(context.Context, int64, int64) (SemanticAsset, error) {
	return SemanticAsset{File: f.file, Format: media.FormatPNG, SourceFingerprint: f.fingerprint}, nil
}

func TestBackfillStreamCandidateUsesBoundEncoder(t *testing.T) {
	for _, state := range []string{"active", "retired"} {
		t.Run(state, func(t *testing.T) {
			encoder, runtime := newBoundFixture(t)
			runtime.generation.State = state
			file := &trackedStream{Reader: bytes.NewReader([]byte("synthetic"))}
			processor := &BackfillProcessor{assets: streamAssetFixture{file, "source-v1"}, encoder: encoder, now: time.Now}
			item, outcome, err := processor.processCandidate(context.Background(), BackfillJob{GenerationID: "generation", LibraryID: 1}, BackfillCandidate{AssetID: 1, SourceFingerprint: "source-v1"})
			if !file.closed {
				t.Fatal("source not closed")
			}
			if state == "retired" {
				if err == nil || outcome != candidateFailed || runtime.sent != 0 {
					t.Fatal("retired generation used")
				}
				return
			}
			if err != nil || outcome != candidateCompleted || len(item.Vector) != 1024 || runtime.sent != 1 {
				t.Fatalf("result=%+v outcome=%v err=%v", item, outcome, err)
			}
		})
	}
}
func TestBackfillStreamRejectsChangedSource(t *testing.T) {
	encoder, runtime := newBoundFixture(t)
	file := &trackedStream{Reader: bytes.NewReader(nil)}
	processor := &BackfillProcessor{assets: streamAssetFixture{file, "changed"}, encoder: encoder, now: time.Now}
	_, outcome, err := processor.processCandidate(context.Background(), BackfillJob{GenerationID: "generation", LibraryID: 1}, BackfillCandidate{AssetID: 1, SourceFingerprint: "old"})
	if !errors.Is(err, ErrSemanticSourceChanged) || outcome != candidateStale || !file.closed || runtime.sent != 0 || runtime.prepared != 0 {
		t.Fatal("source guard failed")
	}
}
