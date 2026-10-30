# FolioPath Agent Instructions

These instructions apply to the entire repository. Keep changes small, preserve the
rules below, and update this file when an accepted architecture decision changes
them.

## Product invariants

- Treat original media as read-only by default. ADR-0042 accepts an opt-in,
  authenticated media deletion capability; CR-2026-060 limits new actions to permanent deletion. CR-2026-058 / DELETE-S1/S2
  contracts and backend plus S3 production-component UI have passed local gates;
  general release remains gated by DELETE-S4; ADR-0044 accepts the explicitly
  authorized 222/9090 candidate deployment. Ordinary scanning, browsing, analysis
  and library configuration removal remain read-only; never edit media contents.
  Writes require files-owned anchored operations, explicit deployment authorization
  and administrator opt-in.
- ADR-0044 supersedes ADR-0043: deletion runs in the existing main-process worker;
  do not add Landlock, a mutation subprocess, or host/PVE boot changes. Keep openat2
  and the single media mount. External programs must not move the target directory
  or its ancestors during deletion; concurrent external rename containment is not
  promised. Deletion owns durable operations and jobs admission.
- CR-2026-060 supersedes the normal trash UX: new deletions are permanent after
  one confirmation dialog, without typed words or an extra checkbox. New UI
  settings always disable recycling and automatic retention. Remove trash UI and
  admission APIs; retain only historical bytes/records and already accepted task
  recovery. Never purge those files or reinterpret accepted operations implicitly.
- Treat the filesystem as the source of truth for media existence and hierarchy;
  the SQLite catalog and thumbnails are derived state. User curation such as
  favorites and manual tags is application state under `/app/data`, never media truth.
- Support multiple libraries configured in the UI. A library is a named directory at
  or below the container's allowed media root (`/library`) and includes descendants by
  default. The empty allowed-root-relative value represents `/library` itself and then
  overlaps every possible additional library.
- Support exactly one media mount target at `/library`. It is read-only by default;
  ADR-0042/0044 and CR-2026-060 permit explicit writable deployment for permanent deletion.
  It may itself be a bind mount or volume, but its descendants must be ordinary
  directories, not nested
  volumes, bind mounts, or other mount points. UI libraries select directories inside
  that one tree; they are not deployment units.
- Require non-empty, instance-unique library names. Allow renaming, but keep the
  library root immutable in the MVP; changing it requires removal and recreation.
- Reject overlapping library roots to prevent duplicate indexing.
- Removing a library removes only its configuration, index, curation assignments,
  jobs, and cache; it never modifies original media.
- Mark an unavailable or unreadable library as offline. Do not interpret it as empty
  and do not purge its index.
- Store a library root relative to `/library`; store asset/directory paths relative to
  that library root. Only the authenticated administration API may expose the former,
  and media APIs expose only IDs or library-relative paths. Never expose host paths.
- Ship the first stable release with first-run single-administrator setup, sessions,
  logout, and CSRF protection. Before that exists, bind previews to loopback or put
  them behind a trusted authenticating proxy; never support anonymous LAN mode.
  Once authentication exists, support direct HTTP on a trusted LAN; TLS termination
  and public-network exposure are deployment concerns outside FolioPath's required
  single-container topology.

## Archive image sources

- ADR-0040 accepts ZIP-S1: read-only ZIP containers project into
  virtual catalog directories; explicit member locators never become disk paths.
  `media` owns the v1:z fingerprint subtype; `files` owns bounded parsing and
  request-scoped temporary extraction under `/app/data/tmp/zip`. Preserve ordinary
  v1 identities, the single media mount, existing pagination, thumbnail quota and
  scan-generation ownership. Failed archives preserve their previous subtree.
  ZIP members stay excluded from AI admission/coverage until a later accepted slice.
- ADR-0041 extends the same source contract to read-only 7z. files owns bounded
  descriptor-only 7zip subprocesses, with prlimit applied before decoder startup.
  Preserve all ZIP identities and global temporary quotas; do not extract archive
  paths into the media tree or admit 7z members to AI.

## Managed AI deployment

