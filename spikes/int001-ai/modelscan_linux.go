//go:build linux

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"time"

	"golang.org/x/sys/unix"
)

type ModelScanReport struct {
	SchemaVersion    int                    `json:"schema_version"`
	GeneratedAt      string                 `json:"generated_at"`
	Root             string                 `json:"root"`
	ReadOnly         bool                   `json:"read_only"`
	Accepted         []AcceptedModelFile    `json:"accepted"`
	AcceptedPackages []AcceptedModelPackage `json:"accepted_packages"`
	Rejected         []RejectedModelFile    `json:"rejected"`
	authenticated    bool
}

type AcceptedModelFile struct {
	ID        string `json:"id"`
	Filename  string `json:"filename"`
	SizeBytes int64  `json:"size_bytes"`
	SHA256    string `json:"sha256"`
}
type AcceptedModelPackage struct {
	ID            string `json:"id"`
	Directory     string `json:"directory"`
	ArtifactCount int    `json:"artifact_count"`
	TotalBytes    int64  `json:"total_bytes"`
	PackageSHA256 string `json:"package_sha256"`
}
type RejectedModelFile struct {
	Filename string `json:"filename"`
	Reason   string `json:"reason"`
}

func ScanModels(root string, catalog ModelCatalog, requireReadOnly bool) (ModelScanReport, error) {
	report := ModelScanReport{SchemaVersion: 1, GeneratedAt: time.Now().UTC().Format(time.RFC3339), Root: root}
	rootFD, err := unix.Open(root, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return report, fmt.Errorf("open model root without following symlinks: %w", err)
	}
	rootFile := os.NewFile(uintptr(rootFD), root)
	defer rootFile.Close()
	var filesystem unix.Statfs_t
	if err := unix.Fstatfs(rootFD, &filesystem); err != nil {
		return report, fmt.Errorf("stat model filesystem: %w", err)
	}
	report.ReadOnly = filesystem.Flags&unix.ST_RDONLY != 0
	if requireReadOnly && !report.ReadOnly {
		return report, fmt.Errorf("model root is writable; mount it read-only")
	}
	entries, err := rootFile.ReadDir(-1)
	if err != nil {
		return report, fmt.Errorf("enumerate model root: %w", err)
	}
	allowedFiles := make(map[string]ModelEntry, len(catalog.Models))
	allowedPackages := make(map[string]ModelEntry, len(catalog.Models))
	for _, model := range catalog.Models {
		if model.Status != "approved" {
			continue
		}
		if catalog.SchemaVersion == 1 {
			allowedFiles[model.Filename] = model
		} else {
			allowedPackages[model.Directory] = model
		}
	}
	for _, entry := range entries {
		if model, known := allowedFiles[entry.Name()]; known {
			accepted, reason := scanSingleModelFile(rootFD, entry, model)
			if reason != "" {
				report.Rejected = append(report.Rejected, RejectedModelFile{entry.Name(), reason})
			} else {
				report.Accepted = append(report.Accepted, accepted)
			}
			continue
		}
		if model, known := allowedPackages[entry.Name()]; known {
			accepted, reason := scanModelPackage(rootFD, entry, model)
			if reason != "" {
				report.Rejected = append(report.Rejected, RejectedModelFile{entry.Name(), reason})
			} else {
				report.AcceptedPackages = append(report.AcceptedPackages, accepted)
			}
			continue
		}
		{
			report.Rejected = append(report.Rejected, RejectedModelFile{entry.Name(), "not present in approved catalog"})
		}
	}
	sort.Slice(report.Accepted, func(i, j int) bool { return report.Accepted[i].Filename < report.Accepted[j].Filename })
	sort.Slice(report.AcceptedPackages, func(i, j int) bool {
		return report.AcceptedPackages[i].Directory < report.AcceptedPackages[j].Directory
	})
	sort.Slice(report.Rejected, func(i, j int) bool { return report.Rejected[i].Filename < report.Rejected[j].Filename })
	report.authenticated = true
	return report, nil
}

// reconcileModelScan converts only kernel-anchored, exact-catalog package
// acceptances into authenticated observations. Missing or rejected catalogued
// generations become unavailable; the active pointer is never changed here.
func reconcileModelScan(ctx context.Context, store *activationStore, report ModelScanReport, catalog ModelCatalog, sourceKind string) error {
	if !report.authenticated {
		return errors.New("model scan report is not authenticated")
	}
	if sourceKind != "managed" && sourceKind != "direct" {
		return errors.New("model source kind is invalid")
	}
	accepted := make(map[string]AcceptedModelPackage, len(report.AcceptedPackages))
	for _, current := range report.AcceptedPackages {
		accepted[current.ID+"\x00"+current.Directory] = current
	}
	for _, model := range catalog.Models {
		if catalog.SchemaVersion != 2 || model.Status != "approved" {
			continue
		}
		key := model.ID + "\x00" + model.Directory
		current, exists := accepted[key]
		if !exists {
			if err := store.markGenerationUnavailable(ctx, model.ID, model.Directory, sourceKind); err != nil {
				return err
			}
			continue
		}
		if current.PackageSHA256 != model.PackageSHA256 {
			return errCatalogEquivocate
		}
		if err := store.reconcileVerifiedPackage(ctx, verifiedPackageObservation{
			ModelID: current.ID, Generation: current.Directory,
			PackageDigest: current.PackageSHA256, SourceKind: sourceKind, authenticated: true,
		}); err != nil {
			return err
		}
	}
	return nil
}

