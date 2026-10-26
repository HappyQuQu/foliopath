package face

import (
	"bytes"
	"context"
	"errors"
	"io"
	"math"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/media"
)

type readSeekCloser struct{ *bytes.Reader }

func (readSeekCloser) Close() error { return nil }

type assetSourceStub struct {
	asset Asset
	err   error
}

func (stub assetSourceStub) OpenFaceAsset(context.Context, int64, int64) (Asset, error) {
	return stub.asset, stub.err
}

type runtimeStub struct {
	candidates []Candidate
	err        error
	limit      int64
}

func (stub *runtimeStub) AnalyzeFaces(_ context.Context, _ io.ReadSeeker, _ media.Format, limit int64) ([]Candidate, error) {
	stub.limit = limit
	return stub.candidates, stub.err
}

func validAsset() Asset {
	return Asset{File: readSeekCloser{bytes.NewReader([]byte("fixture"))}, Format: media.FormatJPEG, SourceFingerprint: "source-v1"}
}

func validCandidate() Candidate {
	return Candidate{Box: Box{X: 0.1, Y: 0.2, Width: 0.3, Height: 0.4}, Detection: 0.95, Quality: 0.8, Embedding: []float32{3, 4}}
}

func TestProcessorValidatesAndNormalizesRuntimeOutput(t *testing.T) {
	runtime := &runtimeStub{candidates: []Candidate{validCandidate()}}
	processor, err := NewProcessor(assetSourceStub{asset: validAsset()}, runtime)
	if err != nil {
		t.Fatal(err)
	}
	observations, err := processor.Analyze(context.Background(), 1, 2, "source-v1")
	if err != nil {
		t.Fatal(err)
	}
	if runtime.limit != MaxInputBytes || len(observations) != 1 || len(observations[0].Embedding) != 2 {
		t.Fatalf("unexpected result: limit=%d observations=%+v", runtime.limit, observations)
	}
	if math.Abs(float64(observations[0].Embedding[0]-0.6)) > 1e-6 ||
		math.Abs(float64(observations[0].Embedding[1]-0.8)) > 1e-6 {
		t.Fatalf("embedding was not normalized: %v", observations[0].Embedding)
	}
}

func TestProcessorRejectsChangedSourceBeforeRuntime(t *testing.T) {
	runtime := &runtimeStub{}
	processor, err := NewProcessor(assetSourceStub{asset: validAsset()}, runtime)
	if err != nil {
		t.Fatal(err)
	}
	_, err = processor.Analyze(context.Background(), 1, 2, "source-v2")
	if !errors.Is(err, ErrSourceChanged) || runtime.limit != 0 {
		t.Fatalf("expected source change before runtime, got %v", err)
	}
}

func TestProcessorRejectsUnsupportedFormatBeforeRuntime(t *testing.T) {
	runtime := &runtimeStub{}
	asset := validAsset()
	asset.Format = media.FormatMP4
	processor, err := NewProcessor(assetSourceStub{asset: asset}, runtime)
	if err != nil {
		t.Fatal(err)
	}
	_, err = processor.Analyze(context.Background(), 1, 2, "source-v1")
	if !errors.Is(err, ErrInvalidInput) || runtime.limit != 0 {
		t.Fatalf("expected unsupported format before runtime, got %v", err)
	}
}

func TestProcessorRejectsUntrustedRuntimeOutput(t *testing.T) {
	tests := []struct {
		name      string
		candidate Candidate
	}{
		{name: "box outside image", candidate: Candidate{Box: Box{X: 0.9, Y: 0, Width: 0.2, Height: 1}, Detection: 1, Quality: 1, Embedding: []float32{1}}},
		{name: "zero box", candidate: Candidate{Box: Box{Width: 0, Height: 1}, Detection: 1, Quality: 1, Embedding: []float32{1}}},
		{name: "non finite score", candidate: Candidate{Box: Box{Width: 1, Height: 1}, Detection: float32(math.NaN()), Quality: 1, Embedding: []float32{1}}},
		{name: "zero vector", candidate: Candidate{Box: Box{Width: 1, Height: 1}, Detection: 1, Quality: 1, Embedding: []float32{0, 0}}},
		{name: "non finite vector", candidate: Candidate{Box: Box{Width: 1, Height: 1}, Detection: 1, Quality: 1, Embedding: []float32{float32(math.Inf(1))}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			processor, err := NewProcessor(assetSourceStub{asset: validAsset()}, &runtimeStub{candidates: []Candidate{test.candidate}})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := processor.Analyze(context.Background(), 1, 2, "source-v1"); !errors.Is(err, ErrInvalidOutput) {
				t.Fatalf("expected invalid output, got %v", err)
			}
		})
	}
}

func TestProcessorRejectsCandidateOverflow(t *testing.T) {
	candidates := make([]Candidate, MaxCandidatesPerAsset+1)
	processor, err := NewProcessor(assetSourceStub{asset: validAsset()}, &runtimeStub{candidates: candidates})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := processor.Analyze(context.Background(), 1, 2, "source-v1"); !errors.Is(err, ErrInvalidOutput) {
		t.Fatalf("expected candidate bound failure, got %v", err)
	}
}
