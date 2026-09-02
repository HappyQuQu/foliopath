package face

import (
	"context"
	"errors"
	"io"
	"math"

	"github.com/HappyQuQu/foliopath/internal/media"
)

const (
	MaxCandidatesPerAsset = 64
	MaxEmbeddingDimension = 4096
	MaxInputBytes         = 250 * 1024 * 1024
	MaxDecodeDimension    = 1600
)

var (
	ErrInvalidInput       = errors.New("invalid face input")
	ErrInvalidOutput      = errors.New("invalid face output")
	ErrSourceChanged      = errors.New("face source changed")
	ErrRuntimeUnavailable = errors.New("face runtime unavailable")
)

type Box struct {
	X, Y, Width, Height float32
}

type Candidate struct {
	Box       Box
	Detection float32
	Quality   float32
	Embedding []float32
}

type Observation struct {
	Box               Box
	Detection         float32
	Quality           float32
	Embedding         []float32
	SourceFingerprint string
}

type Asset struct {
	File              io.ReadSeekCloser
	Format            media.Format
	SourceFingerprint string
}

type AssetSource interface {
	OpenFaceAsset(context.Context, int64, int64) (Asset, error)
}

type DecodedImage struct {
	Width  int
	Height int
	RGB    []byte
}

type ImageDecoder interface {
	DecodeFaceImage(context.Context, io.ReadSeeker, media.Format, int64, int) (DecodedImage, error)
}

type Runtime interface {
	AnalyzeFaces(context.Context, io.ReadSeeker, media.Format, int64) ([]Candidate, error)
}

type Processor struct {
	assets  AssetSource
	runtime Runtime
}

func NewProcessor(assets AssetSource, runtime Runtime) (*Processor, error) {
	if assets == nil || runtime == nil {
		return nil, ErrInvalidInput
	}
	return &Processor{assets: assets, runtime: runtime}, nil
}

func (processor *Processor) Analyze(ctx context.Context, libraryID, assetID int64, expectedFingerprint string) ([]Observation, error) {
	if libraryID < 1 || assetID < 1 || expectedFingerprint == "" || len(expectedFingerprint) > 256 {
		return nil, ErrInvalidInput
	}
	asset, err := processor.assets.OpenFaceAsset(ctx, libraryID, assetID)
	if err != nil {
		return nil, err
	}
	if asset.File == nil {
		return nil, ErrInvalidInput
	}
	defer asset.File.Close()
	if asset.SourceFingerprint != expectedFingerprint {
		return nil, ErrSourceChanged
	}
	if !supportedFaceFormat(asset.Format) {
		return nil, ErrInvalidInput
	}
	candidates, err := processor.runtime.AnalyzeFaces(ctx, asset.File, asset.Format, MaxInputBytes)
	if err != nil {
		return nil, err
	}
	if len(candidates) > MaxCandidatesPerAsset {
		return nil, ErrInvalidOutput
	}
	result := make([]Observation, len(candidates))
	for index, candidate := range candidates {
		normalized, err := validateCandidate(candidate)
		if err != nil {
			return nil, err
		}
		result[index] = Observation{Box: candidate.Box, Detection: candidate.Detection,
			Quality: candidate.Quality, Embedding: normalized, SourceFingerprint: expectedFingerprint}
	}
	return result, nil
}

func supportedFaceFormat(format media.Format) bool {
	switch format {
	case media.FormatJPEG, media.FormatPNG, media.FormatWebP, media.FormatGIF:
		return true
	default:
		return false
	}
}

func validateCandidate(candidate Candidate) ([]float32, error) {
	values := []float32{candidate.Box.X, candidate.Box.Y, candidate.Box.Width, candidate.Box.Height,
		candidate.Detection, candidate.Quality}
	for _, value := range values {
		if value < 0 || value > 1 || math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			return nil, ErrInvalidOutput
		}
	}
	if candidate.Box.Width <= 0 || candidate.Box.Height <= 0 ||
		candidate.Box.X+candidate.Box.Width > 1 || candidate.Box.Y+candidate.Box.Height > 1 ||
		len(candidate.Embedding) == 0 || len(candidate.Embedding) > MaxEmbeddingDimension {
		return nil, ErrInvalidOutput
	}
	var norm float64
	for _, value := range candidate.Embedding {
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			return nil, ErrInvalidOutput
		}
		norm += float64(value) * float64(value)
	}
	if norm == 0 {
		return nil, ErrInvalidOutput
	}
	scale := float32(1 / math.Sqrt(norm))
	result := make([]float32, len(candidate.Embedding))
	for index, value := range candidate.Embedding {
		result[index] = value * scale
	}
	return result, nil
}
