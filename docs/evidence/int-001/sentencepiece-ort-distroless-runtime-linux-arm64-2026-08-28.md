# SentencePiece + ONNX Runtime distroless closure evidence

Status: **native Linux/arm64 combined ABI and restricted-runtime subproof
passed; ADR acceptance and release supply chain remain blocked**.

The isolated
[`Dockerfile.runtime`](../../../spikes/int001-sentencepiece-capi/Dockerfile.runtime)
builds SentencePiece 0.2.1 from a staged source archive whose SHA-256 is
`c1a59e9259c9653ad0ade653dadff074cd31f0a6ff2a11316f67bee4189a8f1b`.
The archive declares version 0.2.1 and contains the Apache-2.0 license; the
license digest is
`cfc7749b96f63bd31c3c42b5c471bf756814053e847c10f3eb003417bc523d30`.
The upstream `v0.2.1` tag resolves to commit
`31646a467d2051eb904e0b45de3a73e91fe1c1e3`. A separate full-tree comparison
attests that all 258 archive files exactly match that official commit's raw Git
blob IDs and executable modes. The archive's particular
upstream distribution URL was not independently attested, so the evidence does
not claim signed or byte-for-byte upstream archive provenance.

The final image combines that library with the already pinned official ONNX
Runtime 1.28.0 arm64 archive on the same no-SSL distroless base used by the ORT
closure spike. It contains no graph or tokenizer model bytes. Those two files
were bind-mounted read-only for the test. The resulting local image is
28,633,295 bytes. In the final default arm64 regression build after the
Dockerfile was parameterized for both supported target architectures, the
runnable manifest is
`sha256:dedc1b601040f45682b46e86469d5d1d042bf7b322cee10e27f8cee4907d9447`,
its config is `sha256:5891f7a4c6b648e77641ce5cd625fafddd164852134ca0cc55882294cdfc6f2b`,
and the local index including BuildKit provenance is
`sha256:7fed8b185244854301e6467ebc1dada76bc8960c83d377ba4f790c533480040d`.
Before that metadata change, repeated identical Dockerfile builds preserved the
runnable manifest and config while regenerated provenance changed the local
index digest. Parameterizing the Dockerfile and replacing the architecture-
specific image label with a common label changed the config and manifest as
expected; it did not change the runtime shared-library bytes. Release evidence
must therefore bind and sign the exact published index, while reproducible-
content comparison uses the platform manifest and artifact digests rather than
silently treating regenerated attestations or intentional metadata changes as
runtime-byte drift.

The SentencePiece build targets only the 42-step `sentencepiece` shared-library
target. It no longer runs the 117-step install graph that also built training
libraries, static archives and five command-line tools. The final image did not
contain those artifacts before or after this change; the narrower target
reduces unnecessary builder scope and produces the same runtime library digest.

Both native inputs are supplied as named local BuildKit contexts and checked
by SHA-256 before extraction. This was an evidence-driven choice rather than a
convenience fallback: a direct official SentencePiece URL download timed out
after 300 seconds, and a subsequent BuildKit remote `ADD --checksum` attempt
also made no completion progress within the same bounded window and was
cancelled. The restored offline-context build completed and verified both the
13,485,527-byte SentencePiece archive and the 8.12 MB ORT archive. A release
pipeline may fetch or mirror these inputs in an earlier authenticated step, but
the compilation step does not depend on reaching GitHub and still rejects any
substituted bytes.

The final binary directly requires `libonnxruntime.so.1`,
`libsentencepiece.so.0`, `libstdc++.so.6`, `libgcc_s.so.1` and `libc.so.6`.
SentencePiece additionally requires libstdc++, libm, libgcc and libc. ORT's
recorded closure additionally names libdl, librt, libpthread and the arm64
loader. The final-stage execution proves the copied SONAME links and base
libraries resolve together; `libsentencepiece.so.0.0.0` has SHA-256
`4606acd764ba8ec2ad4d80901b5ae8ddaabc4439cb54ea85d7e85f4eb20ab9d8`
and `libonnxruntime.so.1.28.0` has SHA-256
`f1ec1a08eb99bd6e5401340f0a2b101381bf4694415480291dc13bcaa30f9ec7`.
Both digests were re-extracted from the final parameterized arm64 image and
match the pre-parameterization runtime artifacts.

Under 4 CPUs, 4 GiB, 64 PIDs, uid/gid 65532, no network, a read-only root
filesystem, all capabilities dropped and `no-new-privileges`, the complete
SentencePiece-to-ORT path processed all 31 fixed queries and 23,808 float
coordinates. It matched the independently generated ORT 1.29 reference with a
maximum absolute difference of `1.811981201171875e-05`; the run completed in
2.26 seconds after the narrowed rebuild.

The supplemental
[`sentencepiece-component.cdx.json`](../../../spikes/int001-sentencepiece-capi/sentencepiece-component.cdx.json)
records the arm64 library, archive, tag commit and license digests. Its
SHA-256 is
`f904fc3bd10d17b6311ca57720a3155076572e10ba8313a14190698d41798514`.
Its composition is deliberately `incomplete`: it has not been merged with the
final image SBOM or bound to signed provenance.

Docker Scout 1.24.0 also inventoried the exact current arm64 image and found 17
base/Go components, but did not recognize the manually copied ONNX Runtime or
SentencePiece libraries. That generated CycloneDX inventory was therefore not
committed or represented as a complete SBOM. The isolated merge utility
flattens Scout's nondeterministic package-to-file nesting, consolidates duplicate
dependency rows, rejects dangling references/duplicate components/wrong
architectures, and adds the two explicit native component records. Two fresh
scans of the same image produced byte-identical normalized output: 1,267
components, 18 dependency rows, UUID
`07c43c80-c569-5fb0-a549-e37ade4a7005`, and SHA-256
`6e340ca72e1cc91fea637cb9af2b1bf463c8eb3ad112613032caeb1616fb9301`.
The composition deliberately remains `incomplete`; this proves a deterministic
merge path, not final SBOM completeness or signing.

A later fixed Grype 0.116.1 scan with database auto-update disabled covered this
exact image and confirmed 15 findings, including one Critical and two High in
`libc6` 2.41-12+deb13u3. The database reports no fixed version for those three.
Adding SentencePiece therefore does not make the release Gate pass. Security
must still provide reviewed VEX/reachability evidence or select a fixed base,
then rescan the exact final digest. Native Linux/amd64, explicit native-library
advisory correlation, exact distribution-archive provenance, final signed
dual-architecture SBOM/provenance, production composition and approval of
ADR-0014 also remain open.

Commands executed from the repository root:

```sh
docker build --platform linux/arm64 \
  --build-context sentencepiece_source=/private/tmp/foliopath-spiece.RbFvKV \
  --build-context onnxruntime_source=/private/tmp/foliopath-int008-capi.QjO26G \
  -f spikes/int001-sentencepiece-capi/Dockerfile.runtime \
  -t foliopath-int001-text-runtime:arm64 .

docker run --rm --platform linux/arm64 --network none --read-only \
  --cpus 4 --memory 4g --pids-limit 64 --cap-drop ALL \
  --security-opt no-new-privileges \
  --mount type=bind,src=spiece.model,dst=/models/spiece.model,readonly \
  --mount type=bind,src=text_encoder.onnx,dst=/models/text_encoder.onnx,readonly \
  foliopath-int001-text-runtime:arm64 \
  -test.v -test.run '^TestPinnedSigLIPTokenizerToTextEncoderParity$' \
  -sentencepiece-model /models/spiece.model \
  -text-onnx-model /models/text_encoder.onnx
```
