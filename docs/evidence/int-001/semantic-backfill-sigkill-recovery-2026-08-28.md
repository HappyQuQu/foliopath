# Semantic backfill process-kill recovery

Status: **restart-safe queue subproof passed; S2A remains incomplete**.

The integration test
[`TestSemanticBackfillRecoversCheckpointAfterProcessKill`](../../../tests/integration/semantic_backfill_restart_test.go)
uses a file-backed migrated SQLite database and a real helper process. The
fixture starts with a queued semantic backfill whose durable progress is one of
two items complete at checkpoint asset ID 101.

The helper opens the same database through the production SQLite store, claims
the job with a 100 ms lease, prints the claimed opaque job ID, and then blocks.
The parent sends an operating-system process kill rather than cancelling a Go
context or closing the store gracefully. After the lease expires, a new store
instance runs the production recovery transaction and claims the requeued job.

Observed result:

- recovery reported exactly one requeued job and no terminal interruption;
- the second claim advanced the attempt count from 1 to 2;
- the claim revision advanced from 2 to 3, invalidating the killed worker;
- checkpoint 101 and completed/total progress 1/2 survived unchanged;
- the job was not reset to the beginning and was not marked complete.

Focused verification:

```sh
go test ./tests/integration \
  -run 'TestSemanticBackfill(RecoversCheckpointAfterProcessKill|ClaimKillHelper)$' \
  -count=1
```

This closes process-kill lease recovery for the semantic backfill queue. It
does not prove strong-kill recovery during managed model publication, native
inference, or a final reviewed-model end-to-end run.
