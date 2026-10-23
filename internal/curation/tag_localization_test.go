package curation

import "testing"

func TestTagLocalizationRespectsEvidenceAndManualNames(t *testing.T) {
	for _, tag := range []Tag{{Name: "car"}, {Name: "car", Source: TagSourceManual}, {Name: "Car", Source: TagSourceAI}, {Name: "自定义", Source: TagSourceAI}} {
		if LocalizeTag(tag) != nil {
			t.Fatalf("guessed localization for %+v", tag)
		}
	}
	for _, tag := range []Tag{{Name: "car", Source: TagSourceAI}, {Name: "car", AutomaticOnly: true}} {
		label := LocalizeTag(tag)
		if label == nil || label.Key != "car" || label.ZhCN != "汽车" || label.En != "Car" {
			t.Fatalf("missing metadata: %+v", label)
		}
		if tag.Name != "car" {
			t.Fatal("changed stored name")
		}
	}
}

func TestCategoryAliasesAndBoundedSearch(t *testing.T) {
	for _, input := range []string{"car", "汽车", "车", "车辆", " CAR "} {
		if AutomaticCategoryKey(input) != "car" {
			t.Fatalf("alias %s", input)
		}
	}
	if AutomaticCategoryKey("我喜欢汽车") != "" {
		t.Fatal("free text must stay literal")
	}
	for _, input := range []string{"汽车", "blue background", "发饰"} {
		if len(MatchingAutomaticCategories(input)) == 0 {
			t.Fatalf("missing search %s", input)
		}
	}
	if len(MatchingAutomaticCategories("")) != 0 || len(MatchingAutomaticCategories("不存在的标签")) != 0 {
		t.Fatal("unexpected expansion")
	}
	for key := range tagCategories {
		if AutomaticCategoryKey(key) != key || len(AutomaticTagProtectionKeys(key)) < 2 {
			t.Fatal(key)
		}
	}
}
