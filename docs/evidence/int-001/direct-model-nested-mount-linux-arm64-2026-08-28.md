# Direct model nested-mount boundary on Linux/arm64

Date: 2026-08-28
Scope: `INT-215`, fixed `/models` direct-package boundary

## Environment

- Docker Desktop native Linux/arm64
- Project-pinned Go builder base:
  `mirror.gcr.io/library/golang@sha256:4ee9ffa999b4583ce281939cdff828763083610292f252279a0cee77473bd9a7`
- Isolated ephemeral container with `CAP_SYS_ADMIN`; repository mounted read-only
- Bind mounts exist only inside the disposable container mount namespace

## Executed verification

```text
go test -tags fsboundary ./tests/integration \
  -run '^TestAIModelSourceRejects(NestedPackageMountDuringScan|MountReplacementAfterScan)$' \
  -count=1 -v

--- PASS: TestAIModelSourceRejectsNestedPackageMountDuringScan
--- PASS: TestAIModelSourceRejectsMountReplacementAfterScan
PASS
ok github.com/HappyQuQu/foliopath/tests/integration
```

The first case bind-mounts an external directory onto a direct-child `.foliomodel` before scanning. Production
`files.Root`/`ModelSource` rejects package capture with `ErrCrossDevice` and never enumerates mounted contents.

The second case scans an ordinary package, records its opaque source identity, then bind-mounts a replacement package
onto the same directory. Reopening a manifest-owned file through the old identity fails with `ErrCrossDevice`; no
replacement bytes are returned. This covers same-filesystem mount transitions, where device/inode comparison alone is
not sufficient.

## Remaining limits

This closes the direct-model nested-mount subcase on native Linux/arm64. It does not provide Linux/amd64 evidence,
managed-store kernel `ENOSPC`, final reviewed-model validation, or a native inference process-kill run.
