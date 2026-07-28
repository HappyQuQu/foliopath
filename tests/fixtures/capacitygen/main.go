package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

const onePixelPNG = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII="

type metrics struct {
	Root            string `json:"root"`
	Directories     int    `json:"directories"`
	Assets          int    `json:"assets"`
	BytesPerAsset   int    `json:"bytesPerAsset"`
	GenerationNanos int64  `json:"generationNanos"`
}

func main() {
	if len(os.Args) != 4 {
		fatal(errors.New("usage: capacitygen ROOT DIRECTORY_COUNT ASSET_COUNT"))
	}
	directoryCount, err := positiveInteger(os.Args[2])
	if err != nil {
		fatal(fmt.Errorf("directory count: %w", err))
	}
	assetCount, err := positiveInteger(os.Args[3])
	if err != nil {
		fatal(fmt.Errorf("asset count: %w", err))
	}
	png, err := base64.StdEncoding.DecodeString(onePixelPNG)
	if err != nil {
		fatal(fmt.Errorf("decode embedded PNG: %w", err))
	}

	started := time.Now()
	paths, err := createDirectories(os.Args[1], directoryCount)
	if err != nil {
		fatal(err)
	}
	for index := 0; index < assetCount; index++ {
		path := filepath.Join(paths[index%len(paths)], fmt.Sprintf("asset-%06d.png", index))
		if err := os.WriteFile(path, png, 0o444); err != nil {
			fatal(fmt.Errorf("write asset %d: %w", index, err))
		}
	}

	result := metrics{
		Root:            os.Args[1],
		Directories:     directoryCount,
		Assets:          assetCount,
		BytesPerAsset:   len(png),
		GenerationNanos: time.Since(started).Nanoseconds(),
	}
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		fatal(fmt.Errorf("encode metrics: %w", err))
	}
}

func positiveInteger(value string) (int, error) {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%q must be a positive integer", value)
	}
	return parsed, nil
}

func createDirectories(root string, count int) ([]string, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create root: %w", err)
	}
	groupCount := min(count, 100)
	paths := make([]string, 0, count)
	for index := 0; index < groupCount; index++ {
		path := filepath.Join(root, fmt.Sprintf("group-%03d", index))
		if err := os.Mkdir(path, 0o755); err != nil {
			return nil, fmt.Errorf("create group %d: %w", index, err)
		}
		paths = append(paths, path)
	}
	for index := groupCount; index < count; index++ {
		parent := paths[index%groupCount]
		path := filepath.Join(parent, fmt.Sprintf("directory-%05d", index))
		if err := os.Mkdir(path, 0o755); err != nil {
			return nil, fmt.Errorf("create directory %d: %w", index, err)
		}
		paths = append(paths, path)
	}
	return paths, nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
