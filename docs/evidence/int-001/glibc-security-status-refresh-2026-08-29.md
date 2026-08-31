# Distroless Debian 13 glibc security status refresh (2026-08-29)

Status: **blocker unchanged; no fixed trixie base is available from the inspected
distroless tags**.

This is a read-only refresh of the isolated INT-001 runtime evidence. It does
not approve ADR-0014, change the production composition, or replace a final
signed SBOM/VEX review.

## Official Debian status

The Debian Security Tracker reported `glibc 2.41-12+deb13u3` in trixie as
`vulnerable` for all three findings on 2026-08-29:

| Finding | Debian trixie status | Debian disposition |
| --- | --- | --- |
| [CVE-2026-5450](https://security-tracker.debian.org/tracker/CVE-2026-5450) | `2.41-12+deb13u3`, vulnerable | `no-dsa` (Minor issue) |
| [CVE-2026-5928](https://security-tracker.debian.org/tracker/CVE-2026-5928) | `2.41-12+deb13u3`, vulnerable | `no-dsa` (Minor issue) |
| [CVE-2026-5435](https://security-tracker.debian.org/tracker/CVE-2026-5435) | `2.41-12+deb13u3`, vulnerable | `no-dsa` (Minor issue) |

The tracker lists fixed versions only for Debian unstable/newer releases. A
`no-dsa` note is not a fixed-package claim and does not satisfy the frozen
Critical/High release gate by itself.

## Current distroless inspection

Commands executed successfully against the current tags:

```sh
docker buildx imagetools inspect gcr.io/distroless/base-nossl-debian13:nonroot
docker buildx imagetools inspect gcr.io/distroless/cc-debian13:nonroot
docker pull --platform linux/arm64 gcr.io/distroless/cc-debian13:nonroot@sha256:f80223ec87492ae8c651d0d966ecf19af48faff2e7b31814eecddc4db8c9c3e6
docker pull --platform linux/amd64 gcr.io/distroless/cc-debian13:nonroot@sha256:1d2e87077bb3b12be8622609c5975fed6a3cba63e68fed53209293be10f7022c
```

Resolved index and child manifests:

| Image | Index digest | linux/arm64 child | linux/amd64 child |
| --- | --- | --- | --- |
| `base-nossl-debian13:nonroot` | `sha256:5cab74e7f8a5e7c5f1c8a9e6268b1f352f053c36c656f493308340bcecbc636c` | `sha256:f0d71cf317129017537c5b1938e3be83d0f8a1edefd37aa6ff3b1ce80a4dc51d` | `sha256:cc74a68b2924afee50ab111f14d86b9f4e1c461d02ac8382708343f97f6b6f33` |
| `cc-debian13:nonroot` | `sha256:c31ff9abcb1910f3ab25c7957bdaf0bfe12a01eb546e8df2282f1c8f682b606c` | `sha256:f80223ec87492ae8c651d0d966ecf19af48faff2e7b31814eecddc4db8c9c3e6` | `sha256:1d2e87077bb3b12be8622609c5975fed6a3cba63e68fed53209293be10f7022c` |

The `/var/lib/dpkg/status.d/libc6` record read directly from both current `cc`
children reports:

| Platform | Package version |
| --- | --- |
| `linux/arm64` | `libc6 2.41-12+deb13u3` |
| `linux/amd64` | `libc6 2.41-12+deb13u3` |

The current tag therefore still contains the exact trixie version Debian marks
vulnerable. Although the `cc` tag's arm64 child digest has changed since the
frozen runtime Dockerfile, the glibc blocker has not. Churning that pin would
not remove the finding and was deliberately avoided.

## Gate consequence

- Keep the prior 1 Critical / 2 High cross-architecture Grype result open.
- Do not promote the isolated SentencePiece/ORT runtime into production.
- Do not reinterpret `no-dsa`, a changed image digest, the dynamic tripwire, or
  indirect reachability as a signed VEX.
- Reopen this check when Debian publishes a fixed trixie `libc6`, distroless
  exposes a child containing that version, or the security owner signs an exact
  call-path/runtime-configuration VEX. A fixed-base path still requires a fresh
  dual-architecture build, complete SBOM and vulnerability scan.
