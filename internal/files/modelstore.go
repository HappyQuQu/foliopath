package files

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
)

const (
	DefaultManagedModelQuota = int64(8 << 30)
	minimumModelReserve      = int64(1 << 30)
)

type ManagedModelStore struct {
	mu         sync.Mutex
	root       string
	quota      int64
	spaceProbe func(string) (int64, int64, error)
}

type ManagedModelReconcileReport struct {
	RemovedStaging     int
	KnownFinals        int
	UnknownEntries     int
	Truncated          bool
	FinalContentHashes []string
}

func NewManagedModelStore(root string, quota int64) (*ManagedModelStore, error) {
	if root == "" || !filepath.IsAbs(root) {
		return nil, fs.ErrInvalid
	}
	if quota == 0 {
		quota = DefaultManagedModelQuota
	}
	if quota < 1<<30 || quota > 64<<30 {
		return nil, fs.ErrInvalid
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, err
	}
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode()&fs.ModeSymlink != 0 {
		return nil, ErrInvalidRoot
	}
	return &ManagedModelStore{root: filepath.Clean(root), quota: quota, spaceProbe: filesystemSpace}, nil
}

// Reconcile removes only interrupted staging directories owned by this store.
// Unknown and final entries are reported but never deleted or activated.
func (store *ManagedModelStore) Reconcile(ctx context.Context) (ManagedModelReconcileReport, error) {
	if ctx == nil {
		return ManagedModelReconcileReport{}, fs.ErrInvalid
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	entries, err := os.ReadDir(store.root)
	if err != nil {
		return ManagedModelReconcileReport{}, err
	}
	const maxEntries = 256
	report := ManagedModelReconcileReport{}
	for index, entry := range entries {
		if err := ctx.Err(); err != nil {
			return ManagedModelReconcileReport{}, err
		}
		if index == maxEntries {
			report.Truncated = true
			break
		}
		name := entry.Name()
		switch {
		case strings.HasPrefix(name, ".partial-") && entry.IsDir() && entry.Type()&fs.ModeSymlink == 0:
			if err := os.RemoveAll(filepath.Join(store.root, name)); err != nil {
				return ManagedModelReconcileReport{}, err
			}
			report.RemovedStaging++
		case isManagedFinalName(name) && entry.IsDir() && entry.Type()&fs.ModeSymlink == 0:
			report.KnownFinals++
			report.FinalContentHashes = append(report.FinalContentHashes, strings.TrimSuffix(name, ".foliomodel"))
		default:
			report.UnknownEntries++
		}
	}
	if report.RemovedStaging > 0 {
		if err := syncDirectory(store.root); err != nil {
			return ManagedModelReconcileReport{}, err
		}
	}
	return report, nil
}

func isManagedFinalName(name string) bool {
	const suffix = ".foliomodel"
	if !strings.HasSuffix(name, suffix) {
		return false
	}
	hash := strings.TrimSuffix(name, suffix)
	if len(hash) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(hash)
	return err == nil && hash == strings.ToLower(hash)
}

func (store *ManagedModelStore) PublishModelPackage(
	ctx context.Context,
	model aimodel.VerifiedPackage,
	manifest aimodel.Manifest,
	opener aimodel.PackageOpener,
) (string, error) {
	if ctx == nil || opener == nil || aimodel.ValidatePackage(model) != nil {
		return "", aimodel.ErrInvalidModel
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil || len(manifestBytes) > aimodel.MaxManifestBytes {
		return "", aimodel.ErrModelIncompatible
	}
	store.mu.Lock()
	defer store.mu.Unlock()

	finalName := model.ContentHash + ".foliomodel"
	finalPath := filepath.Join(store.root, finalName)
	if info, statErr := os.Lstat(finalPath); statErr == nil {
		if !info.IsDir() || info.Mode()&fs.ModeSymlink != 0 {
			return "", ErrChanged
		}
		if err := verifyManagedPackage(finalPath, manifestBytes, manifest); err != nil {
			return "", err
		}
		return "managed:" + model.ContentHash, nil
	} else if !errors.Is(statErr, fs.ErrNotExist) {
		return "", statErr
	}
	usage, err := directoryUsage(store.root, store.quota)
	if err != nil {
		return "", err
	}
	required := model.PackageSizeByte + int64(len(manifestBytes))
	if usage > store.quota-required {
		return "", aimodel.ErrInsufficientSpace
	}
	available, total, err := store.spaceProbe(store.root)
	if err != nil {
		return "", err
	}
	reserve := total / 10
	if reserve < minimumModelReserve {
		reserve = minimumModelReserve
	}
	if available < required || available-required < reserve {
		return "", aimodel.ErrInsufficientSpace
	}

	staging, err := os.MkdirTemp(store.root, ".partial-")
	if err != nil {
		return "", managedStoreWriteError(err)
	}
	keep := false
	defer func() {
		if !keep {
			_ = os.RemoveAll(staging)
		}
	}()
	if err := writeManagedFile(filepath.Join(staging, "manifest.json"), manifestBytes); err != nil {
		return "", managedStoreWriteError(err)
	}
	for _, expected := range manifest.Files {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		reader, size, err := opener(ctx, expected.Name)
		if err != nil {
			return "", err
		}
		if size != expected.Size {
			_ = reader.Close()
			return "", aimodel.ErrModelIncompatible
		}
		err = copyManagedFile(filepath.Join(staging, expected.Name), reader, expected)
		closeErr := reader.Close()
		if err != nil {
			return "", err
		}
		if closeErr != nil {
			return "", closeErr
		}
	}
	if err := syncDirectory(staging); err != nil {
		return "", managedStoreWriteError(err)
	}
	if err := renameNoReplace(staging, finalPath); err != nil {
		if errors.Is(err, fs.ErrExist) {
			if verifyErr := verifyManagedPackage(finalPath, manifestBytes, manifest); verifyErr != nil {
				return "", verifyErr
			}
			return "managed:" + model.ContentHash, nil
		}
		return "", managedStoreWriteError(err)
	}
	keep = true
	if err := syncDirectory(store.root); err != nil {
		return "", managedStoreWriteError(err)
	}
	return "managed:" + model.ContentHash, nil
}

func managedStoreWriteError(err error) error {
	if isStorageExhausted(err) {
		return errors.Join(aimodel.ErrInsufficientSpace, err)
	}
	return err
}

func (store *ManagedModelStore) ValidateManagedModelPackage(ctx context.Context, model aimodel.Model, manifest aimodel.Manifest) error {
	if ctx == nil || ctx.Err() != nil || aimodel.ValidateModel(model) != nil || model.StorageMode != aimodel.StorageManaged ||
		model.SourceIdentity != "managed:"+model.Package.ContentHash {
		return aimodel.ErrInvalidModel
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return aimodel.ErrModelIncompatible
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	return verifyManagedPackage(filepath.Join(store.root, model.Package.ContentHash+".foliomodel"), manifestBytes, manifest)
}

func (store *ManagedModelStore) OpenManagedModelPackageFile(ctx context.Context, model aimodel.Model, name string) (io.ReadCloser, int64, error) {
	if ctx == nil || ctx.Err() != nil || aimodel.ValidateModel(model) != nil || model.StorageMode != aimodel.StorageManaged ||
		model.SourceIdentity != "managed:"+model.Package.ContentHash || name == "" || filepath.Base(name) != name ||
		strings.ContainsAny(name, "/\\\x00") || name == "." || name == ".." {
		return nil, 0, aimodel.ErrInvalidModel
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	root := filepath.Join(store.root, model.Package.ContentHash+".foliomodel")
	rootInfo, err := os.Lstat(root)
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode()&fs.ModeSymlink != 0 {
		return nil, 0, ErrChanged
	}
	path := filepath.Join(root, name)
	before, err := os.Lstat(path)
	if err != nil || !before.Mode().IsRegular() || before.Mode()&fs.ModeSymlink != 0 || before.Mode().Perm()&0o111 != 0 {
		return nil, 0, ErrNotRegular
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	after, err := file.Stat()
	if err != nil || !os.SameFile(before, after) {
		_ = file.Close()
		return nil, 0, ErrChanged
	}
	return file, after.Size(), nil
}

func (store *ManagedModelStore) OpenManagedRuntimeModelFile(ctx context.Context, model aimodel.Model, name string) (aimodel.RuntimeModelFile, error) {
	reader, size, err := store.OpenManagedModelPackageFile(ctx, model, name)
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

func verifyManagedPackage(root string, manifestBytes []byte, manifest aimodel.Manifest) error {
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != len(manifest.Files)+1 {
		return aimodel.ErrModelIncompatible
	}
	expected := make(map[string]aimodel.ManifestFile, len(manifest.Files))
	for _, file := range manifest.Files {
		expected[file.Name] = file
	}
	for _, entry := range entries {
		if entry.Type()&fs.ModeSymlink != 0 || entry.IsDir() {
			return aimodel.ErrModelIncompatible
		}
		name := entry.Name()
		if name == "manifest.json" {
			content, readErr := os.ReadFile(filepath.Join(root, name))
			if readErr != nil || !bytes.Equal(content, manifestBytes) {
				return aimodel.ErrModelIncompatible
			}
			continue
		}
		file, exists := expected[name]
		if !exists {
			return aimodel.ErrModelIncompatible
		}
		opened, openErr := os.Open(filepath.Join(root, name))
		if openErr != nil {
			return openErr
		}
		info, statErr := opened.Stat()
		digest := sha256.New()
		written, copyErr := io.Copy(digest, io.LimitReader(opened, file.Size+1))
		closeErr := opened.Close()
		if statErr != nil || copyErr != nil || closeErr != nil || !info.Mode().IsRegular() ||
			info.Mode().Perm()&0o111 != 0 || written != file.Size ||
			hex.EncodeToString(digest.Sum(nil)) != file.SHA256 {
			return aimodel.ErrModelIncompatible
		}
	}
	return nil
}

func writeManagedFile(name string, content []byte) error {
	file, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := file.Write(content); err != nil {
		_ = file.Close()
		return err
	}
	return errors.Join(file.Sync(), file.Close())
}

func copyManagedFile(name string, reader io.Reader, expected aimodel.ManifestFile) error {
	file, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	digest := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(managedTargetWriter{writer: file}, digest), io.LimitReader(reader, expected.Size+1))
	if copyErr != nil {
		return errors.Join(copyErr, managedStoreWriteError(file.Close()))
	}
	if written != expected.Size || hex.EncodeToString(digest.Sum(nil)) != expected.SHA256 {
		if closeErr := file.Close(); closeErr != nil {
			return managedStoreWriteError(closeErr)
		}
		return aimodel.ErrModelIncompatible
	}
	return managedStoreWriteError(errors.Join(file.Sync(), file.Close()))
}

type managedTargetWriter struct{ writer io.Writer }

func (writer managedTargetWriter) Write(content []byte) (int, error) {
	written, err := writer.writer.Write(content)
	return written, managedStoreWriteError(err)
}

func directoryUsage(root string, limit int64) (int64, error) {
	var total int64
	err := filepath.WalkDir(root, func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return ErrSymlink
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			return ErrSpecialFile
		}
		if total > limit-info.Size() {
			return aimodel.ErrInsufficientSpace
		}
		total += info.Size()
		return nil
	})
	return total, err
}

func syncDirectory(name string) error {
	directory, err := os.Open(name)
	if err != nil {
		return err
	}
	return errors.Join(directory.Sync(), directory.Close())
}
