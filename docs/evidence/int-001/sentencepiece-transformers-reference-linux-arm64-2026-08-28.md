# SigLIP tokenizer Transformers reference conformance — Linux/arm64

Status: **31-case reference matrix passed; ADR-0014 and INT-203 remain open**.

## Reference contract

- Generator: Python 3.12.13, Transformers 4.56.2, SentencePiece 0.2.1.
- Model: `google/siglip-base-patch16-224` revision
  `7fd15f0689c79d79e38b1c2e2e2370a7bf2761ed`.
- `spiece.model`: 798,330 bytes, SHA-256
  `1e5036bed065526c3c212dfbe288752391797c4bb1a284aa18c9a0b23fcaf8ec`.
- Deterministic fixture:
  `spikes/int001-sentencepiece-capi/testdata/siglip-tokenizer-reference-v1.json`,
  35,972 bytes, SHA-256
  `fa12da1f146659256d0607b548b7375cb49af7fc933b0395ad9a32344fb85d0b`.

The committed generator verifies dependency versions and model size/hash before
writing output. The fixture has no timestamp or host path and records every
64-element input-ID sequence rather than only token prefixes.

## Coverage and result

The 31 named cases cover English case/punctuation/whitespace, simplified and
traditional Chinese, mixed Chinese/English, Japanese, Korean, Arabic, Cyrillic,
Devanagari, fullwidth and CJK punctuation, composed/combining accents,
Turkish/Greek/German casing, emoji/ZWJ/variation selectors, Unicode spaces and
line separators, zero-width and bidi controls, embedded NUL, rare CJK byte
fallback, filesystem-like text, digits, and two exact 512-rune truncation
inputs.

The first Linux/arm64 comparison found a real mismatch for Greek final sigma.
Transformers 4.56.2 lowercases non-special text through a non-greedy regex,
which applies Unicode lowercase one code point at a time; whole-string lowercasing
produced contextual final sigma instead. The canonical owner was corrected to
the per-code-point behavior and received a focused unit expectation. The
fixture was regenerated, byte-compared with the committed file, and the full
tagged Linux/arm64 suite then passed:

```text
=== RUN   TestPinnedSigLIPReferenceFixture
--- PASS: TestPinnedSigLIPReferenceFixture (0.02s)
PASS
ok github.com/HappyQuQu/foliopath/spikes/int001-sentencepiece-capi 2.086s
```

## Limits and remaining blockers

- The matrix is broad but not exhaustive over Unicode.
- A same-day contract follow-up resolved literal registered special tokens:
  case-insensitive `</s>` and `<unk>` are rejected before punctuation removal,
  snapshot lookup or inference. Unit and service tests cover exact, uppercase
  and embedded forms; the HTTP adapter maps the stable error to a query-free
  `invalid_request`. They are deliberately invalid FolioPath queries and are
  therefore not token-parity cases.
- Native Linux/amd64 execution, text ONNX embedding parity, package format-v2
  activation, final image ABI/SBOM/provenance and long-duration resource evidence
  remain open.
- The reference Python environment is evidence tooling only and is not a
  production dependency.
