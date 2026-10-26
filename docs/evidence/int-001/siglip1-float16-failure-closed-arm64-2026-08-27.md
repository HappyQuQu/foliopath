# SigLIP 1 float16 split failure-closed smoke

Status: **bounded arm64 subproof passed; `INT-008` remains open**.

The hash-bound SigLIP 1 float16-internal image and text graphs were tested with
CPU memory arena disabled on macOS/arm64 (ONNX Runtime 1.29.0) and native
Linux/arm64 (ONNX Runtime 1.28.0). The Linux run used 4 CPUs, 4 GiB, no network
and a read-only container root filesystem.

Each run rejected an empty file, 64 KiB deterministic random bytes and the
first 64 KiB of the real image graph. Every corrupt load ran in a child process
with a 15-second bound; none timed out or terminated by signal. The valid image
session rejected missing input, float64 input, four channels and the wrong
height. The valid text session rejected missing input, float32 token IDs,
sequence length 65 and batch size 2. Both sessions produced finite output after
the rejected calls.

The first local invocation deliberately failed before model loading because the
supplied prepared-tensor digest was wrong. The run was repeated with the digest
measured from the immutable temporary artifact; the validation was not relaxed.

This is not a general hostile-model proof. It does not cover adversarial but
protobuf-valid graphs, oversized tensor declarations, adapter input-byte
admission, production C/Go cancellation, native Linux/amd64 or final runtime
packaging. Those gaps keep `INT-008` and INT-S0 open.

Machine-readable results are in
[`siglip1-float16-failure-closed-arm64-2026-08-27.json`](siglip1-float16-failure-closed-arm64-2026-08-27.json).
