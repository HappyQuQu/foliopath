# Pure synthetic face/native-memory probe

FR-INT-009/016 / ADR-0024. This is a local regression helper, not a service or release image.
Never mount real media, an active database, model weights, or the original 6010 data here.

The default Dockerfile expects the already loaded, verified local builder/runtime images used by the maintenance record.
Override `BUILD_IMAGE` and `RUNTIME_IMAGE` only with corresponding verified local builds, not unrelated/latest images.

```sh
docker build -f spikes/int001-remote-ml/facememory/Dockerfile -t foliopath:face-memory-regression .
```

Create a **new private temporary directory** for generated inputs. Mount it as `/generated` in a one-off container;
run `FOLIOPATH_SYNTHETIC_FACE_MEMORY=1 /app/face-memory generate`. Generation needs 2 GiB and writable `/tmp`
(512 MiB) and is not the measured 1 GiB application case. It procedurally creates baseline/progressive 8800×11000 JPEGs.

Mount only that generated directory read-only at `/library` for measurement. Use no network, 1 CPU, RAM=1 GiB,
memory-swap=1 GiB (no extra swap), read-only root, and `/tmp` tmpfs=128 MiB; disable the inherited HTTP healthcheck.
Set `FOLIOPATH_SYNTHETIC_FACE_MEMORY=1` and optionally `FOLIOPATH_FACE_MEMORY_ITERATIONS=30` (bounded 1–120).
The probe holds 256 MiB of synthetic live heap and checks fixed RGB hashes/dimensions while exercising RGB→PNG.
`FOLIOPATH_FACE_MEMORY_BASELINE=1` selects the baseline JPEG. Record exit code, OOMKilled, refs and memory peak;
do not call a container that was killed or restarted a pass. A peak at the limit still leaves a capacity caveat.

For native unit tests, override the entrypoint with `/app/imagevips.test`, e.g. `-test.short -test.v`.
Large generated-input tests can be run separately without `-test.short`; unit-test skips must remain explicit.

For the **full** app/scan/thumbnail/inference check, use `tests/release/face_diagnostics_smoke.mjs` with its documented
app/AI/cache variables and `FOLIOPATH_TEST_FACE_MEMORY_FIXTURE` pointing only to this generated `progressive.jpg`.
The script verifies its exact encoded hash, copies it into a fresh synthetic library, and checks successful coverage,
missing-only failures, source hashes and zero automatic restarts before the explicit recovery restart.
This full-app case is required: the isolated decoder can pass while overlapping thumbnail work still causes OOM.

Do not reinterpret this regression as all 500 MP, capacity, long-duration, quality, multi-architecture or release acceptance.
