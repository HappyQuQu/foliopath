//go:build libvips

package imagevips

import (
	"bytes"
	"context"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"runtime"
	"runtime/debug"

	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/davidbyttow/govips/v2/vips"
	"golang.org/x/image/webp"
)

// libvips is process-global. This native working-memory admission is shared by
// thumbnail, face and remote-search preprocessing, not one gate per library or
// feature. Existing bounded workers own waiters; no new goroutine is created.
var largeImageDecodeSlot = make(chan struct{}, NativeLargeImageConcurrency)

// Header probing only selects a resource lane. It does not accept/reject a
// format, change decoder settings or replace the canonical native validation.
// Unknown/oversized headers take the conservative lane. Restore the stream before
// returning, and never open a pathname or read the complete compressed source.
func largeImageSourceNeedsReclaim(source io.ReadSeeker, format media.Format) bool {
	// Do not peek through caller-owned cancellation/no-progress readers. Their
	// canonical read policy must see the very first read; unknown streams simply
	// use the conservative lane without adding reads or changing retry counters.
	switch source.(type) {
	case *os.File, *bytes.Reader:
	default:
		return true
	}
	position, err := source.Seek(0, io.SeekCurrent)
	if err != nil {
		return true
	}
	defer source.Seek(position, io.SeekStart)
	size, err := source.Seek(0, io.SeekEnd)
	if err != nil || largeImageDecodeNeedsReclaim(size, 0, 0) {
		return true
	}
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return true
	}
	header := io.LimitReader(source, 1<<20)
	var config image.Config
	switch format {
	case media.FormatJPEG:
		config, err = jpeg.DecodeConfig(header)
	case media.FormatPNG:
		config, err = png.DecodeConfig(header)
	case media.FormatGIF:
		config, err = gif.DecodeConfig(header)
	case media.FormatWebP:
		config, err = webp.DecodeConfig(header)
	default:
		return true
	}
	return err != nil || largeImageDecodeNeedsReclaim(size, config.Width, config.Height)
}

// The caller closes every ImageRef before finish runs. Native decoder allocations
// are outside Go's heap accounting, so a small Go heap alone cannot drive their
// reclamation. Do not change pixels, shrink policy, model identity or RAM limits.
func beginLargeImageDecode(ctx context.Context) (func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	select {
	case largeImageDecodeSlot <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if err := ctx.Err(); err != nil {
		<-largeImageDecodeSlot
		return nil, err
	}
	runtime.LockOSThread()
	reclaimNativeImageMemory()
	return func() {
		vips.ShutdownThread()
		reclaimNativeImageMemory()
		runtime.UnlockOSThread()
		<-largeImageDecodeSlot
	}, nil
}

func reclaimNativeImageMemory() {
	debug.FreeOSMemory()
	trimNativeImageMemory()
}
