package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"math/rand"
	randv2 "math/rand/v2"
	"os"
	"runtime"
	"sort"
	"time"

	"github.com/coder/hnsw"
)

type ANNReport struct {
	SchemaVersion       int               `json:"schema_version"`
	GeneratedAt         string            `json:"generated_at"`
	Environment         VectorEnvironment `json:"environment"`
	Engine              string            `json:"engine"`
	EngineRevision      string            `json:"engine_revision"`
	EngineLicense       string            `json:"engine_license"`
	Items               int               `json:"items"`
	Dimensions          int               `json:"dimensions"`
	Queries             int               `json:"queries"`
	TopK                int               `json:"top_k"`
	M                   int               `json:"m"`
	EfSearch            int               `json:"ef_search"`
	Seed                int64             `json:"seed"`
	BuildMS             float64           `json:"build_ms"`
	SaveMS              float64           `json:"save_ms"`
	LoadMS              float64           `json:"load_ms"`
	IndexBytes          int64             `json:"index_bytes"`
	QueryP50MS          float64           `json:"query_p50_ms"`
	QueryP95MS          float64           `json:"query_p95_ms"`
	QueryP99MS          float64           `json:"query_p99_ms"`
	MeanRecallAtK       float64           `json:"mean_recall_at_k"`
	MinimumRecallAtK    float64           `json:"minimum_recall_at_k"`
	HeapAllocBytes      uint64            `json:"heap_alloc_bytes"`
	HeapSysBytes        uint64            `json:"heap_sys_bytes"`
	CorruptLoadRejected bool              `json:"corrupt_load_rejected"`
	RecoveryRebuildMS   float64           `json:"recovery_rebuild_ms"`
	RecoverySearchOK    bool              `json:"recovery_search_ok"`
	DeterministicExport bool              `json:"deterministic_export"`
	Caveats             []string          `json:"caveats"`
}

func BenchmarkANN(items, dims, queries, topK int, seed int64, m, efSearch int) (ANNReport, error) {
	report := ANNReport{
		SchemaVersion:  1,
		GeneratedAt:    time.Now().UTC().Format(time.RFC3339),
		Environment:    VectorEnvironment{runtime.GOOS, runtime.GOARCH, runtime.Version(), runtime.GOMAXPROCS(0)},
		Engine:         "github.com/coder/hnsw",
		EngineRevision: "36cab6028fed4adc9c3edf2323a06f0a95c1f030",
		EngineLicense:  "CC0-1.0",
		Items:          items, Dimensions: dims, Queries: queries, TopK: topK, M: m, EfSearch: efSearch, Seed: seed,
		Caveats: []string{
			"random vectors measure ANN capacity and algorithmic recall only, not semantic quality",
			"the candidate import format requires a hostile-file allocation audit before approval",
			"the persisted graph duplicates vectors across HNSW layers and remains fully derived data",
		},
	}
	if items <= 0 || dims <= 0 || queries <= 0 || topK <= 0 || topK > items || m <= 0 || efSearch < topK {
		return report, errors.New("invalid ANN arguments")
	}
	rng := randv2.New(randv2.NewPCG(uint64(seed), uint64(seed)^0x9e3779b97f4a7c15))
	queryVectors := make([][]float32, queries)
	for i := range queryVectors {
		queryVectors[i] = randomUnitVector(rng, dims)
	}
	vectors := make([]float32, items*dims)
	for id := 0; id < items; id++ {
		copy(vectors[id*dims:(id+1)*dims], randomUnitVector(rng, dims))
	}
	groundTruth, _ := queryFloat32(vectors, dims, queryVectors, topK)

	started := time.Now()
	graph := buildHNSW(vectors, dims, seed, m, efSearch)
	report.BuildMS = milliseconds(time.Since(started))

	indexFile, err := os.CreateTemp("", "foliopath-int001-hnsw-*.index")
	if err != nil {
		return report, err
	}
	indexPath := indexFile.Name()
	indexFile.Close()
	defer os.Remove(indexPath)
	secondPath := indexPath + ".second"
	corruptPath := indexPath + ".corrupt"
	defer os.Remove(secondPath)
	defer os.Remove(corruptPath)

	saved := &hnsw.SavedGraph[int]{Graph: graph, Path: indexPath}
	started = time.Now()
	if err := saved.Save(); err != nil {
		return report, err
	}
	report.SaveMS = milliseconds(time.Since(started))
	info, err := os.Stat(indexPath)
	if err != nil {
		return report, err
	}
	report.IndexBytes = info.Size()

	started = time.Now()
	loaded, err := hnsw.LoadSavedGraph[int](indexPath)
	if err != nil {
		return report, err
	}
	report.LoadMS = milliseconds(time.Since(started))
	actual, latencies := queryHNSW(loaded.Graph, queryVectors, topK)
	report.MeanRecallAtK, report.MinimumRecallAtK = recallAtK(groundTruth, actual, topK)
	sort.Float64s(latencies)
	report.QueryP50MS = percentile(latencies, 0.50)
	report.QueryP95MS = percentile(latencies, 0.95)
	report.QueryP99MS = percentile(latencies, 0.99)

	second := &hnsw.SavedGraph[int]{Graph: graph, Path: secondPath}
	if err := second.Save(); err != nil {
		return report, err
	}
	firstHash, err := fileSHA256(indexPath)
	if err != nil {
		return report, err
	}
	secondHash, err := fileSHA256(secondPath)
	if err != nil {
		return report, err
	}
	report.DeterministicExport = firstHash == secondHash

	if err := truncateCopy(indexPath, corruptPath, report.IndexBytes/2); err != nil {
		return report, err
	}
	_, corruptErr := hnsw.LoadSavedGraph[int](corruptPath)
	report.CorruptLoadRejected = corruptErr != nil
	started = time.Now()
	recovered := buildHNSW(vectors, dims, seed, m, efSearch)
	report.RecoveryRebuildMS = milliseconds(time.Since(started))
	report.RecoverySearchOK = len(recovered.Search(queryVectors[0], topK)) == topK

	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	report.HeapAllocBytes = memory.HeapAlloc
	report.HeapSysBytes = memory.HeapSys
	runtime.KeepAlive(vectors)
	return report, nil
}

func buildHNSW(vectors []float32, dims int, seed int64, m, efSearch int) *hnsw.Graph[int] {
	graph := hnsw.NewGraph[int]()
	graph.M = m
	graph.EfSearch = efSearch
	graph.Rng = rand.New(rand.NewSource(seed))
	for id := 0; id < len(vectors)/dims; id++ {
		graph.Add(hnsw.MakeNode(id, vectors[id*dims:(id+1)*dims]))
	}
	return graph
}

func queryHNSW(graph *hnsw.Graph[int], queries [][]float32, topK int) ([][]int, []float64) {
	ids := make([][]int, 0, len(queries))
	latencies := make([]float64, 0, len(queries))
	for _, query := range queries {
		started := time.Now()
		results := graph.Search(query, topK)
		latencies = append(latencies, milliseconds(time.Since(started)))
		keys := make([]int, len(results))
		for index, result := range results {
			keys[index] = result.Key
		}
		ids = append(ids, keys)
	}
	return ids, latencies
}

func truncateCopy(sourcePath, destinationPath string, limit int64) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()
	destination, err := os.OpenFile(destinationPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	_, copyErr := io.CopyN(destination, source, limit)
	closeErr := destination.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func fileSHA256(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()
	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}
