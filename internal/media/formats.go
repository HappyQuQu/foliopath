// Package media owns the supported media format contract and media processing
// semantics. Scanner and API status consume this registry instead of keeping
// independent allowlists.
package media

import (
	"path"
	"strings"
)

type Kind string

const (
	KindImage    Kind = "image"
	KindAnimated Kind = "animated"
	KindVideo    Kind = "video"
)

type Format string

const (
	FormatJPEG Format = "jpeg"
	FormatPNG  Format = "png"
	FormatWebP Format = "webp"
	FormatGIF  Format = "gif"
	FormatMP4  Format = "mp4"
	FormatMOV  Format = "mov"
	FormatMKV  Format = "mkv"
	FormatAVI  Format = "avi"
)

type descriptor struct {
	kind   Kind
	format Format
	mime   string
}

var supportedExtensions = map[string]descriptor{
	".jpg":  {KindImage, FormatJPEG, "image/jpeg"},
	".jpeg": {KindImage, FormatJPEG, "image/jpeg"},
	".png":  {KindImage, FormatPNG, "image/png"},
	".webp": {KindImage, FormatWebP, "image/webp"},
	".gif":  {KindAnimated, FormatGIF, "image/gif"},
	".mp4":  {KindVideo, FormatMP4, "video/mp4"},
	".mov":  {KindVideo, FormatMOV, "video/quicktime"},
	".mkv":  {KindVideo, FormatMKV, "video/x-matroska"},
	".avi":  {KindVideo, FormatAVI, "video/x-msvideo"},
}

var (
	imageMIMETypes = []string{"image/jpeg", "image/png", "image/webp", "image/gif"}
	videoMIMETypes = []string{
		"video/mp4",
		"video/quicktime",
		"video/x-matroska",
		"video/x-msvideo",
	}
)

func ClassifyPath(relativePath string) (Kind, Format, string, bool) {
	descriptor, ok := supportedExtensions[strings.ToLower(path.Ext(relativePath))]
	if !ok {
		return "", "", "", false
	}
	return descriptor.kind, descriptor.format, descriptor.mime, true
}

func ImageMIMETypes() []string {
	return append([]string(nil), imageMIMETypes...)
}

func VideoMIMETypes() []string {
	return append([]string(nil), videoMIMETypes...)
}