- ADR-0039 accepts explicit `standard-v1` automatic model preparation in the existing AI container and an independently embedded main-image profile. Fixed sources/hashes only; never trust live identity as configuration. AI alone may write its dedicated model cache and use network downloads; media/data and the bounded inference scheduler remain unchanged. Legacy offline deployments remain supported. New cache manifests get new identities; never relabel old results. Promotion still requires native amd64/arm64 matching verification.

## Repository and dependency boundaries

- Use a modular Go monolith with one HTTP process and an embedded React SPA.
  ADR-0020 accepts a FolioPath-owned thin AI image based on the pinned upstream
  runtime, with verified private cache snapshots and a versioned identity protocol.
  This is still exactly two deployment units; release approval requires evidence.
  POST-MVP-5 ADR-0019 accepts an optional independent AI inference container.
  The main application retains SQLite, auth, media access and bounded job ownership;
  the AI service gets no media/database mount and no default public port. Internal
  contracts and model-generation compatibility must precede production integration.
- ADR-0021 accepts one bounded, request-scoped inference subprocess inside the AI
  container, not a third deployment unit. Reap it before releasing admission; retain
  the existing resource budget until measured evidence supports a change. Automatic
  tagging owns a separate model identity and bounded job capability. Curation owns
  manual-tag priority and persistent removal suppression; merge tags and advance the
  automatic-tag checkpoint in one fenced SQLite transaction. Never reinterpret old
  candidate suggestions as automatic tags or silently relabel old index identities.
- `cmd/foliopath` is a minimal process entry point. Dependency composition,
  lifecycle, and graceful shutdown belong in `internal/app`.
- Put business capabilities under `internal/` (`auth`, `settings`, `library`,
  `catalog`, `scanner`, `thumbnail`, `media`, `deletion`, and `jobs`). Each capability defines the
  interfaces it uses.
- ADR-0031: `internal/search` owns fused ranking, literal/semantic source evidence
  and snapshot-bound keyset cursors. Reuse catalog normalization, semantic encoding
  and the cursor mechanism. Stream scoped candidates with a bounded result heap;
  never merge only first pages in the browser or claim OCR without an OCR contract.
- ADR-0037: OCR plaintext search uses source-bound, current enabled-library results
  and the deployment OCR identity. Update derived FTS documents with OCR results
  and checkpoints atomically, invalidate snapshot cursors on visibility/source
  changes, and recheck after HTTP hydration. Return only bounded original-text
  line excerpts; never HTML, raw exceptions, confidence percentages or video OCR.
- Put HTTP handlers, DTOs, and middleware under `internal/api`. Handlers call services;
  they must not query SQLite, resolve filesystem paths, or invoke media tools directly.
- Put SQLite implementations under `internal/store/sqlite`, pure relative-path lexical
  rules under `internal/pathpolicy`, and filesystem/open/identity boundary code under
  `internal/files`. Wire concrete implementations only in `internal/app`.
- Put authenticated opaque-token encoding under `internal/cursor`. Each resource
  capability still owns its cursor payload, query binding, ordering, and validation;
  do not duplicate cryptographic cursor codecs.
- Put the Go embed wrapper for Vite output under `internal/webassets`; Vite writes its
  generated production assets to `internal/webassets/dist` before the Go build.
- Dependencies point inward: adapters implement capability-owned interfaces; business
  packages do not import API or concrete storage packages.
- Do not create `pkg/` without a real external consumer. Do not create generic
  `utils`, `common`, or `helpers` packages.
- Do not add a separate worker, Redis, PostgreSQL, GraphQL, SSR, WebSockets, Nginx, or
  another deployable service without an accepted ADR.

## Architecture governance and delivery order

- Treat `docs/architecture.md` and `docs/architecture/` as the system architecture
  entry point. Product scope comes from the accepted PRD/RQ baseline, structural
  decisions come from accepted ADRs, and executable API/data contracts become
  authoritative once their source files exist. Implementation must not silently
  override any of them.
- The current MVP scope is frozen. A new user-visible capability defaults to a later
  version. Moving it into the MVP requires an explicit change record, architecture and
  release impact analysis, and either deferring an item of comparable cost/risk or an
  explicitly accepted scope-budget exception. Safety invariants are never traded away.
