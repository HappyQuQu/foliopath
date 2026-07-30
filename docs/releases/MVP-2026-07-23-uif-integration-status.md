# MVP-2026-07-23 UIF integration status

## Status

- Scope: frozen revision 4; this document does not change the manifest.
- Feature: [`FTR-UIF-001`](../features/frontend-prototype-fidelity.md).
- Current result: `UIF-401～408` completed; `UIF-S4 Integrated Slice Done` is Go.
- Next decision: none inside `FTR-UIF-001`; continue the independent Stage 5
  release-blocker closure work.
- Release status: not released; the existing Release Candidate remains No-Go
  until all independent Stage 5 blockers close.

Historical Architecture, Contract, Backend and Consumer/UI Gate documents keep
the state and pending work that was true when each Gate was signed. This
current additive status points to later evidence instead of rewriting those
records.

## Integrated evidence

| Slice | Result and evidence |
| --- | --- |
| `UIF-401` | Twelve manifest pages compared at one shared `1280 × 720` CSS viewport with the latest prototype and real production routes; no P0/P1/P2 or deferred P3 finding. [Evidence](../evidence/uif-401/README.md) |
| `UIF-402` | Eleven Linux-owned Chromium baselines cover Header, management pages, Browse top/bottom/preview, Search and Viewer; comparison rerun passed `9`. [Evidence](../evidence/uif-402/README.md) |
| `UIF-403` | Real setup/login → account → library/scan → directory q → Search/preview/Viewer → settings/cache → logout/login chain passed; original-media path and SHA-256 manifests remained identical. [Evidence](../evidence/uif-403/README.md) |
| `UIF-404` | Chromium, Firefox, WebKit, axe, keyboard, emulated touch, Chrome forced-colors and reduced-motion applicability passed; physical accessibility remains assigned to S5-006B. [Evidence](../evidence/uif-404/README.md) |
| `UIF-405` | Three-engine 100k scroll/DOM/FPS/RSS and backend 10k-directory/100k-file scan-time concurrency passed with zero budget violations. [Evidence](../evidence/uif-405/README.md) |
| `UIF-406` | `fmt`, architecture, generation, lint, unit, integration and production-container E2E all passed. [Evidence](../evidence/uif-406/README.md) |
| `UIF-407` | PRD, UI, flows, API, data, security, testing, deployment, traceability, risk, README and release status converged on the same implementation/evidence boundary. [Evidence](../evidence/uif-407/README.md) |
| `UIF-408` | Twelve pages were reviewed source/production side by side at all four declared CSS viewports; affected browser, capacity, container and RC readiness checks were rerun, and UIF-S4 was signed Go. [Evidence](../evidence/uif-408/README.md) / [Gate](../gates/MVP-2026-07-23/uif-s4-integrated-slice-done.md) |

The shared bilingual, dual-theme state matrix remains in
[`UIF-317`](../evidence/uif-317/README.md). The distinct twelve-page,
four-breakpoint source/production matrix now exists in `UIF-408`; neither
evidence set is mislabeled as the other.

## Acceptance evidence accepted by UIF-S4

| Acceptance | Accepted evidence |
| --- | --- |
| `UIF-AC-001～003` | Shared Header/management shell and twelve-page comparison (`UIF-401`); real route/actions (`UIF-403`) |
| `UIF-AC-004～006` | Account transaction, directory q and cache cleanup Backend Ready plus real browser chain (`UIF-S2`, `UIF-403`) |
| `UIF-AC-007～009` | Browse top/bottom/preview comparison, shared responsive matrix, Linux baselines and twelve-page four-breakpoint review (`UIF-317`, `UIF-401`, `UIF-402`, `UIF-408`) |
| `UIF-AC-010` | Browser/input/accessibility applicability (`UIF-404`) |
| `UIF-AC-011` | Frontend and backend full-capacity revalidation (`UIF-405`) |
| `UIF-AC-012` | Read-only mount, media path/SHA-256 invariance and complete repository verification (`UIF-403`, `UIF-406`) |

The signed decision is
[`UIF-S4 Integrated Slice Done`](../gates/MVP-2026-07-23/uif-s4-integrated-slice-done.md).

## Scope exclusions preserved

The current production routes do not include missing-cache backfill, rebuild
all, the unified task center, system maintenance, integrity reports,
application backup/diagnostic packages, AI semantic search, OCR or face
recognition. Those prototypes and proposals remain Post-MVP/future inputs and
must not be reintroduced as static controls or local mock success.

## Remaining release work

1. Keep S5-006B physical Firefox, screen reader, 200%/400% zoom, OS
   high-contrast and representative touch/mobile review open.
2. Keep final immutable digest and unresolved supply-chain findings under
   their existing Stage 5 owners.
