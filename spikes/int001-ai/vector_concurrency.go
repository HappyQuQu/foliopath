package main

import (
	"container/heap"
	"context"
	"database/sql"
	"encoding/binary"
	"errors"
	"math/rand/v2"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

type VectorConcurrencyReport struct {
	SchemaVersion            int               `json:"schema_version"`
	GeneratedAt              string            `json:"generated_at"`
	Environment              VectorEnvironment `json:"environment"`
	Items                    int               `json:"items"`
	Dimensions               int               `json:"dimensions"`
	StorageFormat            string            `json:"storage_format"`
	BatchSize                int               `json:"batch_size"`
	BatchDelayMS             float64           `json:"batch_delay_ms"`
	BrowseQueries            int               `json:"browse_queries"`
	SearchQueries            int               `json:"search_queries"`
	BaselineBrowseP95MS      float64           `json:"baseline_browse_p95_ms"`
	ConcurrentBrowseP95MS    float64           `json:"concurrent_browse_p95_ms"`
	BrowseP95DegradationRate float64           `json:"browse_p95_degradation_rate"`
	ConcurrentSearchP95MS    float64           `json:"concurrent_search_p95_ms"`
	RowsAfterCancellation    int               `json:"rows_after_cancellation"`
	RowsAfterRestart         int               `json:"rows_after_restart"`
	DatabaseBytes            int64             `json:"database_bytes"`
	WALBytes                 int64             `json:"wal_bytes"`
	HeapSysBytes             uint64            `json:"heap_sys_bytes"`
	PeakRSSBytes             uint64            `json:"peak_rss_bytes,omitempty"`
	Caveats                  []string          `json:"caveats"`
}

func BenchmarkVectorConcurrency(
	items, dimensions, batchSize, browseQueries, searchQueries int,
	seed int64,
	batchDelay time.Duration,
	storageFormat string,
) (VectorConcurrencyReport, error) {
	report := VectorConcurrencyReport{
		SchemaVersion: 1,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Environment: VectorEnvironment{
			GOOS: runtime.GOOS, GOARCH: runtime.GOARCH,
			GoVersion: runtime.Version(), GOMAXPROCS: runtime.GOMAXPROCS(0),
		},
		Items: items, Dimensions: dimensions, StorageFormat: storageFormat,
		BatchSize:     batchSize,
		BatchDelayMS:  milliseconds(batchDelay),
		BrowseQueries: browseQueries, SearchQueries: searchQueries,
		Caveats: []string{
			"synthetic vectors measure SQLite exact-scan concurrency and recovery only",
			"peak_rss_bytes is read from Linux /proc/self/status when available",
			"a native run on one architecture does not satisfy the dual-architecture gate",
		},
	}
	if items < 100 || dimensions < 4 || batchSize < 1 || browseQueries < 1 || searchQueries < 1 ||
		(storageFormat != "float32" && storageFormat != "float16") {
		return report, errors.New("invalid vector concurrency benchmark arguments")
	}
	file, err := os.CreateTemp("", "foliopath-int001-vector-concurrency-*.db")
	if err != nil {
		return report, err
	}
	databasePath := file.Name()
	if err := file.Close(); err != nil {
		return report, err
	}
	defer os.Remove(databasePath)
	defer os.Remove(databasePath + "-shm")
	defer os.Remove(databasePath + "-wal")
	db, err := sql.Open("sqlite", databasePath)
	if err != nil {
		return report, err
	}
	defer db.Close()
	db.SetMaxOpenConns(4)
	if _, err := db.Exec(`
		PRAGMA journal_mode=WAL;
		PRAGMA synchronous=NORMAL;
		PRAGMA busy_timeout=5000;
		CREATE TABLE assets (
			id INTEGER PRIMARY KEY,
			vector BLOB
		);
	`); err != nil {
		return report, err
	}
	if err := seedAssetIDs(db, items, batchSize); err != nil {
		return report, err
	}
	baseline, err := browseLatency(db, browseQueries)
	if err != nil {
		return report, err
	}
	report.BaselineBrowseP95MS = percentile(baseline, 0.95)

	ctx, cancel := context.WithCancel(context.Background())
	signalled := make(chan struct{})
	backfillDone := make(chan error, 1)
	go func() {
		backfillDone <- backfillVectors(
			ctx, db, items, dimensions, batchSize, seed,
			items/2, signalled, batchDelay, storageFormat,
		)
	}()
	select {
	case <-signalled:
	case err := <-backfillDone:
		cancel()
		return report, err
	case <-time.After(2 * time.Minute):
		cancel()
		return report, errors.New("vector backfill did not reach cancellation boundary")
	}
	searchDone := make(chan struct {
		latencies []float64
		err       error
	}, 1)
	go func() {
		latencies, err := searchLatency(
			db, dimensions, searchQueries, seed, storageFormat,
		)
		searchDone <- struct {
			latencies []float64
			err       error
		}{latencies, err}
	}()
	concurrentBrowse, err := browseLatency(db, browseQueries)
	if err != nil {
		cancel()
		return report, err
	}
	searchResult := <-searchDone
	if searchResult.err != nil {
		cancel()
		return report, searchResult.err
	}
	cancel()
	if err := <-backfillDone; !errors.Is(err, context.Canceled) {
		return report, errors.New("cancelled vector backfill did not stop cooperatively")
	}
	report.ConcurrentBrowseP95MS = percentile(concurrentBrowse, 0.95)
	if report.BaselineBrowseP95MS > 0 {
		report.BrowseP95DegradationRate =
			(report.ConcurrentBrowseP95MS - report.BaselineBrowseP95MS) /
				report.BaselineBrowseP95MS
	}
	report.ConcurrentSearchP95MS = percentile(searchResult.latencies, 0.95)
	if err := db.QueryRow(`SELECT count(*) FROM assets WHERE vector IS NOT NULL`).Scan(
		&report.RowsAfterCancellation,
	); err != nil {
		return report, err
	}
	if report.RowsAfterCancellation < items/2 || report.RowsAfterCancellation >= items {
		return report, errors.New("cancelled vector backfill exposed an unexpected row count")
	}
	if err := backfillVectors(
		context.Background(), db, items, dimensions, batchSize,
		seed+1, 0, nil, 0, storageFormat,
	); err != nil {
		return report, err
	}
	if err := db.QueryRow(`SELECT count(*) FROM assets WHERE vector IS NOT NULL`).Scan(
		&report.RowsAfterRestart,
	); err != nil {
		return report, err
	}
	if report.RowsAfterRestart != items {
		return report, errors.New("restarted vector backfill did not complete")
	}
	if _, err := db.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		return report, err
	}
	if info, err := os.Stat(databasePath); err == nil {
		report.DatabaseBytes = info.Size()
	} else {
		return report, err
	}
	if info, err := os.Stat(databasePath + "-wal"); err == nil {
		report.WALBytes = info.Size()
	} else if !errors.Is(err, os.ErrNotExist) {
		return report, err
	}
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	report.HeapSysBytes = memory.HeapSys
	report.PeakRSSBytes = linuxPeakRSSBytes()
	return report, nil
}