- Every new user-visible capability, architecture change, or high-risk slice must name
  its requirement/quality ID, target version and stage, owner, contracts, and evidence.
  A routine fix inside an approved slice may use a lightweight record linking the
  existing Gate, affected invariant, and regression test. Work with no traceable scope
  remains a proposal and must not grow production code.
- Work backend-first inside each vertical slice: define the use case and failure
  semantics, accept the OpenAPI/data contracts, implement and integration-test the
  backend, then build product UI against the generated client. Only the minimal shell,
  tokens, primitives, and time-boxed disposable prototype directly required by an
  approved S0/S1 slice may proceed earlier; prototypes stay out of the production
  import graph. Feature UI must not invent behavior on mocks or bypass a missing
  backend contract.
- Give every policy and state transition exactly one canonical owner. Other packages
  call that owner through its public service, type, or narrow interface; they do not
  copy validation, error mapping, query keys, retry policy, transaction rules, path
  policy, or task state machines.
- Record a new ADR before changing deployment units, core technology, trust or
  persistence boundaries, module dependency direction, API compatibility policy,
  transaction ownership, job consistency, or shared frontend architecture.
- Add architecture fitness checks as the relevant source tree appears.
  `make arch-check` is the repository-level entry point; a rule described only in
  prose is not considered fully enforced until its planned executable check exists
  and is included in the documented local verification surface.

## Requirements, prototypes, and implementation consistency

- Treat an explicitly accepted prototype as acceptance evidence for an approved
  requirement, not as an independent product branch. Record the requirement/change
  ID and distinguish synthetic fixture data from supported production behavior.
- Before implementation, map every visible control, state, label, navigation target,
  and scope boundary (for example global versus per-library) to its owning product
  requirement and backend contract. A prototype must not imply a writable setting or
  successful action that the accepted contract cannot provide.
- Update accepted requirements, user flows, UI design, stable Storybook states,
  production components, and regression tests in the same change whenever their
  shared behavior changes. Do not declare the slice complete while those artifacts
  describe different information architecture or semantics.
- Once a design is accepted, its stable Storybook story must render the production
  component with clearly labelled synthetic contract fixtures. Remove or archive the
  disposable duplicate screen; do not maintain a second implementation that can
  drift from production.
- Document every intentional difference between requirement, accepted prototype, and
  production implementation, including the missing contract or deferred Gate. Add an
  executable regression or architecture check when the consistency rule can be
  automated.

## Filesystem safety

- All media filesystem access must go through `internal/files`.
- Public APIs accept library IDs, media IDs, and relative paths only; never accept an
  arbitrary absolute path.
- Lexically normalize each configured library root and requested target with
  `internal/pathpolicy`, then resolve and open it only through the kernel-anchored
  `internal/files` boundary. Do not use a pathname pre-check as proof of containment.
- Reject traversal, duplicate-encoding traversal, NUL bytes, and symlink escapes.
- On Linux, anchor all real media opens at the `/library` directory file descriptor and
  use `openat2` with `RESOLVE_BENEATH | RESOLVE_NO_SYMLINKS | RESOLVE_NO_XDEV`. If the
  kernel, flags, seccomp, or LSM cannot provide that boundary, fail closed; never
  silently fall back to `os.Root`, realpath, or device/inode checks.
- Reject every nested mount below `/library`, including same-device and self bind
  mounts. If independent host volumes are needed, the operator must first present one
  mount-crossing-free host root and mount it once at `/library`; FolioPath does not
  promise or configure a particular union technology.
- Treat non-Linux filesystem adapters as development-only evidence. They do not prove
  the Linux-equivalent mount boundary or expand the supported release platforms.
- Do not follow directory symlinks during scanning unless an accepted ADR defines safe
  behavior. Never cross filesystem boundaries implicitly.
