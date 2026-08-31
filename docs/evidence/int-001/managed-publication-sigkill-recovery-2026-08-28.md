# Managed package publication process-kill recovery

Status: **production publisher filesystem boundary passed; full install-worker
recovery remains incomplete**.

Two process-level tests exercise `files.ManagedModelStore` rather than manually
creating simulated staging/final paths:

- [`TestManagedModelStoreReconcilesRealPublishAfterProcessKill`](../../../internal/files/modelstore_process_test.go)
  kills a helper while the production publisher is blocked inside the first
  model-file copy. The parent observes one real `.partial-*` directory and no
  visible final package. A newly constructed store removes exactly that staging
  directory during reconciliation.
- [`TestManagedModelStoreRetainsPublishedFinalAfterProcessKill`](../../../internal/files/modelstore_process_test.go)
  lets the production publisher finish its no-replace rename and parent fsync,
  then kills the helper before any database registration exists. Startup
  reconciliation reports one known final, deletes nothing, and a subsequent
  idempotent publish verifies and reuses the exact content-addressed package.

The test host was already below the contract's `max(1 GiB, filesystem 10%)`
free-space reserve even though it had tens of GiB available. The helper therefore
injects only a deterministic 2 GiB capacity-probe result through an unexported
store seam. Production construction still binds that seam to the real
filesystem probe. Staging, manifest/file writes, SHA-256 checks, fsync,
no-replace rename, kill, and reconciliation all use production code.

Focused verification:

```sh
go test ./internal/files \
  -run 'TestManagedModelStore(ReconcilesRealPublishAfterProcessKill|RetainsPublishedFinalAfterProcessKill|PublishKillHelper)$' \
  -count=1
```

This proves the production managed-filesystem publication boundary on the
current development host. It does not yet combine the durable install operation,
publisher and model-registration transaction in one killed process; nor does it
replace Linux/amd64, real ENOSPC, nested-mount or reviewed-model evidence.
