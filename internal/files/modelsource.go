package files

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"sync"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
)

const modelSourceReadBatch = 32

// ModelSource adapts one trusted fixed root, normally /models, to aimodel's
// candidate source. It never returns a path or filename to the public API.
type ModelSource struct {
	root    *Root
	mu      sync.RWMutex
	sources map[string]string
}

func NewModelSource(root *Root) (*ModelSource, error) {
	if root == nil {
		return nil, errors.New("files model source requires a root")
	}
	return &ModelSource{root: root, sources: map[string]string{}}, nil
}

func (source *ModelSource) ScanModelPackages(
	ctx context.Context,
	maxPackages int,
	maxFiles int,
	maxBytes int64,
) ([]aimodel.RawCandidate, bool, error) {
	if ctx == nil || maxPackages < 1 || maxFiles < 1 || maxBytes < 1 {
		return nil, false, fs.ErrInvalid
	}
	directory, err := source.root.OpenDir("")
	if err != nil {
		return nil, false, err
	}
	defer directory.Close()

	candidates := make([]aimodel.RawCandidate, 0, maxPackages)
	sources := make(map[string]string, maxPackages)
	truncated := false
	for {
		if err := ctx.Err(); err != nil {
			return nil, false, err
		}
		entries, readErr := directory.Read(modelSourceReadBatch)
		for _, entry := range entries {
			if !strings.HasSuffix(entry.Name(), ".foliomodel") {
				continue
			}
			if len(candidates) == maxPackages {
				truncated = true
				continue
			}
			candidate := source.readModelPackage(ctx, entry, maxFiles, maxBytes)
			candidates = append(candidates, candidate)
			if candidate.Failure == nil && candidate.SourceIdentity != "" {
				sources[candidate.SourceIdentity] = entry.Name()
			}
		}
		if errors.Is(readErr, io.EOF) {
			source.mu.Lock()
			source.sources = sources
			source.mu.Unlock()
			return candidates, truncated, nil
		}
		if readErr != nil {
			return nil, false, readErr
		}
	}
}

// OpenModelPackageFile reopens a file only through a source identity produced
// by the most recent scan. Callers cannot supply a path or filename from HTTP;
// fileName must come from the accepted built-in manifest.
func (source *ModelSource) OpenModelPackageFile(
	ctx context.Context,
	sourceIdentity string,
	fileName string,
) (io.ReadCloser, int64, error) {
	if ctx == nil || ctx.Err() != nil || fileName == "" || strings.ContainsAny(fileName, "/\\\x00") ||
		fileName == "." || fileName == ".." {
		return nil, 0, fs.ErrInvalid
	}
	packageName, err := source.resolveCurrentPackage(sourceIdentity)
	if err != nil {
		return nil, 0, err
	}
	file, err := source.root.Open(packageName + "/" + fileName)
	if err != nil {
		return nil, 0, err
	}
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o111 != 0 {
		_ = file.Close()
		return nil, 0, ErrNotRegular
	}
	return file, info.Size(), nil
}

// ValidateDirectModelSource requires the Linux anchored source filesystem to
// remain read-only and the package directory to retain its scanned identity.
func (source *ModelSource) ValidateDirectModelSource(ctx context.Context, sourceIdentity string) error {
	if ctx == nil || ctx.Err() != nil {
		return fs.ErrInvalid
	}
	readOnly, err := source.root.ReadOnly()
	if err != nil {
		return err
	}
	if !readOnly {
		return fs.ErrPermission
	}
	_, err = source.resolveCurrentPackage(sourceIdentity)
	return err
}

func (source *ModelSource) ValidateDirectModelPackage(ctx context.Context, sourceIdentity string, manifest aimodel.Manifest) error {
	if err := source.ValidateDirectModelSource(ctx, sourceIdentity); err != nil {
		return err
	}
	for _, expected := range manifest.Files {
		if err := ctx.Err(); err != nil {
			return err
		}
		file, size, err := source.OpenModelPackageFile(ctx, sourceIdentity, expected.Name)
		if err != nil {
			return err
		}
		digest := sha256.New()
		written, copyErr := io.Copy(digest, io.LimitReader(file, expected.Size+1))
		closeErr := file.Close()
		if copyErr != nil || closeErr != nil || size != expected.Size || written != expected.Size ||
			hex.EncodeToString(digest.Sum(nil)) != expected.SHA256 {
			return aimodel.ErrModelIncompatible
		}
	}
	return nil
}

