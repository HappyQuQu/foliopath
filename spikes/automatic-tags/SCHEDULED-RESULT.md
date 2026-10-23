# ADR-0021 bounded inference: local same-budget evidence

Date: 2026-10-22. Scope: CR-2026-031 / FR-AUTOTAG-001, sidecar and internal
`foliopath-tag-v1` only. **Result: bounded real inference passed at the existing
2304 MiB memory / 1200 MiB tmpfs limits.** This does not establish general tag
accuracy, AMD64 support, full application integration or release acceptance.

## What changed after the confirmed OOM

The [five-session experiment](CO-RESIDENCY-RESULT.md) failed with OOM. Keeping a
single inference child but retaining all 1,168,346,001 snapshot bytes reduced live
sessions, yet the maximum-edge/queued run still reached the memory hard ceiling.
That intermediate configuration was not accepted as a stable memory fix.

The final implementation has one parent-owned scheduler, one running child and at
most one waiting request. Startup verifies every accepted model file, then releases
the temporary snapshots. A child re-verifies/copies only the current capability's
complete pinned subset, loads its sessions and infers. Copy, load and inference
share the request's 26-second deadline including queue wait. Parent cleanup kills
and waits for the child before deleting its request directory and releasing the
slot. No async background copy can outlive a cancelled request. The runtime keeps
the fixed upstream CPU ORT options, including its memory arena; the improvement
does not rely on changing accuracy/model execution options or Python GC.

The final source/cache inputs are fixed; no remote Python or inference download is
used. Current requests see the full CLIP shared config and selected encoder subtree,
the complete `buffalo_l` subtree, or the three JoyTag files. Entries preserve their
exact accepted paths, byte sizes and hashes. Global compatibility identities still
bind the full accepted manifests, not an arbitrary per-request subset.

## Exact image, identities and files

Candidate local image ID:
`sha256:fa7750b57500031a3282181e3b728646717885b339a0640a76b121468165e0b8`.
Tag: `foliopath-inference:automatic-tags-candidate`. Linux ARM64, fixed Immich
v3.1.0 base digest, CPU provider. Weights are not bundled into the image.

| Identity field | Value |
| --- | --- |
| Shared runtime contract SHA256 | `be1d1baaccb2f8532d9df872483b0453e66e1221d5d1768432357dc76c1cfc8a` |
| CLIP/face cache SHA256 | `e6d671f9d9fada323fd77a02d217349f46af8e103eccb297b1ebcac10112efe8` |
| Tag model SHA256 | `f85b7130e6e549b5b0822537007b7482e8c4c8e754c8d9a5bee08e27050e1097` |
| Tag policy SHA256 | `10a57ceb4c4221fb6996bdb8ed5f8b16899e5d9ccfbfcc620ca3bb7566795918` |
| Tag-only cache SHA256 | `f40fd0e4b9bcb69b901353bccd62bd5822bdcff3974e2604f846cc1f28523d73` |

The independent staging directory is
`/Volumes/extended/foliopath/model-experiments/automatic-tags-integration-cache`.
All 49 legacy and 3 JoyTag files were checked before copy and after staging, then
again by the service. It is retained for subsequent isolated application tests.
The active `/Volumes/extended/foliopath/model-cache` was not changed.

## Real final run

`PYTHONDONTWRITEBYTECODE=1 python3 spikes/automatic-tags/run-scheduled.py` created
`fp-tags-scheduled-60994b3d4cd7`. Start `2026-10-22T10:51:03.196040042Z`, finish
`2026-10-22T10:51:43.171505631Z`; Docker exit 0, `OOMKilled=false`.
The disposable container had no network, no ports, read-only root/source mounts,
3 CPU, 2304 MiB RAM/swap ceiling, 256 PIDs, 1200 MiB verified-cache tmpfs and
128 MiB `/tmp`. There was no `/library`, database, Docker socket, live service API
call or server222 connection.

The only media was the existing fictional-adult `azure-01.png` repository fixture
and a derived synthetic 1600×1600 resize (2,070,931 PNG bytes). Tests used the real
FastAPI application/lifespan through TestClient, not fake model responses.

