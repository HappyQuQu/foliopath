package files

import (
	"bytes"
	"context"
	"errors"
	"hash/crc32"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/HappyQuQu/foliopath/internal/media"
)

const sevenZipMemoryBytes = 256 << 20

var sevenZipSlots = make(chan struct{}, 2)

func readSevenZipArchive(ctx context.Context, file *os.File) (*ArchiveManifest, error) {
	var output bytes.Buffer
	if err := runSevenZip(ctx, file, []string{"l", "-slt", "-ba"}, &archiveLimitedWriter{writer: &output, remaining: MaxArchiveDirectoryBytes}); err != nil {
		return nil, err
	}
	return parseSevenZipManifest(output.String())
}

func parseSevenZipManifest(output string) (*ArchiveManifest, error) {
	manifest := &ArchiveManifest{sevenZip: true}
	entries := make(map[string]ArchiveEntry)
	var expanded uint64
	for _, record := range strings.Split(strings.Trim(output, "\r\n"), "\n\n") {
		if record == "" {
			continue
		}
		properties := make(map[string]string)
		for _, line := range strings.Split(record, "\n") {
			key, value, ok := strings.Cut(line, " = ")
			if !ok {
				return nil, ErrArchiveInvalid
			}
			if _, duplicate := properties[key]; duplicate {
				return nil, ErrArchiveInvalid
			}
			properties[key] = value
		}
		entry, err := sevenZipEntry(properties)
		if err != nil {
			return nil, err
		}
		if _, duplicate := entries[entry.Path]; duplicate {
			return nil, ErrArchiveInvalid
		}
		if uint64(entry.SizeBytes) > MaxArchiveExpandedBytes-expanded {
			return nil, ErrArchiveLimit
		}
		expanded += uint64(entry.SizeBytes)
		entries[entry.Path] = entry
		manifest.Entries = append(manifest.Entries, entry)
		if len(entries) > MaxArchiveEntries {
			return nil, ErrArchiveLimit
		}
	}
	for _, entry := range manifest.Entries {
		for parent := path.Dir(entry.Path); parent != "."; parent = path.Dir(parent) {
			if ancestor, ok := entries[parent]; ok && !ancestor.IsDirectory {
				return nil, ErrArchiveInvalid
			}
		}
	}
	return manifest, nil
}

func sevenZipEntry(properties map[string]string) (ArchiveEntry, error) {
	if properties["Encrypted"] == "+" {
		return ArchiveEntry{}, ErrArchiveEncrypted
	}
	if properties["Encrypted"] != "-" {
		return ArchiveEntry{}, ErrArchiveInvalid
	}
	attributes := strings.Fields(properties["Attributes"])
	if len(attributes) == 0 {
		return ArchiveEntry{}, ErrArchiveInvalid
	}
	directory := strings.Contains(attributes[0], "D")
	if len(attributes) > 1 && ((directory && attributes[1][0] != 'd') || (!directory && attributes[1][0] != '-')) {
		return ArchiveEntry{}, ErrArchiveInvalid
	}
	if properties["Symbolic Link"] != "" || properties["Hard Link"] != "" || properties["Alternate Stream"] == "+" {
		return ArchiveEntry{}, ErrArchiveInvalid
	}
	size, err := strconv.ParseInt(properties["Size"], 10, 64)
	if err != nil {
		return ArchiveEntry{}, ErrArchiveLimit
	}
	var checksum uint64
	if !directory {
		if len(properties["CRC"]) != 8 {
			return ArchiveEntry{}, ErrArchiveUnsupported
		}
		checksum, err = strconv.ParseUint(properties["CRC"], 16, 32)
		if err != nil {
			return ArchiveEntry{}, ErrArchiveInvalid
		}
	}
	var modified time.Time
	if value := properties["Modified"]; value != "" {
		modified, err = time.Parse("2006-01-02 15:04:05.999999999", value)
		if err != nil {
			return ArchiveEntry{}, ErrArchiveInvalid
		}
	}
	mtime := int64(0)
	if !modified.IsZero() {
		mtime = modified.UnixNano()
	}
	return validateArchiveEntry(ArchiveEntry{Path: properties["Path"], IsDirectory: directory, SizeBytes: size, MTimeNS: mtime, CRC32: uint32(checksum)})
}

