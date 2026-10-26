# SentencePiece + ONNX Runtime cross-architecture vulnerability evidence

Status: **both image candidates fail the release vulnerability Gate**.

Grype 0.116.1 scanned the exact local arm64 and amd64 text-runtime images with
database auto-update disabled. The retained database is schema v6.1.9, built
2026-08-26, hash-validated and reported valid. The scanner binary, release
archive, database archive, image index/platform/config identifiers and raw
report hashes are fixed in the adjacent machine-readable record.

Both architectures produced the same 15 findings: one Critical, two High,
three Medium, one Low, seven Negligible and one Unknown. The release-blocking
set is identical and belongs to Debian `libc6` 2.41-12+deb13u3:

- Critical `CVE-2026-5450`;
- High `CVE-2026-5928`;
- High `CVE-2026-5435`.

The database lists no fixed version for these three findings. This does not
permit automatic suppression: security must provide reviewed VEX/reachability
evidence or the runtime must move to a fixed base and be rescanned. The earlier
negative ELF-symbol evidence remains only an input to that review. A subsequent
expanded-closure check found that both exact libstdc++ builds directly import
`ungetwc`; therefore `CVE-2026-5928` cannot use a closure-wide “no direct
import” justification.

Grype, like Docker Scout, did not identify the manually copied ONNX Runtime or
SentencePiece shared libraries as packages. Their explicit CycloneDX records
close inventory visibility but do not automatically correlate vulnerability
advisories. A final signed SBOM/VEX pipeline still needs explicit advisory
handling for both native components.

The arm64 image has native runtime evidence. The amd64 result proves package
parity for the QEMU-built image only; it does not close native linux/amd64
runtime, performance or release evidence. No finding was suppressed, and no
release-readiness task is marked complete by this scan.
