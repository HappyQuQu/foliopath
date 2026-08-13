# API contract

`openapi.yaml` is the authoritative HTTP structure for the currently implemented
contract baseline, including the frozen MVP and accepted Post-MVP slices through
`POST-MVP-4`. `openapi.sha256` is an exact revision lock, not a substitute for
semantic compatibility analysis.

Changing the contract requires:

1. an approved requirement/change record and any required ADR;
2. updating `openapi.yaml` and its contract tests;
3. regenerating the TypeScript contract with `make generate`;
4. reviewing the semantic compatibility impact;
5. updating `openapi.sha256` only after that review; and
6. passing `make contract-check generate-check`.

Generated files must never be edited directly.
