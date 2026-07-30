# FIX-2026-07-30: Browse prototype shell scroll height

- Type: routine prototype fix
- Scope: accepted MVP browse prototype
- Delivery gate: existing consumer UI design evidence
- Affected invariant: the browse shell must occupy one viewport baseline without
  duplicating global navigation or creating trailing blank scroll space.

## Problem

The browse prototype nested a second global header and application layout inside
the first shell. Because every application layout has a viewport minimum height,
the duplicate wrapper added an extra viewport of document height and exposed a
blank region when scrolling to the bottom.

## Change

Keep one global header and one application layout on the browse page. Preserve the
existing content, navigation, responsive behavior, and visual tokens.

## Evidence

- Static structure check: one `.global-header` and one `.app-layout` on the browse
  page.
- Browser regression: the browse page reaches its content boundary without a
  trailing empty viewport at desktop and mobile widths.
