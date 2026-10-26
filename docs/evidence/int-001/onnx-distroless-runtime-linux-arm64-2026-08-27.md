# ONNX Runtime distroless closure and vulnerability evidence

Status: **native Linux/arm64 ABI and restricted-runtime subproof passed; release
security and `INT-008` remain open**.

The isolated `spikes/int001-ort-capi` image uses the official ONNX Runtime
1.28.0 Linux/aarch64 archive, fixed by SHA-256, and does not change FolioPath's
production image or dependency graph. The first distroless build failed at
process startup with exit 127 because only `libonnxruntime.so.1.28.0` was copied
while its ELF SONAME requires `libonnxruntime.so.1`. Adding that exact symlink
closed the ABI error; the failure is retained because a successful builder link
did not prove the final image closure.

The refreshed pinned `cc-debian13:nonroot` arm64 image then completed 100
cancel/recovery cycles as uid/gid 65532 with 4 CPUs, 4 GiB, 64 PIDs, no network,
a read-only root filesystem, all capabilities dropped and
`no-new-privileges`. All cancellations returned an error, all recovery outputs
were finite, cancellation P95 was 6.572 ms and retained RSS grew 17,268 KiB.
The image is 28,315,519 bytes and its local manifest-list digest is
`sha256:6fed58ce5c886ed493e933fa15976d57423c02354fa6cb7609769baa6344ef24`.

Grype 0.116.1 with its 2026-08-26 database found 25 matches in that image:
1 Critical, 9 High, 4 Medium, 1 Low, 7 Negligible and 3 Unknown. Seven High
matches belonged to `libssl3t64` 3.5.6 even though the inference process does
not use OpenSSL. An isolated `base-nossl-debian13` comparison therefore copied
only the Debian-tracked libstdc++, libgcc, libgomp and zlib closure, including
package metadata and distribution license files. The no-SSL image passed the
same restricted 100-cycle test with cancellation P95 6.59 ms and RSS growth
17,384 KiB. It is 25,803,506 bytes with local manifest-list digest
`sha256:3db1780cfa1a33811e83cbbeb60ca2dd3a5558f66733343221b62e1eb320aca0`.

The no-SSL scan removed all seven OpenSSL High findings, leaving 1 Critical,
2 High, 3 Medium, 1 Low, 7 Negligible and 1 Unknown. The remaining Critical
`CVE-2026-5450` and High `CVE-2026-5435`/`CVE-2026-5928` are glibc findings.
Debian's tracker marks Debian 13's package vulnerable but classifies each as a
minor `no-dsa` issue. This context is not an automatic waiver: a security owner
must produce reviewed VEX/reachability evidence or select a fixed runtime base.
The Gate therefore remains blocked.

As preliminary reachability input, Darwin `objdump` parsed the actual arm64 ELF
dynamic symbol tables from the no-SSL image. Across 52 harness and 386 ORT
undefined symbols, neither binary directly imports the affected `scanf` family,
`ungetwc`, `ns_printrrf`, `ns_printrr` or `fp_nquery`; exact-name string scans
also found none. The harness directly needs ORT and glibc, while ORT needs
libstdc++, libm, libgcc and glibc. Absence of a direct import does not rule out
indirect glibc-internal calls, runtime symbol lookup or future production code,
so this is not a VEX decision and does not close the findings.

Superseding scope note (2026-08-28): that check covered only the harness and
ORT. The later combined-runtime closure check also included SentencePiece,
libstdc++ and libgcc_s, and found a direct `ungetwc` import in libstdc++ on both
architectures. The original two-file result remains factual but must not be
used as a closure-wide justification for `CVE-2026-5928`.

The CycloneDX result contains 1,264 components for the no-SSL image, but the
raw native ORT shared object is not identified as an ONNX Runtime package. A
supplemental [`onnxruntime-component.cdx.json`](../../../spikes/int001-ort-capi/onnxruntime-component.cdx.json)
now records the Linux/arm64 runtime version and purl, shared-library and archive
digests, source commit, MIT declaration, license-file digest and notices digest.
Its SHA-256 is
`72d6ef2c0983991a85a8f80827ad2d03b0d08310539c82420e6d3a692ef40673`.
It passed structural validation with `ajv-cli` 5.0.0 against the official
CycloneDX 1.6 JSON Schema (schema SHA-256
`18f57f7482593bad9f21b4feed09084640cbeff419d62ad5090c5ceccca5b37d`)
and official JSF/SPDX references; optional format plugins were not enabled.
The document intentionally marks its composition incomplete. It closes only
the explicit-component inventory gap for this arm64 spike: release generation
must merge an architecture-specific component into each final image SBOM and
bind it to vulnerability/VEX results and signed provenance.

This evidence does not cover native Linux/amd64, a production adapter,
production image composition, runtime admission/concurrency, a merged final
SBOM, signed provenance, security/compliance approval or the face-model licensing hold. Machine-readable
details and report digests are in
[`onnx-distroless-runtime-linux-arm64-2026-08-27.json`](onnx-distroless-runtime-linux-arm64-2026-08-27.json).
