package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/catalog"
	"github.com/HappyQuQu/foliopath/internal/curation"
	"github.com/HappyQuQu/foliopath/internal/scanner"
	"github.com/HappyQuQu/foliopath/migrations"
	"github.com/pressly/goose/v3"
)

var automaticTagTestTime = time.Date(2026, 10, 22, 9, 0, 0, 0, time.UTC)

func automaticPlanForAsset(t *testing.T, store *Store, assetID int64, names ...string) curation.AutomaticTagPlan {
	t.Helper()
	plan := curation.AutomaticTagPlan{AssetID: assetID, ModelIdentity: "model:synthetic-tags-v1"}
	if err := store.db.QueryRow(`SELECT library_id,source_fingerprint FROM assets WHERE id=?`, assetID).Scan(&plan.LibraryID, &plan.SourceFingerprint); err != nil {
		t.Fatal(err)
	}
	for index, name := range names {
		plan.Labels = append(plan.Labels, curation.AutomaticLabel{Name: name, Confidence: 1 - float64(index)/100})
	}
	return plan
}

func mergeAutomaticForTest(store *Store, plan curation.AutomaticTagPlan) (curation.AutomaticTagMergeResult, error) {
	var result curation.AutomaticTagMergeResult
	err := store.withWriteTx(context.Background(), func(tx *sql.Tx) error {
		var err error
		result, err = mergeAutomaticAssetTagsTx(context.Background(), tx, plan, automaticTagTestTime)
		return err
	})
	return result, err
}

