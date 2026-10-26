package main

import (
	"errors"
	"math"
	"math/rand/v2"
	"runtime"
	"sort"
	"time"
)

type QuantizationReport struct {
	SchemaVersion int               `json:"schema_version"`
	GeneratedAt   string            `json:"generated_at"`
	Environment   VectorEnvironment `json:"environment"`
	Items         int               `json:"items"`
	Dimensions    int               `json:"dimensions"`
	Queries       int               `json:"queries"`
	TopK          int               `json:"top_k"`
	Seed          int64             `json:"seed"`
	Baseline      QuantizationRun   `json:"baseline"`
	Formats       []QuantizationRun `json:"formats"`
	Caveats       []string          `json:"caveats"`
}

type QuantizationRun struct {
	Format           string  `json:"format"`
	StorageBytes     int64   `json:"storage_bytes"`
	BytesPerVector   int64   `json:"bytes_per_vector"`
	BuildMS          float64 `json:"build_ms"`
	QueryP50MS       float64 `json:"query_p50_ms"`
	QueryP95MS       float64 `json:"query_p95_ms"`
	QueryP99MS       float64 `json:"query_p99_ms"`
	MeanRecallAtK    float64 `json:"mean_recall_at_k"`
	MinimumRecallAtK float64 `json:"minimum_recall_at_k"`
}

func BenchmarkQuantization(items, dims, queries, topK int, seed int64) (QuantizationReport, error) {
	report := QuantizationReport{
		SchemaVersion: 1,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Environment:   VectorEnvironment{runtime.GOOS, runtime.GOARCH, runtime.Version(), runtime.GOMAXPROCS(0)},
		Items:         items, Dimensions: dims, Queries: queries, TopK: topK, Seed: seed,
		Caveats: []string{
			"random vectors measure quantization distortion and exact-scan capacity only, not model quality",
			"storage_bytes is packed payload size and excludes SQLite row/page/WAL overhead",
			"quantization acceptance requires recall on embeddings from the selected real model",
		},
	}
	if items <= 0 || dims <= 0 || queries <= 0 || topK <= 0 || topK > items {
		return report, errors.New("items, dims, queries, and topk must be positive; topk cannot exceed items")
	}
	rng := rand.New(rand.NewPCG(uint64(seed), uint64(seed)^0x9e3779b97f4a7c15))
	queryVectors := make([][]float32, queries)
	for i := range queryVectors {
		queryVectors[i] = randomUnitVector(rng, dims)
	}
	vectors := make([]float32, items*dims)
	for i := 0; i < items; i++ {
		copy(vectors[i*dims:(i+1)*dims], randomUnitVector(rng, dims))
	}

	baselineIDs, baselineLatencies := queryFloat32(vectors, dims, queryVectors, topK)
	report.Baseline = quantizationTiming("float32", int64(len(vectors))*4, int64(dims)*4, 0, baselineLatencies, 1, 1)

	started := time.Now()
	halfVectors := make([]uint16, len(vectors))
	for i, value := range vectors {
		halfVectors[i] = float32ToHalf(value)
	}
	halfBuildMS := milliseconds(time.Since(started))
	halfIDs, halfLatencies := queryFloat16(halfVectors, dims, queryVectors, topK)
	halfMean, halfMinimum := recallAtK(baselineIDs, halfIDs, topK)
	report.Formats = append(report.Formats, quantizationTiming(
		"float16", int64(len(halfVectors))*2, int64(dims)*2, halfBuildMS, halfLatencies, halfMean, halfMinimum,
	))

	started = time.Now()
	int8Vectors := make([]int8, len(vectors))
	int8Scales := make([]float32, items)
	for id := 0; id < items; id++ {
		vector := vectors[id*dims : (id+1)*dims]
		var maximum float32
		for _, value := range vector {
			if absolute := float32(math.Abs(float64(value))); absolute > maximum {
				maximum = absolute
			}
		}
		scale := maximum / 127
		if scale == 0 {
			scale = 1
		}
		int8Scales[id] = scale
		for offset, value := range vector {
			int8Vectors[id*dims+offset] = int8(math.Round(float64(value / scale)))
		}
	}
	int8BuildMS := milliseconds(time.Since(started))
	int8IDs, int8Latencies := queryInt8(int8Vectors, int8Scales, dims, queryVectors, topK)
	int8Mean, int8Minimum := recallAtK(baselineIDs, int8IDs, topK)
	report.Formats = append(report.Formats, quantizationTiming(
		"int8-per-vector-scale", int64(len(int8Vectors))+int64(len(int8Scales))*4,
		int64(dims)+4, int8BuildMS, int8Latencies, int8Mean, int8Minimum,
	))
	runtime.KeepAlive(vectors)
	return report, nil
}

