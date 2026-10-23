# Operator-authorized local face group-pair functional evidence

Date: 2026-08-31

Status: **functional error-merge evidence only; not governed identity ground truth or a release Gate**.

The read-only run sampled at most 100 supported images from each of the nine
operator-provided top-level groups. It decoded 725 images, detected 457 candidates,
and produced 457 finite 128-dimensional SFace embeddings. Group labels and vectors
existed only in process memory; source paths, names, crops and embeddings were not
written to the repository or the media tree.

The bounded scorer compared 13,632 same-group and 90,564 cross-group pairs. At
cosine threshold 0.7 it found 39 cross-group pairs above threshold (FPR 0.0431%);
at 0.8 it found none, while same-group pair recall fell to 7.31%. This confirms the
expected safety/recall tradeoff and demonstrates that a permissive threshold must
not authorize whole-group assignment.

The source folders are useful operator-authorized functional group labels, but are
not audited biometric ground truth: there are only nine groups, folder membership
may contain collaboration images, and no skin-tone, age, lighting, occlusion or
people-count annotations exist. Therefore this result cannot close the required
50-identity × 20-image governed quality/bias Gate or approve a production threshold.

Machine-readable aggregate:
[`face-group-pair-functional-local-arm64-2026-08-31.json`](face-group-pair-functional-local-arm64-2026-08-31.json).