func requireAutomaticMerge(t *testing.T, store *Store, plan curation.AutomaticTagPlan) curation.AutomaticTagMergeResult {
	t.Helper()
	result, err := mergeAutomaticForTest(store, plan)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func requireTagSources(t *testing.T, store *Store, assetID int64, expected map[string]curation.TagSource) curation.AssetState {
	t.Helper()
	state, err := store.GetAssetState(context.Background(), assetID)
	if err != nil {
		t.Fatal(err)
	}
	actual := map[string]curation.TagSource{}
	for _, tag := range state.Tags {
		actual[tag.Name] = tag.Source
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("tag sources=%v, want %v", actual, expected)
	}
	return state
}

func curationTagID(t *testing.T, state curation.AssetState, name string) int64 {
	t.Helper()
	for _, tag := range state.Tags {
		if tag.Name == name {
			return tag.ID
		}
	}
	t.Fatalf("tag %q not in %+v", name, state.Tags)
	return 0
}

func suppressionCount(t *testing.T, store *Store, assetID int64) int {
	t.Helper()
	var count int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM asset_tag_suppressions WHERE asset_id=?`, assetID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func TestAutomaticTagsPreserveManualAndAreIdempotentAcrossModels(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	assetID := catalogAssetID(t, store, "photo-10.jpg")
	service := newCurationServiceForStore(t, store, automaticTagTestTime)
	manual, _, err := service.CreateTag(context.Background(), "Human album")
	if err != nil {
		t.Fatal(err)
	}
	state, _ := service.GetAssetState(context.Background(), assetID)
	if _, err := service.ReplaceAssetTags(context.Background(), assetID, state.Revision, []int64{manual.ID}); err != nil {
		t.Fatal(err)
	}
	plan := automaticPlanForAsset(t, store, assetID, "HUMAN ALBUM", "Machine alpha", "Machine beta", "machine ALPHA")
	result := requireAutomaticMerge(t, store, plan)
	if result.Added != 2 || result.Removed != 0 {
		t.Fatalf("first merge=%+v", result)
	}
	expected := map[string]curation.TagSource{"Human album": curation.TagSourceManual, "Machine alpha": curation.TagSourceAI, "Machine beta": curation.TagSourceAI}
	before := requireTagSources(t, store, assetID, expected)
	if again := requireAutomaticMerge(t, store, plan); again.Added != 0 || again.Removed != 0 {
		t.Fatalf("replay=%+v", again)
	}
	after := requireTagSources(t, store, assetID, expected)
	if after.Revision != before.Revision {
		t.Fatal("identical plan advanced curation revision")
	}
	plan.ModelIdentity = "model:synthetic-tags-v2"
	requireAutomaticMerge(t, store, plan)
	after = requireTagSources(t, store, assetID, expected)
	if after.Revision <= before.Revision {
		t.Fatal("model lineage update did not invalidate revision")
	}
	page, err := service.ListTags(context.Background(), curation.TagListRequest{LibraryID: libraryID, Kinds: []catalog.AssetKind{catalog.KindImage, catalog.KindAnimated}})
	if err != nil || len(page.Items) != 3 {
		t.Fatalf("cloud=%+v, %v", page, err)
	}
	for _, tag := range page.Items {
		if tag.AssetCount != 1 || tag.Source != "" {
			t.Fatalf("global tag must count one image without association source: %+v", tag)
		}
	}
	plan.Labels = nil
	if result := requireAutomaticMerge(t, store, plan); result.Removed != 2 {
		t.Fatalf("zero-label merge=%+v", result)
	}
	requireTagSources(t, store, assetID, map[string]curation.TagSource{"Human album": curation.TagSourceManual})
}

func TestAutomaticTagManualRemoveRestoreAndUnchangedSource(t *testing.T) {
	store, _ := openTestStore(t)
	seedBrowseCatalog(t, store)
	assetID := catalogAssetID(t, store, "photo-10.jpg")
	service := newCurationServiceForStore(t, store, automaticTagTestTime)
	plan := automaticPlanForAsset(t, store, assetID, "Machine alpha", "Machine beta")
	requireAutomaticMerge(t, store, plan)
	state := requireTagSources(t, store, assetID, map[string]curation.TagSource{"Machine alpha": curation.TagSourceAI, "Machine beta": curation.TagSourceAI})
	alpha, beta := curationTagID(t, state, "Machine alpha"), curationTagID(t, state, "Machine beta")
	if _, err := service.ReplaceAssetTags(context.Background(), assetID, state.Revision, []int64{beta}); err != nil {
		t.Fatal(err)
	}
	if suppressionCount(t, store, assetID) != 1 {
		t.Fatal("manual removal did not persist suppression")
	}
	plan.ModelIdentity = "model:replacement-model"
	if result := requireAutomaticMerge(t, store, plan); result.SkippedSuppressed != 1 || result.Added != 0 {
		t.Fatalf("rerun=%+v", result)
	}
	state = requireTagSources(t, store, assetID, map[string]curation.TagSource{"Machine beta": curation.TagSourceAI})
	if _, err := service.ReplaceAssetTags(context.Background(), assetID, state.Revision, []int64{beta, alpha}); err != nil {
		t.Fatal(err)
	}
	if suppressionCount(t, store, assetID) != 0 {
		t.Fatal("manual restoration retained suppression")
	}
	requireTagSources(t, store, assetID, map[string]curation.TagSource{"Machine alpha": curation.TagSourceManual, "Machine beta": curation.TagSourceAI})
	plan.Labels = nil
	requireAutomaticMerge(t, store, plan)
	requireTagSources(t, store, assetID, map[string]curation.TagSource{"Machine alpha": curation.TagSourceManual})
}

func TestAutomaticTagExplicitPromotionUsesCASAndLegacyManualRetryIsIdempotent(t *testing.T) {
	store, _ := openTestStore(t)
	seedBrowseCatalog(t, store)
	assetID := catalogAssetID(t, store, "photo-10.jpg")
	service := newCurationServiceForStore(t, store, automaticTagTestTime)
	plan := automaticPlanForAsset(t, store, assetID, "Machine alpha")
	requireAutomaticMerge(t, store, plan)
	state := requireTagSources(t, store, assetID, map[string]curation.TagSource{"Machine alpha": curation.TagSourceAI})
	id := state.Tags[0].ID
	if _, _, err := service.CreateTag(context.Background(), "Other concurrent update"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.AddAssetTag(context.Background(), assetID, state.Revision, id); !errors.Is(err, curation.ErrPreconditionFailed) {
		t.Fatalf("stale promotion=%v", err)
	}
	if _, err := service.ReplaceAssetTags(context.Background(), assetID, state.Revision, nil); !errors.Is(err, curation.ErrPreconditionFailed) {
		t.Fatalf("stale removal=%v", err)
	}
	if suppressionCount(t, store, assetID) != 0 {
		t.Fatal("stale removal changed protection")
	}
	state = requireTagSources(t, store, assetID, map[string]curation.TagSource{"Machine alpha": curation.TagSourceAI})
	promoted, err := service.AddAssetTag(context.Background(), assetID, state.Revision, id)
	if err != nil || promoted.Tags[0].Source != curation.TagSourceManual {
		t.Fatalf("promotion=%+v, %v", promoted, err)
	}
	replayed, err := service.AddAssetTag(context.Background(), assetID, state.Revision, id)
	if err != nil || replayed.Revision != promoted.Revision {
		t.Fatalf("legacy accepted retry=%+v, %v", replayed, err)
	}
	plan.Labels = nil
	requireAutomaticMerge(t, store, plan)
	requireTagSources(t, store, assetID, map[string]curation.TagSource{"Machine alpha": curation.TagSourceManual})
}

func TestAutomaticTagRenameProtectsAllCurrentAssignmentsAndCaseOnlyEdits(t *testing.T) {
	for _, newName := range []string{"Human renamed", "MACHINE ALPHA"} {
		t.Run(newName, func(t *testing.T) {
			store, _ := openTestStore(t)
			seedBrowseCatalog(t, store)
			first, second := catalogAssetID(t, store, "photo-10.jpg"), catalogAssetID(t, store, "photo-2.jpg")
			service := newCurationServiceForStore(t, store, automaticTagTestTime)
			for _, id := range []int64{first, second} {
				requireAutomaticMerge(t, store, automaticPlanForAsset(t, store, id, "Machine alpha"))
			}
			state := requireTagSources(t, store, first, map[string]curation.TagSource{"Machine alpha": curation.TagSourceAI})
			if _, err := service.RenameTag(context.Background(), state.Tags[0].ID, "Machine alpha"); err != nil {
				t.Fatal(err)
			}
			requireTagSources(t, store, first, map[string]curation.TagSource{"Machine alpha": curation.TagSourceAI})
			if _, err := service.RenameTag(context.Background(), state.Tags[0].ID, newName); err != nil {
				t.Fatal(err)
			}
			for _, id := range []int64{first, second} {
				requireAutomaticMerge(t, store, automaticPlanForAsset(t, store, id, "Machine alpha"))
				requireTagSources(t, store, id, map[string]curation.TagSource{newName: curation.TagSourceManual})
				if suppressionCount(t, store, id) != 1 {
					t.Fatal("rename did not suppress previous name")
				}
			}
		})
	}
}

func TestAutomaticTagGlobalDeleteSuppressesCurrentAssetsButNotFutureAssets(t *testing.T) {
	store, _ := openTestStore(t)
	seedBrowseCatalog(t, store)
	first, second := catalogAssetID(t, store, "photo-10.jpg"), catalogAssetID(t, store, "photo-2.jpg")
	service := newCurationServiceForStore(t, store, automaticTagTestTime)
	requireAutomaticMerge(t, store, automaticPlanForAsset(t, store, first, "Machine alpha"))
	state := requireTagSources(t, store, first, map[string]curation.TagSource{"Machine alpha": curation.TagSourceAI})
	oldID := state.Tags[0].ID
	if _, _, err := service.CreateTag(context.Background(), "ID retention sentinel"); err != nil {
		t.Fatal(err)
	}
	if err := service.DeleteTag(context.Background(), oldID); err != nil {
		t.Fatal(err)
	}
	if result := requireAutomaticMerge(t, store, automaticPlanForAsset(t, store, first, "MACHINE ALPHA")); result.SkippedSuppressed != 1 {
		t.Fatalf("deleted-label rerun=%+v", result)
	}
	requireAutomaticMerge(t, store, automaticPlanForAsset(t, store, second, "Machine alpha"))
	secondState := requireTagSources(t, store, second, map[string]curation.TagSource{"Machine alpha": curation.TagSourceAI})
	if secondState.Tags[0].ID == oldID {
		t.Fatal("fixture did not recreate a distinct tag ID")
	}
	requireAutomaticMerge(t, store, automaticPlanForAsset(t, store, first, "Machine alpha"))
	firstState := requireTagSources(t, store, first, map[string]curation.TagSource{})
	if _, err := service.AddAssetTag(context.Background(), first, firstState.Revision, secondState.Tags[0].ID); err != nil {
		t.Fatal(err)
	}
	if suppressionCount(t, store, first) != 0 {
		t.Fatal("explicit restored label remains suppressed")
	}
	requireTagSources(t, store, first, map[string]curation.TagSource{"Machine alpha": curation.TagSourceManual})
}

func TestAutomaticTagMergeRejectsStaleSourceAndRollsBackWithCaller(t *testing.T) {
	store, _ := openTestStore(t)
	seedBrowseCatalog(t, store)
	assetID := catalogAssetID(t, store, "photo-10.jpg")
	valid := automaticPlanForAsset(t, store, assetID, "Machine alpha")
	for _, change := range []func(*curation.AutomaticTagPlan){
		func(p *curation.AutomaticTagPlan) { p.SourceFingerprint = "v1:999:999" },
		func(p *curation.AutomaticTagPlan) { p.LibraryID++ },
		func(p *curation.AutomaticTagPlan) { p.AssetID = 999999 },
	} {
		plan := valid
		change(&plan)
		if _, err := mergeAutomaticForTest(store, plan); !errors.Is(err, curation.ErrAutomaticTagSourceChanged) {
			t.Fatalf("stale source=%v", err)
		}
	}
	videoPlan := automaticPlanForAsset(t, store, catalogAssetID(t, store, "Album 2/clip.mp4"), "Machine alpha")
	if _, err := mergeAutomaticForTest(store, videoPlan); !errors.Is(err, curation.ErrAutomaticTagSourceChanged) {
		t.Fatalf("video accepted: %v", err)
	}
	sentinel := errors.New("simulated checkpoint conflict")
	err := store.withWriteTx(context.Background(), func(tx *sql.Tx) error {
		if _, err := mergeAutomaticAssetTagsTx(context.Background(), tx, valid, automaticTagTestTime); err != nil {
			return err
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatal(err)
	}
	requireTagSources(t, store, assetID, map[string]curation.TagSource{})
	var count int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM tags WHERE normalized_name='machine alpha'`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("rolled-back tag remains: %d, %v", count, err)
	}
}

