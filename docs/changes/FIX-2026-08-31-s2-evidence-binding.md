# POST-MVP-5 S2 最终证据交叉绑定

- 日期：2026-08-31
- 状态：Accepted evidence tooling；**不等于任一 S2 Gate Go**
- 版本与阶段：`POST-MVP-5` revision 2 / S2 evidence closure
- Requirement：`FR-INT-001～020`、`NFR-INT-001～010`
- Owner：product、ML、QA、security、compliance、release、inference；实现 owner 为 `tests/release`
- 关联 Gate：[INT-S2A](../gates/POST-MVP-5/int-s2a-backend-evidence-ready.md)、
  [INT-S2B](../gates/POST-MVP-5/int-s2b-backend-evidence-ready.md)

## 问题与决定

质量、严格原生双架构和供应链 verifier 各自失败关闭，但三份独立 summary 若来自不同 source commit、
model package 或最终镜像，不能合并为同一个候选的 Gate 证据。新增
`make verify-intelligent-media-s2-evidence`，只消费三个已验证 summary，并强制：

- quality、native、supply-chain 的 source commit 与显式 `RELEASE_SHA` 相同；
- quality、两个原生架构和 supply-chain 的 final model package digest 相同；
- native summary 必须是 `finalModelEvidence=true` 的严格模式，不接受 baseline paired summary；
- native 与 supply-chain 的 amd64/arm64 final image digest 逐架构相同；
- quality Gate、全部 strict native checks 与 supply-chain result 均为 passed。

聚合 summary 记录三份输入文件的实际 SHA-256、governed dataset manifest hash、共同 model digest 和架构，
拒绝 symlink/非普通 summary 文件。该入口不重新解释原始数据或批准，也不能让缺失的外部证据变成通过。

质量评分器同时新增 `RELEASE_SHA` 绑定和可选原子 `SUMMARY_FILE` 输出；summary 明确携带 governed manifest
hash、`sha256:` model package digest 和 product/ML/QA 批准引用，供聚合器交叉核对。

## 验证

- `go test ./tests/release/intelligent_media_s2_evidence`
- `cd spikes/int001-ai && go test ./...`
- `make arch-check`
