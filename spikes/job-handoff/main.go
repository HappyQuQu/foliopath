// Offline Linux maintenance adapter for the one-time upgrade from a binary
// without graceful handoff support. It is not shipped in the product image.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"time"

	"github.com/HappyQuQu/foliopath/internal/jobs"
	store "github.com/HappyQuQu/foliopath/internal/store/sqlite"
)

func main() {
	snapshot := flag.Bool("snapshot", false, "read-only running-claim snapshot")
	stopped := flag.Bool("confirm-owner-stopped", false, "operator confirmed Docker owner is stopped and fully backed up")
	flag.Parse()
	if runtime.GOOS != "linux" || flag.NArg() != 0 || (*snapshot == *stopped) {
		fail()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if *snapshot {
		db, err := sql.Open("sqlite", "file:/app/data/foliopath.db?mode=ro&_pragma=query_only(1)&_pragma=busy_timeout(3000)")
		if err != nil {
			fail()
		}
		defer db.Close()
		claims := []jobs.HandoffClaim{}
		for _, def := range [][3]string{{"automatic_tags", "autotag_jobs", "claim_revision"}, {"semantic", "semantic_jobs", "claimed_revision"}, {"face", "face_analysis_jobs", "claimed_revision"}} {
			rows, err := db.QueryContext(ctx, `SELECT j.id,j.operation_id,j.`+def[2]+`,j.attempt_count,j.checkpoint_id,o.completed_items FROM `+def[1]+` j JOIN ai_model_operations o ON o.id=j.operation_id WHERE j.state='running' AND o.state='running' ORDER BY j.id LIMIT 33`)
			if err != nil {
				fail()
			}
			for rows.Next() {
				claim := jobs.HandoffClaim{Kind: def[0]}
				if err := rows.Scan(&claim.JobID, &claim.OperationID, &claim.ClaimRevision, &claim.AttemptCount, &claim.CheckpointID, &claim.CompletedItems); err != nil {
					fail()
				}
				claims = append(claims, claim)
				if len(claims) > 32 {
					fail()
				}
			}
			if rows.Err() != nil {
				fail()
			}
			rows.Close()
		}
		if err := json.NewEncoder(os.Stdout).Encode(claims); err != nil {
			fail()
		}
		return
	}
	var claims []jobs.HandoffClaim
	decoder := json.NewDecoder(io.LimitReader(os.Stdin, 65537))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&claims); err != nil {
		fail()
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		fail()
	}
	db, err := store.Open(ctx, "/app/data/foliopath.db", store.Options{MaxOpenConnections: 1})
	if err != nil {
		fail()
	}
	defer db.Close()
	if err := db.HandoffStoppedAnalysisJobs(ctx, claims, time.Now()); err != nil {
		fail()
	}
	fmt.Printf("handed off %d exact stopped claims\n", len(claims))
}

func fail() {
	fmt.Fprintln(os.Stderr, "analysis handoff refused or failed; verify stopped owner and exact snapshot")
	os.Exit(1)
}
