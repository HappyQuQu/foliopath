# Prototype Instructions

Run the local server yourself and open the preview in the browser available to this environment. Do not give the user server-start instructions when you can run it.

Before making substantial visual changes, use the Product Design plugin's `get-context` skill when the visual source is unclear or no longer matches the current goal. When the user gives durable prototype-specific design feedback, preferences, or decisions, record them in `AGENTS.md`.

When implementing from a selected generated mock, treat that image as the source of truth for layout, component anatomy, density, spacing, color, typography, visible content, and hierarchy.

Build app UI in `src/`. Keep `.openai/hosting.json`, `worker/index.js`, `scripts/prepare-sites-build.mjs`, and `tests/sites-worker.test.mjs` intact so the same local prototype can be handed to Sites. Before a Sites handoff, run `npm run build` and `npm run test:sites`; the build must leave `dist/client/index.html`, `dist/server/index.js`, and `dist/.openai/hosting.json`.

## FolioPath prototype decisions

- This is a disposable static UI prototype only. Keep it outside the production
  `web/src` import graph and do not connect it to FolioPath APIs.
- Recreate visual direction 1 from the selected ImageGen concept. Light and dark
  modes are two token states of the same information architecture, not separate
  layouts.
- Provide a local theme switch and remember the explicit choice in browser
  storage.
- Responsive behavior follows the accepted UI specification: a fixed directory
  sidebar at desktop widths, an overlay drawer on tablet, and a full-width
  directory panel plus a two-column media grid on mobile.
- Use local mock data and local prototype-only media fixtures. Original media
  remains read-only and no host or container absolute path may appear in UI copy.
- Apply the web-relevant parts of Emil Kowalski's Apple Design skill as a review
  lens: immediate press feedback, spatially consistent drawer/viewer transitions,
  platform-system typography, purposeful restraint, adaptive light/dark colors,
  clear wayfinding, and reduced-motion/high-contrast fallbacks. Do not add
  Apple-like translucency or spring motion where it would conflict with
  FolioPath's quiet, cross-platform UI specification.
- Media detail uses a non-modal, resizable right-side inspector on desktop. It
  must never dim or cover the gallery. At narrower widths it becomes a
  non-overlapping bottom workspace or an in-flow panel.
- The preview may be pinned. When pinned, browsing changes grid selection without
  replacing the preview or interrupting video playback. Double-clicking a media
  card explicitly replaces the pinned preview; previous/next actions also change
  it.
