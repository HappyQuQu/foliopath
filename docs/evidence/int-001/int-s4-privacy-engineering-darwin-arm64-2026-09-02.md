# INT-S4 privacy engineering evidence — darwin/arm64

- Date: 2026-09-02
- Branch: `aifeature`
- Status: **engineering subproof passed / Release No-Go**
- Scope: `INT-406` log, diagnostic/API, live SQLite deletion and cache-boundary checks

## Implemented boundary

The application JSON logger now redacts attribute keys for face IDs, person names, embeddings/vectors, similarity scores,
bounding boxes, landmarks and crop paths in addition to the existing query, path, credential, SQL and raw-error fields.
The regression writes unique sensitive canaries through each of those attribute classes and verifies that none appears in
the serialized log.

Every SQLite connection now requires `PRAGMA secure_delete=ON`. A file-backed regression first proves that a unique canary
was materialized in the database, deletes it, truncates the WAL, and then verifies that the canary is absent from the live
database, WAL and shared-memory artifacts. Existing face clear tests independently verify that derived clear removes
observations/clusters while retaining explicitly classified manual state, manual relationship clear removes anchors/audit
state while retaining explicitly classified derived data, and both leave the synthetic original-media hash and mtime
unchanged.

The existing wire and architecture tests continue to reject face embeddings, vectors, crop data, internal paths and model
scores from the face API, keep semantic query text out of request/error logs, expose only closed media-diagnostic fields,
and prevent the face capability or SQLite adapter from writing logs directly. No face crop cache exists in the production
composition; ordinary thumbnails are media-preview derived state and are not represented as biometric analysis data.

## Executed verification

```text
make test-intelligent-media-privacy
```

The target passed all four focused suites:

- application structured-log redaction;
- face/semantic/diagnostic HTTP wire privacy;
- SQLite derived/manual clear plus secure-delete live-file residue;
- face privacy architecture fitness.

The broader affected-package check also passed:

```text
go test -count=1 ./internal/app ./internal/store/sqlite ./tests/architecture
```

## Boundary of this evidence

This result covers live application files after a successful clear and WAL truncation. It does not claim erasure from
operator backups, storage snapshots, filesystem journals or intentionally retained state; those have separate documented
retention and destruction responsibilities. It also does not supply privacy, compliance or security owner approval, a
lawful biometric ground-truth manifest, or final model/package evidence. Therefore it advances but does not complete
`INT-406`, and the S4 release decision remains No-Go.
