# Prototype Instructions

2026-09-14: user chose the first displayed design, the quiet settings layout.
This is a disposable ADR-0019 / CR-2026-030 prototype, not production AI UI.
Reuse web shared components and tokens; do not copy their implementations.
Capability toggles apply to the explicitly selected library. CLIP is semantic,
buffalo_l is face-only. All service, job and coverage states are examples.
Do not add live API, Docker or media access. Testing stays local, never server222.

Run the local server yourself and open the preview in the browser available to this environment. Do not give the user server-start instructions when you can run it.

Before making substantial visual changes, use the Product Design plugin's `get-context` skill when the visual source is unclear or no longer matches the current goal. When the user gives durable prototype-specific design feedback, preferences, or decisions, record them in `AGENTS.md`.

When implementing from a selected generated mock, treat that image as the source of truth for layout, component anatomy, density, spacing, color, typography, visible content, and hierarchy.

Build app UI in `src/`. Keep `.openai/hosting.json`, `worker/index.js`, `scripts/prepare-sites-build.mjs`, and `tests/sites-worker.test.mjs` intact so the same local prototype can be handed to Sites. Before a Sites handoff, run `npm run build` and `npm run test:sites`; the build must leave `dist/client/index.html`, `dist/server/index.js`, and `dist/.openai/hosting.json`.
