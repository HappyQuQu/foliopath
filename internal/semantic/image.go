package semantic

import "errors"

const (
	SigLIPImageWidth    = 224
	SigLIPImageHeight   = 224
	SigLIPImageChannels = 3
	SigLIPImagePixels   = SigLIPImageWidth * SigLIPImageHeight
	SigLIPImageValues   = SigLIPImagePixels * SigLIPImageChannels
)

var (
	ErrInvalidImageInput          = errors.New("invalid semantic image input")
	ErrImagePreprocessFailed      = errors.New("semantic image preprocessing failed")
	ErrImagePreprocessUnavailable = errors.New("semantic image preprocessing unavailable")
)

// PrepareSigLIPImageTensor converts an exactly 224x224 interleaved uint8 RGB
// image into the fixed float32 CHW tensor expected by the reviewed SigLIP
// image graph. Decode, orientation, alpha removal, color conversion and
// bicubic resize remain the bounded native image adapter's responsibility.
func PrepareSigLIPImageTensor(rgb []byte) ([]float32, error) {
	if len(rgb) != SigLIPImageValues {
		return nil, ErrInvalidImageInput
	}
	result := make([]float32, SigLIPImageValues)
	const scale = float64(1) / 255
	for pixel := 0; pixel < SigLIPImagePixels; pixel++ {
		input := pixel * SigLIPImageChannels
		for channel := 0; channel < SigLIPImageChannels; channel++ {
			// Match the pinned Transformers slow image processor: rescale in
			// float64, downcast to float32, then normalize with float32
			// mean/std 0.5. Keeping these stages separate avoids a silent
			// one-ULP contract change from algebraic simplification.
			rescaled := float32(float64(rgb[input+channel]) * scale)
			result[channel*SigLIPImagePixels+pixel] = (rescaled - float32(0.5)) / float32(0.5)
		}
	}
	return result, nil
}
