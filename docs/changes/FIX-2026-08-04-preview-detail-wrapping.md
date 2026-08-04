# FIX-2026-08-04: Wrap long media preview details

- Type: routine fix inside the accepted non-modal media preview slice
- Requirement / quality ID: FR-MED-003 / NFR-ACC-001
- Target: MVP Stage 3 browse and Stage 4 search
- Owner: shared `web/src/components/patterns/MediaPreview`
- Delivery gate: `docs/gates/MVP-2026-07-23/s4-frontend-media-matrix.md`
- Affected invariant: safe library-relative metadata must remain readable without
  horizontal page overflow at supported preview widths.

## Problem

Every preview detail value was forced onto one line and truncated with an ellipsis.
Long library-relative locations therefore hid the selected asset's path, especially
for Chinese names or uninterrupted filename segments.

## Change

Allow shared preview detail values to wrap normally and break anywhere when a long
segment cannot otherwise fit. Preserve the fixed label column, right alignment,
single-line filename identity, and scrollable details region.

## Evidence

- Shared component regression verifies that a long mixed Chinese and uninterrupted
  path uses normal wrapping with `overflow-wrap: anywhere`.
- Browse and search continue to consume the same shared preview component.