func (store *ArchiveStore) openSevenZipContent(ctx context.Context, source *os.File, member media.ArchiveMember, manifest *ArchiveManifest) (*archiveContentFile, error) {
	var entry ArchiveEntry
	found := false
	for _, candidate := range manifest.Entries {
		if candidate.Path == member.MemberPath {
			entry, found = candidate, true
			break
		}
	}
	if !found || entry.CRC32 != member.CRC32 {
		return nil, media.ErrContentSourceChanged
	}
	kind, _, _, supported := media.ClassifyPath(entry.Path)
	if !supported || kind == media.KindVideo || entry.IsDirectory {
		return nil, ErrArchiveUnsupported
	}
	available, _, err := filesystemSpace(store.root)
	if err != nil {
		return nil, err
	}
	if available < store.safeFreeBytes+archiveOpenLimit*MaxArchiveMemberBytes {
		return nil, ErrArchiveLimit
	}
	temporary, err := os.CreateTemp(store.root, "member-")
	if err != nil {
		return nil, err
	}
	success := false
	defer func() {
		if !success {
			_ = temporary.Close()
			_ = os.Remove(temporary.Name())
		}
	}()
	hash := crc32.NewIEEE()
	limited := &archiveLimitedWriter{writer: io.MultiWriter(temporary, hash), remaining: MaxArchiveMemberBytes}
	if err := runSevenZip(ctx, source, []string{"x", "-so", "-spd", "-y"}, limited, member.MemberPath); err != nil {
		return nil, err
	}
	if MaxArchiveMemberBytes-limited.remaining != entry.SizeBytes || hash.Sum32() != entry.CRC32 {
		return nil, ErrArchiveInvalid
	}
	if err := verifyArchiveSource(source, member); err != nil {
		return nil, err
	}
	if _, err := temporary.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	success = true
	return &archiveContentFile{file: temporary, source: source, member: member, info: sevenZipFileInfo{entry}}, nil
}

// The decoder sees only the already anchored descriptor; member arguments are
// literal selections (-spd), never filesystem output paths or shell input.
func runSevenZip(ctx context.Context, source *os.File, options []string, output io.Writer, members ...string) error {
	if runtime.GOOS != "linux" {
		return ErrArchiveUnsupported
	}
	bounded, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	select {
	case sevenZipSlots <- struct{}{}:
	case <-bounded.Done():
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return ErrArchiveLimit
	}
	defer func() { <-sevenZipSlots }()
	args := []string{"--as=" + strconv.Itoa(sevenZipMemoryBytes), "--cpu=60", "--nofile=32", "--", "/usr/lib/7zip/7z"}
	args = append(args, options...)
	args = append(args, "-t7z", "-sccUTF-8", "-bd", "-mmt=1", "-p", "--", "/proc/self/fd/3")
	args = append(args, members...)
	command := exec.CommandContext(bounded, "/usr/bin/prlimit", args...)
	command.ExtraFiles = []*os.File{source}
	command.Env = []string{"LANG=C.UTF-8", "TZ=UTC"}
	command.WaitDelay = time.Second
	stdout := &archiveCommandWriter{writer: output, cancel: cancel}
	command.Stdout = stdout
	var diagnostics bytes.Buffer
	stderr := &archiveCommandWriter{writer: &archiveLimitedWriter{writer: &diagnostics, remaining: 16 << 10}, cancel: cancel}
	command.Stderr = stderr
	err := command.Run()
	if stdout.err != nil {
		return stdout.err
	}
	if stderr.err != nil {
		return stderr.err
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if bounded.Err() != nil {
		return ErrArchiveLimit
	}
	if err == nil {
		return nil
	}
	if strings.Contains(diagnostics.String(), "Wrong password") {
		return ErrArchiveEncrypted
	}
	var exit *exec.ExitError
	if errors.Is(err, ErrArchiveLimit) || strings.Contains(diagnostics.String(), "Can't allocate required memory!") ||
		(errors.As(err, &exit) && exit.ExitCode() == 8) {
		return ErrArchiveLimit
	}
	return ErrArchiveInvalid
}

type archiveLimitedWriter struct {
	writer    io.Writer
	remaining int64
}

func (writer *archiveLimitedWriter) Write(data []byte) (int, error) {
	if int64(len(data)) > writer.remaining {
		return 0, ErrArchiveLimit
	}
	n, err := writer.writer.Write(data)
	writer.remaining -= int64(n)
	return n, err
}

type sevenZipFileInfo struct{ entry ArchiveEntry }

func (info sevenZipFileInfo) Name() string       { return path.Base(info.entry.Path) }
func (info sevenZipFileInfo) Size() int64        { return info.entry.SizeBytes }
func (info sevenZipFileInfo) Mode() fs.FileMode  { return 0o444 }
func (info sevenZipFileInfo) ModTime() time.Time { return time.Unix(0, info.entry.MTimeNS) }
func (info sevenZipFileInfo) IsDir() bool        { return false }
func (info sevenZipFileInfo) Sys() any           { return nil }

// Stop the child immediately when an output budget or destination write fails.
type archiveCommandWriter struct {
	writer io.Writer
	cancel context.CancelFunc
	err    error
}

func (writer *archiveCommandWriter) Write(data []byte) (int, error) {
	n, err := writer.writer.Write(data)
	if err != nil {
		writer.err = err
		writer.cancel()
	}
	return n, err
}
