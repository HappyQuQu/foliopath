# Operator-authorized nested-group face functional evidence

Date: 2026-08-31

Status: **expanded functional evidence only; not governed identity ground truth or a release Gate**.

The read-only run used depth-two source directories as temporary, in-memory functional groups and sampled at most 20
supported files per directory. It processed 3,070 files from 180 directories, decoded every selected file, detected 1,996
candidates in 1,973 images, and produced 1,996 finite 128-dimensional embeddings across 170 groups with detections.
Source files were not copied or modified. Directory names, source paths, crops and embeddings were not persisted.

The bounded pair scorer evaluated all 12,848 within-directory pairs and a deterministic 100,000-pair sample balanced
across every directory pair with embeddings. This corrects the earlier prefix-truncation bias. At cosine threshold 0.7 it
observed 541 cross-directory candidates (0.541%) and 59.95% within-directory recall. Even at 0.8 it observed six
cross-directory candidates (0.006%) while within-directory recall fell to 19.54%. This larger run reproduces the
safety/recall tradeoff and shows that whole-group operations cannot be enabled from a permissive threshold.

Directory membership is not audited face-level truth: a directory may contain multiple people, multiple faces in one
image, collaboration images, or a subject different from its surrounding folder. The run also lacks detector expected-face
labels and skin-tone, age, lighting, occlusion and people-count annotations. It therefore cannot close the governed
50-identity × 20-image quality/bias Gate, establish deployment prevalence, or approve a production threshold.

Machine-readable aggregate:
[`face-nested-group-functional-local-arm64-2026-08-31.json`](face-nested-group-functional-local-arm64-2026-08-31.json).
