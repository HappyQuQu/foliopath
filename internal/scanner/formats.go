package scanner

import (
	"path"
	"strings"
)

var supportedExtensions = map[string]struct {
	kind   AssetKind
	format MediaFormat
	mime   string
}{
	".jpg":  {AssetKindImage, MediaFormatJPEG, "image/jpeg"},
	".jpeg": {AssetKindImage, MediaFormatJPEG, "image/jpeg"},
	".png":  {AssetKindImage, MediaFormatPNG, "image/png"},
	".webp": {AssetKindImage, MediaFormatWebP, "image/webp"},
	".gif":  {AssetKindAnimated, MediaFormatGIF, "image/gif"},
	".mp4":  {AssetKindVideo, MediaFormatMP4, "video/mp4"},
	".mov":  {AssetKindVideo, MediaFormatMOV, "video/quicktime"},
	".mkv":  {AssetKindVideo, MediaFormatMKV, "video/x-matroska"},
}

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
	descriptor, ok := supportedExtensions[strings.ToLower(path.Ext(relativePath))]
	if !ok {
		return "", "", "", false
	}
	return descriptor.kind, descriptor.format, descriptor.mime, true
}

func IsSystemDirectory(name string) bool {
	_, ok := systemDirectories[strings.ToLower(name)]
	return ok
}
