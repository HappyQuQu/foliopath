package files

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/HappyQuQu/foliopath/internal/media"
)

const archiveOpenLimit = 2

var archiveOpenSlots = make(chan struct{}, archiveOpenLimit)

// ArchiveStore owns request-scoped seekable copies. Admission remains held
// until Close so concurrent HTTP responses also count against the disk budget.
type ArchiveStore struct {
	root          string
	safeFreeBytes int64
}

func NewArchiveStore(root string, safeFreeBytes int64) (*ArchiveStore, error) {
	if !filepath.IsAbs(root) || safeFreeBytes < 0 {
		return nil, fs.ErrInvalid
	}
	return &ArchiveStore{root: root, safeFreeBytes: safeFreeBytes}, nil
}

func (store *ArchiveStore) Start() error {
	if err := os.MkdirAll(store.root, 0o700); err != nil {
		return err
	}
	entries, err := os.ReadDir(store.root)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "member-") && !entry.IsDir() {
			if err := os.Remove(filepath.Join(store.root, entry.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

// OpenMember takes ownership of source on every return path. source must have
// been opened through Root; member names are never used for temporary paths.
func (store *ArchiveStore) OpenMember(ctx context.Context, source *os.File, member media.ArchiveMember) (media.ContentFile, error) {
	select {
	case archiveOpenSlots <- struct{}{}:
	case <-ctx.Done():
		_ = source.Close()
		return nil, ctx.Err()
	}
	var result *archiveContentFile
	defer func() {
		if result == nil {
			_ = source.Close()
			<-archiveOpenSlots
		}
	}()
	if err := verifyArchiveSource(source, member); err != nil {
		return nil, err
	}
	manifest, err := ReadArchive(ctx, source)
	if err != nil {
		return nil, err
	}
	entry, ok := manifest.members[member.MemberPath]
	if !ok || entry.CRC32 != member.CRC32 {
		return nil, media.ErrContentSourceChanged
	}
	kind, _, _, supported := media.ClassifyPath(member.MemberPath)
	if !supported || kind == media.KindVideo || entry.FileInfo().IsDir() {
		return nil, ErrArchiveUnsupported
	}
	available, _, err := filesystemSpace(store.root)
	if err != nil {
		return nil, err
	}
	if available < store.safeFreeBytes+archiveOpenLimit*MaxArchiveMemberBytes {
		return nil, ErrArchiveLimit
	}
	reader, err := entry.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	temporary, err := os.CreateTemp(store.root, "member-")
	if err != nil {
		return nil, err
	}
	defer func() {
		if result == nil {
			_ = temporary.Close()
			_ = os.Remove(temporary.Name())
		}
	}()
	copyCtx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	written, err := io.Copy(temporary, io.LimitReader(archiveContextReader{copyCtx, reader}, MaxArchiveMemberBytes+1))
	if err != nil {
		return nil, err
	}
	if written > MaxArchiveMemberBytes {
		return nil, ErrArchiveLimit
	}
	if written != int64(entry.UncompressedSize64) {
		return nil, ErrArchiveInvalid
	}
	if err := verifyArchiveSource(source, member); err != nil {
		return nil, err
	}
	if _, err := temporary.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	result = &archiveContentFile{file: temporary, source: source, member: member, info: entry.FileInfo()}
	return result, nil
}

func verifyArchiveSource(source *os.File, member media.ArchiveMember) error {
	info, err := source.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() != member.ArchiveSizeBytes || info.ModTime().UnixNano() != member.ArchiveMTimeNS {
		return media.ErrContentSourceChanged
	}
	return nil
}

type archiveContentFile struct {
	file   *os.File
	source *os.File
	member media.ArchiveMember
	info   fs.FileInfo
	once   sync.Once
	err    error
}

func (file *archiveContentFile) Read(buffer []byte) (int, error) { return file.file.Read(buffer) }
func (file *archiveContentFile) Seek(offset int64, whence int) (int64, error) {
	return file.file.Seek(offset, whence)
}

func (file *archiveContentFile) Stat() (fs.FileInfo, error) {
	if err := verifyArchiveSource(file.source, file.member); err != nil {
		return nil, err
	}
	return file.info, nil
}

func (file *archiveContentFile) Close() error {
	file.once.Do(func() {
		file.err = errors.Join(file.file.Close(), file.source.Close(), os.Remove(file.file.Name()))
		<-archiveOpenSlots
	})
	return file.err
}
