package face

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"math"
	"time"
)

var (
	ErrInvalidObservation        = errors.New("invalid face observation")
	ErrFaceGenerationUnavailable = errors.New("face generation unavailable")
)

type ObservationItem struct {
	ID                string
	Box               Box
	Detection         float32
	Quality           float32
	SourceFingerprint string
	Vector            []byte
	CreatedAt         time.Time
}

type ObservationBatch struct {
	GenerationID string
	LibraryID    int64
	AssetID      int64
	Items        []ObservationItem
	UpdatedAt    time.Time
}

type StoredObservation struct {
	ID                string
	GenerationID      string
	LibraryID         int64
	AssetID           int64
	Box               Box
	Detection         float32
	Quality           float32
	SourceFingerprint string
	Vector            []byte
	Revision          int64
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type ObservationRepository interface {
	ReplaceFaceObservations(context.Context, ObservationBatch) error
	ListFaceObservations(context.Context, string, int64, int64) ([]StoredObservation, error)
	DeleteFaceObservationsIfSourceChanged(context.Context, string, int64, int64, string, time.Time) (bool, error)
}

func ValidateObservationBatch(batch ObservationBatch, maximumItems int) error {
	if len(batch.GenerationID) < 8 || len(batch.GenerationID) > 128 || batch.LibraryID < 1 ||
		batch.AssetID < 1 || maximumItems < 1 || len(batch.Items) > maximumItems || batch.UpdatedAt.IsZero() {
		return ErrInvalidObservation
	}
	seenIDs := make(map[string]struct{}, len(batch.Items))
	var fingerprint string
	for _, item := range batch.Items {
		if len(item.ID) < 8 || len(item.ID) > 128 || len(item.SourceFingerprint) < 1 ||
			len(item.SourceFingerprint) > 256 || item.CreatedAt.IsZero() ||
			len(item.Vector) < 2 || len(item.Vector)%2 != 0 || !validBoxAndScores(item.Box, item.Detection, item.Quality) {
			return ErrInvalidObservation
		}
		if fingerprint == "" {
			fingerprint = item.SourceFingerprint
		} else if fingerprint != item.SourceFingerprint {
			return ErrInvalidObservation
		}
		if _, exists := seenIDs[item.ID]; exists {
			return ErrInvalidObservation
		}
		seenIDs[item.ID] = struct{}{}
	}
	return nil
}

func validBoxAndScores(box Box, detection, quality float32) bool {
	values := []float32{box.X, box.Y, box.Width, box.Height, detection, quality}
	for _, value := range values {
		if value < 0 || value > 1 || math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			return false
		}
	}
	return box.Width > 0 && box.Height > 0 && box.X+box.Width <= 1 && box.Y+box.Height <= 1
}

// EncodeEmbedding stores the normalized face vector as little-endian IEEE 754 binary16.
func EncodeEmbedding(vector []float32, dimension int) ([]byte, error) {
	if dimension < 1 || dimension > MaxEmbeddingDimension || len(vector) != dimension {
		return nil, ErrInvalidObservation
	}
	var squaredNorm float64
	for _, value := range vector {
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			return nil, ErrInvalidObservation
		}
		squaredNorm += float64(value) * float64(value)
	}
	if squaredNorm == 0 || math.IsInf(squaredNorm, 0) {
		return nil, ErrInvalidObservation
	}
	encoded := make([]byte, dimension*2)
	inverseNorm := float32(1 / math.Sqrt(squaredNorm))
	for index, value := range vector {
		half, ok := float32ToHalf(value * inverseNorm)
		if !ok {
			return nil, ErrInvalidObservation
		}
		binary.LittleEndian.PutUint16(encoded[index*2:], half)
	}
	return encoded, nil
}

func ValidateEncodedEmbedding(encoded []byte, dimension int) error {
	if dimension < 1 || dimension > MaxEmbeddingDimension || len(encoded) != dimension*2 {
		return ErrInvalidObservation
	}
	var squaredNorm float64
	for index := 0; index < dimension; index++ {
		value := halfToFloat32(binary.LittleEndian.Uint16(encoded[index*2:]))
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			return ErrInvalidObservation
		}
		squaredNorm += float64(value) * float64(value)
	}
	if squaredNorm == 0 || math.IsInf(squaredNorm, 0) {
		return ErrInvalidObservation
	}
	return nil
}

