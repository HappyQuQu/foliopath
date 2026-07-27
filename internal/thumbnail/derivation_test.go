package thumbnail

import (
	"strings"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/media"
)

func TestGridDerivationIsFingerprintAndVersionBoundWithoutPathMaterial(t *testing.T) {
	first, err := GridDerivation(7, 9, media.SourceFingerprint("v1:42:100"))
	if err != nil {
		t.Fatal(err)
	}
	firstPath, err := first.CacheRelativePath()
	if err != nil {
		t.Fatal(err)
	}
	second, err := GridDerivation(7, 9, media.SourceFingerprint("v1:42:101"))
	if err != nil {
		t.Fatal(err)
	}
	secondPath, err := second.CacheRelativePath()
	if err != nil {
		t.Fatal(err)
	}
	if firstPath == secondPath ||
		!strings.HasPrefix(firstPath, "libraries/lib_7/") ||
		strings.Contains(firstPath, "photo") ||
		!strings.HasSuffix(firstPath, ".webp") {
		t.Fatalf("cache paths = %q, %q", firstPath, secondPath)
	}
}
