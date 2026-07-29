package thumbnail

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path"
	"strconv"

	"github.com/HappyQuQu/foliopath/internal/media"
)

type Variant string

const (
	VariantGrid                Variant = "grid"
	VariantStoryboard          Variant = "storyboard"
	GridTransformVersion               = 1
	StoryboardTransformVersion         = 1
)

var ErrInvalidDerivation = errors.New("invalid thumbnail derivation")

type Derivation struct {
	LibraryID         int64
	AssetID           int64
	Variant           Variant
	SourceFingerprint media.SourceFingerprint
	TransformVersion  int
}

func GridDerivation(
	libraryID, assetID int64,
	fingerprint media.SourceFingerprint,
) (Derivation, error) {
	value := Derivation{
		LibraryID: libraryID, AssetID: assetID, Variant: VariantGrid,
		SourceFingerprint: fingerprint, TransformVersion: GridTransformVersion,
	}
	if err := value.Validate(); err != nil {
		return Derivation{}, err
	}
	return value, nil
}

func StoryboardDerivation(
	libraryID, assetID int64,
	fingerprint media.SourceFingerprint,
) (Derivation, error) {
	value := Derivation{
		LibraryID: libraryID, AssetID: assetID, Variant: VariantStoryboard,
		SourceFingerprint: fingerprint, TransformVersion: StoryboardTransformVersion,
	}
	if err := value.Validate(); err != nil {
		return Derivation{}, err
	}
	return value, nil
}

func (value Derivation) Validate() error {
	if value.LibraryID <= 0 || value.AssetID <= 0 ||
		!value.SourceFingerprint.Valid() {
		return ErrInvalidDerivation
	}
	switch value.Variant {
	case VariantGrid:
		if value.TransformVersion != GridTransformVersion {
			return ErrInvalidDerivation
		}
	case VariantStoryboard:
		if value.TransformVersion != StoryboardTransformVersion {
			return ErrInvalidDerivation
		}
	default:
		return ErrInvalidDerivation
	}
	return nil
}

func (value Derivation) CacheRelativePath() (string, error) {
	if err := value.Validate(); err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(
		strconv.FormatInt(value.AssetID, 10) + "\x00" +
			string(value.Variant) + "\x00" +
			value.SourceFingerprint.String() + "\x00" +
			strconv.Itoa(value.TransformVersion),
	))
	encoded := hex.EncodeToString(sum[:])
	return path.Join(
		"libraries",
		"lib_"+strconv.FormatInt(value.LibraryID, 10),
		encoded[:2],
		encoded+".webp",
	), nil
}
