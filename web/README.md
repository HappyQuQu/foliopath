# Web contract foundation

This directory currently contains only the generated OpenAPI contract, the single
typed HTTP client boundary, and strict TypeScript/toolchain configuration. It is not
a product UI and must not grow business features before the applicable Backend Ready
Gate.

Install and verify:

```sh
npm ci
npm run generate:check
npm run check:types
```

Regenerate after an accepted OpenAPI change:

```sh
npm run generate:api
```
