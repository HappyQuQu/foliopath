# Face analysis process-kill recovery — Darwin/arm64

Status: **S2C restart-safety evidence; not face quality or release evidence**.

## Bound source and environment

- Source commit before the uncommitted S2C evidence changes:
  `fdede8c99b3dcef52cc2d9851551521fc0340652`
- Host: Darwin 25.6.0, arm64
- Go: `go1.26.5 darwin/arm64`
- Database: a temporary SQLite file created by the integration test; no developer
  media library was read or modified.

## Command

```sh
go test ./tests/integration \
  -run 'TestFaceAnalysis(RecoversCheckpointAfterProcessKill|ClaimKillHelper)$' \
  -count=1
```

Result: `ok`.

## Observed invariant

The child process opened the production SQLite store and claimed a durable face
analysis job with a one-second lease. The parent waited for the explicit `CLAIMED`
barrier, killed that process, reopened the same database after lease expiry, and ran
the production recovery owner.

Recovery requeued exactly one job and the next claim preserved checkpoint `101`,
completed count `1/2`, and attempt count `2`, while advancing claimed revision from
`2` to `3`. The higher revision prevents the killed worker from committing stale
results if it were ever to resume.

This closes the face-analysis queue process-crash/restart subcase of `INT-250`. It
does not prove detector or embedding quality, native model runtime recovery,
Linux dual-architecture behavior, or the final 4-CPU/4-GiB joint capacity tier.
