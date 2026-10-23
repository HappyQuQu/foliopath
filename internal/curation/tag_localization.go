package curation

import (
	_ "embed"
	"encoding/json"
	"slices"
	"strings"
)

// tagCategoriesJSON is the curation-owned catalog. Inference embeds the same
// source file in its independently versioned output policy; it is not user input.
//
//go:embed tag_categories.json
var tagCategoriesJSON []byte

var tagCategories, tagAliases = loadTagCategories()

type TagLocalization struct {
	Key  string `json:"key"`
	ZhCN string `json:"zhCN"`
	En   string `json:"en"`
}

func loadTagCategories() (map[string][]string, map[string]string) {
	var categories map[string][]string
	if err := json.Unmarshal(tagCategoriesJSON, &categories); err != nil {
		panic("invalid embedded tag categories")
	}
	aliases := make(map[string]string)
	for key, values := range categories {
		if len(values) < 3 || len(key) > MaxTagRunes {
			panic("invalid tag category")
		}
		for _, value := range append([]string{key}, append(append([]string{}, values[:2]...), values[3:]...)...) {
			_, normalized, err := NormalizeTagName(value)
			if err != nil || aliases[normalized] != "" && aliases[normalized] != key {
				panic("invalid or ambiguous tag alias")
			}
			aliases[normalized] = key
		}
	}
	return categories, aliases
}

// LocalizeTag never guesses from the text of a manual/unknown association. A
// mixed-source global tag also stays literal; per-image AI tags can localize.
func LocalizeTag(tag Tag) *TagLocalization {
	if tag.Source != TagSourceAI && (tag.Source != "" || !tag.AutomaticOnly) {
		return nil
	}
	values, found := tagCategories[tag.Name]
	if !found {
		return nil
	}
	return &TagLocalization{Key: tag.Name, ZhCN: values[0], En: values[1]}
}

// AutomaticCategoryKey only recognizes explicitly reviewed exact aliases. It is
// for protecting manual curation, never for rewriting user names.
func AutomaticCategoryKey(name string) string {
	_, key, err := NormalizeTagName(name)
	if err != nil {
		return ""
	}
	return tagAliases[key]
}

func AutomaticTagProtectionKeys(name string) []string {
	key := AutomaticCategoryKey(name)
	if key == "" {
		return []string{name}
	}
	keys := []string{}
	for alias, category := range tagAliases {
		if category == key {
			keys = append(keys, alias)
		}
	}
	slices.Sort(keys)
	return keys
}

// MatchingAutomaticCategories expands a bounded search before SQL pagination;
// filtering a fetched page on the client would drop valid matches and cursors.
func MatchingAutomaticCategories(searchKey string) []string {
	keys := []string{}
	if searchKey == "" {
		return keys
	}
	for alias, category := range tagAliases {
		if strings.Contains(alias, searchKey) {
			keys = append(keys, category)
		}
	}
	slices.Sort(keys)
	return slices.Compact(keys)
}
