package main

import (
	"container/heap"
	"context"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"runtime"
	"sort"
	"time"

	_ "modernc.org/sqlite"
)

type VectorReport struct {
	SchemaVersion  int               `json:"schema_version"`
	GeneratedAt    string            `json:"generated_at"`
	Environment    VectorEnvironment `json:"environment"`
	Backend        string            `json:"backend"`
	Items          int               `json:"items"`
	Dimensions     int               `json:"dimensions"`
	Queries        int               `json:"queries"`
	TopK           int               `json:"top_k"`
	Seed           int64             `json:"seed"`
	FilterModulo   int               `json:"filter_modulo"`
	EligibleItems  int               `json:"eligible_items"`
	BuildMS        float64           `json:"build_ms"`
	QueryP50MS     float64           `json:"query_p50_ms"`
	QueryP95MS     float64           `json:"query_p95_ms"`
	QueryP99MS     float64           `json:"query_p99_ms"`
	HeapAllocBytes uint64            `json:"heap_alloc_bytes"`
	HeapSysBytes   uint64            `json:"heap_sys_bytes"`
	DatabaseBytes  int64             `json:"database_bytes,omitempty"`
	Caveats        []string          `json:"caveats"`
}

type VectorEnvironment struct {
	GOOS       string `json:"goos"`
	GOARCH     string `json:"goarch"`
	GoVersion  string `json:"go_version"`
	GOMAXPROCS int    `json:"gomaxprocs"`
}

func BenchmarkVectors(backend string, items, dims, queries, topK int, seed int64, filterModulo int) (VectorReport, error) {
	report := VectorReport{
		SchemaVersion: 1,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Environment:   VectorEnvironment{runtime.GOOS, runtime.GOARCH, runtime.Version(), runtime.GOMAXPROCS(0)},
		Backend:       backend, Items: items, Dimensions: dims, Queries: queries, TopK: topK, Seed: seed,
		FilterModulo: filterModulo,
		Caveats: []string{
			"random vectors measure capacity and exact-scan latency only; they provide no semantic quality evidence",
			"a local run is not S0 platform evidence until repeated on constrained linux/amd64 and linux/arm64",
		},
	}
	if items <= 0 || dims <= 0 || queries <= 0 || topK <= 0 || filterModulo <= 0 {
		return report, errors.New("items, dims, queries, topk, and filter-modulo must be positive")
	}
	report.EligibleItems = (items + filterModulo - 1) / filterModulo
	if topK > report.EligibleItems {
		return report, errors.New("topk cannot exceed eligible item count")
	}
	rng := rand.New(rand.NewPCG(uint64(seed), uint64(seed)^0x9e3779b97f4a7c15))
	queryVectors := make([][]float32, queries)
	for i := range queryVectors {
		queryVectors[i] = randomUnitVector(rng, dims)
	}
	var latencies []float64
	var err error
	switch backend {
	case "memory":
		latencies, report.BuildMS, err = benchmarkMemory(rng, items, dims, queryVectors, topK, filterModulo)
	case "sqlite":
		latencies, report.BuildMS, report.DatabaseBytes, err = benchmarkSQLite(rng, items, dims, queryVectors, topK, filterModulo)
	default:
		err = fmt.Errorf("unsupported backend %q", backend)
	}
	if err != nil {
		return report, err
	}
	sort.Float64s(latencies)
	report.QueryP50MS = percentile(latencies, 0.50)
	report.QueryP95MS = percentile(latencies, 0.95)
	report.QueryP99MS = percentile(latencies, 0.99)
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	report.HeapAllocBytes = memory.HeapAlloc
	report.HeapSysBytes = memory.HeapSys
	return report, nil
}

func benchmarkMemory(rng *rand.Rand, items, dims int, queries [][]float32, topK, filterModulo int) ([]float64, float64, error) {
	started := time.Now()
	vectors := make([]float32, items*dims)
	for i := 0; i < items; i++ {
		copy(vectors[i*dims:(i+1)*dims], randomUnitVector(rng, dims))
	}
	buildMS := milliseconds(time.Since(started))
	latencies := make([]float64, 0, len(queries))
	for _, query := range queries {
		started = time.Now()
		top := make(scoreHeap, 0, topK)
		for i := 0; i < items; i++ {
			if i%filterModulo != 0 {
				continue
			}
			top.offer(scoredID{i, dot(query, vectors[i*dims:(i+1)*dims])}, topK)
		}
		latencies = append(latencies, milliseconds(time.Since(started)))
	}
	runtime.KeepAlive(vectors)
	return latencies, buildMS, nil
}

