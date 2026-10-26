//go:build libvips && remotemlprobe

package sqlite

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"math"
	"os"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	"github.com/HappyQuQu/foliopath/internal/inference/remoteml"
	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/media/imagevips"
	"github.com/HappyQuQu/foliopath/internal/semantic"
	"github.com/davidbyttow/govips/v2/vips"
)

// Synthetic catalog and bytes only: this source never opens original media.
type remoteProbeSource struct {
	store *Store
	data  []byte
}

func (s remoteProbeSource) OpenSemanticAsset(ctx context.Context, libraryID, assetID int64) (semantic.SemanticAsset, error) {
	var fingerprint string
	if err := s.store.db.QueryRowContext(ctx, `SELECT source_fingerprint FROM assets WHERE library_id=? AND id=?`, libraryID, assetID).Scan(&fingerprint); err != nil {
		return semantic.SemanticAsset{}, err
	}
	return semantic.SemanticAsset{File: semanticReadSeekCloser{bytes.NewReader(s.data)}, Format: media.FormatJPEG, SourceFingerprint: fingerprint}, nil
}

// This opt-in probe validates composition, not deployment identity or activation.
func TestRemoteSemanticPersistedProbe(t *testing.T) {
	origin := os.Getenv("FOLIOPATH_TEST_ML_ORIGIN")
	if origin == "" {
		t.Fatal("FOLIOPATH_TEST_ML_ORIGIN required")
	}
	runtime := imagevips.NewRuntime()
	if err := runtime.Start(); err != nil {
		t.Fatal(err)
	}
	defer runtime.Shutdown()
	client, err := remoteml.New(origin)
	if raw := os.Getenv("FOLIOPATH_TEST_ML_IDENTITY"); raw != "" {
		if client != nil {
			client.Close()
		}
		var expected remoteml.ServiceIdentity
		if err := json.Unmarshal([]byte(raw), &expected); err != nil {
			t.Fatal(err)
		}
		client, err = remoteml.NewVerified(context.Background(), origin, expected)
	}
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	store, path := openTestStore(t)
	library := seedBrowseCatalog(t, store)
	// Deliberately synthetic identity: this is not a claim about the remote instance.
	identity := spaceFixture()
	generation := activateProbeDeployment(t, ctx, store)
	seedSemanticLibrarySettings(t, store, library)
	now := time.Now().UTC()
	admitSemanticBackfill(t, store, library, generation, now, "remote-probe-job")
	job, found, err := store.ClaimSemanticBackfill(ctx, now.Add(time.Second), time.Minute)
	if err != nil || !found {
		t.Fatalf("claim found=%v: %v", found, err)
	}
	encoder, err := semantic.NewBoundEncoder(store, imagevips.New(), client, identity)
	if err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 255, A: 255})
		}
	}
	var data bytes.Buffer
	if err := jpeg.Encode(&data, img, nil); err != nil {
		t.Fatal(err)
	}
	processor, err := semantic.NewStreamBackfillProcessor(store, remoteProbeSource{store, data.Bytes()}, encoder, store, store, func() time.Time { return now.Add(2 * time.Second) }, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := processor.Process(ctx, job); err != nil {
		t.Fatal(err)
	}
	stored, err := scanSemanticJob(store.db.QueryRowContext(ctx, semanticJobSelect+` WHERE job.id=?`, job.ID))
	if err != nil || stored.TotalItems < 1 || stored.CompletedItems != stored.TotalItems || stored.State != semantic.JobSucceeded {
		t.Fatalf("incomplete job: %+v %v", stored, err)
	}
	// Close and reopen the actual temporary database before querying persisted vectors.
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(ctx, path, Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	encoder, err = semantic.NewBoundEncoder(reopened, imagevips.New(), client, identity)
	if err != nil {
		t.Fatal(err)
	}
	query, err := encoder.EncodeSemanticText(ctx, generation, "a red square")
	if err != nil {
		t.Fatal(err)
	}
	matches, err := reopened.SearchSemanticVectors(ctx, semantic.VectorSearchRequest{GenerationID: generation, LibraryID: library, Query: query, Limit: 200})
	if err != nil || int64(len(matches)) != stored.TotalItems {
		t.Fatalf("persisted matches=%d expected=%d: %v", len(matches), stored.TotalItems, err)
	}
	for _, match := range matches {
		if math.IsNaN(float64(match.Score)) || math.IsInf(float64(match.Score), 0) || match.Score < -1.001 || match.Score > 1.001 {
			t.Fatalf("invalid score: %+v", match)
		}
	}
	t.Logf("real offline ML: %d synthetic images committed; database reopened; text vector retrieved %d persisted matches; no quality/activation claim", stored.CompletedItems, len(matches))
	exerciseRemoteVideo(t, ctx, client, data.Bytes())
}

type remoteVideoSource struct{ storyboard semantic.CompleteStoryboard }

func (s remoteVideoSource) OpenCompleteStoryboard(context.Context, int64, int64) (semantic.CompleteStoryboard, error) {
	return s.storyboard, nil
}

func exerciseRemoteVideo(t *testing.T, ctx context.Context, client *remoteml.Client, jpegBytes []byte) {
	t.Helper()
	image, err := vips.LoadImageFromBuffer(jpegBytes, nil)
	if err != nil {
		t.Fatal(err)
	}
	webpBytes, _, err := image.ExportWebp(vips.NewWebpExportParams())
	image.Close()
	if err != nil {
		t.Fatal(err)
	}
	store, _ := openTestStore(t)
	seedStoryboardAsset(t, store.db)
	generation := activateProbeDeployment(t, ctx, store)
	now := time.Now().UTC()
	if _, err := store.db.ExecContext(ctx, `INSERT INTO thumbnails(
		library_id, asset_id, variant, source_fingerprint, transform_version, cache_rel_path, status,
		width, height, byte_size, created_at_ms, last_accessed_at_ms, frame_count, sprite_columns, sprite_rows, cell_width, cell_height
	) VALUES(1,1,'storyboard','v1:42:100',1,'libraries/lib_1/story.webp','ready',1280,180,1000,?,?,4,4,1,320,180)`, now.UnixMilli(), now.UnixMilli()); err != nil {
		t.Fatal(err)
	}
	seedSemanticLibrarySettings(t, store, 1)
	fingerprint, err := semantic.StoryboardFingerprint("v1:42:100", 1, 4)
	if err != nil {
		t.Fatal(err)
	}
	storyboard := semantic.CompleteStoryboard{LibraryID: 1, AssetID: 1, SourceFingerprint: "v1:42:100", StoryboardFingerprint: fingerprint, TransformVersion: 1, PlanSize: 4}
	for i := 0; i < 4; i++ {
		storyboard.Frames = append(storyboard.Frames, semantic.StoryboardFrame{Ordinal: i, TimestampMS: int64(i+1) * 1000,
			Format: media.FormatWebP, Image: semanticReadSeekCloser{bytes.NewReader(webpBytes)}})
	}
	encoder, err := semantic.NewBoundEncoder(store, imagevips.New(), client, spaceFixture())
	if err != nil {
		t.Fatal(err)
	}
	processor, err := semantic.NewStreamVideoProcessor(remoteVideoSource{storyboard}, encoder, store, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := processor.Process(ctx, generation, 1, 1); err != nil {
		t.Fatal(err)
	}
	query, err := encoder.EncodeSemanticText(ctx, generation, "a red square")
	if err != nil {
		t.Fatal(err)
	}
	matches, err := store.SearchVideoSemanticVectors(ctx, semantic.VideoVectorSearchRequest{GenerationID: generation, LibraryID: 1, Query: query, Limit: 10})
	if err != nil || len(matches) != 1 || matches[0].PlanSize != 4 {
		t.Fatalf("video matches=%+v err=%v", matches, err)
	}
	if _, err := store.db.ExecContext(ctx, `DELETE FROM semantic_video_frames WHERE ordinal=3`); err != nil {
		t.Fatal(err)
	}
	matches, err = store.SearchVideoSemanticVectors(ctx, semantic.VideoVectorSearchRequest{GenerationID: generation, LibraryID: 1, Query: query, Limit: 10})
	if err != nil || len(matches) != 0 {
		t.Fatalf("partial plan searchable: %+v %v", matches, err)
	}
	t.Log("real offline ML: four generated storyboard frames persisted and searchable; partial plan excluded; no video decoding or quality claim")
}

// Uses the actual atomic activation commit; model metadata and admission remain
// synthetic fixtures, not a substitute for the unfinished app deployment loader.
func activateProbeDeployment(t *testing.T, ctx context.Context, store *Store) string {
	t.Helper()
	now := time.Now().UTC()
	identity := spaceFixture()
	fingerprint, err := identity.Fingerprint()
	if err != nil {
		t.Fatal(err)
	}
	model := activationModelFixture(now)
	model.StorageMode = aimodel.StorageService
	model.Package.ContentHash = fingerprint
	model.SourceIdentity = fingerprint
	if _, _, err := store.RegisterAIModel(ctx, model); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.CreateAIModelActivation(ctx, activationWorkFixture(model, now)); err != nil {
		t.Fatal(err)
	}
	work, found, err := store.ClaimAIModelActivation(ctx, now)
	if err != nil || !found {
		t.Fatalf("activation claim: %v %v", found, err)
	}
	generation := "aig_probe_deployment"
	if _, err := store.CommitDeploymentActivation(ctx, semantic.DeploymentActivationCommit{Identity: identity,
		Activation: aimodel.ActivationCommit{OperationID: work.Operation.ID, ExpectedRevision: work.Operation.Revision,
			ExpectedAvailabilityRevision: work.ExpectedAvailabilityRevision, UpdatedAt: now,
			Generation: aimodel.Generation{ID: generation, ModelID: model.ID, TransformVersion: 1, OutputSchemaVersion: 1,
				IndexFormatVersion: 1, EmbeddingDimension: 512, State: aimodel.GenerationActive, CreatedAt: now, ActivatedAt: &now, UpdatedAt: now}}}); err != nil {
		t.Fatal(err)
	}
	return generation
}
