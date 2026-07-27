// Package cachefs atomically publishes reconstructible thumbnail bytes below
// the configured application cache root.
package cachefs

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/HappyQuQu/foliopath/internal/thumbnail"
)

type Publisher struct {
	root string
}

func New(root string) (*Publisher, error) {
	if root == "" || !filepath.IsAbs(root) {
		return nil, errors.New("thumbnail cache root must be absolute")
	}
	return &Publisher{root: filepath.Clean(root)}, nil
}

func (publisher *Publisher) Publish(
	ctx context.Context,
	derivation thumbnail.Derivation,
	value []byte,
) (thumbnail.Published, error) {
	if err := ctx.Err(); err != nil {
		return thumbnail.Published{}, err
	}
	relative, err := derivation.CacheRelativePath()
	if err != nil || len(value) == 0 {
		return thumbnail.Published{}, thumbnail.ErrInvalidDerivation
	}
	destination, err := publisher.resolve(relative)
	if err != nil {
		return thumbnail.Published{}, err
	}
	directory := filepath.Dir(destination)
	if err := os.MkdirAll(directory, 0o750); err != nil {
		return thumbnail.Published{}, fmt.Errorf("prepare thumbnail cache directory: %w", err)
	}
	temp, err := os.CreateTemp(directory, ".thumbnail-*.tmp")
	if err != nil {
		return thumbnail.Published{}, fmt.Errorf("create thumbnail temporary file: %w", err)
	}
	tempName := temp.Name()
	committed := false
	defer func() {
		if !committed {
			_ = os.Remove(tempName)
		}
	}()
	if err := temp.Chmod(0o640); err != nil {
		_ = temp.Close()
		return thumbnail.Published{}, fmt.Errorf("set thumbnail permissions: %w", err)
	}
	if _, err := temp.Write(value); err != nil {
		_ = temp.Close()
		return thumbnail.Published{}, fmt.Errorf("write thumbnail: %w", err)
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return thumbnail.Published{}, fmt.Errorf("sync thumbnail: %w", err)
	}
	if err := temp.Close(); err != nil {
		return thumbnail.Published{}, fmt.Errorf("close thumbnail: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return thumbnail.Published{}, err
	}
	if err := os.Rename(tempName, destination); err != nil {
		return thumbnail.Published{}, fmt.Errorf("publish thumbnail: %w", err)
	}
	committed = true
	return thumbnail.Published{
		CacheRelativePath: relative,
		ByteSize:          int64(len(value)),
	}, nil
}

func (publisher *Publisher) AvailableBytes(ctx context.Context) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	target := publisher.root
	var stats syscall.Statfs_t
	for {
		err := syscall.Statfs(target, &stats)
		if err == nil {
			break
		}
		if !errors.Is(err, syscall.ENOENT) {
			return 0, fmt.Errorf("inspect thumbnail cache filesystem: %w", err)
		}
		parent := filepath.Dir(target)
		if parent == target {
			return 0, fmt.Errorf("inspect thumbnail cache filesystem: %w", err)
		}
		target = parent
	}
	available := uint64(stats.Bavail) * uint64(stats.Bsize)
	if available > uint64(^uint64(0)>>1) {
		return int64(^uint64(0) >> 1), nil
	}
	return int64(available), nil
}

func (publisher *Publisher) Remove(ctx context.Context, relative string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	filename, err := publisher.resolve(relative)
	if err != nil {
		return err
	}
	if err := os.Remove(filename); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove thumbnail cache entry: %w", err)
	}
	return nil
}

func (publisher *Publisher) Open(
	ctx context.Context,
	relative string,
) (thumbnail.CacheContent, error) {
	if err := ctx.Err(); err != nil {
		return thumbnail.CacheContent{}, err
	}
	if _, err := publisher.resolve(relative); err != nil {
		return thumbnail.CacheContent{}, err
	}
	root, err := os.OpenRoot(publisher.root)
	if errors.Is(err, os.ErrNotExist) {
		return thumbnail.CacheContent{}, thumbnail.ErrCacheEntryMissing
	}
	if err != nil {
		return thumbnail.CacheContent{}, fmt.Errorf("open thumbnail cache root: %w", err)
	}
	defer root.Close()
	file, err := root.Open(relative)
	if errors.Is(err, os.ErrNotExist) {
		return thumbnail.CacheContent{}, thumbnail.ErrCacheEntryMissing
	}
	if err != nil {
		return thumbnail.CacheContent{}, fmt.Errorf("open thumbnail cache entry: %w", err)
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return thumbnail.CacheContent{}, fmt.Errorf("inspect thumbnail cache entry: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 {
		_ = file.Close()
		return thumbnail.CacheContent{}, thumbnail.ErrCacheEntryMissing
	}
	return thumbnail.CacheContent{Reader: file, ByteSize: info.Size()}, nil
}

func (publisher *Publisher) resolve(relative string) (string, error) {
	if relative == "" || filepath.IsAbs(relative) ||
		filepath.ToSlash(filepath.Clean(relative)) != relative {
		return "", thumbnail.ErrInvalidDerivation
	}
	destination := filepath.Join(publisher.root, filepath.FromSlash(relative))
	if !strings.HasPrefix(destination, publisher.root+string(filepath.Separator)) {
		return "", thumbnail.ErrInvalidDerivation
	}
	return destination, nil
}
