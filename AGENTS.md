# FolioPath Agent Instructions

These instructions apply to the entire repository. Keep changes small, preserve the
rules below, and update this file when an accepted architecture decision changes
them.

## Product invariants

- Treat original media as read-only. Never move, rename, edit, or delete it.
- Treat the filesystem as the source of truth for media existence and hierarchy;
  SQLite and thumbnails are derived state.
- Support multiple libraries configured in the UI. A library is a named subdirectory
  below the container's allowed media root (`/library`) and includes descendants by
  default.
- Reject overlapping library roots to prevent duplicate indexing.
- Removing a library removes only its configuration, index, jobs, and cache.
- Mark an unavailable or unreadable library as offline. Do not interpret it as empty
  and do not purge its index.
- Store and expose only library-relative paths. Never expose host paths.

## Repository and dependency boundaries

- Use a modular Go monolith with one HTTP process and an embedded React SPA.
- `cmd/foliopath` is a minimal process entry point. Dependency composition,
  lifecycle, and graceful shutdown belong in `internal/app`.
- Put business capabilities under `internal/` (`library`, `catalog`, `scanner`,
  `thumbnail`, `media`, and `jobs`). Each capability defines the interfaces it uses.
- Put HTTP handlers, DTOs, and middleware under `internal/api`. Handlers call services;
  they must not query SQLite, resolve filesystem paths, or invoke media tools directly.
- Put SQLite implementations under `internal/store/sqlite` and path-boundary code
  under `internal/files`. Wire concrete implementations only in `internal/app`.
- Put the Go embed wrapper for Vite output under `internal/webassets`; Vite writes its
  generated production assets to `internal/webassets/dist` before the Go build.
- Dependencies point inward: adapters implement capability-owned interfaces; business
  packages do not import API or concrete storage packages.
- Do not create `pkg/` without a real external consumer. Do not create generic
  `utils`, `common`, or `helpers` packages.
- Do not add a separate worker, Redis, PostgreSQL, GraphQL, SSR, WebSockets, Nginx, or
  another deployable service without an accepted ADR.

## Filesystem safety

- All media filesystem access must go through `internal/files`.
- Public APIs accept library IDs, media IDs, and relative paths only; never accept an
  arbitrary absolute path.
- Clean and resolve a configured library root and every accessed target. Verify the
  resolved path remains within `/library` and within that library root.
- Reject traversal, duplicate-encoding traversal, NUL bytes, and symlink escapes.
- Do not follow directory symlinks during scanning unless an accepted ADR defines safe
  behavior. Never cross filesystem boundaries implicitly.
- Keep `/library` read-only. Store the database, settings, scan state, and generated
  cache under `/app/data`.

## Scanning and jobs

- Scan directories incrementally; do not load a complete tree into memory and do not
  start an unbounded goroutine per entry.
- Use bounded global queues and explicit concurrency limits shared by all libraries.
- Identify catalog entries by `library_id` plus normalized relative path.
- Record each full scan with a generation. Upsert discovered entries in bounded
  transactions and remove stale entries only after the entire scan succeeds.
- A cancelled, failed, offline, or partially unreadable scan must preserve previously
  indexed entries. Record the failure for display and retry.
- Correctness must not depend on filesystem watchers. Startup, manual, and scheduled
  reconciliation scans remain authoritative.
- Make jobs restart-safe and idempotent. Write derived files to a temporary file and
  atomically rename them into place.

## SQLite

- Use SQLite in WAL mode for a single FolioPath instance. Do not place `/app/data` on
  SMB or NFS unless support is explicitly designed and documented.
- Every directory, asset, thumbnail, and scan row is scoped by `library_id`; enforce
  uniqueness for normalized relative paths within a library.
- Serialize writes, batch scan mutations, keep transactions bounded, and set a busy
  timeout. Do not hold a transaction while doing filesystem or media processing.