- Three text → image → face → tag cycles: **12 real successful inferences**.
- CLIP text and image vectors: 512 finite values, normalized.
- Face: one real detected synthetic face each cycle, finite normalized 512-vector.
- Tag: `flower` 0.5773978, `dress` 0.4784993, `window` 0.4403350, verified headers.
  These are observed model outputs, not asserted ground truth.
- Maximum supported 1600×1600 image: all four request types completed. The text
  request has no image but was repeated in the same sequence.
- Two overlapping maximum-image requests: one real child plus one waiter; a third
  text request returned **503 busy**; both admitted requests completed.
- **18 successful requests total**, with children reaped and per-request snapshots
  empty after completion; no model sessions were retained between requests.

| Measurement | Final observed value |
| --- | ---: |
| cgroup memory ceiling | 2,415,919,104 bytes / 2304 MiB |
| cgroup memory peak | 2,185,469,952 bytes / about 2084.2 MiB |
| Parent process peak RSS | 109,420 KiB |
| Maximum child process peak RSS | 869,276 KiB |
| `memory.events` max / high / oom / oom_kill | all 0 |
| Text cold request range, three cycles | 1755.203–1891.836 ms |
| Image cold request range, three cycles | 1817.645–2160.899 ms |
| Face cold request range, three cycles | 1898.468–1939.801 ms |
| Tag cold request range, three cycles | 2703.865–2787.943 ms |
| 1600-edge tag request | 2763.383 ms |

Cgroup memory includes source page cache, snapshots and process memory; RSS is not
an interchangeable measure. Earlier snapshot-lifecycle runs peaked at 2,187,194,368
and 2,123,567,104 bytes and also passed; those preceded the real TCP cancellation
fix below and are not the final image. These are bounded smoke observations with normal
host cache variability, not a production capacity or latency SLA. The inherited
image healthcheck reported unhealthy because this isolated TestClient probe does
not open a TCP listener; the actual `/ping`/identity handlers and lifespan ran in
process. It is not evidence of a deployed service health failure.

## Unit and failure verification

The final image passed **28/28** tests with:

```sh
docker run --rm --network none --read-only --memory 2304m --memory-swap 2304m \
  --cpus 3 --pids-limit 256 --tmpfs /tmp:rw,nosuid,size=128m \
  --env PYTHONDONTWRITEBYTECODE=1 --entrypoint python \
  foliopath-inference:automatic-tags-candidate \
  -m unittest discover -s /opt/foliopath-inference -p 'test_*.py' -v
```

Coverage includes actual native subprocess cancellation and timeout, real TCP
disconnect cancellation, queue-inclusive deadline, one active/one waiting admission,
overload, malformed/oversized child output, failed real-worker cache snapshot,
source symlink/hash checks, family-subset completeness, fixed tag identity/headers,
input rejection, candidate-vocabulary rejection, finite/safe/unique/bounded output.
`test_tcp_disconnect.py` starts a real uvicorn loopback listener in the isolated
network-disabled container. It sends a valid multipart image request, waits for a
real native child to start, then shuts down/closes the TCP socket. Within a one-second
assertion deadline the child is OS-reaped and admission is idle; a following request
then succeeds. This covers direct HTTP disconnect, not a reverse-proxy matrix.

Independent review caught a genuine failure in the preceding candidate: with the
old `BaseHTTPMiddleware` identity wrapper, socket shutdown left the child alive
after 3.6 seconds, despite the mocked disconnect test passing. The identity wrapper
is now pure ASGI so it does not interpose a receive task/channel between the request
and transport disconnect. It also wraps the body limiter so ready-service 413/408
responses retain the identity headers; the Go client need not weaken verification.
The new true-TCP regression and all 18 model requests passed after rebuilding; do
not use the superseded `fa42dc...` image/runtime identity.

An initial unit run found a child-stdout high-watermark wait deadlock on oversized
output. It was fixed by cancelling pending I/O, killing the child, draining its
remaining pipe in bounded chunks and then waiting. That regression now passes.
The exact stuck test container was stopped and removed; no application service
was stopped. All subsequent disposable probe/test containers and tmpfs were
removed. Only the verified staging cache above is retained.

`git diff --check -- services/inference spikes/automatic-tags` passed. No
repository-wide Go, browser, full release or generic photo-accuracy gate is claimed
by this sidecar report. The existing local inference service remained healthy and
was not redeployed; main-application persistence/manual protection has its own
transaction and integration evidence.