- Keep `/library` read-only in the default deployment. ADR-0042 is the sole accepted
  finite write exception: explicit deployment authorization plus administrator opt-in,
  deletion-owned use cases and files-owned anchored Linux writes. The trash
  area is an ordinary reserved directory within the same media filesystem, excluded
  from every media discovery/read path; never add a mount or copy/delete fallback.
  Read-boundary evidence alone does not prove write safety. DELETE-S1/S2 local
  gates permit production-component UI development; DELETE-S4 must pass before
  production writable deployment. Store the database, settings, scan
  state, operation/recovery records and generated cache under `/app/data`.

## Scanning and jobs

- Scan directories incrementally; do not load a complete tree into memory and do not
  start an unbounded goroutine per entry.
- Index every readable directory, including empty directories, and maintain direct
  and recursive media counts without hiding directories that contain no media.
- Use bounded global queues and explicit concurrency limits shared by all libraries.
- Identify catalog entries by `library_id` plus normalized relative path.
- Record each full scan with a generation. Upsert discovered entries in bounded
  transactions and remove stale entries only after the entire scan succeeds.
- A cancelled, failed, offline, or partially unreadable scan must preserve previously
  indexed entries. Record the failure for display and retry.
- Correctness must not depend on filesystem watchers. Startup, manual, and scheduled
  reconciliation scans remain authoritative.
- Run a full scan on library creation and startup, and default scheduled full scans
  to every 24 hours with a UI setting to change or disable them.
- Provide cooperative scan cancellation. Preserve the last reliable index and any
  safely committed additions, and require a later full scan to reconcile.
- Scan hidden items, but skip the maintained system-derived/recycle directory list
  and expose skipped counts.
- Make jobs restart-safe and idempotent. Write derived files to a temporary file and
  atomically rename them into place.
- ADR-0027: only signal-marked planned shutdown may hand back automatic-tag,
  image-semantic and face claims after both processor and heartbeat exit. Release
  the exact fenced claim in one audited transaction, undoing only its attempt
  charge and preserving all committed progress and identities. Cancellation wins;
  crashes retain the three-attempt limit. Legacy bootstrap handoff is offline-only
  after verified owner stop/full backup and requires an exact bounded snapshot.
- ADR-0030: automatic tags may recover only explicitly classified temporary tag
  service outages within one 60-second recovery window and at most two extra
  analyses per image. Probe the exact identity before resending, retain the same
  claim/checkpoint, respect cancellation, and never retry identity/output failures
  or revive terminal jobs implicitly. Existing missing scheduling preserves runs.
- ADR-0022 permits an in-flight face job to wait at most 60 seconds per transient
  catalog scan, without holding a transaction; recheck cancellation, availability,
  and the existing lease every second. New admission still requires a ready library.
  Persist only fixed per-asset failure codes with the fenced checkpoint. Keep an
  unresolved failure eligible for missing-only retry even when an older successful
  result exists; never invent historical failure reasons or return raw exceptions.

## SQLite

- Use SQLite in WAL mode for a single FolioPath instance. Do not place `/app/data` on
  SMB or NFS unless support is explicitly designed and documented.
- ADR-0023: never open a live Docker Desktop SQLite database from the macOS/Windows
  host, even for an integrity query. Online checks must share the owning container's
  Linux kernel and lock domain: use `docker exec <app> /app/foliopath database-check`.
  Do not run migrations, checkpoints, or host SQLite against active data. Stop the
  owner before copying a complete backup; host inspection is for detached backups.
- ADR-0038: Docker Desktop deployment data must use a prepared Linux named volume,
  with verified stopped backup/copy before switching. Keep the original host data as
  a detached historical copy; never synchronize it against the live volume. Native
  Linux local bind deployments remain subject to ADR-0023. Preserve every existing
  AI overlay and the single media mount (read-only unless ADR-0042 was explicitly
  enabled); never change its mode implicitly during data maintenance.
- Every directory, asset, thumbnail, and scan row is scoped by `library_id`; enforce
  uniqueness for active normalized relative paths within a library. ADR-0042 plans
  separate trash namespaces so a replacement at the original path cannot inherit
  the recycled asset's identity or curation; the schema change still requires S1/S2.
- Serialize writes, batch scan mutations, keep transactions bounded, and set a busy
  timeout. Do not hold a transaction while doing filesystem or media processing.