func quantizationTiming(format string, storageBytes, bytesPerVector int64, buildMS float64, latencies []float64, meanRecall, minimumRecall float64) QuantizationRun {
	sort.Float64s(latencies)
	return QuantizationRun{
		Format: format, StorageBytes: storageBytes, BytesPerVector: bytesPerVector, BuildMS: buildMS,
		QueryP50MS: percentile(latencies, 0.50), QueryP95MS: percentile(latencies, 0.95), QueryP99MS: percentile(latencies, 0.99),
		MeanRecallAtK: meanRecall, MinimumRecallAtK: minimumRecall,
	}
}

func queryFloat32(vectors []float32, dims int, queries [][]float32, topK int) ([][]int, []float64) {
	return queryEncoded(len(vectors)/dims, queries, topK, func(id int, query []float32) float32 {
		return dot(query, vectors[id*dims:(id+1)*dims])
	})
}

func queryFloat16(vectors []uint16, dims int, queries [][]float32, topK int) ([][]int, []float64) {
	return queryEncoded(len(vectors)/dims, queries, topK, func(id int, query []float32) float32 {
		var score float32
		start := id * dims
		for offset, queryValue := range query {
			score += queryValue * halfToFloat32(vectors[start+offset])
		}
		return score
	})
}

func queryInt8(vectors []int8, scales []float32, dims int, queries [][]float32, topK int) ([][]int, []float64) {
	return queryEncoded(len(scales), queries, topK, func(id int, query []float32) float32 {
		var score float32
		start := id * dims
		for offset, queryValue := range query {
			score += queryValue * float32(vectors[start+offset])
		}
		return score * scales[id]
	})
}

func queryEncoded(items int, queries [][]float32, topK int, score func(int, []float32) float32) ([][]int, []float64) {
	allIDs := make([][]int, 0, len(queries))
	latencies := make([]float64, 0, len(queries))
	for _, query := range queries {
		started := time.Now()
		top := make(scoreHeap, 0, topK)
		for id := 0; id < items; id++ {
			top.offer(scoredID{id: id, score: score(id, query)}, topK)
		}
		ids := make([]int, len(top))
		for i, item := range top {
			ids[i] = item.id
		}
		allIDs = append(allIDs, ids)
		latencies = append(latencies, milliseconds(time.Since(started)))
	}
	return allIDs, latencies
}

func recallAtK(expected, actual [][]int, topK int) (float64, float64) {
	minimum := 1.0
	var total float64
	for queryIndex, expectedIDs := range expected {
		set := make(map[int]struct{}, len(expectedIDs))
		for _, id := range expectedIDs {
			set[id] = struct{}{}
		}
		matches := 0
		for _, id := range actual[queryIndex] {
			if _, exists := set[id]; exists {
				matches++
			}
		}
		recall := float64(matches) / float64(topK)
		total += recall
		if recall < minimum {
			minimum = recall
		}
	}
	return total / float64(len(expected)), minimum
}

func float32ToHalf(value float32) uint16 {
	bits := math.Float32bits(value)
	sign := uint16(bits>>16) & 0x8000
	exponent := int((bits >> 23) & 0xff)
	mantissa := bits & 0x7fffff
	if exponent == 0xff {
		if mantissa == 0 {
			return sign | 0x7c00
		}
		return sign | 0x7e00
	}
	halfExponent := exponent - 127 + 15
	if halfExponent >= 31 {
		return sign | 0x7c00
	}
	if halfExponent <= 0 {
		if halfExponent < -10 {
			return sign
		}
		mantissa |= 0x800000
		shift := uint32(14 - halfExponent)
		halfMantissa := uint16(mantissa >> shift)
		if (mantissa>>(shift-1))&1 != 0 {
			halfMantissa++
		}
		return sign | halfMantissa
	}
	half := sign | uint16(halfExponent<<10) | uint16(mantissa>>13)
	if mantissa&0x1000 != 0 {
		half++
	}
	return half
}

func halfToFloat32(value uint16) float32 {
	sign := uint32(value&0x8000) << 16
	exponent := uint32((value >> 10) & 0x1f)
	mantissa := uint32(value & 0x03ff)
	if exponent == 0 {
		if mantissa == 0 {
			return math.Float32frombits(sign)
		}
		exponent = 1
		for mantissa&0x0400 == 0 {
			mantissa <<= 1
			exponent--
		}
		mantissa &= 0x03ff
		exponent += 127 - 15
	} else if exponent == 0x1f {
		exponent = 0xff
	} else {
		exponent += 127 - 15
	}
	return math.Float32frombits(sign | exponent<<23 | mantissa<<13)
}
