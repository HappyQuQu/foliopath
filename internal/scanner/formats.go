package scanner

import (
	"strings"

	"github.com/HappyQuQu/foliopath/internal/media"
)

var systemDirectories = map[string]struct{}{
	"@eadir":       {},
	".thumbnails":  {},
	"__macosx":     {},
	"#recycle":     {},
	"@recycle":     {},
	".trash":       {},
	".trashes":     {},
	"$recycle.bin": {},
}

func ClassifyPath(relativePath string) (AssetKind, MediaFormat, string, bool) {
	kind, format, mime, ok := media.ClassifyPath(relativePath)
	if !ok {
		return "", "", "", false
	}
	return AssetKind(kind), MediaFormat(format), mime, true
}

func IsSystemDirectory(name string) bool {
	_, ok := systemDirectories[strings.ToLower(name)]
	return ok
}
