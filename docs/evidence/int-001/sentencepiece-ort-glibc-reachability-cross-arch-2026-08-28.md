# Text-runtime glibc reachability input across architectures

Status: **the expanded closure found a direct `ungetwc` import; no VEX decision
is made and the release Gate remains failed**.

The earlier reachability check covered only the Go harness and ONNX Runtime.
That limited statement was accurate for those two files but insufficient for
the combined runtime. The current check covers the external non-glibc ELF
closure: the application, ONNX Runtime, SentencePiece, libstdc++ and libgcc_s.

Debian's tracker describes the three current glibc findings as follows:

- `CVE-2026-5450`: a one-byte heap overflow when a scanf-family `%mc`
  conversion uses an explicit width greater than 1024;
- `CVE-2026-5928`: an `ungetwc` buffer under-read involving character sets
  whose single-byte and multibyte encodings overlap;
- `CVE-2026-5435`: an out-of-bounds write in deprecated `ns_printrrf`,
  `ns_printrr` and `fp_nquery` DNS debugging functions.

Across 575 unique arm64 and 573 unique amd64 undefined symbols, neither image
imports the affected scanf family or the three deprecated DNS functions from
the external closure. Both images do import `ungetwc` through their exact
`libstdc++.so.6.0.33`. Disassembly confirms calls from
`__gnu_cxx::stdio_sync_filebuf<wchar_t>::underflow()` and `pbackfail()` on both
architectures.

A follow-up dynamic tripwire interposed `ungetwc` with `LD_PRELOAD` under the
same no-network, read-only, cap-drop, 4 CPU/4 GiB runtime profile. A dedicated
control probe called `ungetwc`; on both architectures the tripwire printed its
marker and terminated the process with exit 86, proving that interposition was
active. The fixed 31-case tokenizer-to-text-encoder suite then passed with the
same tripwire loaded: arm64 retained maximum absolute difference
`1.811981201171875e-05`, and emulated amd64 retained
`2.09808349609375e-05`. Neither run triggered the tripwire.

This does not prove FolioPath invokes the vulnerable wide-stream path or uses
an overlapping character encoding. It does prove that “no direct affected
import” is not a valid blanket justification for `CVE-2026-5928`. A security
owner would need a reviewed call-path and runtime-configuration argument before
issuing VEX, or the image must move to a fixed glibc base. Absence of direct
imports for the other two findings also cannot exclude glibc-internal paths,
dynamic lookup or future production code.

The dynamic result narrows the current fixed inference path: it did not call
`ungetwc` in these 31 cases under the default container locale. It does not
cover every libstdc++ path, alternate locales/encodings, future production HTTP
composition or native amd64 execution. It therefore remains VEX input rather
than a VEX decision.

The machine-readable record fixes all five binary scopes, aggregate symbol-list
hashes, result hashes and the exact architecture-specific libstdc++ hashes. The
amd64 half remains QEMU package evidence and is not native runtime proof.
