// Bounded read-only audit. Run only inside the owning synthetic Docker container.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"os"
	"time"

	_ "modernc.org/sqlite"
)

func main() {
	memory := flag.Bool("memory", false, "read the current synthetic container's cgroup peak, without opening SQLite")
	flag.Parse()
	if *memory {
		value, err := os.ReadFile("/sys/fs/cgroup/memory.peak")
		if err != nil {
			panic(err)
		}
		if _, err := os.Stdout.Write(value); err != nil {
			panic(err)
		}
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db, err := sql.Open("sqlite", "file:/app/data/foliopath.db?mode=ro&_pragma=query_only(1)&_pragma=busy_timeout(3000)")
	if err != nil {
		panic(err)
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, `SELECT g.id,g.state,COUNT(e.asset_id) FROM semantic_generations g LEFT JOIN semantic_embeddings e ON e.generation_id=g.id GROUP BY g.id,g.state ORDER BY g.id LIMIT 20`)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	type generation struct {
		ID    string `json:"id"`
		State string `json:"state"`
		Count int    `json:"count"`
	}
	result := []generation{}
	for rows.Next() {
		var row generation
		if err := rows.Scan(&row.ID, &row.State, &row.Count); err != nil {
			panic(err)
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		panic(err)
	}
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		panic(err)
	}
}
