package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "int001-ai:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return errors.New("expected validate, vector, vector-concurrency, vector-quant, ann, face-score, quality-score, or scan-models")
	}
	switch args[0] {
	case "validate":
		return runValidate(args[1:])
	case "vector":
		return runVector(args[1:])
	case "vector-concurrency":
		return runVectorConcurrency(args[1:])
	case "vector-quant":
		return runVectorQuant(args[1:])
	case "ann":
		return runANN(args[1:])
	case "face-score":
		return runFaceScore(args[1:])
	case "quality-score":
		return runQualityScore(args[1:])
	case "scan-models":
		return runModelScan(args[1:])
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runQualityScore(args []string) error {
	fs := flag.NewFlagSet("quality-score", flag.ContinueOnError)
	input := fs.String("input", "", "S2B tag/video quality result manifest")
	datasetManifest := fs.String("dataset-manifest", "", "governed source dataset manifest")
	commit := fs.String("commit", "", "source commit that produced the final-model result")
	output := fs.String("output", "", "optional verified quality summary path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *input == "" || *datasetManifest == "" || !regexp.MustCompile(`^[0-9a-f]{40,64}$`).MatchString(*commit) {
		return errors.New("-input, -dataset-manifest and a valid -commit are required")
	}
	dataset, err := ReadS2BQualityDataset(*input)
	if err != nil {
		return err
	}
	if err := ValidateS2BQualityManifest(dataset, *datasetManifest); err != nil {
		return err
	}
	report, err := ScoreS2BQuality(dataset)
	if err != nil {
		return err
	}
	report.SourceCommit = *commit
	if err := writeJSON(report); err != nil {
		return err
	}
	if *output != "" {
		if err := writeJSONFile(*output, report); err != nil {
			return err
		}
	}
	if !report.GatePass {
		return errors.New("S2B quality gate failed")
	}
	return nil
}

func writeJSONFile(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".quality-summary-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	encoder := json.NewEncoder(temporary)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}

func runVectorConcurrency(args []string) error {
	fs := flag.NewFlagSet("vector-concurrency", flag.ContinueOnError)
	items := fs.Int("items", 100000, "vector count")
	dimensions := fs.Int("dims", 512, "vector dimensions")
	batchSize := fs.Int("batch", 256, "bounded SQLite backfill batch size")
	browseQueries := fs.Int("browse-queries", 100, "keyset browse query count")
	searchQueries := fs.Int("search-queries", 10, "exact vector query count")
	batchDelay := fs.Duration("batch-delay", 5*time.Millisecond, "synthetic inference delay between committed batches")
	storageFormat := fs.String("format", "float32", "float32 or float16 SQLite vector BLOB")
	seed := fs.Int64("seed", 20260826, "deterministic seed")
	if err := fs.Parse(args); err != nil {
		return err
	}
	report, err := BenchmarkVectorConcurrency(
		*items, *dimensions, *batchSize, *browseQueries, *searchQueries, *seed,
		*batchDelay, *storageFormat,
	)
	if err != nil {
		return err
	}
	return writeJSON(report)
}

func runValidate(args []string) error {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	datasetPath := fs.String("dataset", "", "dataset manifest path")
	modelsPath := fs.String("models", "", "model catalog path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if (*datasetPath == "") == (*modelsPath == "") {
		return errors.New("provide exactly one of -dataset or -models")
	}
	if *datasetPath != "" {
		manifest, err := ReadDatasetManifest(*datasetPath)
		if err != nil {
			return err
		}
		return writeJSON(map[string]any{"valid": true, "kind": "dataset", "items": len(manifest.Items)})
	}
	catalog, err := ReadModelCatalog(*modelsPath)
	if err != nil {
		return err
	}
	return writeJSON(map[string]any{"valid": true, "kind": "model_catalog", "models": len(catalog.Models)})
}

func runVector(args []string) error {
	fs := flag.NewFlagSet("vector", flag.ContinueOnError)
	backend := fs.String("backend", "memory", "memory or sqlite")
	items := fs.Int("items", 10000, "vector count")
	dims := fs.Int("dims", 256, "vector dimensions")
	queries := fs.Int("queries", 10, "query count")
	topK := fs.Int("topk", 20, "result count")
	seed := fs.Int64("seed", 20260825, "deterministic seed")
	filterModulo := fs.Int("filter-modulo", 1, "only search IDs divisible by this value")
	if err := fs.Parse(args); err != nil {
		return err
	}
	report, err := BenchmarkVectors(*backend, *items, *dims, *queries, *topK, *seed, *filterModulo)
	if err != nil {
		return err
	}
	return writeJSON(report)
}

func runVectorQuant(args []string) error {
	fs := flag.NewFlagSet("vector-quant", flag.ContinueOnError)
	items := fs.Int("items", 100000, "vector count")
	dims := fs.Int("dims", 512, "vector dimensions")
	queries := fs.Int("queries", 20, "query count")
	topK := fs.Int("topk", 20, "result count")
	seed := fs.Int64("seed", 20260825, "deterministic seed")
	if err := fs.Parse(args); err != nil {
		return err
	}
	report, err := BenchmarkQuantization(*items, *dims, *queries, *topK, *seed)
	if err != nil {
		return err
	}
	return writeJSON(report)
}

func runANN(args []string) error {
	fs := flag.NewFlagSet("ann", flag.ContinueOnError)
	items := fs.Int("items", 10000, "vector count")
	dims := fs.Int("dims", 128, "vector dimensions")
	queries := fs.Int("queries", 20, "query count")
	topK := fs.Int("topk", 20, "result count")
	seed := fs.Int64("seed", 20260825, "deterministic seed")
	m := fs.Int("m", 16, "maximum graph neighbors")
	efSearch := fs.Int("ef-search", 64, "search exploration budget")
	if err := fs.Parse(args); err != nil {
		return err
	}
	report, err := BenchmarkANN(*items, *dims, *queries, *topK, *seed, *m, *efSearch)
	if err != nil {
		return err
	}
	return writeJSON(report)
}

func runFaceScore(args []string) error {
	fs := flag.NewFlagSet("face-score", flag.ContinueOnError)
	input := fs.String("input", "", "face embedding/cluster score manifest")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *input == "" {
		return errors.New("-input is required")
	}
	dataset, err := ReadFaceScoreDataset(*input)
	if err != nil {
		return err
	}
	report, err := ScoreFaces(dataset)
	if err != nil {
		return err
	}
	return writeJSON(report)
}

func runModelScan(args []string) error {
	fs := flag.NewFlagSet("scan-models", flag.ContinueOnError)
	root := fs.String("root", "/models", "operator-mounted model root")
	catalogPath := fs.String("catalog", "", "approved model catalog path")
	requireReadOnly := fs.Bool("require-read-only", true, "reject a writable model root")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *catalogPath == "" {
		return errors.New("-catalog is required")
	}
	catalog, err := ReadModelCatalog(*catalogPath)
	if err != nil {
		return err
	}
	report, err := ScanModels(*root, catalog, *requireReadOnly)
	if err != nil {
		return err
	}
	return writeJSON(report)
}

func writeJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
