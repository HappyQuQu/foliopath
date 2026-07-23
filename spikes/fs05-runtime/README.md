# FS-05 runtime spike

This directory is an isolated Stage 0 deployment probe. It is intentionally not
`cmd/foliopath` and exposes no product API. It validates the final-image shape,
SQLite migration startup, health behavior, non-root/read-only boundaries,
graceful shutdown, offline backup/restore, repeated-start upgrade compatibility,
and failure-closed behavior for unwritable, full, or corrupt data directories.

Run from the repository root:

```sh
docker build -f spikes/fs05-runtime/Dockerfile \
  -t foliopath-fs05:local --build-arg VERSION=local .
spikes/fs05-runtime/verify.sh foliopath-fs05:local
```

The probe uses only synthetic configuration and temporary Docker volumes. It
does not read a developer media library.