func (source *ModelSource) OpenDirectRuntimeModelFile(ctx context.Context, identity, name string) (aimodel.RuntimeModelFile, error) {
	reader, size, err := source.OpenModelPackageFile(ctx, identity, name)
	if err != nil {
		return nil, err
	}
	file, ok := reader.(*os.File)
	if !ok {
		_ = reader.Close()
		return nil, ErrChanged
	}
	return newRuntimeModelFile(file, size)
}

func (source *ModelSource) resolveCurrentPackage(sourceIdentity string) (string, error) {
	source.mu.RLock()
	packageName, exists := source.sources[sourceIdentity]
	source.mu.RUnlock()
	if !exists {
		return "", ErrChanged
	}
	identity, err := source.root.CaptureAt(packageName)
	if err != nil {
		return "", err
	}
	current := modelSourceIdentity(identity, nil)
	if current != sourceIdentity {
		return "", ErrChanged
	}
	return packageName, nil
}

func (source *ModelSource) readModelPackage(
	ctx context.Context,
	entry fs.DirEntry,
	maxFiles int,
	maxBytes int64,
) aimodel.RawCandidate {
	if entry.Type()&fs.ModeSymlink != 0 || !entry.IsDir() {
		return aimodel.RawCandidate{Failure: ErrNotDirectory}
	}
	packageName := entry.Name()
	identity, err := source.root.CaptureAt(packageName)
	if err != nil {
		return aimodel.RawCandidate{Failure: err}
	}
	directory, err := source.root.OpenDir(packageName)
	if err != nil {
		return aimodel.RawCandidate{Failure: err}
	}
	defer directory.Close()

	var manifest []byte
	facts := make([]aimodel.FileFact, 0, maxFiles)
	var total int64
	seen := 0
	for {
		if err := ctx.Err(); err != nil {
			return aimodel.RawCandidate{Failure: err}
		}
		entries, readErr := directory.Read(modelSourceReadBatch)
		for _, child := range entries {
			seen++
			if seen > maxFiles+1 || child.Type()&fs.ModeSymlink != 0 || child.IsDir() {
				return aimodel.RawCandidate{Failure: aimodel.ErrModelIncompatible}
			}
			relative := packageName + "/" + child.Name()
			file, openErr := source.root.Open(relative)
			if openErr != nil {
				return aimodel.RawCandidate{Failure: openErr}
			}
			info, statErr := file.Stat()
			if statErr != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o111 != 0 {
				_ = file.Close()
				return aimodel.RawCandidate{Failure: ErrNotRegular}
			}
			if child.Name() == "manifest.json" {
				manifest, err = io.ReadAll(io.LimitReader(file, aimodel.MaxManifestBytes+1))
				_ = file.Close()
				if err != nil || len(manifest) > aimodel.MaxManifestBytes {
					return aimodel.RawCandidate{Failure: aimodel.ErrModelIncompatible}
				}
				continue
			}
			if info.Size() <= 0 || total > maxBytes-info.Size() {
				_ = file.Close()
				return aimodel.RawCandidate{Failure: aimodel.ErrModelIncompatible}
			}
			digest := sha256.New()
			written, copyErr := io.Copy(digest, io.LimitReader(file, info.Size()+1))
			closeErr := file.Close()
			if copyErr != nil || closeErr != nil || written != info.Size() {
				return aimodel.RawCandidate{Failure: aimodel.ErrModelIncompatible}
			}
			total += info.Size()
			facts = append(facts, aimodel.FileFact{
				Name: child.Name(), Size: info.Size(), SHA256: hex.EncodeToString(digest.Sum(nil)), Regular: true,
			})
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return aimodel.RawCandidate{Failure: readErr}
		}
	}
	if len(manifest) == 0 {
		return aimodel.RawCandidate{Failure: aimodel.ErrModelIncompatible}
	}
	device, inode, ok := identity.Key()
	identityDigest := modelSourceIdentityDigest(device, inode, ok, manifest)
	return aimodel.RawCandidate{
		Manifest:       manifest,
		Files:          facts,
		SourceIdentity: "source:" + hex.EncodeToString(identityDigest[:]),
	}
}

func modelSourceIdentity(identity Identity, manifest []byte) string {
	device, inode, ok := identity.Key()
	digest := modelSourceIdentityDigest(device, inode, ok, manifest)
	return "source:" + hex.EncodeToString(digest[:])
}

func modelSourceIdentityDigest(device, inode uint64, platform bool, manifest []byte) [sha256.Size]byte {
	identitySeed := fmt.Sprintf("fallback:%x", sha256.Sum256(manifest))
	if platform {
		identitySeed = fmt.Sprintf("unix:%d:%d", device, inode)
	}
	return sha256.Sum256([]byte(identitySeed))
}