func TestAutomaticTagScannerInvalidationPreservesManualAndSuppressions(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	assetID := catalogAssetID(t, store, "photo-10.jpg")
	service := newCurationServiceForStore(t, store, automaticTagTestTime)
	plan := automaticPlanForAsset(t, store, assetID, "Machine alpha", "Machine beta", "Machine gamma")
	requireAutomaticMerge(t, store, plan)
	state := requireTagSources(t, store, assetID, map[string]curation.TagSource{"Machine alpha": curation.TagSourceAI, "Machine beta": curation.TagSourceAI, "Machine gamma": curation.TagSourceAI})
	alpha, beta := curationTagID(t, state, "Machine alpha"), curationTagID(t, state, "Machine beta")
	state, err := service.AddAssetTag(context.Background(), assetID, state.Revision, alpha)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ReplaceAssetTags(context.Background(), assetID, state.Revision, []int64{alpha, beta}); err != nil {
		t.Fatal(err)
	}
	run, err := store.BeginFullScan(context.Background(), libraryID, scanner.TriggerManual)
	if err != nil {
		t.Fatal(err)
	}
	entries := []scanner.CatalogEntry{{Kind: scanner.CatalogEntryDirectory}, {Kind: scanner.CatalogEntryAsset, RelativePath: "photo-10.jpg", Name: "photo-10.jpg", MTimeNS: 11, SizeBytes: 10, AssetKind: scanner.AssetKindImage, MediaFormat: scanner.MediaFormatJPEG, MIMEType: "image/jpeg"}}
	if err := store.UpsertCatalogBatch(context.Background(), run.ID, entries); err != nil {
		t.Fatal(err)
	}
	requireTagSources(t, store, assetID, map[string]curation.TagSource{"Machine alpha": curation.TagSourceManual})
	if suppressionCount(t, store, assetID) != 1 {
		t.Fatal("scanner source update erased suppression")
	}
	if _, err := mergeAutomaticForTest(store, plan); !errors.Is(err, curation.ErrAutomaticTagSourceChanged) {
		t.Fatalf("old inference accepted after scan: %v", err)
	}
	requireAutomaticMerge(t, store, automaticPlanForAsset(t, store, assetID, "Machine beta", "Machine gamma"))
	entries[1].AssetKind, entries[1].MediaFormat, entries[1].MIMEType = scanner.AssetKindAnimated, scanner.MediaFormatGIF, "image/gif"
	if err := store.UpsertCatalogBatch(context.Background(), run.ID, entries); err != nil {
		t.Fatal(err)
	}
	requireTagSources(t, store, assetID, map[string]curation.TagSource{"Machine alpha": curation.TagSourceManual})
	result := requireAutomaticMerge(t, store, automaticPlanForAsset(t, store, assetID, "Machine beta", "Machine gamma"))
	if result.Added != 1 || result.SkippedSuppressed != 1 {
		t.Fatalf("animated merge=%+v", result)
	}
	if _, err := store.FailFullScan(context.Background(), run.ID, scanner.SkipCounts{}, "partial_tree_unreadable"); err != nil {
		t.Fatal(err)
	}
	if suppressionCount(t, store, assetID) != 1 {
		t.Fatal("failed scan erased human state")
	}
}

