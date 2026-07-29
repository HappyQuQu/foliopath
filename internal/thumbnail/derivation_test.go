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

func TestStoryboardDerivationHasIndependentVariantAndVersionIdentity(t *testing.T) {
	fingerprint := media.SourceFingerprint("v1:42:100")
	grid, err := GridDerivation(7, 9, fingerprint)
	if err != nil {
		t.Fatal(err)
	}
	storyboard, err := StoryboardDerivation(7, 9, fingerprint)
	if err != nil {
		t.Fatal(err)
	}
	gridPath, err := grid.CacheRelativePath()
	if err != nil {
		t.Fatal(err)
	}
	storyboardPath, err := storyboard.CacheRelativePath()
	if err != nil {
		t.Fatal(err)
	}
	if gridPath == storyboardPath || storyboard.Variant != VariantStoryboard {
		t.Fatalf("grid path = %q, storyboard = %#v at %q", gridPath, storyboard, storyboardPath)
	}
}

func TestDerivationRejectsVariantVersionMismatch(t *testing.T) {
	value := Derivation{
		LibraryID:         1,
		AssetID:           2,
		Variant:           VariantStoryboard,
		SourceFingerprint: media.SourceFingerprint("v1:42:100"),
		TransformVersion:  GridTransformVersion + 1,
	}
	if GridTransformVersion+1 == StoryboardTransformVersion {
		value.TransformVersion++
	}
	if err := value.Validate(); err == nil {
		t.Fatal("variant/version mismatch unexpectedly accepted")
	}
}
