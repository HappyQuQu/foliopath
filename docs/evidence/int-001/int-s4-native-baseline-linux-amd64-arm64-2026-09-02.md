# INT-S4 native baseline evidence — linux/amd64 + linux/arm64

Status: **paired native baseline passed; final model/release evidence remains absent**

- Source commit: `5af4da0ea79a8ebfcf1042b83ad737d576343090`
- GitHub Actions run: [`33616238888`](https://github.com/HappyQuQu/foliopath/actions/runs/33616238888), attempt 1
- Paired summary SHA-256: `459a8be9c2506b062ca85312fb2f1cdf207f0469a5101a4d6ac6f5587ab94e4b`
- Workflow result: amd64 passed, arm64 passed, paired verifier passed
- Publication boundary: both Docker publish jobs were skipped by the evidence-only sentinel

## Native identity and step outcomes

| Check | amd64 | arm64 |
| --- | --- | --- |
| Runner | `ubuntu-24.04` | `ubuntu-24.04-arm` |
| Kernel machine | `x86_64` | `aarch64` |
| Go / Docker | `amd64` / `linux/x86_64` | `arm64` / `linux/aarch64` |
| QEMU allowed | false | false |
| Repository / libvips / candidate / matrix / capacity | all success | all success |
| Outcome complete | true | true |

The paired verifier independently reopened both downloaded artifacts and bound them to the same source commit, workflow
run and attempt. A second local invocation of `make verify-intelligent-media-native-evidence` against the downloaded
artifacts produced the same passing summary and SHA-256.

## Enforced 10k/100k capacity

Both jobs ran with `GOMAXPROCS=4`; the capacity result contained no budget violations.

| Metric | Budget | amd64 | arm64 |
| --- | ---: | ---: | ---: |
| Full scan | `<=120,000 ms` | `51,738 ms` | `45,441 ms` |
| Concurrent read P95 | `<=250,000 us` | `1,325 us` | `1,186 us` |
| Concurrent search P95 | `<=500,000 us` | `168,724 us` | `187,593 us` |
| FTS search P95 | `<=250,000 us` | `105,749 us` | `131,246 us` |
| Two-page production keyset P95 | `<=250,000 us` | `182,299 us` | `247,994 us` |
| First / second search page P95 | diagnostic | `143,137 / 33,032 us` | `213,055 / 41,301 us` |
| Peak RSS | `<=1 GiB` | `60,002,304 bytes` | `48,594,944 bytes` |
| DB + WAL | `<=1 GiB` | `160,079,872 bytes` | `158,478,336 bytes` |

The order-first correctness matrix also passed on both architectures. This baseline measures catalog scan/search and
candidate/synthetic face capacity; it does not include the absent final semantic/tag/video/face model workload.

## Candidate and synthetic face boundary

Both architectures loaded the same pinned detector, embedder and public smoke fixture, producing three finite candidate
vectors. The structured evidence explicitly records `productionApproved=false`, `qualityGate=false`, and
`complianceGate=false`. The cross-architecture quantized vector hashes differ, so this preflight is not a numeric parity
claim.

The independent synthetic capacity check passed for 100,000 faces × 512 dimensions in a 4 CPU / 4 GiB container:

| Metric | amd64 | arm64 |
| --- | ---: | ---: |
| Elapsed | `12,012 ms` | `11,648 ms` |
| Runtime memory sys | `421,122,408 bytes` | `416,426,344 bytes` |
| Deterministic result SHA-256 | `ed978ca7f471ba742a38f680cebb5f83481b8f622a70b005b906e654f2b706d4` | same |

The evidence itself records `identityGroundTruth=false` and `qualityGate=false`.

## Gate interpretation

This run closes the previously missing same-commit native amd64/arm64 **baseline** execution and proves the frozen
catalog capacity budgets after the scan/keyset fixes. It does not provide:

- a reviewed final model package or a shared final image digest;
- governed semantic/tag/video/face quality, 50×20 identity ground truth, bias slices or 99.5% core precision;
- strict final-model numeric parity, final joint RSS/rebuild/recovery reports;
- final SBOM/VEX/notices/provenance or owner approvals.

Therefore `INT-402` and `INT-403` advance but remain unchecked, and S4 remains Release No-Go.
