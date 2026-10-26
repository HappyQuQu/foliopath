package semantic

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/media"
)

type boundFixture struct {
	generation        GenerationRuntime
	prepared, sent    int
	onPrepare, onSend func()
	vector            []float32
}

func (f *boundFixture) GetSemanticGenerationRuntime(context.Context, string) (GenerationRuntime, error) {
	return f.generation, nil
}
func (f *boundFixture) PrepareRemoteSemanticImage(context.Context, io.ReadSeeker, media.Format) ([]byte, error) {
	f.prepared++
	if f.onPrepare != nil {
		f.onPrepare()
	}
	return []byte("synthetic"), nil
}
func (f *boundFixture) EncodeText(context.Context, string) ([]float32, error)  { return f.send() }
func (f *boundFixture) EncodeImage(context.Context, []byte) ([]float32, error) { return f.send() }
func (f *boundFixture) send() ([]float32, error) {
	f.sent++
	if f.onSend != nil {
		f.onSend()
	}
	return f.vector, nil
}
func newBoundFixture(t *testing.T) (*BoundEncoder, *boundFixture) {
	t.Helper()
	identity := testSpaceIdentity()
	hash, err := identity.Fingerprint()
	if err != nil {
		t.Fatal(err)
	}
	f := &boundFixture{generation: GenerationRuntime{ID: "generation", State: "active", EmbeddingDimension: 512, SpaceFingerprint: hash}, vector: make([]float32, 512)}
	f.vector[0] = 2
	e, err := NewBoundEncoder(f, f, f, identity)
	if err != nil {
		t.Fatal(err)
	}
	return e, f
}
func TestBoundEncoderBothPaths(t *testing.T) {
	for _, img := range []bool{false, true} {
		e, f := newBoundFixture(t)
		var v []float32
		var err error
		if img {
			v, err = e.EncodeSemanticStream(context.Background(), "generation", bytes.NewReader(nil), media.FormatPNG)
		} else {
			v, err = e.EncodeSemanticText(context.Background(), "generation", "text")
		}
		if err != nil || len(v) != 512 || v[0] != 1 || f.sent != 1 {
			t.Fatalf("result %v calls %d", err, f.sent)
		}
	}
}
func TestBoundEncoderRejectsBeforeWork(t *testing.T) {
	for _, change := range []func(*boundFixture){
		func(f *boundFixture) { f.generation.SpaceFingerprint = "" }, func(f *boundFixture) { f.generation.EmbeddingDimension = 768 }, func(f *boundFixture) { f.generation.State = "retired" }, func(f *boundFixture) { f.generation.ID = "another" },
	} {
		for _, img := range []bool{false, true} {
			e, f := newBoundFixture(t)
			change(f)
			var err error
			if img {
				_, err = e.EncodeSemanticStream(context.Background(), "generation", bytes.NewReader(nil), media.FormatPNG)
			} else {
				_, err = e.EncodeSemanticText(context.Background(), "generation", "text")
			}
			if err == nil || f.sent != 0 || f.prepared != 0 {
				t.Fatal("invalid generation performed work")
			}
		}
	}
}
func TestBoundEncoderDiscardsChangedGeneration(t *testing.T) {
	for _, img := range []bool{false, true} {
		e, f := newBoundFixture(t)
		f.onSend = func() { f.generation.State = "retired" }
		var v []float32
		var err error
		if img {
			v, err = e.EncodeSemanticStream(context.Background(), "generation", bytes.NewReader(nil), media.FormatPNG)
		} else {
			v, err = e.EncodeSemanticText(context.Background(), "generation", "text")
		}
		if err == nil || v != nil {
			t.Fatal("returned stale vector")
		}
	}
	e, f := newBoundFixture(t)
	f.onPrepare = func() { f.generation.SpaceFingerprint = "" }
	if v, err := e.EncodeSemanticStream(context.Background(), "generation", bytes.NewReader(nil), media.FormatPNG); err == nil || v != nil || f.sent != 0 {
		t.Fatal("sent after changed preparation")
	}
}
func TestBoundEncoderStatesAndMalformedVector(t *testing.T) {
	for _, state := range []string{"building", "ready"} {
		e, f := newBoundFixture(t)
		f.generation.State = state
		if _, err := e.EncodeSemanticText(context.Background(), "generation", "text"); err == nil {
			t.Fatal("query accepted inactive")
		}
		if _, err := e.EncodeSemanticStream(context.Background(), "generation", bytes.NewReader(nil), media.FormatPNG); err != nil {
			t.Fatal(err)
		}
	}
	e, f := newBoundFixture(t)
	f.vector = []float32{1}
	if v, err := e.EncodeSemanticText(context.Background(), "generation", "text"); err == nil || v != nil {
		t.Fatal("accepted wrong vector")
	}
}
