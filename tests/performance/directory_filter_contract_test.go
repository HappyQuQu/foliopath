package performance_test

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

const directoryFilterContractCount = 10_000

// TestDirectoryFilterContractSpike keeps the UIF-S1 query decision honest
// without depending on a production migration or repository implementation.
// It models the accepted schema/query shape: a capability-derived normalized
// name key, parent-scoped natural-order index, literal substring filtering,
// and keyset-compatible order.
func TestDirectoryFilterContractSpike(t *testing.T) {
	database, err := sql.Open("sqlite", "file:directory-filter-contract?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open in-memory SQLite: %v", err)
	}
	defer database.Close()
	database.SetMaxOpenConns(1)

	ctx := context.Background()
	if _, err := database.ExecContext(ctx, `
        CREATE TABLE directories (
            id INTEGER PRIMARY KEY,
            library_id INTEGER NOT NULL,
            parent_id INTEGER NOT NULL,
            name TEXT NOT NULL,
            natural_name_key BLOB NOT NULL,
            search_name_key TEXT NOT NULL
        );
        CREATE INDEX directories_browse_children
            ON directories(library_id, parent_id, natural_name_key, name, id);
    `); err != nil {
		t.Fatalf("create directory-filter contract schema: %v", err)
	}

	transaction, err := database.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin directory fixture: %v", err)
	}
	statement, err := transaction.PrepareContext(ctx, `
        INSERT INTO directories(
            id, library_id, parent_id, name, natural_name_key, search_name_key
        ) VALUES (?, 1, 1, ?, ?, ?)
    `)
	if err != nil {
		t.Fatalf("prepare directory fixture: %v", err)
	}
	for id := 1; id <= directoryFilterContractCount; id++ {
		name := fmt.Sprintf("目录 %05d", id)
		searchKey := name
		if id == directoryFilterContractCount {
			name = "京都 Needle 10000"
			searchKey = "京都 needle 10000"
		}
		if _, err := statement.ExecContext(ctx, id, name, []byte(name), searchKey); err != nil {
			t.Fatalf("insert directory %d: %v", id, err)
		}
	}
	if err := statement.Close(); err != nil {
		t.Fatalf("close directory fixture statement: %v", err)
	}
	if err := transaction.Commit(); err != nil {
		t.Fatalf("commit directory fixture: %v", err)
	}

	const filteredQuery = `
        SELECT id, name
        FROM directories
        WHERE library_id = ? AND parent_id = ?
          AND instr(search_name_key, ?) > 0
        ORDER BY natural_name_key, name, id
        LIMIT ?
    `
	var plan string
	if err := database.QueryRowContext(
		ctx,
		"EXPLAIN QUERY PLAN "+filteredQuery,
		1, 1, "needle", 51,
	).Scan(new(int), new(int), new(int), &plan); err != nil {
		t.Fatalf("explain directory filter: %v", err)
	}
	if !strings.Contains(plan, "directories_browse_children") ||
		!strings.Contains(plan, "library_id=? AND parent_id=?") {
		t.Fatalf("directory filter plan = %q, want parent-scoped browse index", plan)
	}

	run := func() {
		rows, err := database.QueryContext(ctx, filteredQuery, 1, 1, "needle", 51)
		if err != nil {
			t.Fatalf("query directory filter: %v", err)
		}
		defer rows.Close()
		found := 0
		for rows.Next() {
			var id int64
			var name string
			if err := rows.Scan(&id, &name); err != nil {
				t.Fatalf("scan directory filter: %v", err)
			}
			found++
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("iterate directory filter: %v", err)
		}
		if found != 1 {
			t.Fatalf("directory filter returned %d rows, want 1", found)
		}
	}

	for warmup := 0; warmup < 5; warmup++ {
		run()
	}
	latencies := make([]time.Duration, 0, 50)
	for sample := 0; sample < cap(latencies); sample++ {
		started := time.Now()
		run()
		latencies = append(latencies, time.Since(started))
	}
	p50 := percentile(latencies, 50)
	p95 := percentile(latencies, 95)
	t.Logf(
		"directory-filter contract: directories=%d plan=%q p50=%s p95=%s",
		directoryFilterContractCount, plan, p50, p95,
	)
	if p95 > 100*time.Millisecond {
		t.Fatalf("directory-filter p95 = %s, budget = 100ms", p95)
	}
}