func benchmarkSQLite(rng *rand.Rand, items, dims int, queries [][]float32, topK, filterModulo int) ([]float64, float64, int64, error) {
	file, err := os.CreateTemp("", "foliopath-int001-vector-*.db")
	if err != nil {
		return nil, 0, 0, err
	}
	name := file.Name()
	if err := file.Close(); err != nil {
		return nil, 0, 0, err
	}
	defer os.Remove(name)
	db, err := sql.Open("sqlite", name)
	if err != nil {
		return nil, 0, 0, err
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `PRAGMA journal_mode=WAL; PRAGMA synchronous=NORMAL; CREATE TABLE vectors (id INTEGER PRIMARY KEY, vector BLOB NOT NULL);`); err != nil {
		return nil, 0, 0, err
	}
	started := time.Now()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, 0, 0, err
	}
	stmt, err := tx.PrepareContext(ctx, `INSERT INTO vectors(id, vector) VALUES (?, ?)`)
	if err != nil {
		tx.Rollback()
		return nil, 0, 0, err
	}
	for i := 0; i < items; i++ {
		if _, err := stmt.ExecContext(ctx, i, encodeVector(randomUnitVector(rng, dims))); err != nil {
			stmt.Close()
			tx.Rollback()
			return nil, 0, 0, err
		}
	}
	if err := stmt.Close(); err != nil {
		tx.Rollback()
		return nil, 0, 0, err
	}
	if err := tx.Commit(); err != nil {
		return nil, 0, 0, err
	}
	buildMS := milliseconds(time.Since(started))
	latencies := make([]float64, 0, len(queries))
	for _, query := range queries {
		started = time.Now()
		rows, err := db.QueryContext(ctx, `SELECT id, vector FROM vectors WHERE id % ? = 0 ORDER BY id`, filterModulo)
		if err != nil {
			return nil, 0, 0, err
		}
		top := make(scoreHeap, 0, topK)
		for rows.Next() {
			var id int
			var blob []byte
			if err := rows.Scan(&id, &blob); err != nil {
				rows.Close()
				return nil, 0, 0, err
			}
			vector, err := decodeVector(blob, dims)
			if err != nil {
				rows.Close()
				return nil, 0, 0, err
			}
			top.offer(scoredID{id, dot(query, vector)}, topK)
		}
		if err := rows.Close(); err != nil {
			return nil, 0, 0, err
		}
		latencies = append(latencies, milliseconds(time.Since(started)))
	}
	info, err := os.Stat(name)
	if err != nil {
		return nil, 0, 0, err
	}
	return latencies, buildMS, info.Size(), nil
}

type scoredID struct {
	id    int
	score float32
}
type scoreHeap []scoredID

func (h scoreHeap) Len() int { return len(h) }
func (h scoreHeap) Less(i, j int) bool {
	return h[i].score < h[j].score || (h[i].score == h[j].score && h[i].id > h[j].id)
}
func (h scoreHeap) Swap(i, j int)   { h[i], h[j] = h[j], h[i] }
func (h *scoreHeap) Push(value any) { *h = append(*h, value.(scoredID)) }
func (h *scoreHeap) Pop() any {
	old := *h
	value := old[len(old)-1]
	*h = old[:len(old)-1]
	return value
}
func (h *scoreHeap) offer(value scoredID, limit int) {
	if h.Len() < limit {
		heap.Push(h, value)
		return
	}
	if (*h)[0].score < value.score || ((*h)[0].score == value.score && (*h)[0].id > value.id) {
		heap.Pop(h)
		heap.Push(h, value)
	}
}

func randomUnitVector(rng *rand.Rand, dims int) []float32 {
	vector := make([]float32, dims)
	var norm float64
	for i := range vector {
		value := float32(rng.Float64()*2 - 1)
		vector[i] = value
		norm += float64(value * value)
	}
	scale := float32(1 / math.Sqrt(norm))
	for i := range vector {
		vector[i] *= scale
	}
	return vector
}

func dot(a, b []float32) float32 {
	var result float32
	for i := range a {
		result += a[i] * b[i]
	}
	return result
}
func encodeVector(vector []float32) []byte {
	blob := make([]byte, len(vector)*4)
	for i, value := range vector {
		binary.LittleEndian.PutUint32(blob[i*4:], math.Float32bits(value))
	}
	return blob
}
func decodeVector(blob []byte, dims int) ([]float32, error) {
	if len(blob) != dims*4 {
		return nil, fmt.Errorf("vector blob has %d bytes, want %d", len(blob), dims*4)
	}
	vector := make([]float32, dims)
	for i := range vector {
		vector[i] = math.Float32frombits(binary.LittleEndian.Uint32(blob[i*4:]))
	}
	return vector, nil
}
func percentile(sortedValues []float64, percentile float64) float64 {
	index := int(math.Ceil(percentile*float64(len(sortedValues)))) - 1
	if index < 0 {
		index = 0
	}
	return sortedValues[index]
}
func milliseconds(duration time.Duration) float64 { return float64(duration.Microseconds()) / 1000 }
