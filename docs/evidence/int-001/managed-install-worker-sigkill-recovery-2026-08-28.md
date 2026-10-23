# Managed install worker process-kill recovery

Date: 2026-08-28
Scope: `INT-209`, `INT-215`; synthetic reviewed package, real process kill

## Exercised boundary

A child process opens the production SQLite store, persists and claims a managed install request, and runs the
production `aimodel.InstallWorker`, `aimodel.Installer`, and `files.ManagedModelStore`. The publisher completes file
validation, staging writes, fsync, and no-replace atomic publication, then blocks before returning to
`RegisterInstalled`. The parent sends an OS process kill at that exact boundary.

A fresh process-equivalent set of store/services then:

1. recovers the durable running operation as `failed/operation_interrupted`;
2. reconciles one complete final and no staging directory;
3. matches its hash to the reviewed synthetic catalog entry and current architecture;
4. revalidates manifest, regular/no-exec files, sizes, and SHA-256 values; and
5. registers exactly one available but inactive model without changing an active pointer.

## Executed verification

```text
go test ./internal/files -run 'TestManagedInstall(RecoversPublishedOrphanAfterProcessKill|KillHelper)$' -count=1 -v
PASS
ok github.com/HappyQuQu/foliopath/internal/files
```

The test injects only the disk-space result because the development volume is below the frozen 10% reserve. Queue,
SQLite transactions, worker execution, package copy/hash/fsync/rename, process kill, reconciliation, validation, and
model registration all use production implementations.

A follow-up unit boundary verifies that post-preflight managed destination `ENOSPC` and `EDQUOT` retain the kernel
error and map to the stable `insufficient_space` operation code. The destination writer owns that mapping; an identical
error returned by the package source reader remains a source failure. This validates classification only and is not a
substitute for the pending real constrained-filesystem run.

## Remaining limits

The package is a synthetic reviewed fixture, not an approved production model. Real managed-store kernel `ENOSPC` and
direct nested mounts are covered by separate Linux/arm64 evidence. Native Linux/amd64, a final reviewed model, and a
process kill during native inference remain explicit `INT-215` blockers.
