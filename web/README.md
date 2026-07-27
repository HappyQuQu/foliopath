# FolioPath Web

This directory contains the production React/Vite application. The current implemented
vertical slice includes the generated OpenAPI contract, single typed HTTP client
boundary, application providers, global safe error boundary, theme/token system,
shared UI primitives, component workbench, and the real administrator setup, login,
session recovery, logout, and general account settings flow.

The remaining library, scan, browse, search, preview, and viewer routes are reserved in
`src/routes/paths.ts` and are implemented only after their corresponding frontend slice
starts. Product code must not introduce mock success paths: API behavior comes through
the generated client and hand-written domain adapters under `src/lib/api`.

Install and verify:

```sh
npm ci
npm run check
npm run build
```

Regenerate after an accepted OpenAPI change:

```sh
npm run generate:api
```

Run the component workbench:

```sh
npm run storybook
```
