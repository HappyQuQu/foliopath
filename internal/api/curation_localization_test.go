package api

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/curation"
)

func TestTagWireLocalizationIsOptionalAndNeverRewritesNames(t *testing.T) {
	for _, source := range []curation.TagSource{"", curation.TagSourceManual, curation.TagSourceAI} {
		tag := curation.Tag{ID: 5, Name: "car", Source: source, CreatedAt: time.Now(), UpdatedAt: time.Now()}
		data, err := json.Marshal(tagWire(tag))
		if err != nil {
			t.Fatal(err)
		}
		var value map[string]any
		if err := json.Unmarshal(data, &value); err != nil {
			t.Fatal(err)
		}
		if value["name"] != "car" || value["id"] != "tag_5" {
			t.Fatal("wire renamed curation")
		}
		localization, found := value["localization"]
		if found != (source == curation.TagSourceAI) {
			t.Fatalf("guessed localization for %s", source)
		}
		if found && localization.(map[string]any)["zhCN"] != "汽车" {
			t.Fatal(localization)
		}
	}
}