func TestAutomaticTagsBoundVisibleSetWithManualPriority(t *testing.T) {
	store, _ := openTestStore(t)
	seedBrowseCatalog(t, store)
	assetID := catalogAssetID(t, store, "photo-10.jpg")
	service := newCurationServiceForStore(t, store, automaticTagTestTime)
	ids := make([]int64, 0, 19)
	for i := 0; i < 19; i++ {
		tag, _, err := service.CreateTag(context.Background(), fmt.Sprintf("Human %02d", i))
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, tag.ID)
	}
	state, _ := service.GetAssetState(context.Background(), assetID)
	if _, err := service.ReplaceAssetTags(context.Background(), assetID, state.Revision, ids); err != nil {
		t.Fatal(err)
	}
	result := requireAutomaticMerge(t, store, automaticPlanForAsset(t, store, assetID, "Machine best", "Machine next"))
	if result.Added != 1 || result.SkippedLimit != 1 {
		t.Fatalf("bounded merge=%+v", result)
	}
	state, _ = service.GetAssetState(context.Background(), assetID)
	if len(state.Tags) != 20 {
		t.Fatalf("visible tags=%d", len(state.Tags))
	}
	if _, err := service.AddAssetTag(context.Background(), assetID, state.Revision, curationTagID(t, state, "Machine best")); err != nil {
		t.Fatal(err)
	}
	result = requireAutomaticMerge(t, store, automaticPlanForAsset(t, store, assetID, "Machine next"))
	if result.Added != 0 || result.Removed != 0 || result.SkippedLimit != 1 {
		t.Fatalf("full manual set=%+v", result)
	}
}

