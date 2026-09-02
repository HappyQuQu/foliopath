package app

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
)

type semanticTextRuntimeSessionStub struct {
	output []float32
	err    error
	closed int
	runs   int
}

func (session *semanticTextRuntimeSessionStub) Encode(context.Context, string) ([]float32, error) {
	session.runs++
	if session.err != nil {
		return nil, session.err
	}
	return append([]float32(nil), session.output...), nil
}

func (session *semanticTextRuntimeSessionStub) Close() error { session.closed++; return nil }

type semanticTextSessionFactoryStub struct {
	sessions []semanticTextRuntimeSession
	opened   []string
	err      error
}

func (factory *semanticTextSessionFactoryStub) ValidateSemanticTextSession(context.Context, string) error {
	return factory.err
}

func (factory *semanticTextSessionFactoryStub) OpenSemanticTextSession(_ context.Context, generationID string) (semanticTextRuntimeSession, error) {
	factory.opened = append(factory.opened, generationID)
	if len(factory.sessions) == 0 {
		return nil, errors.New("unexpected session open")
	}
	session := factory.sessions[0]
	factory.sessions = factory.sessions[1:]
	return session, nil
}

func semanticTextVector(first, second float32) []float32 {
	result := make([]float32, 768)
	result[0], result[1] = first, second
	return result
}

func TestSemanticTextSessionOwnerReusesNormalizesAndSwitches(t *testing.T) {
	first := &semanticTextRuntimeSessionStub{output: semanticTextVector(3, 4)}
	second := &semanticTextRuntimeSessionStub{output: semanticTextVector(0, 2)}
	factory := &semanticTextSessionFactoryStub{sessions: []semanticTextRuntimeSession{first, second}}
	owner, err := newSemanticTextSessionOwner(factory, time.Second, time.Minute, nil)
	if err != nil {
		t.Fatal(err)
	}
	vector, err := owner.EncodeSemanticText(context.Background(), "generation-a", "red armor")
	if err != nil || math.Abs(float64(vector[0]-.6)) > 1e-6 || math.Abs(float64(vector[1]-.8)) > 1e-6 {
		t.Fatalf("vector=%v error=%v", vector[:2], err)
	}
	if _, err := owner.EncodeSemanticText(context.Background(), "generation-a", "blue hair"); err != nil {
		t.Fatal(err)
	}
	if len(factory.opened) != 1 || first.runs != 2 || first.closed != 0 {
		t.Fatalf("reuse opened=%v runs=%d closed=%d", factory.opened, first.runs, first.closed)
	}
	if _, err := owner.EncodeSemanticText(context.Background(), "generation-b", "portrait"); err != nil {
		t.Fatal(err)
	}
	if first.closed != 1 || second.runs != 1 {
		t.Fatalf("switch firstClosed=%d secondRuns=%d", first.closed, second.runs)
	}
	if err := owner.Close(); err != nil || second.closed != 1 {
		t.Fatalf("close=%v count=%d", err, second.closed)
	}
}

func TestSemanticTextSessionOwnerDropsInvalidOrFaultedSession(t *testing.T) {
	for name, session := range map[string]*semanticTextRuntimeSessionStub{
		"runtime": {err: errors.New("runtime failed")},
		"zero":    {output: make([]float32, 768)},
		"short":   {output: []float32{1}},
	} {
		t.Run(name, func(t *testing.T) {
			factory := &semanticTextSessionFactoryStub{sessions: []semanticTextRuntimeSession{session}}
			owner, err := newSemanticTextSessionOwner(factory, time.Second, time.Minute, nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := owner.EncodeSemanticText(context.Background(), "generation-a", "query"); err == nil || session.closed != 1 {
				t.Fatalf("error=%v closed=%d", err, session.closed)
			}
		})
	}
}

func TestSemanticTokenizerManifestFileRequiresExactlyOneRoleBeforeOpen(t *testing.T) {
	valid := aimodel.Manifest{Files: []aimodel.ManifestFile{
		{Name: "image.onnx", Role: "image_encoder"},
		{Name: "spiece.model", Role: "sentencepiece_model"},
		{Name: "text.onnx", Role: "text_encoder"},
	}}
	file, err := semanticTokenizerManifestFile(valid)
	if err != nil || file.Name != "spiece.model" {
		t.Fatalf("file=%+v error=%v", file, err)
	}

	for name, files := range map[string][]aimodel.ManifestFile{
		"missing": {{Name: "text.onnx", Role: "text_encoder"}},
		"duplicate": {
			{Name: "first.model", Role: "sentencepiece_model"},
			{Name: "second.model", Role: "sentencepiece_model"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := semanticTokenizerManifestFile(aimodel.Manifest{Files: files}); !errors.Is(err, aimodel.ErrModelIncompatible) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}
