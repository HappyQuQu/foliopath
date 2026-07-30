# UIF-407 documentation convergence evidence

## Scope

UIF-407 reconciles the current, continuously maintained documentation with the
latest accepted prototype and the actual UIF-401～406 implementation evidence.
It does not rewrite frozen scope manifests or historical Gate decisions.

The review covered:

- root `README.md` and the documentation index;
- product requirements and the FTR-UIF-001 feature/implementation plan;
- UI design and user flows;
- API, data-model and security boundaries;
- testing strategy and candidate deployment status;
- architecture traceability and the risk register;
- release documentation and the UIF development checklist.

## Conflicts closed

1. API design no longer says that authentication and business handlers are
   absent; it identifies the implemented handler and generated-client
   boundary.
2. Account maintenance, directory `q` and cache cleanup are no longer
   described as future backend gaps. Their accepted OpenAPI, migration 13,
   Backend Ready and real browser evidence are linked.
3. The feature and implementation plan no longer stop at Consumer/UI Ready;
   they record UIF-401～407 and leave only UIF-408 plus Stage 5 reruns.
4. The visual evidence is described precisely: twelve page-by-page prototype
   comparisons share a `1280 × 720` CSS viewport, while the four breakpoint
   bilingual/dual-theme matrix is separate responsive and state evidence.
5. Task-center, maintenance, backup/diagnostics, AI search, OCR and face
   recognition remain excluded from the production MVP feature.
6. Deployment status now matches accepted native dual-architecture,
   upgrade/rollback and capacity evidence while retaining the real
   browser/accessibility, digest, supply-chain and RC blockers.

## Release-document handling

`MVP-2026-07-23-scope-r4.md` remains unchanged because an accepted scope
manifest is append-only history. The historical UIF-S0～S3 Gate documents also
remain unchanged. Current facts are recorded in the additive
[`MVP-2026-07-23 UIF integration status`](../../releases/MVP-2026-07-23-uif-integration-status.md),
which links every Integrated Slice evidence package and the remaining
UIF-408/Stage 5 work.

## Outcome

The durable product, UI, API/data/security, testing, deployment, traceability,
risk and release documents now use the same scope and evidence boundary.
UIF-407 is complete, but this evidence does not self-sign UIF-S4 or declare the
candidate released.