- Use keyset/cursor pagination with a stable unique tie-breaker; do not use `OFFSET`
  for media browsing.
- Keep thumbnail bytes out of SQLite.
- Add migrations; never edit a migration that may have shipped.

## Media processing

- ADR-0036 accepts opt-in OCR runtime and authenticated library management. Bind
  a deployment-pinned independent identity; keep libraries off until explicit
  admin requests. Reuse ContentService/files, imagevips, the existing ML client,
  resourcecontrol and jobs.WorkerPool. Recheck originals after processing; only
  definitive preprocessor input failures become bad-source rows. Temporary scan
  waits are bounded to 60 seconds with lease/cancellation checks, without holding
  transactions or repeating inference. Busy deferral and signal-marked, audited
  post-worker handoff preserve progress; crashes keep the three-attempt limit.
  No OCR plaintext/search/UI or automatic missing admission is implied.

- ADR-0035 accepts dormant OCR source-bound storage and fenced per-library
  jobs/epochs with bounded recovery and clear batches. Keep it disabled and
  uncomposed until runtime/API gates. Persist only normalized bounded output or
  fixed source-failure codes, atomically with its checkpoint; never turn model/IO
  faults into bad media. Disable, cancellation, generation change and clear fence
  old writers. Preserve all other capabilities and curation.

- ADR-0034 permits the isolated OCR producer overlay and an explicit request-local
  route adapter around the frozen scheduler. Delegate all admission, deadlines and
  reaping to the original owner; only the private legacy-core admission category
  may map OCR to visual. Public requests/child kind/cache/identity stay OCR. Keep
  all legacy source bytes, identities and actual outputs verified; the OCR runtime
  and immutable overlay digest cover the extension. Never add a second live queue,
  silently edit legacy hashes, or compose persistent OCR jobs/search before gates.

- ADR-0033 additionally accepts OCR's independent `ocr-rgb-png-v1` image
  preprocessor in imagevips, consuming already-opened streams only. Bind the
  canonical OCR input contract explicitly; share existing native-memory admission
  and close image references before releasing it. Do not change legacy semantic,
  face or tag pixels/identities, auto-enable OCR, or treat IO/native failures as
  bad source media. Production OCR composition/persistence remains gated.

- ADR-0032 accepts only the isolated OCR identity/result contract and Go transport
  consumer. OCR owns its own fingerprint and strict bounded plaintext/quadrilateral
  results; remoteml shares the existing HTTP admission/deadline, verifies deployment
  expectations before sending image bytes, and never retries or labels unknown
  service failures as bad media. No production OCR service registration, source-bound
  persistence/jobs, search/UI or activation is authorized by this transport slice.

- POST-MVP-5 model downloads follow ADR-0016: only a deployment-owned fixed
  catalog may supply HTTPS URLs and exact hashes. Browser APIs accept opaque
  download IDs, never URLs; downloads reuse the bounded model-install worker
  and verified atomic managed store. Downloaded is not activated or release-ready.

- ADR-0017 permits an explicit deployment-owned model download HTTP/HTTPS proxy.
  ADR-0018 additionally permits authenticated administrator inherit/direct/custom
  overrides persisted under /app/data with revision conflict protection. Only the
  custom proxy origin is exposed to administrators, never the deployment URL.
  Connectivity tests use one fixed verified source, never a browser-supplied target.
  Never inherit ambient proxy variables. Proxy mode delegates destination DNS/address
  enforcement to the trusted proxy; preserve HTTPS verification, target literal-IP/local-name
  rejection, fixed redirect hosts, hashes and bounded installs. No implicit direct fallback.

- Use libvips through `govips` for image metadata and thumbnails.
- Use `ffprobe` for video metadata and `ffmpeg` for poster extraction.
- The MVP format contract is JPEG, PNG, WebP, and GIF for image indexing/thumbnails,
  plus MP4, MOV, and MKV for video indexing/posters. SVG, HEIC/HEIF, AVIF, and RAW
  are outside the MVP contract.
- The MVP serves supported originals with HTTP Range requests and does not transcode
  video.
