# Model download failure-matrix spike — 2026-08-26

Status: **partial development evidence only; INT-020 and INT-S0 remain open**.

## Boundary tested

The isolated `spikes/int001-ai` module now contains a catalog-owned artifact
fetch state machine. It is not imported by FolioPath, exposes no API, and does
not accept a user URL. The test uses only an in-process `httptest` server and
temporary directories; no model or user media is downloaded. It ran on
macOS/arm64 and native Linux/arm64.

The trusted catalog entry pins exact origin, ETag, byte size and SHA-256. The
spike requires HTTPS for non-test entries, rejects URL credentials/query/
fragment and literal loopback/private/link-local targets, restricts every
redirect to the exact reviewed origin, caps redirects at five, and publishes a
verified file with the existing no-replace primitive.

The catalog transport now resolves the reviewed hostname exactly once per
client, rejects the entire answer if any address is loopback, private,
link-local, CGNAT, benchmark, documentation, multicast or other selected
special-use space, and pins subsequent dials to the accepted addresses. IPv4-
mapped IPv6 is normalized before policy evaluation. The transport does not use
environment proxy variables and keeps the reviewed hostname as the TLS server
name.

## Reproduction and result

```sh
cd spikes/int001-ai
go test -count=1 -run TestFetchTrustedArtifactFailureMatrix ./...
```

Result on 2026-08-26: pass. The automated matrix covers:

| Case | Expected and observed result |
| --- | --- |
| Initial fetch | Exact bytes/hash publish once into the verified name |
| Stable partial + ETag | Sends `Range` and `If-Range`; accepts matching `206` and exact `Content-Range` |
| Changed ETag/object | Rejects resume and removes the stale partial instead of mixing generations |
| Redirect to another origin | Rejects before following the new origin |
| Loopback source | Rejects unless the private test-only exception is set |
| Temporary quota below catalog size | Rejects before network I/O |
| Wrong SHA-256 | Deletes the corrupt partial; candidate never becomes visible |
| Cancelled context | Candidate never becomes visible and the active-generation sentinel is unchanged |
| Mid-stream cancellation | Persists only the received partial, then resumes with the same ETag and publishes exact bytes/hash |
| Existing verified generation | Returns no-replace failure and preserves existing bytes |
| Kernel `ENOSPC` | A native Linux/arm64 128 KiB tmpfs rejects the write; candidate remains invisible and active bytes are unchanged |
| Process `SIGKILL` and restart | A child is killed after a bounded partial is visible; a new process resumes with the pinned ETag and publishes exact bytes/hash |
| Package-publish boundary `SIGKILL` | Real helpers killed before rename, after no-replace rename/before parent fsync, and after parent fsync leave only a complete staging or complete final directory |
| Kernel `ENOSPC` after package completion | A native Linux/arm64 128 KiB tmpfs was filled until a new write returned `ENOSPC`; same-filesystem no-replace rename and parent fsync still published the already-complete package atomically, without changing active |

The mid-stream cancellation case passed ten consecutive local runs. The same
full matrix then passed natively under:

- Linux/arm64 (`aarch64`), Go 1.26.5;
- local development image ID
  `sha256:e54adb751d433f82bbaeb42c198a4add1d62c2c0da8ef62b1e3d2263fa98fe6b`.

The image is a local development build, not a published release artifact.

The kernel disk-full case was run separately with:

```sh
docker run --rm --tmpfs /evidence:rw,size=131072,mode=1777 \
  -e INT001_ENOSPC_DIR=/evidence ... \
  go test -count=1 -run TestFetchTrustedArtifactActualENOSPC ./...
```

It returned the real Linux `ENOSPC` path rather than an injected writer error.

The strong-kill test uses the compiled Go test binary as a real helper process,
waits until a non-empty/incomplete partial is visible, kills that process, and
starts a second helper. It passed five consecutive macOS/arm64 runs and one
native Linux/arm64 run. The active-generation sentinel remained unchanged.

Network-policy tests passed ten consecutive macOS/arm64 runs and one native
Linux/arm64 run. They prove that a mixed public/private answer fails as a whole,
`HTTPS_PROXY` is ignored, the resolver is called once, and the dial target is
the pinned public address. Resolver errors and empty answers now also fail after
exactly one lookup without dialing or an implicit in-request retry; retry
scheduling remains an outer job-policy decision. A pinned loopback end-to-end
fetch exists only behind the private test exception.

