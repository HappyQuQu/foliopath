# Proposed model package format v2 executable contract

Status: **isolated contract matrix passed; ADR-0014 remains proposed and the
production parser remains v1-only**.

The proposed format now has an exact shape rather than only a new role name. It
requires format version 2, exactly three roles (`image_encoder`, `text_encoder`,
`sentencepiece_model`) and four exact contract IDs covering image preprocessing,
text canonicalization, SentencePiece metadata/sequence behavior, and embedding
storage.

The isolated Go validator passed a valid v2 example and rejected:

- a v1 format number carrying v2 fields;
- the v1 generic `tokenizer` role in a v2 manifest;
- unknown contract IDs;
- nested paths and duplicate roles;
- unknown fields, duplicate JSON keys and trailing JSON values.

It also enforces the existing 64 KiB manifest, flat regular-file naming,
lowercase SHA-256, positive size, unique name/role and 4 GiB total bounds.

Executed check:

```text
ok github.com/HappyQuQu/foliopath/spikes/int001-model-package-v2 0.129s
```

This is deliberately not production code. Promotion requires ADR-0014
acceptance, a reviewed built-in catalog entry, scanner/install compatibility
analysis, activation through the FD owner and release supply-chain evidence.
