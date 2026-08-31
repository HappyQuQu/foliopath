# INT-011 production storyboard adapter evidence

Status: **bounded technical evidence; real-content quality remains open**.

## Result

The existing `internal/media/videoffmpeg.Processor.Storyboard` adapter processed
both the four-frame fallback and ten-frame normal plan on native Linux/arm64.
Both outputs passed the canonical `media.ValidateStoryboardResult` validation.
The input MP4 SHA-256 was identical before and after both operations, proving
that this path did not modify original media.

This is the same FFmpeg adapter and bounded command runner already owned by the
media capability. An eventual frame-embedding consumer can consume the
canonical storyboard plan/result; it must not add a second FFmpeg admission or
invoke FFmpeg directly.

## Reproduction facts

- Runtime: native `linux/arm64` Docker Desktop
- FFmpeg: project-pinned `7.1.5`, built by the repository `ffmpeg` target
- Image digest: `sha256:ace3bad090f5fdea66f980fa49d514dca9caae6bb4ffdd0bbd028c3c425f74fb`
- Fixture: generated `testsrc2`, H.264, 320×180, 24 fps, 10 seconds
- Fixture bytes: `322066`
- Fixture SHA-256: `7880c201c8864a7620c39954599f642ab447c8f71c60e8e87ae4e67794850899`
- Test: `TestPinnedFixtureFourAndTenFrameStoryboards`
- Result: PASS in 0.11 seconds

The fixture and test binary were created under a disposable temporary
directory and were not committed.

## What this does not prove

- The synthetic pattern does not measure representative Coser/model video
  sampling quality, semantic retrieval quality, or mean/max-frame ranking.
- It does not satisfy the licensed 100-video acceptance set.
- It does not test semantic embeddings, 4 CPU/4 GiB joint load, native
  Linux/amd64, or concurrent browsing.
- It therefore closes only the extraction/reuse sub-proof, not `INT-011B` or
  the parent `INT-011` task.
