# FolioPath Web

This directory contains the production React/Vite application. Implemented vertical
slices include authentication and administration, multi-library scan/status views,
virtualized browse and search, non-modal preview and full viewer, video hover
storyboards, notifications and processing diagnostics, plus favorites and manual tags.

All server behavior comes through the generated OpenAPI client and hand-written domain
adapters under `src/lib/api`; product UI must not introduce mock success paths. Shared
media cards, collections, preview/viewer primitives, URL state, query keys, localization,
theme and preferences remain canonical owners rather than being copied into features.

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

Run the real-backend authentication browser slice after installing Chromium with
`npm exec --prefix web -- playwright install chromium`:

```sh
make test-web-e2e
```