- Apply timeouts, cancellation, input-size safeguards, and bounded concurrency to all
  subprocesses. Treat media files and metadata as untrusted input.
- ADR-0024: imagevips owns one process-wide, cancellable large-image native-memory
  admission shared by thumbnails, face and semantic preprocessing. Acquire before
  compressed/native working allocations and release only after ImageRef/thread
  cleanup and idle-memory reclamation. Do not replace it with per-feature/library
  gates, change pixels/model identities, or add reads to caller-owned cancellation
  and no-progress streams merely to classify a resource lane.
- ADR-0025: deployment may select only the reviewed English 512-dimensional or
  XLM-Roberta-Large-Vit-B-16Plus 640-dimensional semantic profile. Verify model,
  dimension and identity together; never relabel old vectors. Keep legacy face/tag
  identities and workers separate from the multilingual overlay while sharing one
  inference scheduler. Its bounded snapshots use dedicated disposable disk scratch,
  never media/database storage; large-model tests do not prove the small-model budget.
- ADR-0028: semantic image preprocessing defaults to the unchanged v3 contract.
  The explicit deployment profile `png-v4` permits bounded 64 MiB / 40 MP PNG
  inputs with pre-load header checks and the same native-memory admission. Bind
  this profile to a distinct semantic identity; never relabel old vectors or
  switch in-flight jobs implicitly. Face and automatic-tag preprocessing remain
  unchanged by that semantic setting. ADR-0029 permits an independent deployment
  `tag.image_preprocessing` profile (empty v3 or png-v4), bound to both the tag
  fingerprint and its preprocessor; never inherit the semantic selection. Tag
  identity changes reprocess enabled libraries without clearing manual tags or
  removal suppression, and require verified stopped backup before deployment.
- ADR-0026: curation owns the reviewed automatic-category catalog and exact
  Chinese/English aliases. Localized display must never rename stored tags or infer
  manual provenance. The photo-tag overlay binds its allowed keys and execution
  sources to an independent tag identity, retains legacy semantic/face identities,
  and reuses the one inference scheduler, transform owner and bounded child.
  Display-language preferences never trigger reprocessing; category changes do.
- Derive cache keys from asset identity plus source fingerprint and transform version
  so stale thumbnails cannot be reused.
- Default the thumbnail cache quota to 10 GiB, make it UI-configurable, evict
  reconstructible entries by LRU at the configured waterline, and preserve a safe
  free-space margin for SQLite and temporary writes.

## Frontend

- All product screens, including settings, logs, empty states and errors, use
  user-facing language: state what happened, its practical effect and the next
  useful action. Do not display implementation commentary, ranking algorithms,
  protocol/model identities, hashes, request IDs, internal paths, raw error codes,
  cursor/revision/checkpoint mechanics or contract limitations. Keep these in
  developer documentation and service diagnostics. Show task progress in task
  management; collection/search pages retain actual match evidence and concise
  actionable warnings, without healthy-index statistics or algorithm notes.
  Preserve meaningful destructive-action scope and recovery guidance. Apply the
  same rule in Chinese, English and stable production-component stories.

- Use the latest verified Node.js 24 LTS patch for local tools, CI and Docker
  build/test stages. Pin its exact version in `.node-version`, align package
  engines/npm packageManager and the generated lockfile, use that version file in
  CI, and pin Docker tags plus verified multi-platform digests. Do not use floating
  `latest`/`lts` tags or rewrite historical verification evidence when upgrading.
- Use React, TypeScript, Vite, React Router, TanStack Query, and TanStack Virtual.
- Keep server state in TanStack Query and navigable library, directory, search, sort,
  and cursor state in the URL. Do not add Redux without an accepted ADR.
- Browse large collections with cursor pagination and virtualization; never fetch or
  render an unbounded media list.
- Preserve DOM order, keyboard navigation, focus visibility, semantic controls, and
  useful alt text in gallery and viewer flows.
- Default to an adaptive grid and offer a remembered masonry layout. Desktop uses a
  fixed directory sidebar; mobile uses a directory drawer. Keep motion restrained.
