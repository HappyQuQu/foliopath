// Package semantic owns semantic-generation, embedding and search behavior.
package semantic

import (
	"encoding/binary"
	"errors"
	"math"
)

const SigLIPEmbeddingDimension = 768

var ErrInvalidEmbedding = errors.New("invalid semantic embedding")

// EncodeEmbedding normalizes a finite vector and encodes IEEE 754 binary16 in
// little-endian order. The byte representation is the revision-1 SQLite
// embedding contract.
func EncodeEmbedding(vector []float32, dimension int) ([]byte, error) {
	normalized, err := normalize(vector, dimension)
	if err != nil {
		return nil, err
	}
	encoded := make([]byte, len(normalized)*2)
	for index, value := range normalized {
		half, ok := float32ToHalf(value)
		if !ok {
			return nil, ErrInvalidEmbedding
		}
		binary.LittleEndian.PutUint16(encoded[index*2:], half)
	}
	return encoded, nil
}

// DecodeEmbedding decodes the persisted binary16 vector and normalizes it
// again before scoring. Quantization can change the norm, so callers must not
// score the raw decoded values.
func DecodeEmbedding(encoded []byte, dimension int) ([]float32, error) {
	if dimension < 1 || dimension > 65536 || len(encoded) != dimension*2 {
		return nil, ErrInvalidEmbedding
	}
	vector := make([]float32, dimension)
	for index := range vector {
		value := halfToFloat32(binary.LittleEndian.Uint16(encoded[index*2:]))
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			return nil, ErrInvalidEmbedding
		}
		vector[index] = value
	}
	return normalize(vector, dimension)
}

// NormalizeEmbedding validates and normalizes a transient query vector using
// the same float64 norm contract as persisted image embeddings.
func NormalizeEmbedding(vector []float32, dimension int) ([]float32, error) {
	return normalize(vector, dimension)
}

func normalize(vector []float32, dimension int) ([]float32, error) {
	if dimension < 1 || dimension > 65536 || len(vector) != dimension {
		return nil, ErrInvalidEmbedding
	}
	var squaredNorm float64
	for _, value := range vector {
		converted := float64(value)
		if math.IsNaN(converted) || math.IsInf(converted, 0) {
			return nil, ErrInvalidEmbedding
		}
		squaredNorm += converted * converted
	}
	if squaredNorm == 0 || math.IsInf(squaredNorm, 0) {
		return nil, ErrInvalidEmbedding
	}
	inverseNorm := 1 / math.Sqrt(squaredNorm)
	result := make([]float32, len(vector))
	for index, value := range vector {
		result[index] = float32(float64(value) * inverseNorm)
	}
	return result, nil
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
		if remainder > halfway || (remainder == halfway && halfMantissa&1 != 0) {
			halfMantissa++
		}
		return sign | uint16(halfMantissa), true
	}
	halfExponent := uint16(unbiased + 15)
	halfMantissa := mantissa >> 13
	remainder := mantissa & 0x1fff
	if remainder > 0x1000 || (remainder == 0x1000 && halfMantissa&1 != 0) {
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
