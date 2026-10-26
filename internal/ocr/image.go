package ocr

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"

	"github.com/HappyQuQu/foliopath/internal/media"
)

const (
	MaxSourceBytes       = 64 << 20
	MaxHeaderBytes       = 1 << 20
	MaxSourcePixels      = 40_000_000
	MaxPreparedPixels    = 4 * 1024 * 1024
	MaxPreparedDimension = 2048
	MinImageDimension    = 32
)

var (
	ErrInvalidImageInput     = errors.New("invalid OCR image input")
	ErrPreprocessFailed      = errors.New("OCR image preprocessing failed")
	ErrPreprocessUnavailable = errors.New("OCR image preprocessing unavailable")
)

// InputContract is a versioned compatibility policy, not proof that the native
// runtime is installed. Bind its hash explicitly in a future OCR deployment.
// Keys are lexicographically ordered; bytes are stable across implementations.
const InputContract = `{"alpha":"white","autorotate":true,"color":"srgb-uchar-rgb","formats":["jpeg","png","webp","gif"],"frame":"first","kernel":"lanczos3","max_dimension":2048,"max_header_bytes":1048576,"max_output_bytes":8388608,"max_output_pixels":4194304,"max_source_bytes":67108864,"max_source_pixels":40000000,"metadata":"strip","min_dimension":32,"output":"png","profile":"ocr-rgb-png-v1","upscale":false}`

func InputContractSHA256() string {
	digest := sha256.Sum256([]byte(InputContract))
	return hex.EncodeToString(digest[:])
}

// PreparedImage coordinates are after EXIF correction and aspect-preserving
// scaling. No original path or source fingerprint is inferred by this adapter.
type PreparedImage struct {
	ContractSHA256 string
	PNG            []byte
	Width          int
	Height         int
}

type ImagePreprocessor interface {
	PrepareOCRImage(context.Context, io.ReadSeeker, media.Format) (PreparedImage, error)
}
