package media

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

// ArchiveMember locates a member without interpreting its display path as a
// filesystem path. The containing file and member checksum bind its source.
type ArchiveMember struct {
	ArchivePath      string
	MemberPath       string
	ArchiveSizeBytes int64
	ArchiveMTimeNS   int64
	CRC32            uint32
}

type ArchiveContentSource interface {
	OpenArchiveContent(context.Context, string, ArchiveMember) (ContentFile, error)
}

func NewArchiveSourceFingerprint(sizeBytes, mtimeNS int64, member ArchiveMember) (SourceFingerprint, error) {
	if sizeBytes < 0 || member.ArchiveSizeBytes < 0 {
		return "", ErrInvalidSourceMetadata
	}
	digest := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%s\x00%d:%d:%d",
		member.ArchivePath, member.MemberPath, member.ArchiveSizeBytes, member.ArchiveMTimeNS, member.CRC32)))
	return SourceFingerprint(fmt.Sprintf("v1:z:%d:%d:%s", sizeBytes, mtimeNS,
		base64.RawURLEncoding.EncodeToString(digest[:16]))), nil
}

func archiveFingerprintFields(fingerprint SourceFingerprint) (int64, int64, bool) {
	parts := strings.Split(string(fingerprint), ":")
	if len(parts) != 5 || parts[0] != "v1" || parts[1] != "z" {
		return 0, 0, false
	}
	size, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil || size < 0 {
		return 0, 0, false
	}
	mtime, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		return 0, 0, false
	}
	if digest, err := base64.RawURLEncoding.DecodeString(parts[4]); err != nil || len(digest) != 16 {
		return 0, 0, false
	}
	return size, mtime, true
}

func (fingerprint SourceFingerprint) IsArchiveMember() bool {
	_, _, ok := archiveFingerprintFields(fingerprint)
	return ok
}
