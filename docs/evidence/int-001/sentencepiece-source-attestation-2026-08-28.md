# SentencePiece 0.2.1 source-content attestation

Status: **source content matches the official pinned Git tree; exact upstream
archive-distribution identity remains unclaimed**.

The source archive used by the isolated arm64 build has SHA-256
`c1a59e9259c9653ad0ade653dadff074cd31f0a6ff2a11316f67bee4189a8f1b`.
Its version file reports 0.2.1. GitHub's official repository API resolved tag
`v0.2.1` to commit `31646a467d2051eb904e0b45de3a73e91fe1c1e3` and returned a complete,
non-truncated recursive tree response with 258 blobs. The captured response has
SHA-256 `46968b11ad3a0f388f1de4388c499f165d5c2b0ce50a5fe82a5fae965a89037e`.

The committed
[`verify_source_archive.py`](../../../spikes/int001-sentencepiece-capi/verify_source_archive.py)
checks archive path safety, rejects links and non-regular entries, requires an
exact file set, compares every file using Git's blob hash construction, and
compares executable bits. Results:

- official tree blobs: 258;
- archive regular files: 258;
- exact Git blob matches: 258;
- line-ending-normalized matches required: 0;
- executable blobs/modes: 7/7;
- missing, extra, unsafe or substantive content mismatches: 0.

An initial shell comparison incorrectly reported three CRLF/LF differences
because `git hash-object` applied the current repository's working-tree clean
filter. Repeating it with `--no-filters`, and the verifier's direct Git blob
construction (`SHA-1("blob <size>\\0" + raw bytes)`), matched all 258 official
blob IDs exactly. Official raw endpoints also returned bytes identical to the
archive for the three files affected by that local filter. The committed
verifier deliberately avoids repository attributes and working-tree filters.

This proves that the fixed archive content is equivalent to the official
commit used by the version tag. It does not prove that the archive bytes came
from one particular upstream download URL, and it does not provide signed
upstream provenance. Release construction should either fetch a reviewed URL
by this exact archive digest or reconstruct a deterministic archive from the
pinned commit and record signed build provenance.

Direct fetches of the official tag archive URL did not produce a complete
artifact in the bounded evidence window. The first connection reached the
fixed 300-second timeout after 13,007,573 of the expected 13,485,527 bytes. A
second fresh-file attempt on 2026-08-28 reached 10,155,379 bytes at the same
timeout. GitHub redirected both requests to
`https://codeload.github.com/google/sentencepiece/tar.gz/refs/tags/v0.2.1`;
the codeload response did not support byte-range resume, so the partial files
could not be completed safely. Automatic retry of the second attempt was
stopped rather than starting another unbounded full transfer. These incomplete
responses are discarded as evidence and are not compared or recorded as
upstream artifact hashes. No retry result is represented as successful.

The verification used only public official GitHub repository metadata and raw
content. The recursive tree was fetched from:

```text
https://api.github.com/repos/google/sentencepiece/git/trees/31646a467d2051eb904e0b45de3a73e91fe1c1e3?recursive=1
```

The offline comparison command was:

```sh
python3 spikes/int001-sentencepiece-capi/verify_source_archive.py \
  --archive sentencepiece-0.2.1.tar.gz \
  --tree sentencepiece-v0.2.1-tree.json
```
