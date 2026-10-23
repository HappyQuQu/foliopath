// Opt-in diagnostic: reads only the explicitly mounted /library through files.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/HappyQuQu/foliopath/internal/files"
	_ "golang.org/x/image/webp"
)

type candidate struct{ name, key, kind string }
type entry struct {
	ID     string `json:"id"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
	Kind   string `json:"kind"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "sample preparation failed (no source paths emitted)")
		os.Exit(1)
	}
}

func run() error {
	if os.Getenv("FOLIOPATH_ALLOW_SAMPLE_READ") != "100-readonly" {
		return errors.New("explicit opt in required")
	}
	r, err := files.OpenRoot("/library")
	if err != nil {
		return err
	}
	defer r.Close()
	ro, err := r.ReadOnly()
	if err != nil || !ro {
		return errors.New("read-only mount required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	if os.Getenv("FOLIOPATH_SAMPLE_AUDIT") == "1" {
		count, over16, over100, invalid := 0, 0, 0, 0
		var largest int64
		err = r.Walk(ctx, "", func(name string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}
			switch strings.ToLower(path.Ext(name)) {
			case ".jpg", ".jpeg", ".png", ".webp", ".gif":
			default:
				return nil
			}
			count++
			if count > 100 {
				return errors.New("audit sample limit")
			}
			f, e := r.Open(name)
			if e != nil {
				return e
			}
			config, _, e := image.DecodeConfig(io.LimitReader(f, 64<<20))
			f.Close()
			if e != nil {
				invalid++
				return nil
			}
			pixels := int64(config.Width) * int64(config.Height)
			largest = max(largest, pixels)
			if pixels > 16000000 {
				over16++
			}
			if pixels > 100000000 {
				over100++
			}
			return nil
		})
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(map[string]any{"images": count, "over16MP": over16, "over100MP": over100, "maxPixels": largest, "invalidHeaders": invalid})
	}
	pools := map[string][]candidate{"image": {}, "video": {}}
	visited, skipped := 0, 0
	err = r.Walk(ctx, "", func(name string, d fs.DirEntry, walkErr error) error {
		visited++
		if visited > 20000 {
			return fs.SkipAll
		}
		if walkErr != nil {
			skipped++
			return nil
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(path.Ext(name))
		kind := ""
		switch ext {
		case ".jpg", ".jpeg", ".png", ".webp", ".gif":
			kind = "image"
		case ".mp4", ".mov", ".mkv":
			kind = "video"
		default:
			return nil
		}
		info, e := d.Info()
		if e != nil {
			return e
		}
		limit := int64(64 << 20)
		if kind == "video" {
			limit = 256 << 20
		}
		if !info.Mode().IsRegular() || info.Size() > limit || info.Size() == 0 {
			skipped++
			return nil
		}
		h := sha256.Sum256([]byte("p4-sample-v1/" + name))
		pool := append(pools[kind], candidate{name, hex.EncodeToString(h[:]), kind})
		sort.Slice(pool, func(i, j int) bool { return pool[i].key < pool[j].key })
		if len(pool) > 100 {
			pool = pool[:100]
		}
		pools[kind] = pool
		return nil
	})
	if err != nil {
		return err
	}
	vc := min(10, len(pools["video"]))
	ic := min(100-vc, len(pools["image"]))
	vc = min(100-ic, len(pools["video"]))
	selected := append(pools["image"][:ic], pools["video"][:vc]...)
	if len(selected) != 100 {
		return errors.New("insufficient bounded candidates")
	}
	if err = os.Mkdir("/output/samples", 0700); err != nil {
		return err
	}
	rows := []entry{}
	var total int64
	for i, c := range selected {
		if err = ctx.Err(); err != nil {
			return err
		}
		in, e := r.Open(c.name)
		if e != nil {
			return e
		}
		before, e := in.Stat()
		if e != nil {
			in.Close()
			return e
		}
		maxSize := int64(64 << 20)
		if c.kind == "video" {
			maxSize = 256 << 20
		}
		if before.Size() > maxSize || before.Size() <= 0 || total+before.Size() > 2<<30 {
			in.Close()
			return errors.New("size changed")
		}
		id := fmt.Sprintf("sample-%03d%s", i+1, strings.ToLower(path.Ext(c.name)))
		out, e := os.OpenFile("/output/samples/"+id, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			in.Close()
			return e
		}
		h := sha256.New()
		n, e := io.Copy(io.MultiWriter(out, h), io.LimitReader(in, maxSize+1))
		closeErr := out.Close()
		after, statErr := in.Stat()
		in.Close()
		if e != nil || closeErr != nil || statErr != nil || n != before.Size() || after.Size() != before.Size() || !after.ModTime().Equal(before.ModTime()) {
			return errors.New("copy changed or failed")
		}
		total += n
		if total > 2<<30 {
			return errors.New("sample byte budget")
		}
		rows = append(rows, entry{id, hex.EncodeToString(h.Sum(nil)), n, c.kind})
	}
	return json.NewEncoder(os.Stdout).Encode(struct {
		Profile string  `json:"profile"`
		Visited int     `json:"visited"`
		Skipped int     `json:"skipped"`
		Images  int     `json:"images"`
		Videos  int     `json:"videos"`
		Entries []entry `json:"entries"`
	}{"p4-sample-v1", visited, skipped, ic, vc, rows})
}
