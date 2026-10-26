package catalog

import (
	"path"
	"strings"
)

// LiteralSources uses catalog search profile v1. The path source denotes the
// directory/relative-path evidence only when it adds a match beyond the basename.
func LiteralSources(terms []string, name, relative string) []string {
	if len(terms) == 0 {
		return nil
	}
	name = SearchTextKey(name)
	relativeKey := SearchTextKey(relative)
	parent := path.Dir(relative)
	if parent == "." {
		parent = ""
	}
	parent = SearchTextKey(parent)
	n, p := false, false
	for _, term := range terms {
		inName, inPath := strings.Contains(name, term), strings.Contains(parent, term)
		// A copied relative path may cross the parent/basename separator.
		// Preserve that literal search without labelling a basename hit twice.
		if !inName && !inPath {
			inPath = strings.Contains(relativeKey, term)
			if !inPath {
				return nil
			}
		}
		n = n || inName
		p = p || inPath
	}
	var sources []string
	if n {
		sources = append(sources, "filename")
	}
	if p {
		sources = append(sources, "path")
	}
	return sources
}