func TestAutomaticTagSuppressionPersistsAndCascadesOnlyWithAsset(t *testing.T) {
	store, filename := openTestStore(t)
	seedBrowseCatalog(t, store)
	assetID := catalogAssetID(t, store, "photo-10.jpg")
	service := newCurationServiceForStore(t, store, automaticTagTestTime)
	plan := automaticPlanForAsset(t, store, assetID, "Machine alpha")
	requireAutomaticMerge(t, store, plan)
	state, _ := store.GetAssetState(context.Background(), assetID)
	if _, err := service.ReplaceAssetTags(context.Background(), assetID, state.Revision, nil); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(context.Background(), filename, Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if result := requireAutomaticMerge(t, reopened, plan); result.SkippedSuppressed != 1 {
		t.Fatalf("persistent suppression=%+v", result)
	}
	if _, err := store.db.Exec(`UPDATE libraries SET status='offline' WHERE id=?`, plan.LibraryID); err != nil {
		t.Fatal(err)
	}
	if suppressionCount(t, reopened, assetID) != 1 {
		t.Fatal("offline erased suppression")
	}
	if _, err := store.db.Exec(`DELETE FROM assets WHERE id=?`, assetID); err != nil {
		t.Fatal(err)
	}
	if suppressionCount(t, reopened, assetID) != 0 {
		t.Fatal("deleted asset retained orphan suppression")
	}
}

func TestAutomaticTagMigrationPreservesLegacyManualAssignments(t *testing.T) {
	ctx := context.Background()
	filename := filepath.Join(t.TempDir(), "legacy-tags.db")
	db, err := sql.Open("sqlite", buildDSN(filename, time.Second))
	if err != nil {
		t.Fatal(err)
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.UpTo(ctx, 40); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
        INSERT INTO libraries(id,name,name_key,root_rel_path,status,current_generation,created_at_ms,updated_at_ms,name_sort_key)
        VALUES(1,'Archive','archive','archive','ready',1,1,1,X'01');
        INSERT INTO directories(id,library_id,parent_id,relative_path,name,mtime_ns,last_seen_generation,direct_asset_count,recursive_asset_count)
        VALUES(1,1,NULL,'','',0,1,1,1);
        INSERT INTO assets(id,library_id,directory_id,relative_path,name,kind,media_format,mime_type,size_bytes,mtime_ns,last_seen_generation,source_fingerprint)
        VALUES(1,1,1,'old.jpg','old.jpg','image','jpeg','image/jpeg',42,10,1,'v1:42:10');
        INSERT INTO tags(id,name,normalized_name,created_at_ms,updated_at_ms) VALUES(99999,'Legacy manual','legacy manual',1,1);
        INSERT INTO asset_tags(asset_id,library_id,tag_id,created_at_ms) VALUES(1,1,99999,1);
        INSERT INTO ai_tag_reviews(library_id,asset_id,tag_id,decision,accepted_curation_revision,reviewed_at_ms)
        VALUES(1,1,99999,'accepted',1,1);`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	store, err := Open(ctx, filename, Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	requireTagSources(t, store, 1, map[string]curation.TagSource{"Legacy manual": curation.TagSourceManual})
	requireAutomaticMerge(t, store, automaticPlanForAsset(t, store, 1, "Legacy manual", "Machine alpha"))
	requireTagSources(t, store, 1, map[string]curation.TagSource{"Legacy manual": curation.TagSourceManual, "Machine alpha": curation.TagSourceAI})
	var metadata int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM asset_tags WHERE tag_id=99999 AND (ai_model_identity IS NOT NULL OR ai_source_fingerprint IS NOT NULL OR ai_confidence IS NOT NULL)`).Scan(&metadata); err != nil || metadata != 0 {
		t.Fatalf("legacy metadata=%d %v", metadata, err)
	}
	if _, err := store.db.Exec(`INSERT INTO asset_tags(asset_id,library_id,tag_id,created_at_ms,source) VALUES(1,1,1,1,'ai')`); err == nil {
		t.Fatal("AI row without provenance accepted")
	}
}

func TestAutomaticTagCurationReadsOneConsistentSnapshot(t *testing.T) {
	store, _ := openTestStore(t)
	seedBrowseCatalog(t, store)
	assetID := catalogAssetID(t, store, "photo-10.jpg")
	plan := automaticPlanForAsset(t, store, assetID, "Machine alpha")
	requireAutomaticMerge(t, store, plan)
	tx, err := store.db.BeginTx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	before, err := readAssetCurationState(context.Background(), tx, assetID)
	if err != nil {
		t.Fatal(err)
	}
	plan.Labels = []curation.AutomaticLabel{{Name: "Machine beta", Confidence: .9}}
	done := make(chan error, 1)
	go func() { _, err := mergeAutomaticForTest(store, plan); done <- err }()
	// A reader holds one stable snapshot even while an independent writer attempts
	// a complete label replacement. Both WAL readers and serialized readers are valid.
	again, err := readAssetCurationState(context.Background(), tx, assetID)
	if err != nil || !reflect.DeepEqual(before, again) {
		t.Fatalf("mixed curation snapshot: %+v / %+v, %v", before, again, err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	after := requireTagSources(t, store, assetID, map[string]curation.TagSource{"Machine beta": curation.TagSourceAI})
	if after.Revision <= before.Revision {
		t.Fatal("changed data did not return a newer revision")
	}
}

func TestAutomaticTagConcurrentCurationReadsNeverMixRevisionAndLabels(t *testing.T) {
	store, _ := openTestStore(t)
	seedBrowseCatalog(t, store)
	assetID := catalogAssetID(t, store, "photo-10.jpg")
	plan := automaticPlanForAsset(t, store, assetID, "Machine alpha")
	requireAutomaticMerge(t, store, plan)
	initial, err := store.GetAssetState(context.Background(), assetID)
	if err != nil {
		t.Fatal(err)
	}
	type committed struct {
		revision int64
		name     string
	}
	writes := make(chan committed, 40)
	done := make(chan error, 1)
	go func() {
		for index := 0; index < 40; index++ {
			name := "Machine beta"
			if index%2 == 1 {
				name = "Machine alpha"
			}
			plan.Labels = []curation.AutomaticLabel{{Name: name, Confidence: .9}}
			var revision int64
			err := store.withWriteTx(context.Background(), func(tx *sql.Tx) error {
				if _, err := mergeAutomaticAssetTagsTx(context.Background(), tx, plan, automaticTagTestTime); err != nil {
					return err
				}
				return tx.QueryRow(`SELECT revision FROM curation_state WHERE singleton_key=1`).Scan(&revision)
			})
			if err != nil {
				done <- err
				return
			}
			writes <- committed{revision: revision, name: name}
		}
		done <- nil
	}()
	observed := make([]curation.AssetState, 0, 100)
	for index := 0; index < 100; index++ {
		state, err := store.GetAssetState(context.Background(), assetID)
		if err != nil {
			t.Fatal(err)
		}
		observed = append(observed, state)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	close(writes)
	commits := map[int64]string{initial.Revision: initial.Tags[0].Name}
	for value := range writes {
		commits[value.revision] = value.name
	}
	for _, state := range observed {
		if len(state.Tags) != 1 || commits[state.Revision] != state.Tags[0].Name || state.Tags[0].Source != curation.TagSourceAI {
			t.Fatalf("read mixed state: %+v; committed label at revision is %q", state, commits[state.Revision])
		}
	}
}

func TestAutomaticTagFailedGlobalEditsRollbackAllProtectionChanges(t *testing.T) {
	store, _ := openTestStore(t)
	seedBrowseCatalog(t, store)
	assetID := catalogAssetID(t, store, "photo-10.jpg")
	service := newCurationServiceForStore(t, store, automaticTagTestTime)
	requireAutomaticMerge(t, store, automaticPlanForAsset(t, store, assetID, "Machine alpha"))
	state, _ := service.GetAssetState(context.Background(), assetID)
	id := state.Tags[0].ID
	if _, _, err := service.CreateTag(context.Background(), "Existing human name"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.RenameTag(context.Background(), id, "Existing human name"); !errors.Is(err, curation.ErrTagNameConflict) {
		t.Fatalf("rename conflict=%v", err)
	}
	requireTagSources(t, store, assetID, map[string]curation.TagSource{"Machine alpha": curation.TagSourceAI})
	if suppressionCount(t, store, assetID) != 0 {
		t.Fatal("rename conflict left a suppression")
	}
	// Legacy vocabulary retains its FK guard. A failed delete must not leave an
	// invisible suppression even though protection is written before DELETE.
	if _, err := store.db.Exec(`INSERT INTO ai_tag_vocabulary_entries(snapshot_id,tag_id) SELECT id,? FROM ai_tag_vocabulary_snapshots WHERE state='active'`, id); err != nil {
		t.Fatal(err)
	}
	if err := service.DeleteTag(context.Background(), id); err == nil {
		t.Fatal("expected legacy vocabulary delete restriction")
	}
	requireTagSources(t, store, assetID, map[string]curation.TagSource{"Machine alpha": curation.TagSourceAI})
	if suppressionCount(t, store, assetID) != 0 {
		t.Fatal("failed global deletion left a suppression")
	}
}