- Use keyset/cursor pagination with a stable unique tie-breaker; do not use `OFFSET`
  for media browsing.
- Keep thumbnail bytes out of SQLite.
- Add migrations; never edit a migration that may have shipped.

## Media processing

- Use libvips through `govips` for image metadata and thumbnails.
- Use `ffprobe` for video metadata and `ffmpeg` for poster extraction.
- The MVP serves supported originals with HTTP Range requests and does not transcode
  video.
- Apply timeouts, cancellation, input-size safeguards, and bounded concurrency to all
  subprocesses. Treat media files and metadata as untrusted input.
- Derive cache keys from asset identity plus source fingerprint and transform version
  so stale thumbnails cannot be reused.

## Frontend

- Use React, TypeScript, Vite, React Router, TanStack Query, and TanStack Virtual.
- Keep server state in TanStack Query and navigable library, directory, search, sort,
  and cursor state in the URL. Do not add Redux without an accepted ADR.
- Browse large collections with cursor pagination and virtualization; never fetch or
  render an unbounded media list.
- Preserve DOM order, keyboard navigation, focus visibility, semantic controls, and
  useful alt text in gallery and viewer flows.
- The library picker may browse only server-approved directories below `/library`.

## API and generated files

- Version public endpoints under `/api/v1` and keep `api/openapi.yaml` authoritative.
- API errors use one documented response shape. Do not leak absolute paths, SQL,
  subprocess output, or stack traces.
- Generated OpenAPI clients, generated SQL, embedded frontend assets, and other files
  marked generated must not be edited manually. Change their source and regenerate.
- Generation must be deterministic. Commit generated source required to build, but do
  not commit generated Vite output, runtime databases, logs, temporary files, or media
  caches.

## Tests and verification

- Put unit tests beside Go and TypeScript source. Put cross-component tests in
  `tests/integration`, browser tests in `tests/e2e`, and synthetic media in
  `tests/fixtures`.
- Tests must not read or modify a developer's real media library. Use temporary
  directories and synthetic fixtures.
- Cover path traversal and symlink escape, overlapping libraries, offline roots,
  interrupted scans, stale-generation cleanup, cursor stability, Range responses,
  and corrupt media.
- Before completing a change, run the relevant available checks and report exactly
  what ran. The expected full verification surface is:

  ```sh
  make fmt
  make generate-check
  make lint
  make test
  make test-integration
  make test-e2e
  ```

- Do not claim a check passed unless it was executed successfully. If a target does
  not exist yet or cannot run in the environment, say so explicitly.

## Documentation

- Keep `docs/README.md` as the documentation index and distinguish accepted
  decisions, MVP plans, proposals, and future ideas.
- Update `docs/product-requirements.md` and the affected flow in
  `docs/user-flows.md` when user-visible scope or acceptance behavior changes.
- Update `docs/ui-design.md` for navigation, responsive, accessibility, component,
  or motion behavior changes; do not let implementation become the only UI spec.
- Update `docs/api-design.md` while the API is still a proposal. Once it exists,
  keep `api/openapi.yaml` authoritative and make the prose design point to it.
- Update `api/openapi.yaml` for API behavior or schema changes.
- Add an ADR under `docs/adr/` before changing an architectural constraint in this
  file; record context, decision, alternatives, and consequences.
- Update `docs/security.md` for trust-boundary, path, authentication, or exposure
  changes; update `docs/data-model.md` for persistence semantics.
- Update `docs/deployment.md` for volume, permission, backup, restore, upgrade, or
  health-check changes; update `docs/testing-strategy.md` for verification gates.
- Keep feasibility assumptions and spike evidence in `docs/feasibility-study.md`,
  update `docs/risk-register.md` when risk or mitigation changes, and do not begin a
  feature that fails the applicable gates in `docs/development-readiness.md`.
- Update `README.md` for user-visible behavior, supported formats, configuration,
  volume mounts, deployment, or operational changes.
