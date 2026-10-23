# Multi-file model package safety spike — 2026-08-25

Status: development evidence only; native Linux execution and signed-manifest
verification remain pending.

## Contract exercised

- Catalog schema v1 remains available for a single ONNX artifact such as YuNet
  or SFace.
- Catalog schema v2 identifies one model generation by a direct child directory,
  1–128 declared direct-child artifacts, each artifact's byte size and SHA-256,
  and a deterministic package digest over the sorted artifact manifest.
- Directory and artifact names are restricted to one conservative path segment.
  Traversal, duplicate artifacts, mixed v1/v2 fields and a package digest that
  does not match the artifact manifest fail validation.
- The Linux scanner uses `openat2` with `RESOLVE_BENEATH`,
  `RESOLVE_NO_SYMLINKS` and `RESOLVE_NO_XDEV` for the package directory and each
  artifact. Missing, undeclared, non-regular, symlinked, size-mismatched or
  hash-mismatched artifacts reject the whole package.
- An already-verified package is published from a same-parent staging directory
  by a no-replace atomic rename followed by parent-directory sync. An existing
  generation is never replaced; a failed publish preserves staging for diagnosis.

The package digest domain is `foliopath-model-package-v1` followed by each
artifact's sorted filename, decimal byte size and lowercase SHA-256, each on its
own newline. This digest identifies the artifact manifest; it is not a signature.

## Real candidate manifests

The pinned runtime files for both semantic development candidates are represented
in `testdata/model-catalog.siglip-candidates.json`:

| Candidate | Runtime artifacts | Runtime bytes | Package digest |
| --- | ---: | ---: | --- |
| SigLIP 2 base patch16 224 | 7 | 1,539,453,393 | `893e402e…562a` |
| SigLIP base patch16 224 | 7 | 815,871,927 | `1df799c0…af17` |

README and Git metadata are deliberately excluded from the executable package;
their provenance still belongs in release evidence. Both catalog entries remain
`candidate`, so the scanner will not load them as approved models.

## Executed evidence

- macOS arm64: strict v1/v2 manifest tests, package digest/path/duplicate negative
  tests, and no-replace atomic publish/recovery tests passed.
- Linux amd64 and arm64: the complete test binary, including `openat2` package
  rejection cases, cross-compiled successfully.
- Native Linux execution was not available: Docker daemon was stopped and no
  Podman or Colima runtime existed. Cross-compilation is not Linux safety evidence.

## Remaining blockers

1. Execute the package scanner matrix natively on linux/amd64 and linux/arm64,
   including read-only roots, nested mounts and replacement during hashing.
2. Define and verify a signed catalog envelope; package SHA-256 alone cannot prove
   who authorized a model or prevent an attacker from replacing both files and an
   unsigned local manifest.
3. Connect download resume/quota/error handling to a staging directory, verify the
   complete package, then publish it. The current publish primitive intentionally
   accepts only an already-verified staging directory.
4. Freeze generation selection and rollback ownership in the eventual database;
   directory publication must not become an implicit active-generation switch.
