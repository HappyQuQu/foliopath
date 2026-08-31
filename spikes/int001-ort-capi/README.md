# INT-008 Go/C API spike

This nested Go module keeps ONNX Runtime, cgo flags and the shared library out
of FolioPath's production dependency graph. It must be compiled in a disposable
native Linux environment with reviewed ONNX Runtime headers and libraries
provided through `CGO_CFLAGS`, `CGO_LDFLAGS` and `LD_LIBRARY_PATH`.

The harness uses C-allocated tensor memory, disables the CPU memory arena,
maps runtime failures to numeric errors without exposing raw messages, runs a
fixed embedded control graph, rejects an invalid graph, cancels a SigLIP image
inference and proves recovery. It is evidence code, not a production adapter.

The adjacent `Dockerfile` is likewise an isolated Linux/arm64 evidence image.
It pins the official ONNX Runtime archive by SHA-256, builds the harness with
the reviewed C header/library, copies the runtime license and third-party
notices, runs as uid/gid 65532 on a pinned distroless C/C++ base and expects all
model fixtures to be supplied as read-only mounts. It must not be referenced by
the production Dockerfile or release workflow. Native amd64 requires its own
official archive/hash and native execution evidence; emulation is not a
substitute.

`Dockerfile.nossl` is a second isolated comparison fixture. It starts from the
pinned distroless `base-nossl` image and copies only the Debian-tracked C++
runtime libraries required by the official ORT shared object, together with
their package metadata and distribution license files. It exists to test
whether OpenSSL can be removed from the final inference closure; it is not a
release-image decision.

`onnxruntime-component.cdx.json` is a supplemental CycloneDX 1.6 component for
the official Linux/aarch64 ORT archive because the image scanner identifies the
shared object only as a file, not as a package. It records the runtime, archive,
source, license and notices hashes and deliberately declares an incomplete
composition. A release pipeline must validate and merge an architecture-specific
component into the final image SBOM; this spike document alone is not a complete
image SBOM, vulnerability result, VEX or signed provenance statement.
