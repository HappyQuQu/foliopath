package sqlite

import (
	"context"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/catalog"
	"github.com/HappyQuQu/foliopath/internal/curation"
)

func TestLocalizedTagProjectionSearchAndManualAliases(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	first := catalogAssetID(t, store, "photo-10.jpg")
	second := catalogAssetID(t, store, "photo-2.jpg")
	service := newCurationServiceForStore(t, store, automaticTagTestTime)
	requireAutomaticMerge(t, store, automaticPlanForAsset(t, store, first, "car", "blue_background"))
	requireAutomaticMerge(t, store, automaticPlanForAsset(t, store, catalogAssetID(t, store, "Album 2/Nested/photo.jpg"), "car"))
	for _, query := range []string{"汽车", "车辆", "Car", "蓝色", "blue background"} {
		page, err := service.ListTags(context.Background(), curation.TagListRequest{Search: query, LibraryID: libraryID, Kinds: []catalog.AssetKind{catalog.KindImage}, Limit: 1})
		if err != nil || len(page.Items) != 1 || curation.LocalizeTag(page.Items[0]) == nil {
			t.Fatalf("%s: %+v %v", query, page, err)
		}
	}
	state, _ := service.GetAssetState(context.Background(), first)
	carID := curationTagID(t, state, "car")
	beforeRevision := state.Revision
	for _, tag := range state.Tags {
		if curation.LocalizeTag(tag) == nil {
			t.Fatal("missing per-asset AI localization")
		}
	}
	if _, err := service.AddAssetTag(context.Background(), first, state.Revision, carID); err != nil {
		t.Fatal(err)
	}
	page, err := service.ListTags(context.Background(), curation.TagListRequest{Search: "汽车", LibraryID: libraryID})
	if err != nil || len(page.Items) != 1 || page.Items[0].Name != "car" || page.Items[0].ID != carID || curation.LocalizeTag(page.Items[0]) != nil {
		t.Fatalf("mixed tag translated or lost: %+v %v", page, err)
	}
	manual, _ := service.GetAssetState(context.Background(), first)
	for _, tag := range manual.Tags {
		if tag.ID == carID && (tag.Source != curation.TagSourceManual || curation.LocalizeTag(tag) != nil) {
			t.Fatal("manual text translated")
		}
	}
	// Translation reads never write or advance a revision; the manual addition did.
	after, _ := service.GetAssetState(context.Background(), first)
	if after.Revision != manual.Revision || after.Revision <= beforeRevision {
		t.Fatal("read projection changed revision")
	}
	chinese, _, err := service.CreateTag(context.Background(), "汽车")
	if err != nil {
		t.Fatal(err)
	}
	after, _ = service.GetAssetState(context.Background(), second)
	if _, err := service.ReplaceAssetTags(context.Background(), second, after.Revision, []int64{chinese.ID}); err != nil {
		t.Fatal(err)
	}
	plan := automaticPlanForAsset(t, store, second, "car", "boots")
	requireAutomaticMerge(t, store, plan)
	current := requireTagSources(t, store, second, map[string]curation.TagSource{"汽车": curation.TagSourceManual, "boots": curation.TagSourceAI})
	bootsID := curationTagID(t, current, "boots")
	if _, err := service.ReplaceAssetTags(context.Background(), second, current.Revision, []int64{bootsID}); err != nil {
		t.Fatal(err)
	}
	plan.ModelIdentity = "model:new-policy-v2"
	if result := requireAutomaticMerge(t, store, plan); result.SkippedSuppressed != 1 {
		t.Fatalf("Chinese removal bypassed: %+v", result)
	}
	requireTagSources(t, store, second, map[string]curation.TagSource{"boots": curation.TagSourceAI})
}

func TestLocalizedSearchKeepsKeysetAndDoesNotGuessManualTags(t *testing.T) {
	store, _ := openTestStore(t)
	libraryID := seedBrowseCatalog(t, store)
	asset := catalogAssetID(t, store, "photo-10.jpg")
	service := newCurationServiceForStore(t, store, automaticTagTestTime)
	manual, _, err := service.CreateTag(context.Background(), "car")
	if err != nil {
		t.Fatal(err)
	}
	state, _ := service.GetAssetState(context.Background(), asset)
	if _, err := service.AddAssetTag(context.Background(), asset, state.Revision, manual.ID); err != nil {
		t.Fatal(err)
	}
	page, err := service.ListTags(context.Background(), curation.TagListRequest{Search: "汽车", LibraryID: libraryID})
	if err != nil || len(page.Items) != 0 {
		t.Fatalf("guessed manual alias: %+v %v", page, err)
	}
	requireAutomaticMerge(t, store, automaticPlanForAsset(t, store, asset, "blue_background", "black_background", "green_background"))
	request := curation.TagListRequest{Search: "背景", LibraryID: libraryID, Limit: 1}
	seen := map[int64]bool{}
	for {
		page, err := service.ListTags(context.Background(), request)
		if err != nil || len(page.Items) != 1 || seen[page.Items[0].ID] {
			t.Fatalf("unstable localized page: %+v %v", page, err)
		}
		seen[page.Items[0].ID] = true
		if page.NextCursor == "" {
			break
		}
		request.Cursor = page.NextCursor
	}
	if len(seen) != 3 {
		t.Fatalf("lost localized matches: %v", seen)
	}
}
