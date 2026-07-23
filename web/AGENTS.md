# FolioPath Web Instructions

These instructions apply to `web/` in addition to the repository-level
`AGENTS.md`.

## Current gate

- The current directory is an engineering/contract foundation, not an approved
  product UI.
- Do not add routes, pages, business features, mock product behavior, or a second
  API shape until the corresponding slice has passed Backend Ready.
- A disposable design prototype must remain outside the production import graph
  and be removed or explicitly promoted through the normal Gate.

## API boundary

- `src/lib/api/generated/schema.ts` is generated from `api/openapi.yaml`. Never
  edit it directly; run `make generate` from the repository root.
- `src/lib/api/client.ts` is the only raw `openapi-fetch` client. Do not call
  `fetch` or import `openapi-fetch` elsewhere.
- Product code must consume reviewed domain adapters built under `src/lib/api`;
  features must not import generated types or the raw client directly.
- CSRF, error mapping, query keys, invalidation, retries, and URL codecs each
  require one canonical owner. Do not reimplement them per feature.

## Components and styles

- Shared primitives belong in `src/components/ui`; cross-feature workflows
  belong in `src/components/patterns`. Add a reviewed variant instead of
  creating a near-duplicate component.
- Design tokens and theme rules belong in the central style source. Feature code
  must not invent a second spacing, color, radius, motion, or z-index system.
- Preserve semantic controls, DOM order, visible focus, keyboard operation,
  reduced motion, and bounded/virtualized large collections.

## Verification

- Use the Node and npm versions pinned by `.node-version`, `packageManager`, and
  `package-lock.json`.
- Run `npm ci`, `npm run generate:check`, `npm run check:types`, and the relevant
  component/E2E checks once those layers exist.
- Do not bypass peer dependency or audit failures with `--force` or
  `--legacy-peer-deps`.
