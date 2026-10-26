package ocr

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"testing"

	"github.com/HappyQuQu/foliopath/internal/media"
)

type analyzerFile struct {
	*bytes.Reader
	closed   bool
	closeErr error
}

func (f *analyzerFile) Close() error { f.closed = true; return f.closeErr }
func (f *analyzerFile) Stat() (fs.FileInfo, error) {
	return nil, errors.New("not called: content owner already verified")
}

type analyzerContent struct {
	calls  int
	files  []*analyzerFile
	mutate func(int, *media.Content)
	err    error
}

func (c *analyzerContent) Open(_ context.Context, _ int64) (media.Content, error) {
	c.calls++
	if c.err != nil {
		return media.Content{}, c.err
	}
	f := &analyzerFile{Reader: bytes.NewReader([]byte("synthetic"))}
	c.files = append(c.files, f)
	v := media.Content{File: f, LibraryID: 1, Name: "synthetic.png", Fingerprint: "v1:10:20"}
	if c.mutate != nil {
		c.mutate(c.calls, &v)
	}
	return v, nil
}

type analyzerImages struct {
	err    error
	mutate func(*PreparedImage)
}

func (i analyzerImages) PrepareOCRImage(ctx context.Context, _ io.ReadSeeker, _ media.Format) (PreparedImage, error) {
	p := PreparedImage{ContractSHA256: InputContractSHA256(), PNG: []byte("synthetic prepared png"), Width: 1000, Height: 700}
	if i.mutate != nil {
		i.mutate(&p)
	}
	return p, i.err
}

type analyzerRuntime struct {
	calls  int
	err    error
	onCall func()
	result Result
}

func (r *analyzerRuntime) Identity() Identity {
	i := identityFixture()
	i.InputContractSHA256 = InputContractSHA256()
	i.OutputContractSHA256 = OutputContractSHA256()
	return i
}
func (r *analyzerRuntime) Recognize(ctx context.Context, _ []byte) (Result, error) {
	r.calls++
	if r.onCall != nil {
		r.onCall()
	}
	if r.result.Lines == nil && r.err == nil {
		return Result{Lines: []Line{}}, nil
	}
	return r.result, r.err
}
func TestOCRAnalyzerSourceChecksAndClosesBeforeReturning(t *testing.T) {
	c := &analyzerContent{}
	r := &analyzerRuntime{result: Result{Lines: []Line{lineFixture()}}}
	a, err := NewAnalyzer(c, analyzerImages{}, r)
	if err != nil {
		t.Fatal(err)
	}
	v, err := a.Analyze(context.Background(), 1, Candidate{AssetID: 1, SourceFingerprint: "v1:10:20"})
	if err != nil || v.Outcome != "ready" || len(v.Result.Lines) != 1 || c.calls != 2 || r.calls != 1 {
		t.Fatal(v, c.calls, r.calls, err)
	}
	for _, f := range c.files {
		if !f.closed {
			t.Fatal("open source leaked")
		}
	}
}
func TestOCRAnalyzerChangedOrCrossLibraryNeverBecomesBadMedia(t *testing.T) {
	for _, phase := range []string{"before", "after", "library", "io", "close"} {
		t.Run(phase, func(t *testing.T) {
			c := &analyzerContent{mutate: func(n int, v *media.Content) {
				switch phase {
				case "before":
					v.Fingerprint = "v1:20:30"
				case "after":
					if n == 2 {
						v.Fingerprint = "v1:20:30"
					}
				case "library":
					v.LibraryID = 2
				case "close":
					v.File.(*analyzerFile).closeErr = errors.New("private IOError")
				}
			}}
			if phase == "io" {
				c.err = media.ErrContentUnavailable
			}
			r := &analyzerRuntime{}
			a, _ := NewAnalyzer(c, analyzerImages{err: ErrInvalidMedia}, r)
			v, err := a.Analyze(context.Background(), 1, Candidate{AssetID: 1, SourceFingerprint: "v1:10:20"})
			if err == nil || v.Outcome == "failed" || r.calls != 0 {
				t.Fatal(v, err)
			}
			for _, f := range c.files {
				if !f.closed {
					t.Fatal("file leaked")
				}
			}
		})
	}
}
func TestOCRAnalyzerSourceFailureVersusProgramAndServiceFailure(t *testing.T) {
	for _, code := range []error{ErrInvalidMedia, ErrInputLimit, ErrPreprocessFailed, errors.New("untrusted exception /path"), ErrRuntimeUnavailable, ErrRuntimeBusy, ErrRuntimeOutput} {
		t.Run(code.Error(), func(t *testing.T) {
			c := &analyzerContent{}
			r := &analyzerRuntime{}
			images := analyzerImages{}
			switch code {
			case ErrRuntimeUnavailable, ErrRuntimeBusy, ErrRuntimeOutput:
				r.err = code
			default:
				images.err = code
			}
			a, _ := NewAnalyzer(c, images, r)
			v, err := a.Analyze(context.Background(), 1, Candidate{AssetID: 1, SourceFingerprint: "v1:10:20"})
			if code == ErrInvalidMedia || code == ErrInputLimit {
				if err != nil || v.Outcome != "failed" || v.Result.Lines != nil {
					t.Fatal(v, err)
				}
			} else if err == nil || v.Outcome == "failed" {
				t.Fatal(v, err)
			}
			if c.calls != 2 {
				t.Fatal("failure source not rechecked", c.calls)
			}
		})
	}
	// A service rejecting prepared bytes is not a corrupt original.
	c := &analyzerContent{}
	a, _ := NewAnalyzer(c, analyzerImages{}, &analyzerRuntime{err: ErrInvalidMedia})
	v, err := a.Analyze(context.Background(), 1, Candidate{AssetID: 1, SourceFingerprint: "v1:10:20"})
	if err == nil || v.Outcome == "failed" {
		t.Fatal(v, err)
	}
}
func TestOCRAnalyzerContractMismatchAndCancel(t *testing.T) {
	for _, mutate := range []func(*PreparedImage){func(p *PreparedImage) { p.ContractSHA256 = "wrong" }, func(p *PreparedImage) { p.Width = 0 }, func(p *PreparedImage) { p.PNG = nil }} {
		c := &analyzerContent{}
		r := &analyzerRuntime{}
		a, _ := NewAnalyzer(c, analyzerImages{mutate: mutate}, r)
		_, err := a.Analyze(context.Background(), 1, Candidate{AssetID: 1, SourceFingerprint: "v1:10:20"})
		if !errors.Is(err, ErrPreprocessFailed) || r.calls != 0 || !c.files[0].closed {
			t.Fatal(err)
		}
	}
	c := &analyzerContent{}
	ctx, cancel := context.WithCancel(context.Background())
	r := &analyzerRuntime{onCall: cancel}
	a, _ := NewAnalyzer(c, analyzerImages{}, r)
	_, err := a.Analyze(ctx, 1, Candidate{AssetID: 1, SourceFingerprint: "v1:10:20"})
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
