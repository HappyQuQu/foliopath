//go:build libvips

// int001-vips-input generates bounded semantic inputs through FolioPath's
// production image adapter. It is an isolated S0 tool and must only receive an
// explicitly authorized fixture directory, never a user's media library.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"

	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/media/imagevips"
)

var safeJPEGName = regexp.MustCompile(`^[0-9]+\.jpg$`)

type manifest struct {
	SchemaVersion  int              `json:"schema_version"`
	DatasetID      string           `json:"dataset_id"`
	EvidenceClass  string           `json:"evidence_class"`
	InputPipeline  string           `json:"input_pipeline"`
	InputTransform map[string]any   `json:"input_transform,omitempty"`
	LegalBasis     string           `json:"legal_basis"`
	SourceAPI      string           `json:"source_api"`
	Items          []item           `json:"items"`
	Queries        []map[string]any `json:"queries"`
	Caveats        []string         `json:"caveats"`
}

type item struct {
	ID       string         `json:"id"`
	Filename string         `json:"filename"`
	SHA256   string         `json:"sha256"`
	Source   source         `json:"source"`
	Prepared map[string]any `json:"prepared,omitempty"`
}

type source struct {
	PageID            int    `json:"page_id"`
	Title             string `json:"title"`
	RevisionTimestamp string `json:"revision_timestamp"`
	OriginalSHA1      string `json:"original_sha1"`
	Bytes             int64  `json:"bytes"`
	MIME              string `json:"mime"`
	Width             int    `json:"width"`
	Height            int    `json:"height"`
	URL               string `json:"url"`
	PageURL           string `json:"page_url"`
	Artist            string `json:"artist"`
	License           string `json:"license"`
	LicenseURL        string `json:"license_url"`
}

func main() {
	sourceManifest := flag.String("source-manifest", "", "verified public pilot manifest")
	sourceImages := flag.String("source-images", "", "verified public pilot image directory")
	outputManifest := flag.String("output-manifest", "", "derived manifest path")
	outputImages := flag.String("output-images", "", "derived image directory")
	executionMode := flag.String("execution-mode", "", "native or qemu")
	flag.Parse()
	if *sourceManifest == "" || *sourceImages == "" || *outputManifest == "" || *outputImages == "" {
		fatal(errors.New("all four path arguments are required"))
	}
	if *executionMode != "native" && *executionMode != "qemu" {
		fatal(errors.New("execution-mode must be native or qemu"))
	}
	encoded, err := os.ReadFile(*sourceManifest)
	if err != nil {
		fatal(err)
	}
	var value manifest
	if err := json.Unmarshal(encoded, &value); err != nil {
		fatal(err)
	}
	if value.SchemaVersion != 1 || value.EvidenceClass != "public-license-pilot" ||
		value.LegalBasis != "public-license" || len(value.Items) == 0 {
		fatal(errors.New("expected a non-empty schema v1 public-license pilot"))
	}
	if err := os.MkdirAll(*outputImages, 0o700); err != nil {
		fatal(err)
	}
	vipsRuntime := imagevips.NewRuntime()
	if err := vipsRuntime.Start(); err != nil {
		fatal(err)
	}
	defer vipsRuntime.Shutdown()
	processor := imagevips.New()
	expected := make(map[string]struct{}, len(value.Items))
	var outputBytes int64
	for index := range value.Items {
		current := &value.Items[index]
		if current.ID == "" || !safeJPEGName.MatchString(current.Filename) ||
			current.Source.MIME != "image/jpeg" || current.Source.PageID <= 0 {
			fatal(fmt.Errorf("invalid pilot item %q", current.ID))
		}
		sourcePath := filepath.Join(*sourceImages, current.Filename)
		actualDigest, err := fileSHA256(sourcePath)
		if err != nil {
			fatal(err)
		}
		if actualDigest != current.SHA256 {
			fatal(fmt.Errorf("source SHA-256 mismatch: %s", current.ID))
		}
		file, err := os.Open(sourcePath)
		if err != nil {
			fatal(err)
		}
		result, processErr := processor.Process(context.Background(), file, media.FormatJPEG)
		closeErr := file.Close()
		if processErr != nil {
			fatal(fmt.Errorf("process %s: %w", current.ID, processErr))
		}
		if closeErr != nil {
			fatal(closeErr)
		}
		if len(result.Thumbnail.Bytes) < 12 || string(result.Thumbnail.Bytes[:4]) != "RIFF" ||
			result.Thumbnail.Width > media.GridThumbnailWidth ||
			result.Thumbnail.Height > media.GridThumbnailHeight {
			fatal(fmt.Errorf("invalid bounded thumbnail: %s", current.ID))
		}
		filename := fmt.Sprintf("%d.webp", current.Source.PageID)
		expected[filename] = struct{}{}
		target := filepath.Join(*outputImages, filename)
		if err := atomicWrite(target, result.Thumbnail.Bytes); err != nil {
			fatal(err)
		}
		current.Filename = filename
		thumbnailDigest := sha256.Sum256(result.Thumbnail.Bytes)
		current.SHA256 = hex.EncodeToString(thumbnailDigest[:])
		current.Prepared = map[string]any{
			"bytes": len(result.Thumbnail.Bytes), "width": result.Thumbnail.Width,
			"height": result.Thumbnail.Height, "format": "image/webp",
		}
		outputBytes += int64(len(result.Thumbnail.Bytes))
	}
	entries, err := os.ReadDir(*outputImages)
	if err != nil {
		fatal(err)
	}
	var extras []string
	for _, entry := range entries {
		if _, ok := expected[entry.Name()]; !ok {
			extras = append(extras, entry.Name())
		}
	}
	if len(extras) != 0 {
		sort.Strings(extras)
		fatal(fmt.Errorf("unexpected output entries: %v", extras))
	}
	value.InputPipeline = fmt.Sprintf(
		"foliopath-govips-grid-thumbnail-linux-%s-%s-development",
		runtime.GOARCH, *executionMode,
	)
	value.InputTransform = map[string]any{
		"output_maximum_dimension": media.GridThumbnailWidth,
		"output_format":            "webp", "output_quality": media.GridWebPQuality,
		"production_adapter": true, "native_execution": *executionMode == "native",
	}
	value.Caveats = append(value.Caveats,
		fmt.Sprintf(
			"Inputs were prepared by the production govips adapter in Linux/%s %s mode; dual-architecture evidence requires a matching native run on the other architecture.",
			runtime.GOARCH, *executionMode,
		),
	)
	encoded, err = json.MarshalIndent(value, "", "  ")
	if err != nil {
		fatal(err)
	}
	encoded = append(encoded, '\n')
	if err := atomicWrite(*outputManifest, encoded); err != nil {
		fatal(err)
	}
	summary, _ := json.MarshalIndent(map[string]any{
		"schema_version": 1, "items": len(value.Items), "prepared_bytes": outputBytes,
		"input_pipeline": value.InputPipeline, "production_adapter": true,
		"native_execution": *executionMode == "native",
	}, "", "  ")
	fmt.Println(string(summary))
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func atomicWrite(path string, content []byte) error {
	temporary := path + ".part"
	file, err := os.OpenFile(temporary, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	remove := true
	defer func() {
		if remove {
			_ = os.Remove(temporary)
		}
	}()
	if _, err := file.Write(content); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporary, path); err != nil {
		return err
	}
	remove = false
	return nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
