package sqlite

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/semantic"
)

func spaceFixture() semantic.SpaceIdentity {
	return semantic.SpaceIdentity{Version: 1, Protocol: "immich-ml-v3.1.0", Model: "ViT-B-32__openai", Dimension: 512,
		RuntimeSHA256: strings.Repeat("a", 64), CacheManifestSHA256: strings.Repeat("b", 64), PreprocessingSHA256: strings.Repeat("c", 64)}
}

type spaceTransport struct {
	calls int
	after func()
}

func (s *spaceTransport) EncodeText(context.Context, string) ([]float32, error)  { return s.encode() }
func (s *spaceTransport) EncodeImage(context.Context, []byte) ([]float32, error) { return s.encode() }
func (s *spaceTransport) encode() ([]float32, error) {
	s.calls++
	if s.after != nil {
		s.after()
	}
	v := make([]float32, 512)
	v[0] = 1
	return v, nil
}
func (*spaceTransport) PrepareRemoteSemanticImage(context.Context, io.ReadSeeker, media.Format) ([]byte, error) {
	return []byte("synthetic"), nil
}

func TestBoundEncoderUsesPersistedGeneration(t *testing.T) {
	store, _ := openTestStore(t)
	ctx := context.Background()
	id := seedEmbeddingGeneration(t, store, 512)
	if _, err := store.db.Exec(`UPDATE semantic_generations SET state='building' WHERE id=?`, id); err != nil {
		t.Fatal(err)
	}
	if err := store.BindSemanticGenerationSpace(ctx, id, spaceFixture()); err != nil {
		t.Fatal(err)
	}
	transport := &spaceTransport{}
	encoder, err := semantic.NewBoundEncoder(store, transport, transport, spaceFixture())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := encoder.EncodeSemanticText(ctx, id, "query"); err == nil || transport.calls != 0 {
		t.Fatal("query used building generation")
	}
	if _, err := encoder.EncodeSemanticStream(ctx, id, bytes.NewReader(nil), media.FormatPNG); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`UPDATE semantic_generations SET state='active' WHERE id=?`, id); err != nil {
		t.Fatal(err)
	}
	if _, err := encoder.EncodeSemanticText(ctx, id, "query"); err != nil {
		t.Fatal(err)
	}
	transport.after = func() {
		if _, err := store.db.Exec(`UPDATE semantic_generations SET state='retired' WHERE id=?`, id); err != nil {
			t.Fatal(err)
		}
	}
	if vector, err := encoder.EncodeSemanticText(ctx, id, "query"); err == nil || vector != nil {
		t.Fatal("returned vector after persistent retirement")
	}
}

func TestGenerationSpaceRejectsExistingVectors(t *testing.T) {
	store, _ := openTestStore(t)
	ctx := context.Background()
	libraryID := seedBrowseCatalog(t, store)
	id := seedEmbeddingGeneration(t, store, 512)
	vector := make([]float32, 512)
	vector[0] = 1
	encoded, err := semantic.EncodeEmbedding(vector, 512)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.PutSemanticEmbeddingBatch(ctx, semantic.EmbeddingBatch{GenerationID: id, LibraryID: libraryID, Items: []semantic.EmbeddingItem{{AssetID: catalogAssetID(t, store, "photo-10.jpg"), SourceFingerprint: "v1:10:10", Vector: encoded, CreatedAt: time.Now().UTC()}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`UPDATE semantic_generations SET state='building' WHERE id=?`, id); err != nil {
		t.Fatal(err)
	}
	if err := store.BindSemanticGenerationSpace(ctx, id, spaceFixture()); !errors.Is(err, semantic.ErrIncompatibleSpace) {
		t.Fatalf("bound populated generation: %v", err)
	}
}

func TestGenerationSpacePersistsAcrossConnections(t *testing.T) {
	store, path := openTestStore(t)
	ctx := context.Background()
	id := seedEmbeddingGeneration(t, store, 512)
	if _, err := store.db.Exec(`UPDATE semantic_generations SET state='building' WHERE id=?`, id); err != nil {
		t.Fatal(err)
	}
	if err := store.BindSemanticGenerationSpace(ctx, id, spaceFixture()); err != nil {
		t.Fatal(err)
	}
	other, err := Open(ctx, path, Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	got, err := other.GetSemanticGenerationRuntime(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if err := semantic.RequireSpace(got.SpaceFingerprint, spaceFixture()); err != nil {
		t.Fatal(err)
	}
}

func TestGenerationSpaceBinding(t *testing.T) {
	store, _ := openTestStore(t)
	ctx := context.Background()
	id := seedEmbeddingGeneration(t, store, 512)
	identity := spaceFixture()
	legacy, err := store.GetSemanticGenerationRuntime(ctx, id)
	if err != nil || legacy.SpaceFingerprint != "" {
		t.Fatalf("legacy changed: %+v %v", legacy, err)
	}
	if err := store.BindSemanticGenerationSpace(ctx, id, identity); !errors.Is(err, semantic.ErrIncompatibleSpace) {
		t.Fatalf("bound active legacy: %v", err)
	}
	if _, err := store.db.Exec(`UPDATE semantic_generations SET state='building' WHERE id=?`, id); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := store.BindSemanticGenerationSpace(ctx, id, identity); err != nil {
			t.Fatal(err)
		}
	}
	got, err := store.GetSemanticGenerationRuntime(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if err := semantic.RequireSpace(got.SpaceFingerprint, identity); err != nil {
		t.Fatal(err)
	}
	changed := identity
	changed.CacheManifestSHA256 = strings.Repeat("d", 64)
	if err := store.BindSemanticGenerationSpace(ctx, id, changed); !errors.Is(err, semantic.ErrIncompatibleSpace) {
		t.Fatalf("rebound: %v", err)
	}
	if _, err := store.db.Exec(`UPDATE semantic_generation_spaces SET fingerprint=? WHERE generation_id=?`, strings.Repeat("e", 64), id); err == nil {
		t.Fatal("binding mutable")
	}
	if _, err := store.db.Exec(`UPDATE semantic_generations SET embedding_dimension=768 WHERE id=?`, id); err == nil {
		t.Fatal("bound dimension mutable")
	}
	if _, err := store.db.Exec(`DELETE FROM semantic_generations WHERE id=?`, id); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM semantic_generation_spaces`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("cascade: %d %v", count, err)
	}
}

func TestGenerationSpaceRejectsDimensionAndMissing(t *testing.T) {
	store, _ := openTestStore(t)
	ctx := context.Background()
	id := seedEmbeddingGeneration(t, store, 768)
	if _, err := store.db.Exec(`UPDATE semantic_generations SET state='building' WHERE id=?`, id); err != nil {
		t.Fatal(err)
	}
	if err := store.BindSemanticGenerationSpace(ctx, id, spaceFixture()); !errors.Is(err, semantic.ErrIncompatibleSpace) {
		t.Fatalf("accepted wrong dimension: %v", err)
	}
	if err := store.BindSemanticGenerationSpace(ctx, "aig_missing", spaceFixture()); !errors.Is(err, semantic.ErrSemanticGenerationUnavailable) {
		t.Fatalf("missing: %v", err)
	}
	invalid := spaceFixture()
	invalid.Version = 0
	if err := store.BindSemanticGenerationSpace(ctx, id, invalid); !errors.Is(err, semantic.ErrInvalidSpaceIdentity) {
		t.Fatalf("invalid: %v", err)
	}
}
