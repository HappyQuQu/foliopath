# 智能媒体最终供应链证据校验入口

- 日期：2026-08-31
- 状态：Accepted evidence tooling；**不等于供应链 Gate Go**
- 版本与阶段：`POST-MVP-5` revision 2 / S2A、S2B evidence closure
- Requirement：`NFR-INT-007～010`
- Owner：security、compliance、release、inference；实现 owner 为 `tests/release`
- 关联 Gate：[INT-S2A Backend Evidence Ready](../gates/POST-MVP-5/int-s2a-backend-evidence-ready.md)、
  [INT-S2B Backend Evidence Ready](../gates/POST-MVP-5/int-s2b-backend-evidence-ready.md)
- 影响不变量：不修改生产模型 catalog、runtime composition、API、SQLite 或媒体文件

## 问题与决定

此前 Gate 已要求最终审核模型、双架构完整 SBOM、签名 provenance、notices、漏洞处置及负责人签署，
但没有一个 POST-MVP-5 专用入口把这些外部产物绑定到同一 source commit 和 model package。新增
`make verify-intelligent-media-supply-chain`，只验收真实外部证据，不生成或伪造批准。

输入 manifest 必须：

- 绑定 `POST-MVP-5-r2`、目标 source commit、catalog 与最终 `.foliomodel` 的真实 SHA-256；
- 列出并取得 ONNX Runtime、SentencePiece、SigLIP 的版本、license、notices 与再分发批准引用；
- 同时提供 native Linux amd64/arm64 的 image digest、complete SBOM、provenance、签名验证结果和漏洞报告；
- 对 Critical/High 为零的结果直接失败关闭计数异常；若仍有阻断 finding，则必须同时提供真实 VEX 文件
  和 security approval reference；
- 提供 security、compliance、release、inference 四类不透明批准引用。

校验器逐个读取 manifest 引用的普通文件并复算 SHA-256，拒绝绝对路径、目录逃逸和任一路径组件中的
symlink。通过只表示 manifest 编码的证据一致、完整且可交给 Gate 复审；它不验证批准者身份，也不能把
当前 glibc finding、候选许可或未签署文件自动变成已接受结论。

## 验证

- `go test ./tests/release/intelligent_media_supplychain_evidence`
- 完整仓库检查按任务清单的统一验证面执行。