- Every product page must adapt to the full workspace available after persistent
  navigation. Do not wrap an entire desktop page in an arbitrary fixed `max-width`
  that leaves large unused side gutters. Keep readable line-length limits on prose,
  forms, or focused tasks only; let page shells, dashboards, settings summaries,
  tables, and collections use fluid width and add responsive columns where useful.
  Verify changed pages at narrow mobile, tablet, ordinary desktop, and at least a
  1920px-wide desktop viewport; wide-screen evidence is required when layout changes.
- Search defaults to the current library and can switch to the current directory
  (optionally recursive) or all libraries. Directories sort naturally by name;
  recursive/search views default to descending file modification time.
- Ship Simplified Chinese and English, follow the browser language by default, and
  expose a language setting.
- The MVP viewer includes fit, zoom/pan, 1:1, previous/next, fullscreen, and basic
  file information. It excludes a full EXIF panel, explicit download button, and
  mobile swipe navigation.
- The library picker may browse only server-approved directories below `/library`.
- Follow the dependency direction defined in `docs/architecture/frontend.md`:
  `app/routes -> features -> shared components/lib/styles`. Shared components must not
  import features, routes, generated API code, or server-state clients. A feature must
  not import another feature's private files.
- Keep one canonical implementation of each shared UI primitive and cross-feature
  interaction pattern. Do not create feature-local buttons, icon buttons, fields,
  dialogs, sheets, menus, banners, toasts, loading/empty/error states, or virtual-list
  controllers when the design system already owns that semantic role. Add a reviewed
  variant to the owner instead of copying it.
- Put design tokens in the central token source and component styles beside their
  canonical component. Feature code must not invent raw colors, spacing scales,
  radii, shadows, z-index layers, motion constants, or a second theme mechanism.
- Access HTTP only through the generated client boundary and its hand-written domain
  adapters. Query-key factories, URL codecs, API-error mapping, and invalidation rules
  each have one owner; do not duplicate them across screens.
- Document shared primitives and stable states in the component workbench. New or
  changed primitives require behavior and accessibility tests before first use.
  Establish focused visual regression before their API stabilizes or a second feature
  consumes them; complete the theme/locale/breakpoint matrix before release candidate.

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

- 2026-10-27 user decision: test sources, fixtures, snapshot baselines, and test
  output are local-only and excluded by `.gitignore`. Untrack existing files while
  preserving local copies. Do not force-add new test files. Keep test commands and
  CI configuration, but a fresh clone lacks the local test suite and cannot
  reproduce the full verification surface below without those files.
- 2026-10-27 explicit release exception: retain only the 25 allowlisted build and
  native-release verification inputs in `.gitignore`, as recorded in
  `docs/changes/RELEASE-2026-10-27-directory-layout.md`. These include production
  Dockerfile dependencies, native image smoke/recovery helpers, and their minimal
  synthetic fixtures. Ordinary test suites, snapshots, and outputs stay local-only.

- ADR-0026: build each release image once on its native architecture, verify the same
  immutable digest, and promote tags only from two matching successful records.
  Preserve SPDX SBOM and SLSA provenance when merging indexes. Build caches are
  disposable CI state and never replace verification or mount application/media data.

- Put unit tests beside Go and TypeScript source. Put cross-component tests in
  `tests/integration`, browser tests in `tests/e2e`, and synthetic media in
  `tests/fixtures`.
- Tests must not read or modify a developer's real media library. Use temporary
  directories and synthetic fixtures.
- Cover path traversal and symlink escape, overlapping libraries, offline roots,
  interrupted scans, stale-generation cleanup, cursor stability, Range responses,
  and corrupt media.
- Use roughly 100,000 media items and 10,000 directories on a four-core, 4 GiB target
  environment as the primary capacity acceptance tier. Treat it as a target to
  benchmark, never as an already-proven performance claim.
- Before completing a change, run the relevant available checks and report exactly
  what ran. The expected full verification surface is:

  ```sh
  make fmt
  make arch-check
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
- Keep `docs/architecture/traceability.md` current for implemented FR/NFR slices;
  update module ownership, delivery Gate, and fitness-function status in the same
  change when their facts change.
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
