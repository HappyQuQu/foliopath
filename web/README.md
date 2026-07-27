# Web engineering foundation

This directory contains the production React/Vite engineering foundation: the
generated OpenAPI contract, single typed HTTP client boundary, application providers,
global safe error boundary, theme/token system, shared UI primitives, component
workbench, and focused tests.

It is not yet a product UI. Product routes and business features remain intentionally
unregistered until the applicable Backend Ready Gate is recorded. React Router is
installed for that next gated step; do not introduce mock success paths to activate
routes early.

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
