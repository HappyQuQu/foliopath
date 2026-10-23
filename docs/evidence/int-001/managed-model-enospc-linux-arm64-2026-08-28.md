# Managed model real ENOSPC on Linux/arm64

Date: 2026-08-28
Scope: `INT-215`, managed publisher post-preflight disk exhaustion

## Environment

- Docker Desktop native Linux/arm64
- Project-pinned Go builder base:
  `mirror.gcr.io/library/golang@sha256:4ee9ffa999b4583ce281939cdff828763083610292f252279a0cee77473bd9a7`
- Disposable container with `CAP_SYS_ADMIN`
- 128 KiB tmpfs mounted only inside the container test namespace

## Executed verification

```text
go test -tags fsboundary ./internal/files \
  -run '^TestManagedModelStoreActualENOSPCFailsClosed$' -count=1 -v

--- PASS: TestManagedModelStoreActualENOSPCFailsClosed
PASS
ok github.com/HappyQuQu/foliopath/internal/files
```

The test injects only the capacity-probe result so execution proceeds beyond the already-tested reserve preflight.
Production `ManagedModelStore` then attempts to publish a synthetic package whose first model file is 512 KiB onto the
128 KiB tmpfs. The destination write/sync path receives the real kernel `ENOSPC`.

Assertions prove that the returned error retains both `aimodel.ErrInsufficientSpace` and `unix.ENOSPC`, no
`.foliomodel` final becomes visible, and deferred cleanup removes every `.partial-*` staging directory.

## Remaining limits

This closes the managed-store real-ENOSPC subcase on Linux/arm64. Native inference process-kill is covered separately
for the current candidate image encoder. This evidence does not provide native Linux/amd64 results or approve a
production model.
