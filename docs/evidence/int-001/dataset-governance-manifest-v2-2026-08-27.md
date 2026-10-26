# Dataset governance manifest v2 evidence

Status: **validator subproof only; real face ground truth remains missing**.

On 2026-08-27, `make spike-ai` passed on Darwin/arm64 with Go 1.26.5 after
adding the strict dataset-manifest v2 intake rules. Schema v1 is restricted to
synthetic fixtures. Every non-synthetic dataset must use v2 and state an
allowlisted evaluation purpose and access role, a parseable retention date, the
fixed deletion procedure, and an explicit redistribution policy.

For biometric ground truth, the validator additionally rejects public media
licensing as the processing authority, requires prohibited redistribution, a
privacy-review reference, the privacy-reviewer access role, and—when used for
verification or clustering—an opaque identity ID on every item. Written
authorization requires an opaque authorization reference. References are
bounded identifiers; the manifest does not accept names, documents, URLs, or
filesystem paths in those fields.

The negative matrix covers missing governance, a non-synthetic v1 manifest,
public-license biometric data, missing authorization/privacy/retention fields,
unknown classes/uses/roles/deletion actions, inconsistent redistribution,
unsafe identity references, missing clustering identities, identity fields on
ordinary media, and path traversal. File digests and the machine-readable rule
matrix are recorded in the adjacent JSON file.

The checked-in authorized-face file is deliberately a template whose purpose
and `*-REQUIRED` references say that approval is absent. Passing validation
only proves structural completeness. It does not verify a signature or lawful
basis, and no real face image, consent record, name, or identity ground truth
was used. Controlled storage, access audit, deletion rehearsal, representative
quality evaluation, privacy/legal sign-off, `INT-005B`, `INT-007B`, `INT-015`,
and INT-S0 therefore remain open.
