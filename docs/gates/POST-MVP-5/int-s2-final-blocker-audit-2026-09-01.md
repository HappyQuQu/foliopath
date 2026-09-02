# POST-MVP-5 revision 2：S2 最终完成审计（2026-09-01）

- 范围：S2A、S2B、S2C 全部后端实现、测试、证据与 Gate
- S2 判断：**Backend Ready**
- 发布判断：**Release No-Go**
- Stage decision：[CR-2026-022](../../changes/CR-2026-022-s2-backend-release-gate-separation.md)

## 完成性复核

逐项复核权威任务清单、production composition、三个 S2 Gate 和当前证据后：

- S2A：`INT-201～216` 完成；semantic format v2、SentencePiece/ORT image+text runtime、model lifecycle、
  backfill/clear/search、transport/fault matrix 与本地 100k 容量均有正式 owner 和回归；
- S2B：`INT-221～228` 完成；tag/video repository、durable worker、curation/review、storyboard-only frame
  source、ranking/cursor、quality scorer/validator 与失败回退已实现；
- S2C：`INT-241～251` 完成；face package/generation、observation/cluster/person/manual relationship、worker/
  API、清除/备份、安全读取、source/offline/kill、100k×512，以及授权 `coser` 私有功能矩阵已完成。

S2 完成不等于功能可发布。产品用户要求本阶段只验证功能实现并保持发布 fail-closed；CR-2026-022 因而
把最终发行输入恢复到 S4，而不是删除或降低它们。

## Production fail-closed 证明

- production reviewed AI catalog 仍为空；任何真实 semantic activation/search 都稳定 unavailable；
- semantic fail-closed route 已组合，用于验证完整 auth/CSRF/rate-limit/error boundary；
- face runtime、session 和 route 不进入 production app composition；设置默认关闭；
- candidate model、QEMU/foreign-arch preflight、synthetic capacity 和 `coser` 结果均未被标记为 release、
  quality、identity 或 compliance approval；
- 原媒体保持只读，私有人脸 crop/vector/path/review CSV 只在 `/tmp`，不进入 Git/CI/API/日志。

## S4 Release 阻塞

以下真实外部输入仍缺失并由 `INT-401～411` 持有：

1. 最终审核 semantic/face model package、精确权重许可/训练来源结论、catalog entry 与 threshold profile；
2. governed semantic/tag/video/face dataset、逐项结果及 product/ML/QA/privacy/compliance 批准；
3. 同 source/package/final-image digest 的 native Linux amd64+arm64 paired artifact；
4. 最终模型联合 100k backfill/search/video/face/SQLite/HTTP/browse、RSS 与空间结果；
5. 完整 SBOM、VEX、notices、签名 provenance 和 security/compliance/inference/release 签署。

五个最终入口继续对缺失输入失败关闭：

```text
make verify-intelligent-media-quality
make verify-intelligent-media-face-quality
make verify-intelligent-media-native-model-evidence
make verify-intelligent-media-supply-chain
make verify-intelligent-media-s2-evidence
```

它们通过自身单元测试只证明 verifier 正确，不能产生 S4 Go。S4 任一输入缺失或失败时，catalog、face
composition 和发行 UI 必须继续关闭。

## 本地容量刷新

2026-09-01 强制复跑通过：100k/10k 搜索 `searchKeysetP95Us=130238`、并发浏览 P95 `369 µs`、并发搜索
P95 `66276 µs`、peak Go heap `51,979,024` bytes、DB+WAL `157,274,112` bytes、零预算违例；100k×512
人脸聚类为 7.174 秒、Go memory sys `409,381,208` bytes。详见
[local capacity evidence](../../evidence/int-001/s2-local-capacity-refresh-darwin-arm64-2026-09-01.md)。

## 最终验证

本次 S2 closeout 实际执行并成功：

```text
make fmt
make arch-check
make generate-check
make lint
make test
make test-integration
make test-e2e
make spike-capacity
make test-face-capacity
/tmp/foliopath-coser-review/venv312/bin/python -m unittest \
  face_ground_truth_prepare_test.py face_functional_smoke_test.py \
  face_arcface_functional_smoke_test.py
git diff --check
```

Python 私有复核工具为 14 tests passed；Docker E2E 完成 production libvips-tag build、启动/health/auth/
migration/SIGTERM 与媒体只读 smoke。最终 release verifier 仍缺真实 S4 输入，因此没有被声明为通过。