func linuxPeakRSSBytes() uint64 {
	contents, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return 0
	}
	for line := range strings.SplitSeq(string(contents), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 3 || fields[0] != "VmHWM:" || fields[2] != "kB" {
			continue
		}
		kilobytes, err := strconv.ParseUint(fields[1], 10, 64)
		if err == nil {
			return kilobytes * 1024
		}
	}
	return 0
}

func seedAssetIDs(db *sql.DB, items, batchSize int) error {
	ctx := context.Background()
	for start := 0; start < items; start += batchSize {
		end := min(start+batchSize, items)
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		statement, err := tx.PrepareContext(ctx, `INSERT INTO assets(id) VALUES (?)`)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
		for id := start; id < end; id++ {
			if _, err := statement.ExecContext(ctx, id); err != nil {
				statement.Close()
				_ = tx.Rollback()
				return err
			}
		}
		if err := statement.Close(); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func backfillVectors(
	ctx context.Context,
	db *sql.DB,
	items, dimensions, batchSize int,
	seed int64,
	signalAfter int,
	signalled chan<- struct{},
	batchDelay time.Duration,
	storageFormat string,
) error {
	rng := rand.New(rand.NewPCG(uint64(seed), uint64(seed)^0x9e3779b97f4a7c15))
	didSignal := false
	for start := 0; start < items; start += batchSize {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		end := min(start+batchSize, items)
		vectors := make([][]byte, end-start)
		for index := range vectors {
			vectors[index] = encodeStoredVector(
				randomUnitVector(rng, dimensions), storageFormat,
			)
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		statement, err := tx.PrepareContext(
			ctx,
			`UPDATE assets SET vector = ? WHERE id = ? AND vector IS NULL`,
		)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
		for index, vector := range vectors {
			if _, err := statement.ExecContext(ctx, vector, start+index); err != nil {
				statement.Close()
				_ = tx.Rollback()
				return err
			}
		}
		if err := statement.Close(); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		if !didSignal && signalAfter > 0 && end >= signalAfter {
			close(signalled)
			didSignal = true
		}
		if batchDelay > 0 {
			timer := time.NewTimer(batchDelay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
	}
	return nil
}

func browseLatency(db *sql.DB, queries int) ([]float64, error) {
	latencies := make([]float64, 0, queries)
	after := -1
	for query := 0; query < queries; query++ {
		started := time.Now()
		rows, err := db.Query(
			`SELECT id FROM assets WHERE id > ? ORDER BY id LIMIT 100`, after,
		)
		if err != nil {
			return nil, err
		}
		count := 0
		for rows.Next() {
			if err := rows.Scan(&after); err != nil {
				rows.Close()
				return nil, err
			}
			count++
		}
		if err := rows.Close(); err != nil {
			return nil, err
		}
		if count == 0 {
			after = -1
		}
		latencies = append(latencies, milliseconds(time.Since(started)))
	}
	sort.Float64s(latencies)
	return latencies, nil
}

func searchLatency(
	db *sql.DB,
	dimensions, queries int,
	seed int64,
	storageFormat string,
) ([]float64, error) {
	rng := rand.New(rand.NewPCG(uint64(seed), uint64(seed)^0xd1b54a32d192ed03))
	latencies := make([]float64, 0, queries)
	for queryIndex := 0; queryIndex < queries; queryIndex++ {
		query := randomUnitVector(rng, dimensions)
		started := time.Now()
		rows, err := db.Query(`SELECT id, vector FROM assets WHERE vector IS NOT NULL ORDER BY id`)
		if err != nil {
			return nil, err
		}
		top := make(scoreHeap, 0, 20)
		for rows.Next() {
			var id int
			var blob []byte
			if err := rows.Scan(&id, &blob); err != nil {
				rows.Close()
				return nil, err
			}
			score, err := dotStoredVector(query, blob, dimensions, storageFormat)
			if err != nil {
				rows.Close()
				return nil, err
			}
			top.offer(scoredID{id: id, score: score}, 20)
		}
		if err := rows.Close(); err != nil {
			return nil, err
		}
		runtime.KeepAlive(top)
		latencies = append(latencies, milliseconds(time.Since(started)))
	}
	sort.Float64s(latencies)
	return latencies, nil
}

func encodeStoredVector(vector []float32, storageFormat string) []byte {
	if storageFormat == "float32" {
		return encodeVector(vector)
	}
	blob := make([]byte, len(vector)*2)
	for index, value := range vector {
		binary.LittleEndian.PutUint16(blob[index*2:], float32ToHalf(value))
	}
	return blob
}

func dotStoredVector(
	query []float32,
	blob []byte,
	dimensions int,
	storageFormat string,
) (float32, error) {
	if storageFormat == "float32" {
		vector, err := decodeVector(blob, dimensions)
		if err != nil {
			return 0, err
		}
		return dot(query, vector), nil
	}
	if len(blob) != dimensions*2 {
		return 0, errors.New("invalid float16 vector blob length")
	}
	var score float32
	for index, queryValue := range query {
		score += queryValue * halfToFloat32(
			binary.LittleEndian.Uint16(blob[index*2:]),
		)
	}
	return score, nil
}

var _ heap.Interface = (*scoreHeap)(nil)