func DecodeEmbedding(encoded []byte, dimension int) ([]float32, error) {
	if err := ValidateEncodedEmbedding(encoded, dimension); err != nil {
		return nil, err
	}
	vector := make([]float32, dimension)
	for index := range vector {
		vector[index] = halfToFloat32(binary.LittleEndian.Uint16(encoded[index*2:]))
	}
	if !normalizeInPlace(vector) {
		return nil, ErrInvalidObservation
	}
	return vector, nil
}

// ObservationID is stable only for the exact generation, source lineage,
// geometry, and embedding. It never depends on detector output order.
func ObservationID(generationID string, libraryID, assetID int64, observation Observation) string {
	hash := sha256.New()
	hash.Write([]byte(generationID))
	hash.Write([]byte{0})
	var integer [8]byte
	binary.LittleEndian.PutUint64(integer[:], uint64(libraryID))
	hash.Write(integer[:])
	binary.LittleEndian.PutUint64(integer[:], uint64(assetID))
	hash.Write(integer[:])
	hash.Write([]byte(observation.SourceFingerprint))
	hash.Write([]byte{0})
	for _, value := range []float32{observation.Box.X, observation.Box.Y, observation.Box.Width, observation.Box.Height} {
		var bits [4]byte
		binary.LittleEndian.PutUint32(bits[:], math.Float32bits(value))
		hash.Write(bits[:])
	}
	for _, value := range observation.Embedding {
		var bits [4]byte
		binary.LittleEndian.PutUint32(bits[:], math.Float32bits(value))
		hash.Write(bits[:])
	}
	return "face_" + hex.EncodeToString(hash.Sum(nil)[:20])
}

func float32ToHalf(value float32) (uint16, bool) {
	bits := math.Float32bits(value)
	sign := uint16(bits >> 16 & 0x8000)
	exponent := int((bits >> 23) & 0xff)
	mantissa := bits & 0x7fffff
	if exponent == 0xff {
		return 0, false
	}
	unbiased := exponent - 127
	if unbiased > 15 {
		return 0, false
	}
	if unbiased < -24 {
		return sign, true
	}
	if unbiased < -14 {
		mantissa |= 0x800000
		shift := uint32(-unbiased - 1)
		halfMantissa := mantissa >> shift
		remainder := mantissa & ((uint32(1) << shift) - 1)
		halfway := uint32(1) << (shift - 1)
		if remainder > halfway || remainder == halfway && halfMantissa&1 != 0 {
			halfMantissa++
		}
		return sign | uint16(halfMantissa), true
	}
	halfExponent := uint16(unbiased + 15)
	halfMantissa := mantissa >> 13
	remainder := mantissa & 0x1fff
	if remainder > 0x1000 || remainder == 0x1000 && halfMantissa&1 != 0 {
		halfMantissa++
		if halfMantissa == 0x400 {
			halfMantissa = 0
			halfExponent++
			if halfExponent >= 0x1f {
				return 0, false
			}
		}
	}
	return sign | halfExponent<<10 | uint16(halfMantissa), true
}

func halfToFloat32(value uint16) float32 {
	sign := uint32(value&0x8000) << 16
	exponent := uint32(value>>10) & 0x1f
	mantissa := uint32(value & 0x03ff)
	switch exponent {
	case 0:
		if mantissa == 0 {
			return math.Float32frombits(sign)
		}
		exponent = 113
		for mantissa&0x0400 == 0 {
			mantissa <<= 1
			exponent--
		}
		mantissa &= 0x03ff
		return math.Float32frombits(sign | exponent<<23 | mantissa<<13)
	case 0x1f:
		return math.Float32frombits(sign | 0x7f800000 | mantissa<<13)
	default:
		return math.Float32frombits(sign | (exponent+112)<<23 | mantissa<<13)
	}
}
