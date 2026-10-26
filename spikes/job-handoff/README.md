# Planned analysis-job handoff maintenance

See ADR-0027 and `docs/changes/FIX-2026-10-23-planned-job-handoff.md`.
This Linux-only adapter is not part of the product command/API surface.

Build `./spikes/job-handoff` for the local container architecture with CGO disabled.
`--snapshot` opens the fixed `/app/data/foliopath.db` read-only and emits at most
32 running automatic-tag/image-semantic/face claim snapshots. It may be executed
inside the owning container; it does not migrate or write the database.

For **legacy bootstrap only**, stop the owner and verify Docker reports not running,
take a complete `/app/data` and configuration backup, then take a **fresh stopped**
snapshot inside a disposable Linux maintenance container. Supply that exact JSON
on stdin with `--confirm-owner-stopped` in the same offline Linux lock domain.
The adapter applies migrations and calls the canonical Store transition once.
A mismatch, duplicate or stale snapshot fails closed; the batch transaction rolls
back, but schema migration may already have completed. Do not run it against a
live owner, reset attempts by hand, or use it to revive historical failed jobs.
Preserve the old AI identity while its jobs continue. Restore the entire stopped
backup and matching old configuration if rollback is necessary.

Never use host SQLite on mounted Docker data, including stopped original fixture
data. Prefer container-side checks; host inspection is for detached backups only.
`docker_smoke.py` creates 80 synthetic images and fresh data, exercises two real
legacy SIGKILL recoveries, exact offline handoff, three candidate SIGTERM restarts,
and a final SIGKILL that must still exhaust the original crash budget. Only this
stopped synthetic fixture's lease timestamp is advanced in a Linux container to
avoid waiting 120 seconds each turn. Attempt counters are never injected.
All reads/writes use the Docker Linux kernel; hashes, manual tags and favorites
must survive. Environment variables name explicitly verified images/cache/tool
and an existing **synthetic** PNG. They must never point at a real media library.
Test containers/data are retained for evidence; identify the printed random project
before cleaning its containers and disposable scratch. Do not remove other projects.
