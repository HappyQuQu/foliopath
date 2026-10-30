package scanner

import (
	"path"
	"strings"

	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/pathpolicy"
)

func IsArchivePath(relativePath string) bool {
	extension := path.Ext(relativePath)
	return strings.EqualFold(extension, ".zip") || strings.EqualFold(extension, ".7z")
}

var systemDirectories = map[string]struct{}{
	pathpolicy.TrashDirectory: {},
	"@eadir":                  {},
	".thumbnails":             {},
	"__macosx":                {},
	"#recycle":                {},
	"@recycle":                {},
	".trash":                  {},
	".trashes":                {},
	"$recycle.bin":            {},
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
