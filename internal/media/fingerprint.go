package media

import (
	"errors"
	"strconv"
	"strings"
)

const sourceFingerprintPrefix = "v1:"

var ErrInvalidSourceMetadata = errors.New("invalid source metadata")

// SourceFingerprint is the versioned identity of source bytes used to
// invalidate derived metadata and cache entries. It deliberately describes
// source state, not content equality or cross-path deduplication.
type SourceFingerprint string

func NewSourceFingerprint(sizeBytes, mtimeNS int64) (SourceFingerprint, error) {
	if sizeBytes < 0 {
		return "", ErrInvalidSourceMetadata
	}
	return SourceFingerprint(
		sourceFingerprintPrefix +
			strconv.FormatInt(sizeBytes, 10) + ":" +
			strconv.FormatInt(mtimeNS, 10),
	), nil
}

func (fingerprint SourceFingerprint) String() string {
	return string(fingerprint)
}

func (fingerprint SourceFingerprint) Matches(sizeBytes, mtimeNS int64) bool {
	expected, err := NewSourceFingerprint(sizeBytes, mtimeNS)
	return err == nil && fingerprint == expected
}

func (fingerprint SourceFingerprint) Valid() bool {
	parts := strings.Split(string(fingerprint), ":")
	if len(parts) != 3 || parts[0]+":" != sourceFingerprintPrefix {
		return false
	}
	size, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || size < 0 {
		return false
	}
	if _, err := strconv.ParseInt(parts[2], 10, 64); err != nil {
		return false
	}
	return true
}
