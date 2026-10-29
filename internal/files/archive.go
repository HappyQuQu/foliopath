package files

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
	"unicode/utf8"

	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/pathpolicy"
)

const (
	MaxArchiveEntries        = 10_000
	MaxArchiveDirectoryBytes = 8 << 20
	MaxArchiveMemberBytes    = 64 << 20
	MaxArchiveExpandedBytes  = uint64(4) << 30
	MaxArchiveDepth          = 32
)

var (
	ErrArchiveEncrypted   = errors.New("encrypted archive")
	ErrArchiveUnsupported = errors.New("unsupported archive")
	ErrArchiveLimit       = errors.New("archive limit exceeded")
	ErrArchiveInvalid     = errors.New("invalid archive")
)

type ArchiveEntry struct {
	Path        string
	IsDirectory bool
	SizeBytes   int64
	MTimeNS     int64
	CRC32       uint32
}

type ArchiveManifest struct {
	Entries  []ArchiveEntry
	members  map[string]*zip.File
	sevenZip bool
}

// ReadArchive validates a bounded central directory before exposing any member.
// The caller owns file; this function never opens a filesystem path.
func ReadArchive(ctx context.Context, file *os.File) (*ArchiveManifest, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, ErrArchiveInvalid
	}
	var signature [6]byte
	if _, err := file.ReadAt(signature[:], 0); err != nil {
		return nil, ErrArchiveInvalid
	}
	var manifest *ArchiveManifest
	if signature == [6]byte{'7', 'z', 0xbc, 0xaf, 0x27, 0x1c} {
		manifest, err = readSevenZipArchive(ctx, file)
	} else {
		if err := validateArchiveDirectory(file, info.Size()); err != nil {
			return nil, err
		}
		reader, openErr := zip.NewReader(file, info.Size())
		if openErr != nil {
			return nil, errors.Join(ErrArchiveInvalid, openErr)
		}
		manifest, err = archiveManifest(ctx, reader)
	}
	if err != nil {
		return nil, err
	}

	after, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if after.Size() != info.Size() || after.ModTime() != info.ModTime() {
		return nil, media.ErrContentSourceChanged
	}
	return manifest, nil
}

func validateArchiveDirectory(file *os.File, size int64) error {
	const endSize = 22
	if size < endSize {
		return ErrArchiveInvalid
	}
	// The final record bounds allocations made by archive/zip. ZIP64 and split
	// volumes are outside the initial format contract.
	tail := make([]byte, min(size, 65_535+endSize))
	if _, err := file.ReadAt(tail, size-int64(len(tail))); err != nil {
		return err
	}
	for offset := len(tail) - endSize; offset >= 0; offset-- {
		if !bytes.Equal(tail[offset:offset+4], []byte{'P', 'K', 5, 6}) {
			continue
		}
		end := tail[offset:]
		if int(binary.LittleEndian.Uint16(end[20:22]))+endSize != len(end) {
			continue
		}
		count := binary.LittleEndian.Uint16(end[10:12])
		directorySize := binary.LittleEndian.Uint32(end[12:16])
		directoryOffset := binary.LittleEndian.Uint32(end[16:20])
		if binary.LittleEndian.Uint16(end[4:6]) != 0 || binary.LittleEndian.Uint16(end[6:8]) != 0 ||
			binary.LittleEndian.Uint16(end[8:10]) != count || count == 0xffff ||
			directorySize == 0xffffffff || directoryOffset == 0xffffffff {
			return ErrArchiveUnsupported
		}
		if count > MaxArchiveEntries || directorySize > MaxArchiveDirectoryBytes {
			return ErrArchiveLimit
		}
		endOffset := size - int64(len(end))
		if int64(directoryOffset)+int64(directorySize) != endOffset {
			return ErrArchiveInvalid
		}
		return nil
	}
	return ErrArchiveInvalid
}

func archiveManifest(ctx context.Context, reader *zip.Reader) (*ArchiveManifest, error) {
	if len(reader.File) > MaxArchiveEntries {
		return nil, ErrArchiveLimit
	}
	manifest := &ArchiveManifest{
		Entries: make([]ArchiveEntry, 0, len(reader.File)),
		members: make(map[string]*zip.File, len(reader.File)),
	}
	var expanded uint64
	for _, member := range reader.File {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		entry, err := archiveEntry(member)
		if err != nil {
			return nil, err
		}
		if _, duplicate := manifest.members[entry.Path]; duplicate {
			return nil, ErrArchiveInvalid
		}
		if member.UncompressedSize64 > MaxArchiveExpandedBytes-expanded {
			return nil, ErrArchiveLimit
		}
		expanded += member.UncompressedSize64
		manifest.members[entry.Path] = member
		manifest.Entries = append(manifest.Entries, entry)
	}
	for _, entry := range manifest.Entries {
		for parent := path.Dir(entry.Path); parent != "."; parent = path.Dir(parent) {
			if ancestor, ok := manifest.members[parent]; ok && !ancestor.FileInfo().IsDir() {
				return nil, ErrArchiveInvalid
			}
		}
	}
	return manifest, nil
}

func archiveEntry(member *zip.File) (ArchiveEntry, error) {
	if member.Flags&1 != 0 || member.Flags&0x40 != 0 {
		return ArchiveEntry{}, ErrArchiveEncrypted
	}
	if member.Method != zip.Store && member.Method != zip.Deflate {
		return ArchiveEntry{}, ErrArchiveUnsupported
	}
	name := strings.TrimSuffix(member.Name, "/")
	info := member.FileInfo()
	if info.Mode()&fs.ModeSymlink != 0 || (!info.IsDir() && !info.Mode().IsRegular()) {
		return ArchiveEntry{}, ErrArchiveInvalid
	}
	if info.IsDir() != strings.HasSuffix(member.Name, "/") {
		return ArchiveEntry{}, ErrArchiveInvalid
	}
	return validateArchiveEntry(ArchiveEntry{
		Path: name, IsDirectory: info.IsDir(), SizeBytes: int64(member.UncompressedSize64),
		MTimeNS: info.ModTime().UnixNano(), CRC32: member.CRC32,
	})
}

func validateArchiveEntry(entry ArchiveEntry) (ArchiveEntry, error) {
	name := entry.Path
	if !utf8.ValidString(name) {
		return ArchiveEntry{}, ErrArchiveUnsupported
	}
	if _, err := pathpolicy.Normalize(name); err != nil || name == "" {
		return ArchiveEntry{}, ErrArchiveInvalid
	}
	if strings.Count(name, "/") >= MaxArchiveDepth || len(name) > 4096 {
		return ArchiveEntry{}, ErrArchiveLimit
	}
	kind, _, _, supported := media.ClassifyPath(name)
	if entry.SizeBytes < 0 || uint64(entry.SizeBytes) > MaxArchiveExpandedBytes ||
		(supported && kind != media.KindVideo && entry.SizeBytes > MaxArchiveMemberBytes) {
		return ArchiveEntry{}, ErrArchiveLimit
	}
	return entry, nil
}

type archiveContextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (reader archiveContextReader) Read(buffer []byte) (int, error) {
	if err := reader.ctx.Err(); err != nil {
		return 0, err
	}
	return reader.reader.Read(buffer)
}
