# INT-203 SentencePiece C API spike

This directory is an isolated Linux/cgo feasibility harness for proposed
[ADR-0014](../../docs/adr/0014-siglip-sentencepiece-tokenizer-runtime.md). It is
not imported by FolioPath production packages and does not authorize adding
SentencePiece to the release image.

The wrapper deliberately exposes only model open/close, fixed metadata and
integer token IDs. It canonicalizes text through `internal/semantic`, rejects a
model whose SigLIP metadata differs from the pinned contract, truncates to 63
pieces and fills the fixed 64-element sequence with EOS/pad ID 1. Native errors
are collapsed to an opaque error and are not returned to an API or log. The
test-only FD path accepts a bounded regular file and loads it synchronously via
`/proc/self/fd`; encode and close are serialized on the native handle.

Run it only in a disposable native Linux environment with reviewed official
SentencePiece headers/library and the pinned SigLIP `spiece.model` supplied as
read-only inputs:

```sh
CGO_ENABLED=1 \
CGO_CFLAGS="-I/path/to/sentencepiece/include" \
CGO_CXXFLAGS="-I/path/to/sentencepiece/include" \
CGO_LDFLAGS="-L/path/to/sentencepiece/lib -Wl,-rpath,/path/to/sentencepiece/lib" \
go test -tags sentencepiece ./spikes/int001-sentencepiece-capi -count=1 \
  -args -sentencepiece-model /path/to/spiece.model
```

The committed `generate_reference_fixture.py` requires Python 3.12,
Transformers 4.56.2 and SentencePiece 0.2.1. It verifies the pinned model bytes
before regenerating the deterministic 31-case fixture under `testdata`; Python
is evidence tooling only and is not part of the proposed runtime.

`generate_text_embedding_fixture.py` similarly pins a reviewed split text graph,
the tokenizer fixture, ONNX Runtime 1.29.0 and NumPy 2.5.2. It records all 31
raw 768-D float32 outputs for later Linux ORT 1.28 C API parity testing; it does
not add Python or ORT to production.

With both `sentencepiece` and `onnxruntime` build tags, the isolated text
adapter feeds those 64 IDs into the pinned ORT 1.28 C API graph and compares all
31×768 coordinates with the reference. This path is evidence-only and accepts
an explicit graph path; production must instead use the reviewed FD owner.
The harness also covers observed active-run cancellation, serialized concurrent
reuse and a fixed 10-warm-up/10-measured load-close RSS check. A flat measured
slope does not erase the recorded cold-to-stable resident expansion.

Default builds exclude this package. Linux/arm64 evidence now covers fixed token
IDs, bounded malformed inputs, FD loading, concurrent callers, closed-handle
behavior, pre-entry cancellation and 100 load/close cycles. Production
integration remains blocked on ADR acceptance, native amd64 parity, complete
source attestation, exact-image vulnerability disposition, merged SBOM/signed
provenance and production model-package v2 composition. Registered `</s>` and
`<unk>` literals are rejected by the canonical owner before this adapter.
SentencePiece provides no mid-call cancellation primitive.

`Dockerfile.runtime` is the isolated arm64 combined-runtime closure fixture. It
requires named `sentencepiece_source` and `onnxruntime_source` build contexts
containing the two reviewed archives; the build checks both pinned SHA-256
values before extraction. This avoids making a slow or blocked GitHub
connection part of the release build while still failing closed on substituted
bytes. The final no-SSL distroless image contains only the test binary, the two
native runtimes, their dynamic C/C++/OpenMP/zlib closure, license files and
deterministic reference fixtures. The 441 MiB graph and tokenizer model remain
external read-only test inputs and must not be baked into the image. The
builder targets only the SentencePiece inference shared library; training
libraries, static archives and SentencePiece command-line tools are not built
or copied.

The defaults select the pinned arm64 ORT archive and Debian library directory.
An amd64 preflight uses the same Dockerfile with
`ONNXRUNTIME_ARCHIVE=onnxruntime-linux-x64-1.28.0.tgz`, archive SHA-256
`a3e1b79d7bb1bf09696ce675f49e4064e6c81f6202b8225624fff0e93f8d6407`
and `RUNTIME_LIBDIR=/usr/lib/x86_64-linux-gnu`. Running that image through QEMU
also requires the exact amd64 `cc-debian13` and `base-nossl-debian13` child
manifest digests recorded in the evidence. Running that image through QEMU is
packaging evidence only and cannot satisfy the native Linux/amd64 Gate.

`verify_source_archive.py` verifies the staged source archive against a pinned,
complete GitHub Git-tree API response without extracting files to the working
tree. It rejects unsafe paths, links, missing/extra files, content drift and
executable-mode drift. The verifier constructs raw Git blob IDs itself so local
`.gitattributes` or `core.autocrlf` settings cannot change the result.