func markCatalogSourceUnavailable(ctx context.Context, store *activationStore, catalog ModelCatalog, sourceKind string) error {
	for _, model := range catalog.Models {
		if catalog.SchemaVersion == 2 && model.Status == "approved" {
			if err := store.markGenerationUnavailable(ctx, model.ID, model.Directory, sourceKind); err != nil {
				return err
			}
		}
	}
	return nil
}

func scanSingleModelFile(rootFD int, entry os.DirEntry, model ModelEntry) (AcceptedModelFile, string) {
	if entry.Type()&os.ModeSymlink != 0 || !entry.Type().IsRegular() {
		return AcceptedModelFile{}, "not a regular non-symlink file"
	}
	actualHash, size, reason := hashAnchoredFile(rootFD, entry.Name(), model.SizeBytes)
	if reason != "" {
		return AcceptedModelFile{}, reason
	}
	if actualHash != model.SHA256 {
		return AcceptedModelFile{}, "sha256 mismatch or read failure"
	}
	return AcceptedModelFile{model.ID, entry.Name(), size, actualHash}, ""
}

func scanModelPackage(rootFD int, entry os.DirEntry, model ModelEntry) (AcceptedModelPackage, string) {
	if entry.Type()&os.ModeSymlink != 0 || !entry.IsDir() {
		return AcceptedModelPackage{}, "not a non-symlink package directory"
	}
	packageFD, err := unix.Openat2(rootFD, entry.Name(), &unix.OpenHow{
		Flags:   unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC | unix.O_NOFOLLOW,
		Resolve: unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_XDEV,
	})
	if err != nil {
		return AcceptedModelPackage{}, "kernel-anchored open rejected package: " + err.Error()
	}
	packageFile := os.NewFile(uintptr(packageFD), entry.Name())
	defer packageFile.Close()
	entries, err := packageFile.ReadDir(129)
	if err != nil && err != io.EOF {
		return AcceptedModelPackage{}, "enumerate package: " + err.Error()
	}
	if len(entries) > 128 {
		return AcceptedModelPackage{}, "package exceeds 128 direct entries"
	}
	wanted := make(map[string]ModelArtifact, len(model.Artifacts))
	for _, artifact := range model.Artifacts {
		wanted[artifact.Filename] = artifact
	}
	seen := make(map[string]struct{}, len(entries))
	var totalBytes int64
	for _, artifactEntry := range entries {
		artifact, known := wanted[artifactEntry.Name()]
		if !known {
			return AcceptedModelPackage{}, "package contains undeclared entry " + artifactEntry.Name()
		}
		if artifactEntry.Type()&os.ModeSymlink != 0 || !artifactEntry.Type().IsRegular() {
			return AcceptedModelPackage{}, "artifact is not a regular non-symlink file: " + artifactEntry.Name()
		}
		actualHash, size, reason := hashAnchoredFile(packageFD, artifactEntry.Name(), artifact.SizeBytes)
		if reason != "" || actualHash != artifact.SHA256 {
			return AcceptedModelPackage{}, "artifact verification failed: " + artifactEntry.Name()
		}
		seen[artifactEntry.Name()] = struct{}{}
		totalBytes += size
	}
	if len(seen) != len(wanted) {
		return AcceptedModelPackage{}, "package is missing one or more declared artifacts"
	}
	return AcceptedModelPackage{
		ID:            model.ID,
		Directory:     model.Directory,
		ArtifactCount: len(seen),
		TotalBytes:    totalBytes,
		PackageSHA256: model.PackageSHA256,
	}, ""
}

func hashAnchoredFile(directoryFD int, filename string, expectedSize int64) (string, int64, string) {
	fd, err := unix.Openat2(directoryFD, filename, &unix.OpenHow{
		Flags:   unix.O_RDONLY | unix.O_CLOEXEC | unix.O_NOFOLLOW,
		Resolve: unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_XDEV,
	})
	if err != nil {
		return "", 0, "kernel-anchored open rejected entry: " + err.Error()
	}
	file := os.NewFile(uintptr(fd), filename)
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() != expectedSize {
		return "", 0, "type or declared size mismatch"
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, io.LimitReader(file, expectedSize+1)); err != nil {
		return "", 0, "sha256 mismatch or read failure"
	}
	return hex.EncodeToString(hash.Sum(nil)), info.Size(), ""
}