A later TLS subproof generated a private test CA and a certificate valid only
for `models.example.test`, then used the production-like transport with that
hostname, SNI and a once-resolved loopback address set. The first validated
address returned a deterministic dial failure, the second completed a real TLS
handshake and the downloader published only after ETag/size/SHA-256 validation.
The same server was rejected without the test CA. Both tests passed on
macOS/arm64 and native Linux/arm64 in a 4 CPU/4 GiB/no-network/read-only-root
container; the Linux test binary SHA-256 was
`e945871244398c7b52c8210fef8d3a0445ecafe534bf2f7cb1eba86673b2ba2f`.
This proves TLS verification and fallback after an explicit dial error, not
real CDN rotation. A separate blocking-dial test now gives every validated
address a five-second maximum (or the shorter outer deadline), then proves the
next address is attempted after the first attempt context expires. This closes
the primitive timeout behavior, not public DNS/CDN or outer retry policy.

The signed-catalog primitive uses Ed25519 with a domain-separated binary
message covering schema version, key ID, monotonic sequence, issue/expiry time,
payload length and exact payload bytes. Strict envelope parsing, a 6 MiB
envelope/4 MiB payload bound, valid JSON payload, trusted key lookup, validity
window, rollback checkpoint and same-sequence/different-payload equivocation
checks all fail closed. The matrix covers unknown/wrong keys, payload and
metadata tampering, future/expired catalogs, rollback, equivocation, unknown
fields, trailing JSON and oversized input. It passed ten macOS/arm64 runs and
one native Linux/arm64 run.

An isolated SQLite activation registry models only already-published immutable
package generations and accepts only the internal authenticated result returned
by the Ed25519 verifier. Catalog checkpoint and active-model pointer advance in
one transaction. A trigger-injected failure after checkpoint update but before
the active update rolls both changes back; missing or digest-mismatched
generations, stale catalog sequence and same-sequence equivocation preserve the
old active state. Successful activation is idempotent and survives database
close/reopen. The verifier-plus-activation matrix passed ten macOS/arm64 runs
and one native Linux/arm64 run.

The native Linux scanner-to-registry reconciliation accepts only a scan report
marked complete by the kernel-anchored `openat2` scanner. An exact catalogued
filesystem orphan is registered as available but never activated. Removing or
corrupting the active package marks that generation unavailable while retaining
the active pointer and checkpoint; restoring exact bytes marks it available
again. Unknown directories remain rejected and are neither registered nor
deleted. This matrix passed ten consecutive native Linux/arm64 runs plus one
post-hardening run.

A capability-gated native Linux/arm64 test creates a real read-only bind mount
for the direct source. The scanner accepts it only with the read-only filesystem
flag, registry provenance remains immutably `direct`, disappearance marks the
generation unavailable without changing active/checkpoint, and remounting the
same exact package restores availability. Attempting to re-register the direct
generation as managed fails as source equivocation. No source file is copied,
renamed or deleted. This test requires `SYS_ADMIN` only inside the disposable
test container and is not evidence that the release container needs that
capability.

The package-directory publisher now exposes test-only phase hooks around its
existing no-replace rename and parent-directory fsync. A real helper process was
stopped at each hook and killed. Before rename, restart found the complete
staging package and published it normally. At both post-rename boundaries,
restart found no staging directory and one complete final package. No phase
produced both names, neither name, or partial model bytes. The matrix passed on
macOS/arm64 and native Linux/arm64. This validates process-crash atomicity; it
does not simulate host power loss or storage-controller durability.

A second real `ENOSPC` run completed and synced the staged package first, wrote
the active sentinel, then filled a 128 KiB tmpfs until the kernel rejected a
write with `ENOSPC`. The same-filesystem no-replace rename and parent-directory
fsync still succeeded because they did not copy model bytes. The final package
was complete, staging disappeared, and active remained unchanged. The test also
accepts only an `ENOSPC` failure that preserves a complete staging package and
exposes no final directory, so either filesystem outcome remains atomic.

## What this does not prove

- Public-CA catalog TLS, DNS TTL/legitimate CDN address rotation, outer job
  retry/backoff policy, production key custody/rotation/
  revocation, durable checkpoint ownership, response-header hardening and
  deployment-configured origins remain open.
- Temporary-space reservation, parallel request admission and host power-loss
  durability remain open. Production migration ownership and filesystem/DB
  reconciliation for every managed/direct-source lifecycle remain open despite
  the isolated SQLite, managed-package and direct read-only mount results.
- The matrix has not run natively on Linux/amd64.

Accordingly this closes only the explicitly checked `INT-020` evidence
sub-items; it does not justify
an online model-download product contract or a domestic mirror claim. If the
remaining controls fail, the defined fallback is `/models:ro` plus managed
offline copy only.
