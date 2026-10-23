# Operator-authorized local face functional smoke

Date: 2026-08-31

This isolated run used an operator-authorized, local-only set of network images to test existing candidate models. It
did not train or publish a model, infer real identities, copy source images, persist face crops or embeddings, upload
artifacts, or exercise production FolioPath face composition. The source filesystem path and filenames are deliberately
absent from the report.

The run deterministically sampled at most 15 supported images from each of nine top-level groups. All 135 images decoded;
YuNet returned 79 candidates and SFace produced 79 finite, non-zero 128-dimensional embeddings with no invalid output.
On the macOS arm64 development host, the reproducible entry-point run measured detector P50/P95 at 15.319/17.786 ms
and embedding P50/P95 at 7.963/8.990 ms.
Both model files matched the hashes pinned in the candidate catalog before execution. Temporary models and the temporary
Python environment were removed after the run.

The machine-readable aggregate is
[`face-functional-local-arm64-2026-08-31.json`](face-functional-local-arm64-2026-08-31.json). Candidate count is not
detector recall: the source has multiple people and no face-level identity or bounding-box ground truth. This record only
closes the local functional question “can decode → detect → align → embed execute on ordinary images?” It does not close
`INT-005B`, `INT-007`, `INT-015`, `INT-241`, `INT-250`, S2C Backend Evidence Ready, model licensing, native Linux dual-
architecture evidence, quality/bias acceptance, or any release Gate.

The reusable entry point is covered by `make spike-ai`; its standard-library tests reject placeholder authorization,
unbounded arguments, media-root or child symlinks, oversized samples and model size/hash tampering without requiring
the real media or model files in CI.
