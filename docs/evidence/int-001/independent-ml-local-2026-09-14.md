# Independent ML local probe — 2026-09-14

Scope: POST-MVP-5 / CR-2026-030 / ADR-0019. User authorized the split-service
direction and copying existing server222 models to /Volumes/extended/foliopath.
No production service was changed and no model was uploaded to HF.

Server222's Immich server reports v3.1.0. Its machine-learning Compose section is
commented out, but /root/immich/model-cache contains ViT-B-32__openai and buffalo_l.
Copied cache: /Volumes/extended/foliopath/model-cache, about 783 MiB.
The first staging copy remains under
/Users/qu/Library/Caches/FolioPath/immich-model-test.NxOs6s/model-cache;
it was not automatically deleted after the user changed destination.

Four ONNX SHA-256 values match the source:

An additional checksum-mode rsync dry run over the complete copied cache returned
no differences (including tokenizer and configuration files, not only ONNX).

| File | Bytes | SHA-256 |
| --- | ---: | --- |
| CLIP visual | 351613724 | 33a3df41ceef21acdf371af00f6dd0456ec1f9eba24d03a7720f9c3734e40859 |
| CLIP textual | 254193396 | b80cf0af751533a6712d92247f0ddc0c95208748bc59a1a27f33e67be6864e3b |
| buffalo_l detection | 16923827 | 5838f7fe053675b1c7a08b633df49e7af5495cee0493c7dcf6697200b85b5b91 |
| buffalo_l recognition | 174383866 | 6afbf406aa229a439abbca7436cc42be254d4e3af6200d8b7ae4c1fec0c18f2f |

Local Docker runs Linux/arm64. Official ML image:
ghcr.io/immich-app/immich-machine-learning:v3.1.0
@sha256:5a0839dc5303cd7215bcd2180a26aed3af41675aefb3e75e5157e9f10ad16e6e.
Two-container offline HTTP probe in spikes/int001-remote-ml passed:
imageDimensions=512, textDimensions=512, repeatStable=true, blankImageFaceCount=0.
Restarting the ML container and rerunning the same offline probe also passed.
The test container was then stopped to release resources; its Compose is retained.
Compose config validation, make arch-check, make fmt-check and git diff --check passed.
ML server is capped at 2 CPUs / 2 GiB; these limits are not a production capacity proof.
No media/database mount, no public port, internal-only network, read-only model cache.

This is **not** a FolioPath main-application integration. Production protocol,
generation migration, model identity/dimension validation, failure semantics,
downloads/supply-chain validation and UI migration remain to be implemented.
Automated probe uses only synthetic pixels; copying a real library does not add it
to the automated test suite.

## User-requested media copy

Copied server222 /mnt/r0data/picture/coser/ to
/Volumes/extended/foliopath/library/coser/ with rsync -a --safe-links --partial,
without deletion flags. Transfer exited successfully. Both sides contain 6,221
regular files totaling 60,710,223,481 bytes. A subsequent read-only size/mtime
rsync check reported zero files to transfer. This is not an independent whole-
library cryptographic checksum audit. Source files were not changed.
The copied real library was not opened by the automated inference probe.
